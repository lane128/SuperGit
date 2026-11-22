package lobedesk

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"supergit/internal/appicon"
	"supergit/internal/gitrepo"
)

// toastOwner says which window shows a queued notice. A repository result
// belongs to the main window even while the reader is in Settings, and a
// settings result belongs to the panel; the other window leaves it queued
// instead of consuming it out of view.
type toastOwner int

const (
	toastMain toastOwner = iota
	toastSettings
)

type toastNotice struct {
	// root is captured when a repository result is queued; settings notices
	// have no repository scope, even if later adopted by the main window.
	root        string
	kind        string
	title       string
	description string
	action      string
	onAction    func()
	timeout     time.Duration
}

type undoDiscardState struct {
	path  string
	patch string
}

// busyLabel names an operation in the title bar while it runs. The value is
// an i18n key; an operation that passes no label keeps the general line,
// which is right for the sub-second work (staging, toggling a checkbox) that
// never waits long enough for the bar to say anything.
type busyLabel string

const (
	busyCommit     busyLabel = "busyCommit"
	busyPush       busyLabel = "busyPush"
	busyForcePush  busyLabel = "busyForcePush"
	busyPull       busyLabel = "busyPull"
	busyMerge      busyLabel = "busyMerge"
	busyRebase     busyLabel = "busyRebase"
	busyCherryPick busyLabel = "busyCherryPick"
	busyRevert     busyLabel = "busyRevert"
	busySwitch     busyLabel = "busySwitch"
	busyReset      busyLabel = "busyReset"
	busyCheckout   busyLabel = "busyCheckout"
	// The operations below are quick on a small repo and slow on a big one: a
	// stage-all over thousands of files, a stash that rewrites the whole
	// worktree, a submodule that fetches. They get a name so the title bar says
	// what is running instead of the general line.
	busyStageAll     busyLabel = "busyStageAll"
	busyStageTracked busyLabel = "busyStageTracked"
	busyUnstageAll   busyLabel = "busyUnstageAll"
	busyDiscard      busyLabel = "busyDiscard"
	busySaveStash    busyLabel = "busySaveStash"
	busyApplyStash   busyLabel = "busyApplyStash"
	busyPopStash     busyLabel = "busyPopStash"
	busyDropStash    busyLabel = "busyDropStash"
	busySubmodule    busyLabel = "busySubmodule"
	// The rest of the actions an operation can wait on: the network ones (a
	// tag push, a remote branch delete), the ones that rewrite a working tree
	// (a worktree added or removed, an undone commit, a deleted file), and
	// the ones that hand control to git's own continue/abort/skip, which run
	// the same hooks a commit does.
	busyDeleteFile         busyLabel = "busyDeleteFile"
	busyPushTag            busyLabel = "busyPushTag"
	busyDeleteRemoteBranch busyLabel = "busyDeleteRemoteBranch"
	busyCreateWorktree     busyLabel = "busyCreateWorktree"
	busyRemoveWorktree     busyLabel = "busyRemoveWorktree"
	busyUndoCommit         busyLabel = "busyUndoCommit"
	busyContinue           busyLabel = "busyContinue"
	busyAbort              busyLabel = "busyAbort"
	busySkip               busyLabel = "busySkip"
)

type app struct {
	historyCounts                                     *historyFileCounts
	sidebarHidden, ignoreDiffWhitespace, expandMerges bool
	hiddenBranches                                    map[string][]string // Immutable snapshots keyed by checkout root.
	comparison                                        revisionComparison
	preview                                           filePreviewSlot
	snapshotView                                      snapshotFileView
	fileHistoryRevision                               string
	window                                            *mygo.Window
	repositoryTabs                                    *repositoryTabs
	// openDirectoryDialog defaults to the native chooser; tests exercise the same dispatch boundary.
	openDirectoryDialog   func(mygo.OpenDialogOptions) ([]string, error)
	saveFileDialog        func(mygo.SaveDialogOptions) (string, error)
	repositoryChooserOpen bool
	// settingsWin is the settings panel's window, created on first open and
	// replaced after the reader closes it. A goroutine may invalidate it
	// while the main thread creates or replaces it, so it is atomic.
	settingsWin atomic.Pointer[mygo.Window]
	ctx         context.Context
	cancel      context.CancelFunc

	repo               *gitrepo.Repository
	repositoryAlias    string
	recentRepositories []string // successful checkout roots, newest first
	cloneForm          cloneFormState
	cloneOpeningRoot   string // identifies the read following a successful clone
	newRepoForm        newRepositoryFormState
	newRepoOpeningRoot string
	loading            bool
	activationRefresh  bool
	restoredCheckout   bool      // Restored tab awaiting its first verified snapshot.
	loadSince          time.Time // used to delay repository-read progress in the title bar
	busy               bool
	// opCancel stops the operation in flight, if there is one. A fetch on a
	// hanging network has nothing else to end it before the three-minute
	// timeout, so the title bar's progress line carries a cancel button.
	opCancel context.CancelFunc
	// progressSince is when the running operation started. The title bar
	// waits a moment before saying anything, so staging one file does not
	// flash a spinner and a cancel button for a few dozen milliseconds.
	progressSince time.Time
	// busyKey names the running operation for the title bar. Empty means the
	// caller did not name it, and the bar shows its general line instead.
	busyKey string
	// fetching is true while a fetch runs, automatic or manual, so the title
	// bar can show progress without claiming other busy work is a fetch.
	fetching bool
	loadErr  string

	status gitrepo.Status
	// headMessage is the complete message of the commit HEAD points to, read on every
	// reload. The commit list can be scoped to another ref, so the amend box
	// cannot take its prefill from the first row.
	headMessage       string
	branches          []gitrepo.Branch
	remoteBranches    []gitrepo.Branch
	remoteBranchesErr string
	commits           []gitrepo.Commit
	remotes           []gitrepo.Remote
	stashes           []gitrepo.Stash
	tags              []gitrepo.Tag
	worktrees         []gitrepo.Worktree
	submodules        []gitrepo.Submodule
	worktreeForm      worktreeFormState
	addRemoteForm     addRemoteFormState
	worktreeFormSeq   uint64       // changes when a new form replaces the prior lifecycle
	runOnMain         func(func()) // test seam; nil uses mygo.RunOnMain
	operation         gitrepo.Operation
	historyErr        string
	branchesErr       string
	stashesErr        string
	tagsErr           string
	worktreesErr      string
	submodulesErr     string

	selectedStash            string
	stashTrackedCount        int
	stashAdded, stashDeleted int
	stashTimeline            stashTimelineCache

	page         string
	locale       locale
	themeMode    string
	uiFont       string
	uiFontOpen   bool
	codeFont     string
	codeFontOpen bool
	codeFontSize float32
	splitDiff    bool
	// palette is the key of the color scheme in appThemes; the theme mode
	// picks its light or dark half.
	palette string
	// dockIcon remembers the drawing the Dock shows, so the view can ask for
	// one every frame and the system is only touched when the appearance
	// changes.
	dockIcon *appicon.Switcher

	selectedPath       string
	selectedStaged     bool
	workspaceSelection workspaceFileSelection
	// filesCollapsed holds the workspace sections the reader folded. A long
	// staged list can be put aside while the waiting edits are read. It is a
	// per-repository view state: reloads keep it, opening another repository
	// clears it.
	filesCollapsed map[string]bool
	// filesFoldSel remembers what folding a section did to the file
	// selection, so opening it again can put the selection back.
	filesFoldSel map[string]filesFoldSel
	// filesFolded is the settings file's per-repository fold list, newest
	// first. It is loaded once and kept current by rememberFilesFolds, so
	// opening another repository finds its own sections, not this one's.
	filesFolded     []foldedRepo
	conflictEditors map[conflictEditorKey]*conflictEditorState
	diff            gitrepo.Diff
	diffCache       worktreeDiffCache
	diffFind        *diffFindState
	diffFindPane    ui.Handle
	localWatch      localRepositoryWatch
	diffLoading     bool
	diffErr         string
	lineSel         map[int]bool
	lineAnchor      int
	lineCursor      int
	revealLine      bool
	linePressIndex  int
	lineDragHeld    bool
	lineDragging    bool
	lineDragMoved   bool
	undoDiscard     undoDiscardState
	// The diff's scroll position and its box from the last layout, read
	// while a line drag holds near an edge to keep scrolling.
	diffScroll       ui.ScrollState
	diffHoverSeenX   float32
	diffViewportRect ui.Rect
	// The details panes keep their scroll positions apart: the commit's
	// Summary, its files and its patch. The diff's box is read while a line
	// drag holds near its edge.
	commitInfoScroll    ui.ScrollState
	commitFilesScroll   ui.ScrollState
	commitFilesViewport ui.Rect
	commitDiffScroll    ui.ScrollState
	commitDiffViewport  ui.Rect

	// A pane can be hidden from the settings page; hiding the files and the
	// patch leaves the Summary with the whole width.
	detailsInfoHidden    bool
	detailsChangesHidden bool
	detailsDiffHidden    bool
	// frameSeq counts the main window's builds. historyPage stores the rows it
	// built for the frame, and historyRows hands those to the list, the
	// selection and the details pane instead of filtering the history again.
	frameSeq          uint64
	historyRowsCache  []gitrepo.Commit
	historyRowsFrame  uint64
	historyRowsSet    bool
	historyMergeFolds historyMergeFolds
	// historyGraph is the lane layout the history list drew this frame. The
	// details pane reads it so a ref badge wears the color of the line the
	// graph actually gave that ref, not just its hash slot.
	historyGraph       *commitGraph
	commitGraphCache   commitGraphCache
	filteredHistory    filteredHistoryCache
	reachableHistory   reachableHistoryCache
	historyGraphScroll ui.ScrollState
	// historyScroll keeps the commit list where the reader left it. The page
	// is torn down on every switch, and a scroll area without a stored state
	// starts over at the top when it comes back.
	historyScroll    ui.ScrollState
	historyRowExtent float32 // Row height used when the current scroll offset was laid out.
	// The pane dividers: history's list and details share a row, as do the
	// stash page's list and details and the branch page's. Each keeps its own
	// share and drag state, and a zero share means the page's default still
	// applies.
	historySplit splitDrag
	sidebarSplit splitDrag
	filesSplit   splitDrag
	stashSplit   splitDrag
	branchSplit  splitDrag
	// historyDetailsHidden collapses the history page's details, leaving the
	// commit list the full width. A toggle in the page header brings it back.
	historyDetailsHidden bool
	// Tab identity includes repository, SHA and the file's Index/worktree version.
	detailContexts              detailContextsState
	detailTabStrip              ui.ScrollState
	detailReveal                bool
	commitMessageFocusRoot      string
	commitMessageFocusSeq       uint64
	detailFocusPending          bool
	detailFocusContext          detailContextKey
	detailFocusSeq              uint64
	detailFocusTarget           detailTabKey
	detailFileFocus             *detailFileKey
	detailFileFocusContext      detailContextKey
	detailFileRemembered        *detailFileKey
	detailFileRememberedContext detailContextKey
	overviewEntries             []worktreeOverviewEntry
	overviewSeq                 uint64
	overviewRoot                string
	overviewCancel              context.CancelFunc
	overviewQueue               *overviewReadQueue
	overviewLineTarget          *overviewPatchTarget
	// worktreeSelected makes the list's working-tree row the selection. The
	// files and diff panes then read the working tree instead of a commit.
	worktreeSelected bool
	// The history list's display preferences: line thickness and node scale
	// for the graph, and whether the second line shows the author and the
	// commit time. Zero values mean the standard style; hidden columns are
	// stored as "hidden" so a fresh app shows them.
	historyLine float32
	historyNode float32
	hideAuthor  bool
	hideTime    bool
	// The other page lists keep their place the same way: each page unmounts
	// on a switch, so the position has to live in the app.
	branchesScroll ui.ScrollState
	// branchDetailScroll keeps the branch page's details where the reader left
	// them; it resets when the selected branch changes.
	branchDetailScroll ui.ScrollState
	workspacesScroll   ui.ScrollState
	settingsScroll     ui.ScrollState
	settingsSection    string
	settingsPositions  map[string]detailScrollPosition
	// sidebarScroll keeps the Locations list where the reader left it, and
	// sidebarOpen the groups they folded. A group never touched keeps its
	// default: branches open, the rest folded.
	sidebarScroll ui.ScrollState
	sidebarOpen   map[string]bool
	// branchesScrollQuery is the search the list was last laid out with. A new
	// query lists different rows, so it starts from the top.
	branchesScrollQuery string

	commitText     string
	amend          bool
	commitIdentity commitIdentityState
	// An ordinary saved draft waits for a successful snapshot outside a Git operation.
	unrestoredDraft string
	// pendingMessage is the message seeded from Git during a merge,
	// cherry-pick or revert. Only text that still matches it may be replaced
	// when the page reloads, so whatever the user typed survives.
	pendingMessage string
	// The commit diff's own line selection. Commits are read-only, so it
	// only chooses the rows a copy applies to.
	commitSel        map[int]bool
	commitAnchor     int
	commitCursor     int
	commitReveal     bool
	commitPressIndex int
	commitPressSide  patchSide
	commitDragHeld   bool
	commitDragging   bool
	commitDragMoved  bool

	selectedSHA           string
	refreshSelectedCommit gitrepo.Commit // Metadata retained while a replacement history is incomplete.
	commitFiles           []gitrepo.ChangedFile
	// commitNoBlame lists the selected commit's paths that git blame cannot
	// read: deleted files and submodule pointers. It is built once per load,
	// so a menu does not walk the file list for every diff row.
	commitNoBlame map[string]bool
	// commitNoBlameSHA is the commit the map was built for. A selection that
	// changed before the new files landed must not read the old commit's map.
	commitNoBlameSHA   string
	commitFilesLoaded  bool
	commitFilesLoading bool
	commitFilesErr     string
	commitPath         string
	commitMessage      string
	commitMessageErr   string
	// commitSig is Git's verdict on the selected commit's signature, fetched
	// apart from the files and the message because verifying a signature can
	// take a moment and must not hold up the rest of the details.
	commitSig            gitrepo.Signature
	commitDiff           gitrepo.Diff
	commitDiffLoading    bool
	commitDiffErr        string
	commitDiffByFile     bool // The combined patch exceeds the preview budget.
	commitFileTooLarge   bool
	commitFileReadCancel context.CancelFunc
	// commitSections split the commit's stacked diff into one section per
	// file; commitCollapsed records the sections the reader closed. The diff
	// holds every file's lines, so the selection and copy work across the
	// whole commit.
	commitSections  []commitDiffSection
	commitCollapsed map[string]bool
	// commitCollapsedSel keeps the rows a closed section took out of the
	// selection, so opening it again puts the reader's selection back. The
	// rows are only ever dropped for the time they are hidden; a reload of
	// the patch starts a new diff and forgets them.
	commitCollapsedSel map[string]commitHiddenSel

	historyQuery string
	// historyTips pins the traversal the loaded commits belong to, so paging
	// past the first page keeps offsets stable when refs move.
	historyTips        []string
	historyMore        bool
	historyLoadingMore bool
	historyPageErr     string
	historyReadCtx     context.Context
	historyStream      *gitrepo.HistoryStream
	historyReadCancel  context.CancelFunc
	historyPaused      bool
	historyStashExtent float32
	historyViewport    ui.Rect
	// pendingJumpSHA is the commit a parent link asked for that is not in the
	// loaded window yet. Paging continues until it appears.
	historyNavigation     historyNavigationState
	messageMenuGeneration uint64
	messageCommitRequest  *messageCommitRequest
	pendingJumpSHA        string
	historyRef            historyRef
	historyRefOpen        bool
	historyRefPalette     historyRefPaletteState
	historyRefQuery       string
	// The ref selector keeps its search field focused, so Up/Down move a
	// highlight of its own; cursorQuery and cursorPlaced place it when the
	// popup opens and when the query changes.
	historyRefCursor       int
	historyRefCursorQuery  string
	historyRefCursorPlaced bool
	// historyRefReveal asks the next frame to scroll the highlight into view.
	// Scrolling every frame would fight the reader's own wheel.
	historyRefReveal bool
	// historyPinned is the tip the walk was restarted from when a ref filter
	// names something outside the loaded window; "" walks every ref.
	historyPinned string
	// historyReload counts ref-filter reloads, so a slower answer for an
	// earlier filter cannot replace the list of a later one.
	historyReload int
	// historyReloading says a walk from a new tip is still out. The rows in
	// a.commits belong to the previous filter until it answers, so the list
	// shows a placeholder instead of them.
	historyReloading   bool
	branchQuery        string
	branchSel          string
	revealBranch       bool
	worktreeSel        string
	fileFilter         string
	fileCommits        []gitrepo.Commit
	fileHistoryErr     string
	fileHistoryLoading bool
	fileHistorySeq     uint64
	fileHistoryTips    []string
	fileHistoryMore    bool
	fileHistoryCtx     context.Context
	fileHistoryCancel  context.CancelFunc
	fileHistoryPaused  bool
	blameRoot          string
	blameOpen          bool
	blamePath          string
	// blameRev is the commit whose version of blamePath is blamed; empty means
	// the worktree. A file opened from a commit's patch must not read the
	// worktree, which may hold a different file by now.
	blameRev        string
	blame           []gitrepo.BlameLine
	blameErr        string
	blameLoading    bool
	blameSeq        uint64
	blameIgnoreWS   bool
	revealSelection bool
	paletteOpen     bool

	revealGraphSelection bool
	// showGraph draws the commit lane graph on the history page.
	showGraph bool
	// autoFetch fetches every remote after a repository is opened, the way
	// Sublime Merge's automatic fetch keeps ahead/behind fresh. autoFetchPrune
	// also drops remote-tracking branches Git no longer has.
	autoFetch      bool
	autoFetchPrune bool
	autoFetchEvery int // minutes between background fetches; 0 = only on open
	// lastAutoFetch is when the last fetch started, automatic or manual, so
	// the periodic check does not fetch again before the interval has passed
	// and a manual fetch restarts the clock.
	lastAutoFetch                   time.Time
	stashSaveOpen                   bool
	syncOpen                        bool
	syncCommits                     *syncCommitPage
	confirm                         confirmState
	prompt                          promptState
	reset                           resetState
	checkoutBranch                  checkoutBranchState
	commandPalette                  commandPaletteState
	commandPaletteGeneration        uint64
	quickCheckout                   quickCheckoutState
	quickCheckoutGeneration         uint64
	childNavigation                 childNavigationState
	childNavigationGeneration       uint64
	quickRepositorySwitch           quickRepositorySwitchState
	quickRepositorySwitchGeneration uint64
	// Consumed synchronously by openRepository for this switch attempt's error context.
	repositorySwitchTarget string

	// notices holds a queue per window: a notice waits for the window that
	// shows it, and a result for the other window does not overwrite it.
	notices       [2][]*toastNotice
	noticeDetails [2]noticeDetailsState
	// Measured each main-window frame, so overlays leave branch actions clear.
	mainToastBottomInset float32
	// shownNoticeRoot scopes the main UI engine's already-flushed toasts.
	shownNoticeRoot string

	loadSeq           uint64
	discardPrepareSeq uint64
	diffSeq           uint64
	commitSeq         uint64
}

func Run(initialPath string) error {
	defaults := defaultSettings()
	a := &app{
		page:             "history",
		locale:           locale(defaults.Language),
		themeMode:        defaults.Theme,
		palette:          defaults.Palette,
		uiFont:           uiFontKeyOf(defaults.UIFont),
		codeFont:         codeFontKeyOf(defaults.CodeFont),
		dockIcon:         appicon.New(func(png []byte) error { return mygo.App.Dock.SetIcon(png) }),
		lineAnchor:       -1,
		lineCursor:       -1,
		linePressIndex:   -1,
		commitAnchor:     -1,
		commitCursor:     -1,
		commitPressIndex: -1,
		showGraph:        defaults.ShowGraph != nil && *defaults.ShowGraph,
	}
	a.applySettings(loadSettings())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.ctx, a.cancel = context.WithCancel(ctx)
	tabs := newRepositoryTabs(ctx, a)
	offOpen := mygo.App.OnOpenFile(tabs.queueRepositoryFolder)
	defer offOpen()

	mygo.App.WhenReady(func() {
		go a.watchAutoFetch()
		// MyGo only accepts a locale the app declares among its languages;
		// without this, SetLocale("zh-Hans") falls back to English.
		mygo.App.SetLanguages("zh-Hans", "en")
		a.syncLocale()
		width, height := windowSize()
		a.window = mygo.NewWindow(mygo.WindowOptions{
			Title:                "SuperGit",
			Width:                width,
			Height:               height,
			MinWidth:             minWindowWidth,
			MinHeight:            minWindowHeight,
			TitleBarStyle:        mygo.TitleBarHidden,
			TrafficLightPosition: &mygo.Point{X: 12, Y: 12},
			BackgroundColor:      "light-dark(#f7f8fa, #09090b)",
			Content:              ui.View(tabs.view),
		})
		tabs.installRepositoryMenu()
		a.startLocalWatching()
		tabs.startSession(repositorySessionPath(), initialPath)
	})
	return mygo.App.Run()
}

func (a *app) invalidate() {
	if a.window != nil {
		a.window.Invalidate()
	}
	if w := a.settingsWindowSlot().Load(); w != nil && !w.IsDestroyed() {
		w.Invalidate()
	}
}

func (a *app) queue(kind, title, description string) {
	a.queueFor(toastMain, kind, title, description)
}

// queueFor queues a notice the given window shows.
func (a *app) queueFor(owner toastOwner, kind, title, description string) {
	a.pushNotice(owner, &toastNotice{kind: kind, title: title, description: description})
}

// pushNotice puts a notice at the end of its window's queue.
func (a *app) pushNotice(owner toastOwner, n *toastNotice) {
	if owner == toastMain && a.repo != nil {
		n.root = a.repo.Root
	}
	a.notices[owner] = append(a.notices[owner], n)
}

// takeNotice removes and returns the first notice queued for a window.
func (a *app) takeNotice(owner toastOwner) *toastNotice {
	queue := a.notices[owner]
	if len(queue) == 0 {
		return nil
	}
	n := queue[0]
	a.notices[owner] = queue[1:]
	return n
}

// adoptOrphanedNotice gives the main window the notices whose settings window
// closed before showing them, so a failed save is not lost with the panel.
func (a *app) adoptOrphanedNotice() {
	if len(a.notices[toastSettings]) == 0 {
		return
	}
	if w := a.settingsWindowSlot().Load(); w == nil || w.IsDestroyed() {
		a.notices[toastMain] = append(a.notices[toastMain], a.notices[toastSettings]...)
		a.notices[toastSettings] = nil
	}
}

func (a *app) flushNotice(c *ui.Context, owner toastOwner) {
	root := ""
	if a.repo != nil {
		root = a.repo.Root
	}
	if owner == toastMain {
		sessionChanged := false
		if tabs := a.repositoryTabs; tabs != nil {
			sessionChanged = tabs.shownNoticeSession != a
			tabs.shownNoticeSession = a
		}
		if sessionChanged || a.shownNoticeRoot != root {
			// The engine owns visible toasts for the whole window.
			c.CloseToast("")
			a.shownNoticeRoot = root
			c.After(0)
		}
	}
	n := a.takeNotice(owner)
	for n != nil && n.root != "" && n.root != root {
		n = a.takeNotice(owner)
	}
	if n == nil {
		return
	}
	typ := feedback.ToastInfo
	switch n.kind {
	case "success":
		typ = feedback.ToastSuccess
	case "error":
		typ = feedback.ToastError
	case "warning":
		typ = feedback.ToastWarning
	}
	timeout := n.timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	// Keep actionable failures available until the user dismisses them or
	// follows the recovery/details action. Ordinary and undo timers stay intact.
	if n.kind == "error" && (needsNoticeDetails(n) || n.onAction != nil) {
		timeout = -1
	}
	title, description, actionName := n.title, n.description, n.action
	onAction := n.onAction
	if needsNoticeDetails(n) {
		if longNoticeTitle(n) {
			title = a.t("errorDetailsTitle")
		}
		description = a.t("errorDetailsHint")
		actionName = a.t("viewErrorDetails")
		onAction = func() {
			a.openNoticeDetails(owner, n)
		}
	}
	if onAction != nil && n.root != "" {
		action, actionRoot := onAction, n.root
		onAction = func() {
			if a.activeRepositorySession() == a && !a.localWatch.closed && !a.activationRefresh && !a.tabSnapshotUnavailable() && a.repo != nil && a.repo.Root == actionRoot {
				action()
			}
		}
	}
	feedback.ShowToast(c, feedback.ToastProps{
		Title:       title,
		Description: description,
		Type:        typ,
		Timeout:     timeout,
		Action:      actionName,
		OnAction:    onAction,
	})
	// A toast added while the view builds cannot ask for the frame that
	// draws it (requestFrame ignores in-frame calls), and nothing else may
	// invalidate after this build. Ask for one frame more so the toast
	// host picks it up.
	c.After(0)
}

// loadFailureTitle names a failed read by what the reader was doing: a first
// open has nothing behind it, a reload leaves the old contents, and a switch
// leaves the previous repository on screen.
func (a *app) loadFailureTitle(firstOpen, switching bool) string {
	switch {
	case firstOpen:
		return a.t("openFailed")
	case switching:
		return a.t("switchRepoFailed")
	default:
		return a.t("reloadFailed")
	}
}

func (a *app) openRepository(path string, reset bool) {
	if reset && a.repositoryTabs != nil {
		a.repositoryTabs.open(a, path)
		return
	}
	a.readRepository(path, reset)
}

func (a *app) readRepository(path string, reset bool) {
	a.readRepositoryAtRoot(path, reset, "")
}

func (a *app) readRepositoryAtRoot(path string, reset bool, previousRoot string) {
	if a.busy || strings.TrimSpace(path) == "" {
		return
	}
	restoring := a.restoredCheckout
	reset = reset || restoring
	switchTarget := a.repositorySwitchTarget
	a.repositorySwitchTarget = ""
	if switchTarget != path {
		switchTarget = ""
	}
	if previousRoot == "" {
		// Picker entries are saved checkout roots. A removed nested checkout
		// must not silently open the repository that now contains its path.
		previousRoot = switchTarget
	}
	a.cancelLocalProbe()
	a.resetHistoryTraversal()
	a.resetFileHistory()
	// Cancelling for refresh interrupts known partial traversals. Keep that
	// fact visible if opening/status fails before replacement reads can start.
	a.historyPaused = a.historyMore
	a.fileHistoryPaused = a.fileFilter != "" && (a.fileHistoryTips == nil || a.fileHistoryMore)
	a.loadSeq++
	seq := a.loadSeq
	// This read replaces whatever the last one was doing. A history walk still
	// out belongs to the superseded read: its callback is dropped as stale, so
	// clearing its flags here is the only chance to do it. Left set, they keep
	// the list on "Rereading history…" with dead arrows until the reader
	// changes refs again.
	a.historyLoadingMore = false
	a.historyReloading = false
	// A jump the dropped walk was going to make has no walk left to make it.
	a.pendingJumpSHA = ""
	a.loading = true
	a.loadSince = time.Now()
	// loadErr stays up while the read runs: the banner already on screen is
	// the feedback for the retry, and its button turns into "Retrying…". It is
	// cleared when the read succeeds, or replaced by the new failure.
	a.invalidate()
	if reset {
		a.workspaceSelection = workspaceFileSelection{}
		// A different repository lists a different history; the position kept
		// for the previous one means nothing here. The other page lists start
		// over too.
		a.historyScroll.Y = 0
		a.branchesScroll.Y = 0
		a.historyStashExtent = 0
		a.workspacesScroll.Y = 0
	}
	// First paint stays bounded; automatic pages restore the full traversal.
	historyLimit := 200
	// The first page belongs to the repository refresh, like status and refs.
	// Scope changes may cancel pagination while this snapshot is still loading.
	// Its loadSeq guard already rejects superseded repository snapshots.
	historyCtx := a.ctx
	hiddenBranches := a.hiddenBranches
	// A reload keeps a ref filter's walk pinned to that ref instead of
	// swapping the list back to every ref behind the filter.
	pinned := ""
	if !reset {
		pinned = a.historyPinned
	}
	// Distinguishes the first open's failure (nothing to read yet) from a
	// reload's (the old contents are still on screen, and stale).
	hadRepo := a.repo != nil
	clonedPath := ""
	if reset && path == a.cloneOpeningRoot {
		clonedPath = path
	}
	a.cloneOpeningRoot = ""
	createdPath := ""
	if reset && path == a.newRepoOpeningRoot {
		createdPath = path
	}
	a.newRepoOpeningRoot = ""
	var watchEntries []gitrepo.Entry
	aliasRoot := ""
	if a.repositoryTabs != nil && a.repo != nil {
		aliasRoot = a.repo.Root
	}
	// Tab identities are canonical roots obtained from Git at open time.
	if (!reset || restoring) && a.repositoryTabs != nil && a.repo != nil {
		previousRoot = a.repo.Root
	}
	if !reset {
		watchEntries = append(watchEntries, a.status.Entries...)
	}

	go func() {
		repo, openErr := gitrepo.Open(a.ctx, path)
		if openErr == nil && previousRoot != "" && repo.Root != previousRoot {
			openErr = fmt.Errorf("checkout root changed from %q to %q", previousRoot, repo.Root)
		}
		var status gitrepo.Status
		var statusText string
		var localStamp string
		var headMessage string
		var identity commitIdentityState
		var history gitrepo.HistoryPage
		var branches []gitrepo.Branch
		var remoteBranches []gitrepo.Branch
		var remotes []gitrepo.Remote
		var stashes []gitrepo.Stash
		var tags []gitrepo.Tag
		var worktrees []gitrepo.Worktree
		var submodules []gitrepo.Submodule
		var operation gitrepo.Operation
		var operationMessage string
		var statusErr, historyErr, branchesErr, remoteBranchesErr, stashesErr, tagsErr, worktreesErr, submodulesErr error
		if openErr == nil {
			var reads sync.WaitGroup
			// The commit list does not depend on the working tree, so it starts
			// beside the status read rather than after it. On a repository whose
			// history walk is the slow part, that walk now overlaps the status
			// scan instead of adding to it.
			reads.Go(func() {
				if pinned != "" {
					history, historyErr = repo.HistoryPage(historyCtx, []string{pinned}, 0, historyLimit)
				} else {
					history, historyErr = repo.ReadVisibleHistory(historyCtx, historyLimit, hiddenBranches[repo.Root])
				}
			})
			// The displayed rows and the watcher's baseline are read from one
			// porcelain status. Reading it twice made a large open pay for the
			// slowest command twice.
			status, statusText, statusErr = repo.StatusWithText(a.ctx)
			if statusErr == nil {
				// Establish the baseline from the same observation the window
				// shows. A save after it is newer than the baseline and wakes
				// the watcher; a save before it is already on screen.
				localStamp, _ = readLocalWatchStampWithStatus(a.ctx, repo, watchEntries, statusText)
				// The remaining reads are independent of each other: run them
				// together so opening a large repository waits for the slowest
				// one instead of their sum.
				reads.Go(func() {
					// Read apart from the page: the amend box acts on HEAD even
					// when the list is showing another branch.
					if status.Head != "" {
						headMessage, _ = repo.CommitMessage(a.ctx, status.Head)
						headMessage = strings.TrimSuffix(headMessage, "\n")
					}
				})
				reads.Go(func() { identity = readCommitIdentity(a.ctx, repo, status.Head != "") })
				reads.Go(func() { branches, branchesErr = repo.Branches(a.ctx) })
				reads.Go(func() { remoteBranches, remoteBranchesErr = repo.RemoteBranches(a.ctx) })
				reads.Go(func() { remotes, _ = repo.Remotes(a.ctx) })
				reads.Go(func() { stashes, stashesErr = repo.Stashes(a.ctx) })
				reads.Go(func() { tags, tagsErr = repo.Tags(a.ctx) })
				reads.Go(func() { worktrees, worktreesErr = repo.Worktrees(a.ctx) })
				reads.Go(func() { submodules, submodulesErr = repo.Submodules(a.ctx) })
				reads.Go(func() {
					operation, _ = repo.Operation(a.ctx)
					if commitOperation(operation.Kind) {
						operationMessage, _ = repo.OperationMessage(a.ctx)
					}
				})
			}
			reads.Wait()
		}

		var alias string
		if openErr == nil && statusErr == nil && aliasRoot != repo.Root {
			alias = readRepositoryAlias(repo.Root)
		}
		a.hopToMain(func() {
			if seq != a.loadSeq {
				return
			}
			a.loading = false
			pendingTab := a.repositoryTabs != nil && a.repositoryTabs.opening == a
			if pendingTab {
				defer a.repositoryTabs.opened(a)
			}
			if tabs := a.repositoryTabs; tabs != nil && tabs.active == a && !hadRepo && openErr == nil {
				if existing := tabs.findCheckout(repo.Root); existing != nil {
					if tabs.canChange() {
						a.loadErr = ""
						tabs.activate(existing, false)
					}
					a.invalidate()
					return
				}
			}
			failTitle := a.loadFailureTitle(!hadRepo, reset)
			if openErr != nil || statusErr != nil {
				if restoring && a.repositoryTabs.skipFailedRestore(a) {
					return
				}
				err := openErr
				if err == nil {
					err = statusErr
				}
				detail := err.Error()
				var worktreeErr *gitrepo.WorktreeOpenError
				if errors.As(err, &worktreeErr) {
					detail = worktreeErr.Unwrap().Error()
				}
				var guidance string
				if clonedPath != "" {
					failTitle = a.t("clonedOpenFailed")
					detail = a.tf("clonedOpenFailedHint", clonedPath, detail)
				} else if createdPath != "" {
					failTitle = a.t("createdOpenFailed")
					detail = a.tf("createdOpenFailedHint", createdPath, detail)
				} else if pendingTab || (hadRepo && reset) {
					detail = a.tf("quickRepositorySwitchFailedHint", path, detail)
				} else if !hadRepo {
					guidance = a.tf("openRepositoryCheckFolder", path)
					detail = guidance + "\n\n" + detail
				}
				a.loadErr = detail
				if guidance != "" {
					a.loadErr = guidance
				}
				a.queue("error", failTitle, detail)
				a.invalidate()
				return
			}

			a.repo = repo
			if aliasRoot != repo.Root {
				a.repositoryAlias = alias
			}
			if tabs := a.repositoryTabs; tabs != nil && tabs.active == a && !hadRepo {
				// Replacing Welcome can widen the active tab without adding a tab.
				tabs.reveal = true
				tabs.forgetClosedRepository(repo.Root)
				tabs.sessionState.tabsChanged = true
			}
			a.restoredCheckout = false
			if tabs := a.repositoryTabs; restoring && tabs.active == a {
				tabs.acceptCheckoutSelection(a)
			}
			a.rememberRepository(repo.Root)
			if a.activationRefresh && !reset && (a.status.Head != status.Head || a.status.Branch != status.Branch) {
				// An Amend choice belongs to the displayed checkout and commit.
				// Keep the text so the reader can choose what to do with the draft.
				a.amend = false
			}
			a.status = status
			a.localWatch.stamp = localStamp
			if localStamp != "" {
				a.localWatch.lastError = ""
			}
			a.invalidateWorktreeOverview()
			a.headMessage = headMessage
			a.applyCommitIdentity(seq, repo.Root, identity)
			if !reset {
				a.refreshSelectedCommit, _ = a.selectedCommit()
			} else {
				a.refreshSelectedCommit = gitrepo.Commit{}
			}
			a.commits = history.Commits
			a.historyTips = history.Tips
			a.historyMore = history.HasMore
			a.branches = branches
			a.remoteBranches = remoteBranches
			a.remotes = remotes
			a.stashes = stashes
			a.tags = tags
			a.worktrees = worktrees
			a.submodules = submodules
			a.operation = operation
			a.loadErr = ""
			a.activationRefresh = false
			a.historyPaused = false
			a.historyErr = ""
			a.branchesErr = ""
			a.remoteBranchesErr = ""
			a.stashesErr = ""
			a.tagsErr = ""
			a.worktreesErr = ""
			a.submodulesErr = ""
			if historyErr != nil {
				a.historyErr = historyErr.Error()
			}
			// The filtered ref may have moved, or been deleted, while the
			// walk was pinned to its old tip.
			if !reset && a.historyRef.Kind != "" {
				name := a.historyRef.Name
				switch tip := a.historyRefTip(a.historyRef); {
				case tip == "":
					// The walk loaded the old tip's commits, which are no
					// longer a ref anyone can browse; go back to every ref.
					a.historyRef = historyRef{}
					a.historyPinned = ""
					a.reloadHistoryFromTip("")
					a.queue("info", a.t("refGoneTitle"), a.tf("refGoneDetail", name))
				case a.historyPinned != "" && tip != a.historyPinned:
					a.historyPinned = tip
					a.reloadHistoryFromTip(tip)
				}
			}
			if branchesErr != nil {
				a.branchesErr = branchesErr.Error()
			}
			if remoteBranchesErr != nil {
				a.remoteBranchesErr = remoteBranchesErr.Error()
			}
			if stashesErr != nil {
				a.stashesErr = stashesErr.Error()
				// A partial result can still carry rows, but the old selection
				// and its patch were read before this failure; they do not
				// describe a row the reader can trust now.
				a.clearStashDetail()
			}
			if tagsErr != nil {
				a.tagsErr = tagsErr.Error()
			}
			if worktreesErr != nil {
				a.worktreesErr = worktreesErr.Error()
			}
			if submodulesErr != nil {
				a.submodulesErr = submodulesErr.Error()
			}
			if a.window != nil && (a.repositoryTabs == nil || a.repositoryTabs.active == a) {
				a.window.SetTitle("SuperGit · " + filepath.Base(repo.Root))
			}
			if reset {
				// Descendants on other refs can appear above HEAD in the graph.
				// Open on the checked-out commit when it is in the loaded page.
				a.worktreeSelected = false
				a.detailContexts = detailContextsState{}
				a.detailTabStrip = ui.ScrollState{}
				a.selectedPath = ""
				a.selectedStaged = false
				// The folds are per repository: this one opens with what the
				// reader left folded last time, a new one starts unfolded.
				a.filesCollapsed = a.foldedFor(repo.Root)
				a.filesFoldSel = nil
				a.diff = gitrepo.Diff{}
				a.clearLineSelection()
				a.undoDiscard = undoDiscardState{}
				selected := ""
				if historyLists(a.commits, a.status.Head) {
					selected = a.status.Head
				}
				a.setSelectedCommit(selected)
				a.commitFiles = nil
				a.commitPath = ""
				a.resetCommitDiff()
				a.commitText = ""
				a.pendingMessage = ""
				a.amend = false
				a.historyQuery = ""
				a.historyRef = historyRef{}
				a.historyRefOpen = false
				a.historyRefQuery = ""
				a.historyPinned = ""
				a.historyLoadingMore = false
				a.historyReloading = false
				a.pendingJumpSHA = ""
				a.branchQuery = ""
				a.branchSel = ""
				a.revealBranch = false
				a.fileFilter = ""
				a.resetFileHistory()
				a.fileCommits = nil
				a.fileHistoryErr = ""
				// Drop results for another root while preserving window-owned
				// settings results and still-valid same-root actions, in order.
				for owner, queue := range a.notices {
					kept := queue[:0]
					for _, n := range queue {
						if n.root == "" || n.root == repo.Root {
							kept = append(kept, n)
						}
					}
					clear(queue[len(kept):])
					a.notices[owner] = kept
				}
				a.clearStashDetail()
				a.clearBlame()
			}
			a.applyOperationMessage(operationMessage)
			if a.unrestoredDraft != "" && a.operation.Kind == "" && a.commitText == "" {
				a.commitText, a.unrestoredDraft = a.unrestoredDraft, ""
			}
			if a.fileFilter != "" {
				// Refresh starts a new immutable file traversal. A filter chosen
				// while repository status was loading is the current scope too.
				a.resetFileHistory()
				a.fileCommits = nil
				a.fileHistoryTips = a.startFileHistoryTips()
				a.fileHistoryMore = true
				a.fileHistoryErr = ""
				a.loadFileHistory()
			}

			if restoring {
				// Keep the user's periodic fetch interval without fetching on restore.
				a.lastAutoFetch = time.Now()
				if tabs := a.repositoryTabs; tabs.sessionState.restoring && tabs.active == a {
					tabs.finishSessionRestore()
				}
			} else if reset && !pendingTab {
				a.fetchOnOpen()
			}
			a.reconcileSelection()
			a.reconcileWorktreeTabs(a.overviewKeys())
			if a.worktreeSelected {
				a.loadWorktreeOverview()
			}
			a.invalidate()
			if a.selectedPath != "" {
				a.loadDiff()
			}
			if a.selectedSHA != "" {
				a.loadCommitDetail(true)
			}
			if a.selectedStash != "" {
				a.loadStashDiff()
			}
			a.loadMoreHistory()
		})
	}()
}

func (a *app) reload() {
	if a.repo == nil {
		return
	}
	a.openRepository(a.repo.Root, false)
}

// applyOperationMessage keeps the commit field in step with the message Git
// prepared for a merge, cherry-pick or revert. A field the user has edited is
// never overwritten; one still holding the seeded text is updated when the
// pending message changes, and cleared when the operation ends.
func (a *app) applyOperationMessage(message string) {
	if !commitOperation(a.operation.Kind) {
		if a.pendingMessage != "" && a.commitText == a.pendingMessage {
			a.commitText = ""
		}
		a.pendingMessage = ""
		return
	}
	if message == "" || (a.commitText != "" && a.commitText != a.pendingMessage) {
		return
	}
	a.commitText = message
	a.pendingMessage = message
}

func (a *app) reconcileSelection() {
	entries := a.status.Entries
	if len(entries) == 0 {
		a.selectedPath = ""
		a.selectedStaged = false
		a.diff = gitrepo.Diff{}
	} else {
		found := false
		for _, entry := range entries {
			if entry.Path != a.selectedPath {
				continue
			}
			found = true
			staged := entry.Staged() && !entry.Conflict
			unstaged := entry.Unstaged()
			if a.selectedStaged && !staged {
				a.selectedStaged = false
			}
			if !a.selectedStaged && !unstaged {
				a.selectedStaged = true
			}
			break
		}
		if !found {
			a.selectedPath, a.selectedStaged = firstWorkspaceEntry(entries)
			a.diff = gitrepo.Diff{}
		}
		// A folded section takes its rows out of the list and out of the
		// keyboard order. A selection left inside one would keep the details
		// showing a file the reader cannot find, so it moves to the first row
		// still on screen. With every section folded there is no row to move to
		// and the selection stays.
		if refs := a.fileRefs(); len(refs) > 0 {
			visible := false
			for _, ref := range refs {
				if ref.path == a.selectedPath && ref.staged == a.selectedStaged {
					visible = true
					break
				}
			}
			if !visible {
				a.selectedPath, a.selectedStaged = refs[0].path, refs[0].staged
				a.diff = gitrepo.Diff{}
			}
		}
	}

	a.reconcileWorkspaceSelection()
	a.reconcileHistorySelection()
}

func (a *app) reconcileHistorySelection() {
	if a.selectedStash != "" {
		found := false
		for _, stash := range a.stashes {
			if stash.SHA == a.selectedStash {
				found = true
				break
			}
		}
		if !found {
			a.clearStashDetail()
		}
	}
	// A stash is the subject only when the reader picks it: the list puts the
	// working tree and the commits first, and auto-picking a stash on every
	// reload would describe a row that is not the one selected above it.

	if a.fileFilter == "" && a.selectedSHA != "" {
		found := false
		for _, commit := range a.commits {
			if commit.SHA == a.selectedSHA {
				found = true
				break
			}
		}
		if !found && !a.historyMore && a.historyErr == "" && a.historyPageErr == "" {
			a.setSelectedCommit("")
			a.commitFiles = nil
			a.commitPath = ""
			a.resetCommitDiff()
		}
	}
	// The working tree's row keeps the selection across a reload: auto-picking
	// the first commit would leave two rows marked as selected and hand the
	// arrows a commit the reader never chose.
	if a.fileFilter == "" && a.selectedSHA == "" && len(a.commits) > 0 && !a.worktreeSelected {
		a.setSelectedCommit(a.commits[0].SHA)
		a.commitFiles = nil
		a.commitPath = ""
		a.resetCommitDiff()
	}
	// A repository with no commits has no row but the working tree, and its
	// commit editor is where the first commit comes from.
	if a.fileFilter == "" && a.historyErr == "" && a.selectedSHA == "" && a.selectedStash == "" &&
		len(a.commits) == 0 && !a.worktreeSelected {
		a.worktreeSelected = true
	}
}

func (a *app) historySource() []gitrepo.Commit {
	if a.fileFilter != "" {
		return a.fileCommits
	}
	if a.historyRef.Kind != "" {
		// A ref outside the loaded window restarts the walk at its tip, so
		// the loaded commits are already that ref's history.
		if a.historyPinned != "" {
			return a.commits
		}
		return a.cachedReachableHistory(a.historyRefTip(a.historyRef))
	}
	return a.commits
}

func (a *app) hasSelectedIn(commits []gitrepo.Commit) bool {
	if a.selectedSHA == "" {
		return false
	}
	for _, commit := range commits {
		if commit.SHA == a.selectedSHA {
			return true
		}
	}
	return false
}

func firstWorkspaceEntry(entries []gitrepo.Entry) (string, bool) {
	state := &app{status: gitrepo.Status{Entries: entries}}
	for _, group := range state.workingFileGroups() {
		if len(group.Entries) > 0 {
			return group.Entries[0].Path, group.Staged
		}
	}
	return "", false
}

func (a *app) selectFile(path string, staged bool) {
	a.workspaceSelection = workspaceFileSelection{}
	a.previewWorkspaceFile(path, staged)
}

func (a *app) previewWorkspaceFile(path string, staged bool) {
	if a.worktreeSelected {
		a.openDetailFile(detailFileKey{Path: path, Staged: staged})
		return
	}
	// Workspace reconciliation also uses this entry while history is in front.
	// Preloading a working file must not open a tab in that historical commit.
	if path == "" {
		return
	}
	if a.selectedPath == path && a.selectedStaged == staged && (a.repo == nil || a.diff.Text != "" || a.diffLoading) {
		return
	}
	a.selectedPath, a.selectedStaged = path, staged
	a.diffScroll = ui.ScrollState{}
	a.diff = gitrepo.Diff{}
	a.diffErr = ""
	a.clearLineSelection()
	a.loadDiff()
	a.invalidate()
}

func (a *app) loadDiff() {
	if a.repo == nil || a.selectedPath == "" {
		return
	}
	a.diffSeq++
	seq := a.diffSeq
	repo, path, staged := a.repo, a.selectedPath, a.selectedStaged
	ignoreWhitespace := a.ignoreDiffWhitespace
	epoch := a.loadSeq
	a.diffLoading = true
	a.diffErr = ""
	go func() {
		diff, err := repo.DisplayDiff(a.ctx, path, staged, ignoreWhitespace)
		a.hopToMain(func() {
			if seq != a.diffSeq || a.loadSeq != epoch || a.repo == nil || a.repo.Root != repo.Root || a.selectedPath != path || a.selectedStaged != staged {
				return
			}
			a.diffLoading = false
			if err != nil {
				a.diffErr = err.Error()
			} else {
				a.diff = diff
				a.diffCache.put(repo.Root, epoch, detailFileKey{Path: path, Staged: staged}, diff)
				if a.overviewLineTarget == nil {
					a.clearLineSelection()
				}
			}
			a.invalidate()
		})
	}()
}

// setSelectedCommit moves the selection without loading anything. The paths
// that assign the SHA straight -- a reload, a reset, a file's history replacing
// the list -- come through here so the diff's line selection is dropped when
// the commit really changes: those row indexes belong to the previous diff, and
// Cmd+C would otherwise copy the same rows out of the new one.
func (a *app) setSelectedCommit(sha string) {
	if sha != a.selectedSHA {
		a.clearCommitSelection()
		// Message and signature belong to the previous commit; clear them before
		// rendering a different target while its detail read is still pending.
		a.commitMessage = ""
		a.commitMessageErr = ""
		a.commitSig = gitrepo.Signature{}
		// A different commit opens its own info, file list and diff at the top.
		// The rows kept their offsets, so a selection that moved on a reload
		// showed the new commit scrolled to where the old one was read.
		a.commitInfoScroll.Y = 0
		a.commitFilesScroll.Y = 0
		a.commitDiffScroll.Y = 0
	}
	a.selectedSHA = sha
	a.detailTabs()
}

// worktreeHasSelectedFile reports whether the selected path is still one of
// the working tree's files, so the working-tree row can keep it open.
func (a *app) worktreeHasSelectedFile() bool {
	if a.selectedPath == "" {
		return false
	}
	for _, entry := range a.status.Entries {
		if entry.Path == a.selectedPath {
			return true
		}
	}
	return false
}

// selectWorktree makes the working tree the page's selection. The files
// column lists what is uncommitted and the diff column shows it, with the
// commit box on the Summary tab.
func (a *app) selectWorktree() {
	// Explicit selection supersedes a message action even when the reader later
	// returns to the same source before a queued completion runs.
	a.messageCommitRequest = nil
	a.messageMenuGeneration++
	a.pendingJumpSHA = ""
	if a.selectedStash != "" {
		// The stash stops being the details' subject; selecting a commit or
		// the working tree leaves it in the list, unselected.
		a.clearStashDetail()
	}
	if a.worktreeSelected {
		return
	}
	a.saveDetailScroll()
	a.worktreeSelected = true
	a.setSelectedCommit("")
	a.commitFiles = nil
	a.commitFilesLoaded = false
	a.commitPath = ""
	a.commitMessage = ""
	a.commitMessageErr = ""
	// The Files column remembers where each subject's list stood; choosing the
	// working tree puts its own position back rather than the commit's.
	a.commitFilesScroll = ui.ScrollState{X: a.detailTabs().FilesScroll.X, Y: a.detailTabs().FilesScroll.Y}
	// The file the reader already had stays open, so choosing the working
	// tree changes what the panes describe, not which file the workspace
	// shows. Only when that file is gone does the first working file take
	// its place. The Summary tab stays in front: the commit box is why the
	// row was chosen.
	if !a.worktreeHasSelectedFile() {
		if path, staged := firstWorkspaceEntry(a.status.Entries); path != "" {
			a.selectedPath, a.selectedStaged = path, staged
			a.diff = gitrepo.Diff{}
			a.loadDiff()
		}
	}
	a.applyDetailTab(detailTabKey{})
	a.loadWorktreeOverview()
}

func (a *app) selectCommit(sha string) {
	if sha == "" {
		return
	}
	if a.selectedStash != "" {
		a.clearStashDetail()
	}
	a.messageCommitRequest = nil
	a.messageMenuGeneration++
	if !a.historyNavigation.selecting && (sha != a.selectedSHA || a.historyNavigation.pending != nil) {
		a.clearHistoryNavigation()
	}
	// A newer explicit selection supersedes a parent jump awaiting pagination.
	a.pendingJumpSHA = ""
	a.saveDetailScroll()
	a.worktreeSelected = false
	if a.selectedSHA == sha && a.commitFilesLoaded {
		return
	}
	a.selectedSHA = sha
	a.detailTabs()
	a.commitFiles = nil
	a.commitFilesLoaded = false
	a.commitPath = ""
	a.commitMessage = ""
	a.commitMessageErr = ""
	a.commitSig = gitrepo.Signature{}
	a.resetCommitDiff()
	a.commitDiffErr = ""
	a.commitInfoScroll.Y = 0
	// The Files column puts the historical list's own position back, as it
	// does when the working tree takes the details.
	a.commitFilesScroll = ui.ScrollState{X: a.detailTabs().FilesScroll.X, Y: a.detailTabs().FilesScroll.Y}
	a.commitDiffScroll.Y = 0
	a.clearCommitSelection()
	a.applyDetailTab(detailTabKey{})
	a.loadCommitDetail(false)
	a.invalidate()
}

func (a *app) loadCommitDetail(force bool) {
	if a.selectedStash != "" {
		a.loadStashDiff()
		return
	}
	if a.repo == nil || a.selectedSHA == "" {
		return
	}
	if !force && a.commitFilesLoaded {
		return
	}
	a.commitSeq++
	seq := a.commitSeq
	repo, sha := a.repo, a.selectedSHA
	root := repo.Root
	ignoreWhitespace := a.ignoreDiffWhitespace
	keepPath := a.commitPath
	if !force && keepPath != "" {
		keepPath = ""
	}
	filterPath := a.fileFilter
	if a.commitDiffByFile {
		a.resetCommitDiff()
	}
	a.commitFilesLoading = true
	a.commitFilesErr = ""
	a.commitDiffErr = ""
	a.commitDiffLoading = true
	go func() {
		// Verifying a signature can start a GPG or SSH process; it gets its own
		// goroutine so a slow keyring never delays the files and the diff.
		go func() {
			signature, err := repo.CommitSignature(a.ctx, sha)
			if err != nil {
				return
			}
			a.hopToMain(func() {
				if seq != a.commitSeq || a.selectedSHA != sha || a.repo == nil || a.repo.Root != root || a.worktreeSelected {
					return
				}
				a.commitSig = signature
				a.invalidate()
			})
		}()
		files, filesErr := repo.CommitFiles(a.ctx, sha)
		// Message failure stays local; valid file and patch data remain usable.
		message, messageErr := repo.CommitMessage(a.ctx, sha)
		path := ""
		if filesErr == nil {
			// In a file's history the commit's details should open on that
			// file, not on whichever file the commit happened to change
			// first.
			path = preferredCommitPath(files, keepPath, filterPath)
		}
		var diffs []gitrepo.FileDiff
		var diffErr error
		if filesErr == nil {
			// One command answers for every file, so the details can show the
			// whole commit's diff instead of waiting for a click per file.
			diffs, diffErr = repo.DisplayCommitDiff(a.ctx, sha, ignoreWhitespace)
		}
		a.hopToMain(func() {
			if seq != a.commitSeq || a.selectedSHA != sha || a.repo == nil || a.repo.Root != root || a.worktreeSelected {
				return
			}
			a.commitFilesLoading = false
			a.commitDiffLoading = false
			a.commitMessage = message
			a.commitMessageErr = ""
			if messageErr != nil {
				a.commitMessageErr = messageErr.Error()
			}
			if filesErr != nil {
				a.commitFilesErr = filesErr.Error()
				a.commitDiffErr = filesErr.Error()
				a.commitNoBlame = nil
				a.commitNoBlameSHA = ""
			} else {
				a.commitFiles = files
				a.commitNoBlame = commitNoBlamePaths(files)
				a.commitNoBlameSHA = sha
				a.commitFilesLoaded = true
				a.commitPath = path
				if errors.Is(diffErr, gitrepo.ErrDiffPreviewLimit) {
					a.loadLargeCommitTab()
				} else if diffErr != nil {
					a.commitDiffErr = diffErr.Error()
				} else {
					a.commitDiff, a.commitSections = buildCommitDiff(diffs)
					addSubmoduleNoBlame(a.commitNoBlame, a.commitSections)
					// A commit opened from a file's history should show that
					// file, not the first one git listed.
					if filterPath != "" && path != "" {
						a.scrollCommitDiffToFile(path)
					}
				}
			}
			a.invalidate()
		})
	}()
}

// selectCommitFile opens or activates a file tab.
func (a *app) selectCommitFile(path string) {
	if a.repo == nil || a.currentDetailContext().CommitSHA == "" || path == "" {
		return
	}
	if selection := a.historicalFileSelectionState(); selection != nil {
		selection.Files = map[string]bool{path: true}
		selection.Anchor = path
	}
	a.openDetailFile(detailFileKey{Path: path})
}

// beginOperation starts the deadline for a git command and marks the app
// busy. The returned cancel is what the title bar's cancel button reaches, so
// every operation that sets busy itself goes through here rather than building
// its own context.
func (a *app) beginOperation(label busyLabel) (context.Context, context.CancelFunc) {
	a.cancelLocalProbe()
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Minute)
	a.busy = true
	a.busyKey = string(label)
	a.opCancel = cancel
	a.progressSince = time.Now()
	// Nothing repaints on its own while the progress line waits out its
	// delay, so ask for one frame when the wait ends. A finished operation
	// makes it a no-op.
	time.AfterFunc(progressDelay, func() { mygo.RunOnMain(a.invalidate) })
	a.invalidate()
	return ctx, cancel
}

// finishOperation clears what beginOperation set, once the command is done
// whatever the outcome.
func (a *app) finishOperation() {
	a.busy = false
	a.busyKey = ""
	a.opCancel = nil
}

func (a *app) perform(label, success string, fn func(context.Context, *gitrepo.Repository) error) {
	a.performAs("", label, success, fn)
}

// performAs is perform with a name for the title bar. The name is what the
// reader waiting on a push or a rebase should see instead of the general
// "Working…", which says only that something is happening.
func (a *app) performAs(busy busyLabel, label, success string, fn func(context.Context, *gitrepo.Repository) error) {
	a.performDoneAs(busy, label, success, nil, fn)
}

// performDone is perform with a hook that runs on the UI thread once the work
// finishes, whatever the outcome. Callers use it for state that mirrors the
// operation, such as the title bar's fetch spinner.
func (a *app) performDone(label, success string, done func(error), fn func(context.Context, *gitrepo.Repository) error) {
	a.performDoneAs("", label, success, done, fn)
}

// performDoneAs is performDone with the operation's name for the title bar. It
// reports whether the work started: a caller that keeps state for a retry
// needs to know when nothing was launched.
func (a *app) performDoneAs(busy busyLabel, label, success string, done func(error), fn func(context.Context, *gitrepo.Repository) error) bool {
	if a.repo == nil || a.busy || a.loading || a.tabSnapshotUnavailable() {
		return false
	}
	repo := a.repo
	ctx, cancel := a.beginOperation(busy)
	go func() {
		defer cancel()
		err := fn(ctx, repo)
		// Capture the result before deferred cleanup cancels this context.
		// The UI continuation may run after the worker has already returned.
		cancelled := operationCancelled(ctx, err)
		a.hopToMain(func() {
			a.finishOperation()
			switch {
			case cancelled:
				// Stopping through the cancel button is not a failure, and
				// the git error underneath ("signal: killed") would read as
				// one.
				a.queue("info", a.t("operationCancelled"), "")
			case err != nil:
				// An empty label lets a completion handler own the failure in its form.
				if label != "" {
					a.queue("error", label, a.operationErrorDescription(err))
				}
			case success != "":
				a.queue("success", success, "")
			}
			if done != nil {
				// The error lets a caller restore what the operation would
				// have replaced: a failed worktree creation keeps the typed
				// path and branch in its form.
				done(err)
			}
			a.reload()
			a.invalidate()
		})
	}()
	return true
}

// hopToMain hands the operation's completion to the UI thread. The runOnMain
// seam lets a test take the continuation and run it itself: without an event
// loop the completion would run on the worker's goroutine, and asserting the
// restored form from the test would race with that write.
func (a *app) hopToMain(fn func()) {
	if a.runOnMain != nil {
		a.runOnMain(fn)
		return
	}
	mygo.RunOnMain(fn)
}

// operationCancelled reports whether the operation stopped because someone
// pressed cancel. The context's own error is the signal, not the error git
// returned: a killed process usually reports "signal: killed", which would
// otherwise be shown as a failure.
func operationCancelled(ctx context.Context, err error) bool {
	return err != nil && ctx.Err() == context.Canceled
}

// cancelOperation stops whatever git command is running. The work runs in its
// own context with a three-minute timeout, and a fetch against a network that
// has stopped answering has no other way out before that timeout.
func (a *app) cancelOperation() {
	if a.opCancel != nil {
		a.opCancel()
	}
}

func (a *app) toggleStage(entry gitrepo.Entry, staged bool) {
	target, ok := a.fileActionTarget(entry, staged)
	if !ok {
		return
	}
	a.runFileStage(target)
}

func (a *app) runFileStage(target fileActionTarget) {
	action := a.prepareFileStage(target)
	if action == nil {
		return
	}
	label := a.t("stageFile")
	if target.Staged {
		label = a.t("unstageFile")
	}
	a.perform(label, "", action)
}

func (a *app) stageAll() { a.runBatchStage(false) }

func (a *app) unstageAll() { a.runBatchStage(true) }

func (a *app) commit() {
	if !a.canCommit() {
		return
	}
	message, amend := a.commitText, a.amend
	success := a.t("commitSuccess")
	if amend {
		success = a.t("amendSuccess")
	}
	// Share the operation lifecycle: capture failure/cancellation before worker
	// cleanup and deliver completion on the UI thread. Keep the draft on errors.
	a.performDoneAs(busyCommit, a.t("commitFailed"), success, func(err error) {
		if err == nil {
			a.commitText = ""
			a.amend = false
		}
	}, func(ctx context.Context, repo *gitrepo.Repository) error {
		if amend {
			return repo.AmendCommit(ctx, message)
		}
		return repo.Commit(ctx, message)
	})
}

func (a *app) switchBranchAt(name, sha string) {
	if a.repo == nil || name == "" || name == a.status.Branch {
		return
	}
	a.performAs(busySwitch, a.t("switch"), a.t("switchSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.SwitchBranchChecked(ctx, name, sha)
	})
}

func (a *app) switchBranch(name string) {
	if a.repo == nil || name == "" || name == a.status.Branch {
		return
	}
	a.performAs(busySwitch, a.t("switch"), a.t("switchSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.SwitchBranch(ctx, name)
	})
}

// fetchOnOpen runs the automatic fetch the settings page enables. It stays
// silent on success, because an automatic background step should not toast on
// every repository switch; a failure still surfaces as a toast.
func (a *app) fetchOnOpen() {
	if !a.autoFetch || a.repo == nil || len(a.remotes) == 0 {
		return
	}
	a.startFetch(a.t("fetchFailed"), "", a.autoFetchPrune)
}

// startFetch fetches every remote and lights the title bar's progress label.
// Automatic and manual fetches share it, so both show the same feedback and
// both restart the interval: a manual fetch moments after opening should not
// be repeated by the timer. The attempt is recorded before it runs, so a
// failure waits a full interval instead of retrying every tick.
func (a *app) startFetch(label, success string, prune bool) {
	mode := gitrepo.FetchDefault
	if prune {
		mode = gitrepo.FetchWithPrune
	}
	a.fetchRemotes(a.remotes, mode, label, success)
}

func (a *app) fetchRemotes(targets []gitrepo.Remote, mode gitrepo.FetchMode, label, success string) {
	if a.repo == nil || a.busy || a.loading || a.tabSnapshotUnavailable() || len(targets) == 0 {
		return
	}
	remotes := append([]gitrepo.Remote(nil), targets...)
	a.lastAutoFetch = time.Now()
	a.fetching = true
	a.performDone(label, success, func(error) {
		a.fetching = false
	}, func(ctx context.Context, repo *gitrepo.Repository) error {
		for _, remote := range remotes {
			if err := repo.FetchRemote(ctx, remote.Name, mode); err != nil {
				return err
			}
		}
		return nil
	})
}

// watchAutoFetch posts a periodic check to the UI thread. The timer itself
// touches no app state: the closure that decides whether to fetch runs on the
// main thread, so the repository, busy flag and settings are read the same
// way as every other UI read.
func (a *app) watchAutoFetch() {
	ctx := a.ctx
	if a.repositoryTabs != nil {
		ctx = a.repositoryTabs.ctx
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mygo.RunOnMain(func() { a.activeRepositorySession().tickAutoFetch() })
		}
	}
}

// tickAutoFetch repeats the automatic fetch on the selected interval. It
// shares fetchOnOpen's guards, so a busy app or a repository without remotes
// simply waits for the next tick.
func (a *app) tickAutoFetch() {
	if a.localWatch.closed || a.repositoryChooserOpen || a.repositoryOpenPending() || a.hasPendingDialog(toastMain) {
		return
	}
	if !autoFetchDue(a.autoFetch, a.autoFetchEvery, a.lastAutoFetch, time.Now()) {
		return
	}
	a.fetchOnOpen()
}

// autoFetchDue is the periodic fetch's clock: enabled, an interval above zero,
// and at least that long since the last attempt. Keeping it separate makes the
// timing testable without running a timer.
func autoFetchDue(enabled bool, everyMinutes int, last, now time.Time) bool {
	if !enabled || everyMinutes <= 0 {
		return false
	}
	return now.Sub(last) >= time.Duration(everyMinutes)*time.Minute
}

func (a *app) fetchAll() {
	if len(a.remotes) == 0 {
		a.queue("warning", a.t("noRemote"), "")
		a.invalidate()
		return
	}
	a.startFetch(a.t("fetchFailed"), a.t("fetchSuccess"), false)
}

func (a *app) fetchPruneAll() {
	if len(a.remotes) == 0 {
		a.queue("warning", a.t("noRemote"), "")
		a.invalidate()
		return
	}
	a.startFetch(a.t("fetchPruneFailed"), a.t("fetchPruned"), true)
}

func (a *app) pullFF() {
	if a.status.Upstream == "" {
		a.queue("warning", a.t("noUpstream"), "")
		a.invalidate()
		return
	}
	a.performAs(busyPull, a.t("pullFailed"), a.t("pullSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.PullFF(ctx)
	})
}

// openResetToUpstream resolves the current branch's upstream first, then
// opens the same reset dialog a commit row uses, so the destructive modes and
// their confirmation stay in one place.
func (a *app) openResetToUpstream() {
	if a.repo == nil {
		return
	}
	if a.status.Upstream == "" {
		a.queue("warning", a.t("noUpstream"), "")
		a.invalidate()
		return
	}
	repo, upstream := a.repo, a.status.Upstream
	go func() {
		sha, err := repo.ResolveRef(a.ctx, upstream)
		mygo.RunOnMain(func() {
			if a.repo != repo {
				return
			}
			if err != nil {
				a.queue("error", a.t("resetUpstreamFailed"), err.Error())
				a.invalidate()
				return
			}
			subject := upstream
			for _, commit := range a.commits {
				if commit.SHA == sha {
					subject = commit.Subject
					break
				}
			}
			a.openResetDialog(gitrepo.Commit{SHA: sha, Subject: subject})
		})
	}()
}

// pushTarget names the remote and destination a push should use: the
// upstream when there is one, otherwise the first remote and the branch name.
func (a *app) pushTarget() (remote, destination string) {
	remote, destination = a.remotes[0].Name, a.status.Branch
	if upstream := a.status.Upstream; upstream != "" {
		if i := strings.IndexByte(upstream, '/'); i > 0 {
			remote, destination = upstream[:i], upstream[i+1:]
		}
	}
	return remote, destination
}

// branchPushTarget resolves where a branch's Push action sends it: its
// upstream when it tracks one, otherwise the first remote under its own name.
func (a *app) branchPushTarget(branch gitrepo.Branch) (remote, destination string) {
	if len(a.remotes) == 0 {
		return "", ""
	}
	remote, destination = a.remotes[0].Name, branch.Name
	if upstream := branch.Upstream; upstream != "" {
		if i := strings.IndexByte(upstream, '/'); i > 0 {
			remote, destination = upstream[:i], upstream[i+1:]
		}
	}
	return remote, destination
}

// confirmPushBranch publishes a branch chosen in the branch list, which may
// not be the checked-out one. The first push of a branch also records its
// upstream so later pushes need no guessing.
func (a *app) confirmPushBranch(branch gitrepo.Branch) {
	remote, destination := a.branchPushTarget(branch)
	if remote == "" {
		a.queue("warning", a.t("noRemote"), "")
		a.invalidate()
		return
	}
	var target gitrepo.Remote
	for _, candidate := range a.remotes {
		if candidate.Name == remote {
			target = candidate
			break
		}
	}
	if target.Name == "" {
		a.queue("warning", a.t("noRemote"), "")
		a.invalidate()
		return
	}
	setUpstream := branch.Upstream == ""
	a.askConfirm(a.t("pushDialogTitle"), a.tf("pushMessage", branch.Name, remote, destination, shortSHA(branch.SHA)), a.t("pushButton"),
		false, func() {
			a.performAs(busyPush, a.t("pushFailed"), a.t("pushSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.PushChecked(ctx, target, branch, destination, setUpstream)
			})
		})
}

func (a *app) confirmPush() {
	branch := a.status.Branch
	if branch == "" || len(a.remotes) == 0 {
		a.queue("warning", a.t("noRemote"), "")
		a.invalidate()
		return
	}
	a.confirmPushBranch(gitrepo.Branch{Name: branch, SHA: a.status.Head, Upstream: a.status.Upstream})
}

// confirmForcePush retains the displayed source and remote baseline. Later
// fetches must not broaden the lease granted by this confirmation.
func (a *app) confirmForcePush() {
	branch := a.status.Branch
	if branch == "" || len(a.remotes) == 0 {
		a.queue("warning", a.t("noRemote"), "")
		a.invalidate()
		return
	}
	remote, destination := a.pushTarget()
	var target gitrepo.Remote
	for _, candidate := range a.remotes {
		if candidate.Name == remote {
			target = candidate
			break
		}
	}
	if target.Name == "" || a.remoteBranchesErr != "" {
		a.queue("warning", a.t("forcePushTargetChanged"), a.remoteBranchesErr)
		a.invalidate()
		return
	}
	source := gitrepo.Branch{Name: branch, SHA: a.status.Head}
	lease := ""
	for _, ref := range a.remoteBranches {
		if ref.Name == remote+"/"+destination {
			lease = ref.SHA
			break
		}
	}
	leaseLabel := shortSHA(lease)
	if lease == "" {
		leaseLabel = a.t("forcePushRemoteAbsent")
	}
	a.askConfirm(a.t("forcePushTitle"), a.tf("forcePushMessage", branch, remote, destination, shortSHA(source.SHA), leaseLabel),
		a.t("forcePush"), true, func() {
			a.performAs(busyForcePush, a.t("forcePushFailed"), a.t("forcePushSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.PushForceChecked(ctx, target, source, destination, lease)
			})
		})
}

func (a *app) repositoryChooserBlocked() bool {
	return a.busy || a.loading || a.repositoryChooserOpen || a.repositoryOpenPending() || a.hasPendingDialog(toastMain)
}

func (a *app) chooseRepository() {
	if (a.window == nil && a.openDirectoryDialog == nil) || a.repositoryChooserBlocked() {
		return
	}
	a.repositoryChooserOpen = true
	defer func() { a.repositoryChooserOpen = false }()
	open := a.openDirectoryDialog
	if open == nil {
		open = mygo.Dialog.Open
	}
	paths, err := open(mygo.OpenDialogOptions{
		Parent:    a.window,
		Title:     a.t("openRepo"),
		Directory: true,
	})
	if err != nil {
		a.queue("error", a.t("openFailed"), err.Error())
		a.invalidate()
		return
	}
	if len(paths) == 0 || a.busy || a.loading || a.hasPendingDialog(toastMain) {
		return
	}
	a.repositoryChooserOpen = false
	a.openRepository(paths[0], true)
}

func (a *app) setLocale(l locale) {
	if a.locale == l {
		return
	}
	a.locale = l
	a.syncLocale()
	if tabs := a.repositoryTabs; tabs != nil {
		tabs.installRepositoryMenu()
	}
	a.retitleSettingsWindow()
	a.saveSettingsFor(toastSettings)
	a.invalidate()
}

// syncLocale keeps MyGo and Lobe's built-in strings (toast dismiss labels,
// empty-state text) in the same language as the app's own copy.
func (a *app) syncLocale() {
	tag := "zh-Hans"
	if a.locale == localeEnglish {
		tag = "en"
	}
	mygo.App.SetLocale(tag)
}
