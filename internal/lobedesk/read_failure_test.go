package lobedesk

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

// A failed reload keeps the old contents on screen, so the page has to say
// they are stale and offer the read again. Without the banner the old commits,
// status and branch names still looked live.
func TestAFailedReloadSaysTheContentsAreStale(t *testing.T) {
	const err = "fatal: cannot read history"
	a := worktreePageApp()
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	a.selectedSHA = "aaaa"
	a.commitFilesLoaded = true
	a.loadErr = err
	tt := pageTester(a, a.contentCard)

	if _, ok := tt.Find(a.t("reloadFailed")); !ok {
		t.Error("the page does not say the refresh failed")
	}
	if _, ok := tt.Find(a.tf("reloadStaleHint", err)); !ok {
		t.Error("the page does not say the contents are from the last read")
	}
	if _, ok := tt.Find("first"); !ok {
		t.Error("the old commit list was dropped instead of kept as stale")
	}
	retry, ok := tt.AccessByRole(ui.RoleButton, a.t("retry"))
	if !ok {
		t.Fatal("a failed reload offers no retry")
	}
	before := a.loadSeq
	tt.ClickAt(retry.Bounds.X+retry.Bounds.W/2, retry.Bounds.Y+retry.Bounds.H/2)
	if a.loadSeq == before {
		t.Error("Retry did not start a new read")
	}
}

// A first open has nothing stale behind it, so the empty page keeps naming the
// failure as an open failure rather than a refresh.
func TestAFirstOpenFailureIsNotCalledARefresh(t *testing.T) {
	a := &app{locale: localeChinese, loadErr: "not a git repository"}
	tt := pageTester(a, a.emptyPage)
	if _, ok := tt.Find(a.t("reloadFailed")); ok {
		t.Error("an open with no repository behind it is called a refresh failure")
	}
}

// Switching repositories and refreshing one are different failures: the first
// leaves the previous repository on screen, the second the same one, and the
// toast has to say which happened.
func TestReadFailuresNameWhatTheReaderWasDoing(t *testing.T) {
	a := &app{locale: localeChinese}
	if got := a.loadFailureTitle(true, false); got != a.t("openFailed") {
		t.Errorf("first open failure says %q", got)
	}
	if got := a.loadFailureTitle(false, false); got != a.t("reloadFailed") {
		t.Errorf("refresh failure says %q", got)
	}
	if got := a.loadFailureTitle(false, true); got != a.t("switchRepoFailed") {
		t.Errorf("switch failure says %q", got)
	}
}

// Each ref group owns its failure: the tag error belongs under the tags title,
// and the page header must not claim the repository has no tags.
func TestAFailedTagReadKeepsItsOwnSection(t *testing.T) {
	const err = "fatal: cannot read tags"
	a := &app{locale: localeChinese, tagsErr: err}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt := pageTester(a, a.branchesPage)

	if _, ok := tt.Find(err); !ok {
		t.Error("the page hides the tag read error")
	}
	if !tt.HasText("标签读取失败") {
		t.Error("the header counts tags even though their read failed")
	}
	if tt.HasText("0 个标签") {
		t.Error("the header claims there are no tags after a failed read")
	}
	if _, ok := tt.Find(a.t("tags")); !ok {
		t.Error("the tag section has no title to attach the error to")
	}
	if _, ok := tt.Find(a.t("noTags")); ok {
		t.Error("the page offers an empty tag section beside the error")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retryTags")); !ok {
		t.Error("the failed tag read offers no retry")
	}
}

// The ref picker covers the history list. With Tab on its Retry, an arrow used
// to move and load the commit list behind the popup.
func TestTheRefPickerKeepsTheHistoryArrows(t *testing.T) {
	a := worktreePageApp()
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.selectedSHA = "aaaa"
	a.commitFilesLoaded = true
	a.historyRefOpen = true
	a.moveSelection(1)
	if a.selectedSHA != "aaaa" {
		t.Errorf("an arrow behind the ref picker selected %q", a.selectedSHA)
	}
	if a.worktreeSelected {
		t.Error("an arrow behind the ref picker selected the working tree")
	}
}

// While a read is already running, Retry either does nothing (busy) or starts
// a second full read (loading); it says so instead of looking available.
func TestRetryIsDisabledWhileAReadIsRunning(t *testing.T) {
	a := &app{locale: localeChinese, loading: true, branchesErr: "fatal: cannot read refs"}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt := pageTester(a, a.branchesPage)
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retryLocalBranches")); ok {
		t.Error("Retry looks available while a read is already running")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retrying")); !ok {
		t.Error("the failed read does not say a retry is already running")
	}
}

// A fetch or a stage is not a read: the button stays "Retry", just disabled,
// instead of claiming a retry that is not happening.
func TestRetryDoesNotCallAnotherOperationARetry(t *testing.T) {
	a := &app{locale: localeChinese, busy: true, branchesErr: "fatal: cannot read refs"}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt := pageTester(a, a.branchesPage)
	retry, ok := tt.AccessByRole(ui.RoleButton, a.t("retryLocalBranches"))
	if !ok {
		t.Fatal("the failed read lost its retry button while busy")
	}
	if !retry.Disabled {
		t.Error("the retry button is enabled while another operation holds the repository")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retrying")); ok {
		t.Error("a fetch is named as a retry")
	}
}

// A failed patch used to be a dead end: the files were loaded, so tapping the
// same commit returned without asking git again.
func TestAFailedCommitPatchOffersRetry(t *testing.T) {
	a := &app{locale: localeChinese, ctx: canceledContext()}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.selectedSHA = "aaaa"
	a.commitFilesLoaded = true
	a.commitDiffErr = "fatal: bad object"
	tt := pageTester(a, a.commitDiffView)
	retry, ok := tt.AccessByRole(ui.RoleButton, a.t("retry"))
	if !ok {
		t.Fatal("the failed patch offers no retry")
	}
	before := a.commitSeq
	tt.ClickAt(retry.Bounds.X+retry.Bounds.W/2, retry.Bounds.Y+retry.Bounds.H/2)
	if a.commitSeq == before {
		t.Error("Retry did not ask for the commit again")
	}
	if !a.commitDiffLoading {
		t.Error("Retry did not put the patch back into its loading state")
	}
}

// When the file list itself failed, its column owns the error and the retry;
// the patch column says why it cannot load instead of repeating both.
func TestAFailedFileListDoesNotRepeatItsRetryInThePatch(t *testing.T) {
	a := &app{locale: localeChinese}
	a.commitFilesErr = "fatal: bad object"
	a.commitDiffErr = "fatal: bad object"
	tt := pageTester(a, a.commitDiffView)
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retry")); ok {
		t.Error("the patch column offers a second retry for the file list's failure")
	}
	if _, ok := tt.Find(a.t("diffBlockedByFiles")); !ok {
		t.Error("the patch column does not say why it cannot load")
	}
}

// The old rows are hidden behind the list's error placeholder; the footer and
// the arrows must not page or select them.
func TestAFailedHistoryListIsNotPagedOrWalked(t *testing.T) {
	a := worktreePageApp()
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.selectedSHA = "aaaa"
	a.commitFilesLoaded = true
	a.historyErr = "fatal: cannot read history"
	a.historyMore = true
	tt := pageTester(a, a.historyPage)
	if _, ok := tt.Find(a.t("loadMoreHistory")); ok {
		t.Error("the failed list still offers to load more rows")
	}
	a.loadMoreHistory()
	if a.historyLoadingMore {
		t.Error("the failed list started another page")
	}
	a.moveHistorySelection(1)
	if a.selectedSHA != "aaaa" {
		t.Error("the arrows moved onto rows the error hides")
	}
}

// A failed Stash read is not an empty Stash list.
func TestAFailedStashReadNamesTheFailure(t *testing.T) {
	a := &app{locale: localeChinese, stashesErr: "fatal: cannot read stashes"}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt := pageTester(a, a.historyPage)
	if !tt.HasText(a.t("stashesReadFailed")) {
		t.Error("the list counts a stash column it could not read")
	}
	if tt.HasText("0 个 Stash") {
		t.Error("the failed Stash read is reported as an empty list")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retry")); !ok {
		t.Error("the failed Stash read offers no retry")
	}
}

// A worktree creation that cannot start, or that git rejects, used to throw
// away the whole form. The typed path, branch and start come back with it.
func TestAFailedWorktreeCreationKeepsTheTypedValues(t *testing.T) {
	a := &app{locale: localeChinese, busy: true}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.worktreeForm = worktreeFormState{
		open:   true,
		path:   "/tmp/supergit-topic",
		branch: "topic",
		start:  "main",
	}
	a.submitWorktreeForm()
	if !a.worktreeForm.open {
		t.Fatal("the failed creation closed the form and lost the input")
	}
	if a.worktreeForm.path != "/tmp/supergit-topic" || a.worktreeForm.branch != "topic" || a.worktreeForm.start != "main" {
		t.Errorf("the form came back as %+v", a.worktreeForm)
	}
}

// Switching refs and having the walk fail used to leave the previous ref's
// rows, count and details under the new ref's name. The failed walk now drops
// them: the header says the read failed and the details describe nothing.
func TestAFailedRefSwitchDropsTheOldWalk(t *testing.T) {
	old := gitrepo.Commit{SHA: "old", Subject: "old subject"}
	a := &app{
		locale:            localeChinese,
		page:              "history",
		commits:           []gitrepo.Commit{old},
		selectedSHA:       old.SHA,
		historyQuery:      "old",
		historyMore:       true,
		commitFiles:       []gitrepo.ChangedFile{{Path: "a.go"}},
		commitFilesLoaded: true,
		commitPath:        "a.go",
		commitSeq:         7,
	}
	a.failHistoryReload("fatal: bad revision")
	if len(a.commits) != 0 || len(a.historyTips) != 0 || a.historyMore {
		t.Errorf("the failed walk kept %d commits, %d tips, more=%v", len(a.commits), len(a.historyTips), a.historyMore)
	}
	if a.selectedSHA != "" || a.commitPath != "" || a.commitFiles != nil || a.commitFilesLoaded {
		t.Errorf("the failed walk kept selection %q, path %q", a.selectedSHA, a.commitPath)
	}
	if a.commitSeq != 8 {
		t.Errorf("the failed walk left commit sequence at %d; want 8 so a pending detail load is dropped", a.commitSeq)
	}
	tt := pageTester(a, a.historyPage)
	if _, ok := tt.Find("old subject"); ok {
		t.Error("the failed walk still lists the old ref's commit")
	}
	if _, ok := tt.Find(a.t("commitsReadFailed")); !ok {
		t.Error("the header does not say the commit read failed")
	}
	if _, ok := tt.Find("1 个提交"); ok {
		t.Error("the header still counts the walk that failed")
	}
	if _, ok := tt.Find(a.t("selectCommit")); !ok {
		t.Error("the details still describe the old commit")
	}
}

// While the reload banner is up it owns the page's one Retry: a second button
// on the list below would repeat the same full reread under another name.
func TestTheReloadBannerOwnsThePagesOnlyRetry(t *testing.T) {
	const err = "fatal: cannot read refs"
	a := &app{locale: localeChinese, page: "branches", loadErr: err, branchesErr: err}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt := pageTester(a, a.contentCard)
	if _, ok := tt.Find(err); !ok {
		t.Fatal("the page hides the failed read")
	}
	buttons := 0
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleButton && info.Name == a.t("retry") {
			buttons++
		}
	}
	if buttons != 1 {
		t.Errorf("the page shows %d Retry buttons; want the banner's one", buttons)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retryLocalBranches")); ok {
		t.Error("the list repeats the banner's retry")
	}
}

// A retry keeps the banner up, so the read it started has visible feedback:
// the button says it is running and cannot be pressed again.
func TestARetryKeepsItsBannerWhileTheReadRuns(t *testing.T) {
	const err = "fatal: cannot read refs"
	a := &app{locale: localeChinese, page: "branches", loadErr: err, loading: true}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt := pageTester(a, a.contentCard)
	retry, ok := tt.AccessByRole(ui.RoleButton, a.t("retrying"))
	if !ok {
		t.Fatal("the banner disappears or goes quiet while the retry runs")
	}
	if !retry.Disabled {
		t.Error("the running retry can be pressed again")
	}
	if _, ok := tt.Find(a.t("reloadFailed")); !ok {
		t.Error("the banner drops its failure title while retrying")
	}
}

// A Stash list read can come back with part of the list and an error. The
// patch chosen before that failure belonged to a row the result no longer
// vouches for, so the list drops it instead of leaving it beside the error.
func TestAFailedStashListDropsTheOldPatch(t *testing.T) {
	a := &app{
		locale:        localeChinese,
		page:          "history",
		stashes:       []gitrepo.Stash{{SHA: "aaaa"}, {SHA: "bbbb"}},
		stashesErr:    "fatal: cannot read stashes",
		selectedStash: "aaaa",
		stashDiff:     gitrepo.Diff{Lines: []gitrepo.DiffLine{{Kind: "add", Text: "old patch line"}}},
		stashSeq:      3,
	}
	a.clearStashDetail()
	if a.selectedStash != "" || len(a.stashDiff.Lines) != 0 || a.stashDiffErr != "" {
		t.Errorf("the failed list kept %q, %d lines, error %q", a.selectedStash, len(a.stashDiff.Lines), a.stashDiffErr)
	}
	if a.stashSeq != 4 {
		t.Errorf("the failed list left the stash sequence at %d; want 4 so a pending patch load is dropped", a.stashSeq)
	}
	// Partial rows are still listed, but a failed read must not auto-pick one
	// and load a patch the error says cannot be vouched for.
	a.reconcileSelection()
	if a.selectedStash != "" {
		t.Errorf("the failed list auto-selected %q", a.selectedStash)
	}
	tt := pageTester(a, a.historyPage)
	if _, ok := tt.Find("old patch line"); ok {
		t.Error("the failed list still shows the old patch")
	}
	// With no stash selected the details describe the working tree again.
	if _, ok := tt.Find(a.t("noChangesShort")); !ok {
		t.Error("the details pane does not return to the working tree")
	}
	// The list's own error still offers the full reread under its name.
	if _, ok := tt.Find(a.stashesErr); !ok {
		t.Error("the failed list lost its error line")
	}
}

// The ref picker belongs to the history header. A page shortcut switches pages
// without the popover's own close path, and the open flag it leaves behind
// swallows the arrows of whichever page comes next; returning to history would
// also reopen it with the old query.
func TestTheRefPickerClosesWhenAnotherPageOpens(t *testing.T) {
	a := worktreePageApp()
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.branches = []gitrepo.Branch{{Name: "main", Current: true}, {Name: "feature"}}
	a.historyRefOpen = true
	a.historyRefQuery = "mai"
	a.openPage("branches")
	if a.historyRefOpen || a.historyRefQuery != "" {
		t.Fatalf("leaving history kept the ref picker open (open=%t query=%q)", a.historyRefOpen, a.historyRefQuery)
	}
	a.moveSelection(1)
	if a.branchSel != "local:main" {
		t.Errorf("the first arrow on the branches page selected %q; want the first branch", a.branchSel)
	}
}

// Every other failed read turns its Retry into "Retrying…" while a read runs;
// the ref picker's retry is a full reread too, so it must not start another.
func TestTheRefPickerRetryWaitsForTheReadItStarted(t *testing.T) {
	a := &app{
		locale: localeChinese, page: "history",
		repo:           &gitrepo.Repository{Root: t.TempDir()},
		branchesErr:    "fatal: cannot read refs",
		historyRefOpen: true,
		loading:        true,
	}
	tt := pageTester(a, a.historyRefMenu)
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("retrying"))
	if !ok {
		t.Fatal("the picker's retry says nothing while its read is running")
	}
	if !button.Disabled {
		t.Error("the picker's retry is clickable while a read is already running")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retry")); ok {
		t.Error("the picker offers a second Retry beside the running read")
	}
}
