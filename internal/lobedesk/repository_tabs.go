package lobedesk

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"supergit/internal/gitrepo"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"golang.org/x/text/unicode/norm"
)

// The window renders one session at a time. Each session owns its Git state,
// drafts and asynchronous callbacks even while it is offscreen.
type repositoryTabs struct {
	ctx                context.Context
	active             *app
	lastCheckout       *app
	sessions           []*app
	opening            *app
	pendingAlias       *app
	pendingClose       *app
	closed             []closedRepositoryTab
	reopening          closedRepositoryTab
	reopenMenuItem     *mygo.MenuItem
	menuRecentRoot     string
	menuRecentRoots    []string
	settings           atomic.Pointer[mygo.Window]
	scroll             ui.ScrollState
	reveal             bool
	stripWidth         float32
	stripTabCount      int
	stripButtonWidth   float32
	stripOverflowSpace float32
	stripOpening       bool
	shownNoticeSession *app
	focusTab           *app
	sessionState       repositorySessionState
	sessionStarted     bool
	repositoryFolders  []string
	updater            *updateController
}

func newRepositoryTabs(ctx context.Context, first *app) *repositoryTabs {
	tabs := &repositoryTabs{ctx: ctx, active: first, sessions: []*app{first}}
	first.repositoryTabs = tabs
	return tabs
}

func (tabs *repositoryTabs) view(c *ui.Context) {
	a := tabs.active
	a.renderWorkbench(c, func() { tabs.strip(c, a) })
	// Mount dialogs after the frame that dismisses the context menu.
	if target := tabs.pendingClose; target != nil {
		tabs.pendingClose = nil
		tabs.askClose(target)
		c.After(0)
	} else if target := tabs.pendingAlias; target != nil {
		tabs.pendingAlias = nil
		tabs.askAlias(target)
		c.After(0)
	}
	if tabs.openNextRepositoryFolder() || tabs.active != a {
		c.After(0)
	}
}

func (tabs *repositoryTabs) newSession(source *app) *app {
	ctx, cancel := context.WithCancel(tabs.ctx)
	next := &app{
		repositoryTabs: tabs, ctx: ctx, cancel: cancel, runOnMain: source.runOnMain,
		openDirectoryDialog: source.openDirectoryDialog, saveFileDialog: source.saveFileDialog,
		page: "history", lineAnchor: -1, lineCursor: -1, linePressIndex: -1,
		commitAnchor: -1, commitCursor: -1, commitPressIndex: -1,
	}
	next.copyWindowPreferences(source)
	return next
}

// Copy the shared preferences at activation. Git data, text fields and scroll
// positions remain in their original session.
func (a *app) copyWindowPreferences(source *app) {
	a.locale, a.themeMode, a.palette, a.dockIcon = source.locale, source.themeMode, source.palette, source.dockIcon
	a.sidebarHidden, a.expandMerges = source.sidebarHidden, source.expandMerges
	a.syncHiddenBranches(source.hiddenBranches)
	a.setIgnoreDiffWhitespace(source.ignoreDiffWhitespace)
	a.uiFont, a.codeFont, a.codeFontSize, a.splitDiff = source.uiFont, source.codeFont, source.codeFontSize, source.splitDiff
	a.diffPalette = source.diffPaletteKey()
	a.showGraph, a.historyLine, a.historyNode = source.showGraph, source.historyLine, source.historyNode
	a.hideAuthor, a.hideTime = source.hideAuthor, source.hideTime
	a.autoFetch, a.autoFetchPrune, a.autoFetchEvery = source.autoFetch, source.autoFetchPrune, source.autoFetchEvery
	a.autoCheckUpdates, a.autoDownloadUpdates = source.autoCheckUpdates, source.autoDownloadUpdates
	a.skippedUpdateVersion, a.lastUpdateCheckAt = source.skippedUpdateVersion, source.lastUpdateCheckAt
	a.detailsInfoHidden, a.detailsChangesHidden, a.detailsDiffHidden = source.detailsInfoHidden, source.detailsChangesHidden, source.detailsDiffHidden
	a.historyDetailsHidden = source.historyDetailsHidden
	a.sidebarSplit.share, a.historySplit.share, a.filesSplit.share = source.sidebarSplit.share, source.historySplit.share, source.filesSplit.share
	a.stashSplit.share, a.branchSplit.share = source.stashSplit.share, source.branchSplit.share
	a.filesFolded = source.filesFolded
	a.recentRepositories = source.recentRepositories
	a.settingsSection, a.settingsPositions, a.settingsScroll = source.settingsSection, source.settingsPositions, source.settingsScroll
}

// canSwitch covers tab navigation, which stays open while the active tab is
// only fetching. Structural changes such as opening or replacing a repository
// use canChange.
func (tabs *repositoryTabs) canSwitch() bool {
	a := tabs.active
	// An existing checkout owns its reads. Navigating away need not wait for
	// them, unlike startup restoration which is still assembling the tab list.
	readAllowsSwitch := !a.loading || (a.repo != nil && !tabs.sessionState.restoring)
	return !a.repositoryOpenPending() && !a.localWatch.closed && readAllowsSwitch && !a.exclusiveBusy() && !a.repositoryChooserOpen &&
		!a.hasPendingDialog(toastMain) && !a.hasPendingDialog(toastSettings)
}

func (tabs *repositoryTabs) canChange() bool {
	a := tabs.active
	return !a.repositoryOpenPending() && !a.localWatch.closed && !a.loading && !a.busy && !a.repositoryChooserOpen &&
		!a.hasPendingDialog(toastMain) && !a.hasPendingDialog(toastSettings)
}

func (a *app) canCycleRepositoryTabs() bool {
	tabs := a.repositoryTabs
	return tabs != nil && tabs.active == a && len(tabs.sessions) > 1 && tabs.canSwitch()
}

func (a *app) cycleRepositoryTabs(delta int) {
	if !a.canCycleRepositoryTabs() {
		return
	}
	tabs := a.repositoryTabs
	index := slices.Index(tabs.sessions, a)
	count := len(tabs.sessions)
	tabs.activate(tabs.sessions[(index+count+delta)%count], false)
}

func (a *app) repositoryOpenPending() bool {
	return a.repositoryTabs != nil && a.repositoryTabs.opening != nil
}

func (a *app) activeRepositorySession() *app {
	if a.repositoryTabs != nil {
		return a.repositoryTabs.active
	}
	return a
}

func (a *app) settingsWindowSlot() *atomic.Pointer[mygo.Window] {
	if a.repositoryTabs != nil {
		return &a.repositoryTabs.settings
	}
	return &a.settingsWin
}

func (tabs *repositoryTabs) newTab() {
	if !tabs.canChange() {
		return
	}
	next := tabs.newSession(tabs.active)
	tabs.sessions = append(tabs.sessions, next)
	tabs.focusTab = next
	tabs.activate(next, true)
}

func (tabs *repositoryTabs) findCheckout(root string) *app {
	for _, session := range tabs.sessions {
		if session.repo != nil && session.repo.Root == root {
			return session
		}
	}
	return nil
}

func (tabs *repositoryTabs) open(source *app, path string) {
	if strings.TrimSpace(path) == "" || source != tabs.active || !tabs.canChange() {
		return
	}
	if existing := tabs.findCheckout(path); existing != nil {
		if source.repo == nil {
			source.loadErr = ""
		}
		tabs.activate(existing, false)
		return
	}
	if source.repo == nil {
		source.readRepository(path, true)
		return
	}
	next := tabs.newSession(source)
	next.cloneOpeningRoot, next.newRepoOpeningRoot = source.cloneOpeningRoot, source.newRepoOpeningRoot
	next.repositorySwitchTarget = source.repositorySwitchTarget
	source.cloneOpeningRoot, source.newRepoOpeningRoot, source.repositorySwitchTarget = "", "", ""
	source.cancelLocalProbe()
	tabs.opening = next
	next.readRepository(path, true)
	source.invalidate()
}

func (tabs *repositoryTabs) opened(next *app) {
	if tabs.opening != next {
		return
	}
	reopening := tabs.reopening
	tabs.opening, tabs.reopening = nil, closedRepositoryTab{}
	source := tabs.active
	if next.repo == nil {
		for _, notice := range next.notices[toastMain] {
			notice.root = ""
			source.pushNotice(toastMain, notice)
		}
		next.stopRepositorySession()
		source.invalidate()
		return
	}
	if existing := tabs.findCheckout(next.repo.Root); existing != nil {
		next.stopRepositorySession()
		if tabs.canChange() {
			tabs.activate(existing, false)
		}
		source.invalidate()
		return
	}
	tabs.forgetClosedRepository(next.repo.Root)
	canActivate := tabs.canChange()
	if reopening.root == "" {
		tabs.sessions = append(tabs.sessions, next)
	} else if source.repo == nil && len(tabs.sessions) == 1 && canActivate {
		// Reopening the last checkout fills the otherwise empty workbench.
		tabs.sessions[0] = next
	} else {
		tabs.sessions = slices.Insert(tabs.sessions, min(reopening.index, len(tabs.sessions)), next)
	}
	tabs.sessionState.tabsChanged = true
	if canActivate {
		tabs.activate(next, true)
		if reopening.root != "" {
			tabs.focusTab = next
			if !slices.Contains(tabs.sessions, source) {
				source.stopRepositorySession()
			}
		}
		next.fetchOnOpen()
	} else {
		source.invalidate()
	}
}

func (tabs *repositoryTabs) activate(next *app, fresh bool) {
	previous := tabs.active
	if next == previous {
		return
	}
	previous.cancelLocalProbe()
	previous.syncOpen = false
	previous.dismissSyncCommits()
	next.copyWindowPreferences(previous)
	next.window = previous.window
	next.localWatch.focused = previous.localWatch.focused
	next.notices[toastSettings] = previous.notices[toastSettings]
	previous.notices[toastSettings] = nil
	tabs.active, tabs.reveal = next, true
	if previous.repo != nil && !previous.restoredCheckout {
		tabs.lastCheckout = previous
	}
	title := "SuperGit"
	if next.repo != nil {
		title += " · " + filepath.Base(next.repo.Root)
		next.localWatch.next = time.Time{}
		if next.restoredCheckout {
			if !next.fetching && !next.loading {
				next.reload()
			}
		} else {
			tabs.acceptCheckoutSelection(next)
			next.rememberRepository(next.repo.Root)
			// A fetch or an in-flight read owns this tab's refresh. Mark the
			// activation so the eventual read still re-checks HEAD; starting
			// a second validation now would race the first read's completion.
			if !fresh && (next.fetching || next.loading) {
				next.activationRefresh = true
			} else if !fresh {
				next.revalidateActivatedRepository()
			}
		}
	} else {
		tabs.refreshRepositoryRecents()
	}
	if next.window != nil {
		next.window.SetTitle(title)
	}
	next.invalidate()
}

// Check the checkout before enabling operations on a returning tab. An
// unchanged snapshot keeps its selection, pagination and History navigation.
func (a *app) revalidateActivatedRepository() {
	repo, seq, baseline := a.repo, a.loadSeq, a.localWatch.stamp
	expectedRefs := a.localWatch.refStamp
	entries := slices.Clone(a.status.Entries)
	submodules := a.nextSubmoduleProbes()
	forceReload := a.loadErr != "" || baseline == ""
	a.activationRefresh = true
	a.checkingTab = true
	a.loading, a.loadSince = true, time.Now()
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, localBackgroundInterval)
		defer cancel()
		unchanged := false
		var refs string
		var updates []gitrepo.Submodule
		if !forceReload {
			current, err := gitrepo.Open(ctx, repo.Root)
			if err == nil && current.Root == repo.Root {
				stamp, err := readLocalWatchStamp(ctx, current, entries)
				unchanged = err == nil && stamp == baseline
				if unchanged {
					refs, err = current.RefStamp(ctx)
					unchanged = err == nil
					updates = readSubmoduleUpdates(ctx, repo, submodules)
				}
			}
		}
		a.hopToMain(func() {
			if a.localWatch.closed || a.loadSeq != seq || a.repo != repo {
				return
			}
			a.checkingTab = false
			if a.activeRepositorySession() != a {
				// No writes can be initiated here. Revalidate on the next visit
				// instead of leaving this inactive tab stuck in its loading state.
				a.activationRefresh, a.loading = false, false
				a.invalidate()
				return
			}
			if unchanged && !a.applySubmoduleUpdates(updates) {
				if refs != expectedRefs {
					a.refreshRefs()
					return
				}
				a.activationRefresh, a.loading = false, false
				a.localWatch.next = time.Now().Add(localProbeInterval)
				a.invalidate()
				return
			}
			a.reload()
		})
	}()
}

func (a *app) tabSnapshotUnavailable() bool {
	return a.repositoryTabs != nil && a.repo != nil && a.loadErr != ""
}

func (tabs *repositoryTabs) canClose(target *app) bool {
	return slices.Contains(tabs.sessions, target) && (target.repo != nil || len(tabs.sessions) > 1) &&
		tabs.canSwitch() && !target.busy && !target.fetching && (!target.loading || target != tabs.active)
}

func (tabs *repositoryTabs) askClose(target *app) {
	if !tabs.canClose(target) {
		return
	}
	if target.repo == nil || (target.commitText == "" && target.unrestoredDraft == "" && len(target.conflictEditors) == 0) {
		tabs.close(target)
		return
	}
	a := tabs.active
	a.askConfirmInWindow(toastMain, "", a.t("closeRepositoryTab"), a.tf("closeRepositoryDraft", target.repo.Root), a.t("close"), false, func() {
		if tabs.canClose(target) {
			tabs.close(target)
		}
	})
}

func (tabs *repositoryTabs) close(target *app) {
	i := slices.Index(tabs.sessions, target)
	if i < 0 {
		return
	}
	tabs.sessions = slices.Delete(tabs.sessions, i, i+1)
	if target.repo != nil {
		tabs.rememberClosedRepository(target.repo.Root, i)
		state := &tabs.sessionState
		state.tabsChanged = true
		if state.path != "" {
			if state.discardedDrafts == nil {
				state.discardedDrafts = make(map[string]bool)
			}
			state.discardedDrafts[target.repo.Root] = true
		}
	}
	if target == tabs.active {
		if len(tabs.sessions) == 0 {
			tabs.sessions = append(tabs.sessions, tabs.newSession(target))
		}
		next := tabs.sessions[min(i, len(tabs.sessions)-1)]
		tabs.focusTab = next
		tabs.activate(next, false)
	}
	if tabs.lastCheckout == target {
		tabs.lastCheckout = nil
	}
	target.stopRepositorySession()
	tabs.active.invalidate()
}

func (a *app) stopRepositorySession() {
	a.closeWorkspaceClean()
	a.localWatch.closed = true
	a.loadSeq++
	a.cancelLocalProbe()
	a.cancelOperation()
	a.resetHistoryTraversal()
	a.resetFileHistory()
	a.invalidateWorktreeOverview()
	if a.cancel != nil {
		a.cancel()
	}
}

func (tabs *repositoryTabs) shutdown() {
	for _, session := range tabs.sessions {
		session.stopRepositorySession()
	}
	if tabs.opening != nil {
		tabs.opening.stopRepositorySession()
		tabs.opening = nil
	}
}

func (tabs *repositoryTabs) strip(c *ui.Context, a *app) {
	visibleTabs := tabs.sessions
	tk := theme.TokensOf(c)
	blocked := !tabs.canChange()
	tabBlocked := !tabs.canSwitch()
	const gap = 4
	width, _ := c.Size()
	buttonWidth := scaledFont(workbenchButtonBaseHeight)
	opening := tabs.opening != nil
	if width != tabs.stripWidth || buttonWidth != tabs.stripButtonWidth || len(visibleTabs) != tabs.stripTabCount || opening != tabs.stripOpening {
		// ScrollArea publishes its extent after layout.
		c.After(0)
		tabs.reveal = true
	}
	space := tabs.stripOverflowSpace
	if len(visibleTabs) == 0 {
		space = 0
	} else if !opening && !tabs.stripOpening {
		// Compare against the viewport without the overflow controls so their
		// own width cannot keep them visible when all tabs would fit.
		space = 0
		if tabs.scroll.MaxX > tabs.stripOverflowSpace+0.5 {
			space = 3 * (buttonWidth + gap)
		}
	}
	if space != tabs.stripOverflowSpace {
		tabs.reveal = true
		c.After(0)
	}
	tabs.stripWidth, tabs.stripButtonWidth, tabs.stripTabCount = width, buttonWidth, len(visibleTabs)
	tabs.stripOverflowSpace, tabs.stripOpening = space, opening
	stripGround := ui.Hex("#E7ECF1")
	if tk.Dark {
		stripGround = ui.Hex("#151C25")
	}
	windowTitleBar(c).Height(max(float32(52), c.TitleBar().Height)).Background(stripGround).Gap(gap).Children(func() {
		var activate, close, alias, dragSource, dragTarget *app
		viewport := ui.Box(c).Grow(1).Basis(0).MinWidth(0).Height(46).PaddingX(3)
		clip := viewport.Bounds()
		names := tabs.checkoutTabNames(c, visibleTabs, repositoryCaptionWidth(repositoryCardWidth(clip.W, len(visibleTabs))))
		viewport.Children(func() {
			layout.ScrollArea(c, layout.ScrollAreaProps{Direction: layout.Horizontal, State: &tabs.scroll}, func() {
				ui.Row(c).Height(44).Gap(6).AlignItems(ui.Center).Role(ui.RoleTabList).Label(a.t("repositoryTabs")).Children(func() {
					for _, session := range visibleTabs {
						row, clicked, closed, editAlias := tabs.tab(c.Key(session), a, session, names[session], tabBlocked, clip)
						if editAlias {
							alias = session
						}
						if !tabBlocked && len(visibleTabs) > 1 {
							if source := tabs.tabDropTarget(row, session, tk.Primary); source != nil {
								dragSource, dragTarget = source, session
							}
						}
						if clicked {
							activate = session
						}
						if closed {
							close = session
						}
					}
				})
			}).Fill()
		})
		if dragTarget != nil && tabs.moveTab(dragSource, dragTarget) {
			tabs.reveal = true
			c.After(0)
		}
		if close != nil {
			tabs.pendingClose = close
		} else if alias != nil {
			tabs.pendingAlias = alias
		} else if activate != nil && tabs.canSwitch() {
			tabs.activate(activate, false)
			c.After(0)
		}
		if space > 0 {
			// Keep part of the previous view visible so the next group of tabs
			// is easy to place. The active session is unchanged by scrolling.
			step := max(buttonWidth, clip.W*0.8)
			left := workbenchIconButton(c.Key("repository-tabs-left"), lucide.ChevronLeft(), a.t("scrollRepositoryTabsLeft"), buttonNormal, tabBlocked || tabs.scroll.X <= 0)
			right := workbenchIconButton(c.Key("repository-tabs-right"), lucide.ChevronRight(), a.t("scrollRepositoryTabsRight"), buttonNormal, tabBlocked || tabs.scroll.X >= tabs.scroll.MaxX)
			if left.Clicked() {
				tabs.scroll.X = max(0, tabs.scroll.X-step)
				c.After(0)
			} else if right.Clicked() {
				tabs.scroll.X = min(tabs.scroll.MaxX, tabs.scroll.X+step)
				c.After(0)
			}
			// DialogBase restores focus to its trigger on dismissal.
			canOpen := !blocked && len(visibleTabs) > 1 && a.canOpenRepositoryPicker()
			button := workbenchIconButton(c.Key("repository-switcher"), lucide.List(), a.t("switchRepositoryTab"), buttonNormal, !canOpen && !a.quickRepositorySwitch.open)
			if button.Clicked() && canOpen {
				a.openRepositoryPicker(true)
				c.After(0)
			}
		}
		label := a.t("newRepositoryTab")
		button := styleWorkbenchButton(c, ui.ButtonBase(c.Key("new-repository-tab")), buttonNormal, blocked, false).
			Label(label).Shrink(0)
		button.Children(func() {
			ui.Icon(c, lucide.Plus()).FontSize(scaledFont(14))
			ui.Text(c, label).FontSize(scaledFont(12)).SingleLine()
		})
		if button.Clicked() {
			tabs.newTab()
			c.After(0)
		}
		a.updateTitleBarBadge(c)
		if tabs.opening != nil {
			display.Text(c, a.t("repoOpening"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).SingleLine().Shrink(0)
		}
	})
}

// checkoutTabContext finds the shortest parent suffix that distinguishes a
// checkout from every open checkout with the same name.
func (tabs *repositoryTabs) checkoutTabContext(session *app) string {
	root := filepath.Clean(session.repo.Root)
	base := filepath.Base(root)
	var peers []string
	for _, other := range tabs.sessions {
		if other != session && other.repo != nil && filepath.Base(other.repo.Root) == base {
			peers = append(peers, filepath.Dir(other.repo.Root))
		}
	}
	return distinguishingParentSuffix(filepath.Dir(root), peers)
}

func distinguishingParentSuffix(parent string, peers []string) string {
	if len(peers) == 0 {
		return ""
	}
	var suffix string
	for dir := parent; ; dir = filepath.Dir(dir) {
		if filepath.Dir(dir) == dir {
			return parent
		}
		suffix = filepath.Join(filepath.Base(dir), suffix)
		if !slices.ContainsFunc(peers, func(peer string) bool {
			return peer == suffix || strings.HasSuffix(peer, string(filepath.Separator)+suffix)
		}) {
			return suffix
		}
	}
}

type checkoutTabName struct {
	name, caption string
}

// Equivalent Unicode spellings and zero-width spaces can identify separate
// checkouts. Compare what the caption displays, while retaining the exact path.
func checkoutCaptionText(text string) string {
	return norm.NFC.String(strings.ReplaceAll(text, "​", ""))
}

func (tabs *repositoryTabs) checkoutTabNames(c *ui.Context, sessions []*app, widths ...float32) map[*app]checkoutTabName {
	captionWidth := repositoryCaptionWidth(168)
	if len(widths) > 0 {
		captionWidth = widths[0]
	}
	fits := func(text string) bool {
		width, _ := c.MeasureText(0, ui.Span{Text: text, Size: scaledFont(13), Weight: 500})
		return width <= captionWidth
	}
	fit := func(text string) string {
		if fits(text) {
			return text
		}
		// The current UI scale always fits at least five characters. Look for
		// a longer fit without measuring every prefix of a long checkout name.
		caption := elideMiddle(text, 5)
		for lo, hi := 6, len([]rune(text))-1; lo <= hi; {
			mid := lo + (hi-lo)/2
			candidate := elideMiddle(text, mid)
			if fits(candidate) {
				caption, lo = candidate, mid+1
			} else {
				hi = mid - 1
			}
		}
		return caption
	}
	names := make(map[*app]checkoutTabName, len(sessions))
	groups := make(map[string][]*app)
	for _, session := range sessions {
		if session.repo == nil {
			continue
		}
		base := filepath.Base(session.repo.Root)
		caption := checkoutCaptionText(base)
		name := checkoutTabName{base, caption}
		if session.repositoryAlias != "" {
			name = checkoutTabName{session.repositoryAlias, session.repositoryAlias}
		} else if suffix := tabs.checkoutTabContext(session); suffix != "" {
			name = checkoutTabName{base + " · " + suffix, elideMiddle(caption, 12) + " · " + elideMiddle(checkoutCaptionText(suffix), 20)}
		}
		name.caption = fit(name.caption)
		groups[name.caption] = append(groups[name.caption], session)
		names[session] = name
	}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		// Keep the first differing portion instead of showing identical ends.
		slices.SortFunc(group, func(a, b *app) int {
			if order := strings.Compare(checkoutCaptionText(names[a].name), checkoutCaptionText(names[b].name)); order != 0 {
				return order
			}
			return strings.Compare(a.repo.Root, b.repo.Root)
		})
		firstText := checkoutCaptionText(names[group[0]].name)
		lastText := checkoutCaptionText(names[group[len(group)-1]].name)
		if firstText == lastText {
			continue // Numbering distinguishes names with no visible differences.
		}
		first, last := []rune(firstText), []rune(lastText)
		common := 0
		for common < min(len(first), len(last)) && first[common] == last[common] {
			common++
		}
		for _, session := range group {
			name := names[session]
			fragment := elideAround(checkoutCaptionText(name.name), common)
			base := checkoutCaptionText(filepath.Base(session.repo.Root))
			name.caption = elideMiddle(base, 7) + " · " + fragment
			if !fits(name.caption) {
				name.caption = fit(fragment)
			}
			names[session] = name
		}
	}

	// Number remaining collisions in root order so dragging tabs does not
	// rename them. Reserve literal captions and every generated caption.
	counts := make(map[string]int, len(names))
	nameCounts := make(map[string]int, len(names))
	for _, name := range names {
		counts[name.caption]++
		nameCounts[checkoutCaptionText(name.name)]++
	}
	var duplicates []*app
	for session, name := range names {
		if counts[name.caption] > 1 {
			duplicates = append(duplicates, session)
		}
	}
	slices.SortFunc(duplicates, func(a, b *app) int { return strings.Compare(a.repo.Root, b.repo.Root) })
	number := 0
	for _, session := range duplicates {
		name := names[session]
		for {
			number++
			caption := fit(strconv.Itoa(number) + " · " + name.caption)
			if counts[caption] == 0 {
				counts[caption] = 1
				name.caption = caption
				if session.repositoryAlias != "" && nameCounts[checkoutCaptionText(name.name)] > 1 {
					name.name = strconv.Itoa(number) + " · " + name.name
				}
				names[session] = name
				break
			}
		}
	}
	return names
}

func (tabs *repositoryTabs) tab(c *ui.Context, a, session *app, labels checkoutTabName, blocked bool, viewport ui.Rect) (row ui.Element, activate, closeTab, editAlias bool) {
	name, root, closeTarget := a.t("welcomeTab"), "", a.t("welcomeTab")
	caption := name
	icon := lucide.Folder()
	if session.repo != nil {
		root = session.repo.Root
		closeTarget = root
		name, caption = labels.name, labels.caption
		icon = lucide.FolderGit2()
	}
	tk := theme.TokensOf(c)
	selected := session == a
	row = ui.Row(c).Shrink(0).Width(repositoryCardWidth(viewport.W, len(tabs.sessions))).Height(36).Padding(0, 5, 0, 0).AlignItems(ui.Center)
	ink := paintRepositoryTab(row, tk, session.repositoryColor, selected)
	row.Children(func() {
		tip := root
		if session.repositoryAlias != "" {
			tip = session.repositoryAlias + "\n" + root
		}
		tab := ui.ButtonBase(c).Row().Grow(1).Basis(0).MinWidth(0).Height(36).Padding(0, 8, 0, 12).AlignItems(ui.Center).Gap(8).
			Role(ui.RoleTab).Label(name).Description(root).AccessSelected(selected).Tooltip(tip).Disabled(blocked && !selected)
		styleRepositoryTabTarget(tab, ink)
		if !blocked && len(tabs.sessions) > 1 {
			tab.Drag(repositoryTabDrag{session: session})
		}
		canClose := tabs.canClose(session)
		// When close is unavailable, keep its space inside the selectable tab
		// instead of leaving a disabled button over part of an inactive tab.
		wholeTab := !selected && !blocked && !canClose
		dirty := session.repo != nil && session.status.HasWorkspaceChanges()
		if wholeTab && dirty {
			tab.Description(tip + " · " + a.t("repositoryTabUncommitted"))
		}
		tab.Children(func() {
			ui.Icon(c, icon).FontSize(scaledFont(15)).TextColor(ink).Shrink(0)
			// Selection is carried by the fixed line alone; the label keeps
			// its normal weight so switching tabs does not reflow the caption.
			ui.Text(c, caption).FontSize(scaledFont(13)).FontWeight(500).TextColor(ink).SingleLine().Grow(1).Basis(0).MinWidth(0)
			if wholeTab {
				slot := ui.Box(c).Width(24).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
				if dirty {
					slot.Children(func() { size := scaledFont(6); ui.Box(c).Size(size, size).Radius(size / 2).Background(ink) })
				}
			}
		})
		if tabs.focusTab == session {
			tab.Focus()
			row.ScrollIntoView()
			tabs.focusTab = nil
		}
		if !blocked {
			i, n := slices.Index(tabs.sessions, session), len(tabs.sessions)
			switch {
			case tab.Shortcut(0, ui.KeyLeft):
				tabs.focusTab = tabs.sessions[(i+n-1)%n]
			case tab.Shortcut(0, ui.KeyRight):
				tabs.focusTab = tabs.sessions[(i+1)%n]
			case tab.Shortcut(0, ui.KeyHome):
				tabs.focusTab = tabs.sessions[0]
			case tab.Shortcut(0, ui.KeyEnd):
				tabs.focusTab = tabs.sessions[n-1]
			}
			if tabs.focusTab != nil {
				c.After(0)
			}
		}
		activate = tab.Clicked() && !blocked
		if wholeTab {
			return
		}
		if canClose {
			middleClose := ui.Local(tab, "middle-close", func() bool { return false })
			closeTab = closeTab || *middleClose
			*middleClose = false
			bounds := tab.Bounds()
			tab.HandleInput(func(ev ui.InputEvent) bool {
				if ev.Button != 2 {
					return false
				}
				switch ev.Kind {
				case ui.InputPointerDown:
					return true // Capture the gesture so it cannot move to another tab.
				case ui.InputPointerUp:
					*middleClose = !ev.Canceled && ev.X >= 0 && ev.X < bounds.W && ev.Y >= 0 && ev.Y < bounds.H
					return true
				}
				return false
			})
		}
		// Keep the close affordance out of the clipped strip edge. Its layout
		// slot stays in place so suppressing it does not change scroll extent.
		bounds := row.Bounds()
		clipped := bounds.W > 0 && viewport.W > 0 &&
			(bounds.X < viewport.X-.5 || bounds.X+bounds.W > viewport.X+viewport.W+.5)
		closeLabel := a.t("close") + " " + closeTarget
		close := ui.ButtonBase(c).Disabled(!canClose || clipped).Shrink(0).Label(closeLabel).Tooltip(closeLabel)
		styleRepositoryClose(close, ink)
		if dirty {
			close.Description(a.t("repositoryTabUncommitted"))
		}
		close.Children(func() {
			if dirty && !close.Hovered() && !close.Focused() {
				size := scaledFont(6)
				ui.Box(c).Width(size).Height(size).Radius(size / 2).Background(ink)
			} else {
				ui.Icon(c, lucide.X()).FontSize(scaledFont(14))
			}
		})
		if clipped {
			close.Opacity(0)
		}
		closeTab = closeTab || (canClose && !clipped && close.Clicked())
	})
	if tabs.canClose(session) {
		menuClose, menuAlias := tabs.tabMenu(c.Key("tab-menu"), row, a, session)
		closeTab, editAlias = closeTab || menuClose, menuAlias
	}
	if selected && tabs.reveal {
		row.ScrollIntoView()
		tabs.reveal = false
	}
	return row, activate, closeTab, editAlias
}
