package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func TestFileTabsOpenActivateAndClose(t *testing.T) {
	a := &app{worktreeSelected: true}
	first, second := detailFileKey{Path: "dir/a.go"}, detailFileKey{Path: "other/a.go", Staged: true}
	a.openDetailFile(first)
	a.openDetailFile(second)
	a.openDetailFile(first)
	if len(a.detailTabs().OpenFiles) != 2 || a.selectedPath != first.Path {
		t.Fatal("file click did not reuse and activate its tab")
	}
	a.closeDetailFile(first)
	if a.selectedPath != second.Path || !a.selectedStaged {
		t.Fatal("closing active tab did not show neighbor's version")
	}
	a.closeDetailFile(second)
	if !a.detailTabs().Active.Summary {
		t.Fatal("closing final file did not return to first tab")
	}
}

func TestDraftSurvivesClosingAFileTab(t *testing.T) {
	a := &app{locale: localeEnglish, worktreeSelected: true, commitText: "draft title\n\ndraft body", amend: true}
	first, second := detailFileKey{Path: "dir/a.go"}, detailFileKey{Path: "other/b.go", Staged: true}
	a.openDetailFile(first)
	a.openDetailFile(second)
	tt := pageTester(a, a.historyDiffPane)
	click := func(role ui.Role, label string) {
		t.Helper()
		node, ok := tt.AccessByRole(role, label)
		if !ok {
			t.Fatalf("missing %q", label)
		}
		tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
	}
	click(ui.RoleTab, "a.go · "+a.t("unstaged"))
	if a.selectedPath != first.Path {
		t.Fatal("tab click did not change the patch target")
	}
	click(ui.RoleButton, a.t("close")+" "+first.Path)
	if a.selectedPath != second.Path {
		t.Fatal("close click did not show neighbor")
	}
	click(ui.RoleTab, a.t("commit"))
	if !a.detailTabs().Active.Summary || a.commitText != "draft title\n\ndraft body" || !a.amend {
		t.Fatal("summary or draft state was lost")
	}
}

// Closing the first tab while rendering three tabs must not mutate the slice
// currently being ranged over, duplicate the last key or lose a surviving tab.
func TestClosingFirstOfThreeTabsKeepsUniqueRenderedKeys(t *testing.T) {
	a := &app{locale: localeEnglish, worktreeSelected: true}
	for _, path := range []string{"first.go", "second.go", "third.go"} {
		a.openDetailFile(detailFileKey{Path: path})
	}
	tt := pageTester(a, a.historyDiffPane)
	close, ok := tt.AccessByRole(ui.RoleButton, a.t("close")+" first.go")
	if !ok {
		t.Fatal("first tab close button missing")
	}
	tt.ClickAt(close.Bounds.X+close.Bounds.W/2, close.Bounds.Y+close.Bounds.H/2)
	tt.Frame()
	if len(a.detailTabs().OpenFiles) != 2 {
		t.Fatal("close lost surviving tabs")
	}
	for _, label := range []string{"second.go", "third.go"} {
		if _, ok := tt.AccessByRole(ui.RoleTab, label+" · "+a.t("unstaged")); !ok {
			t.Fatalf("surviving tab %s disappeared", label)
		}
	}
}

func TestSummaryDoesNotPretendASingleFileIsSelected(t *testing.T) {
	a := &app{selectedSHA: "aaaa", commitPath: "a.go", commitFiles: []gitrepo.ChangedFile{{Path: "a.go", Status: "M"}}}
	tt := pageTester(a, a.commitFileList)
	row, ok := tt.AccessByRole(ui.RoleOption, "a.go")
	if !ok {
		t.Fatal("file row missing")
	}
	if row.Selected {
		t.Fatal("summary announces a single file as open")
	}
}

func TestDetailTabScrollIsKeptPerTarget(t *testing.T) {
	for _, worktree := range []bool{true, false} {
		a := &app{worktreeSelected: worktree}
		first, second := detailFileKey{Path: "a"}, detailFileKey{Path: "b"}
		a.openDetailFile(first)
		if worktree {
			a.diffScroll.Y = 123
		} else {
			a.commitDiffScroll.Y = 123
		}
		a.openDetailFile(second)
		if worktree {
			a.diffScroll.Y = 456
		} else {
			a.commitDiffScroll.Y = 456
		}
		a.activateDetailTab(detailTabKey{File: first})
		got := a.commitDiffScroll.Y
		if worktree {
			got = a.diffScroll.Y
		}
		if got != 123 {
			t.Fatalf("worktree=%v: first tab lost scroll: %v", worktree, got)
		}
		a.activateDetailTab(detailTabKey{File: second})
		got = a.commitDiffScroll.Y
		if worktree {
			got = a.diffScroll.Y
		}
		if got != 456 {
			t.Fatalf("worktree=%v: second tab lost scroll: %v", worktree, got)
		}
	}
}

func worktreePageApp() *app {
	return &app{
		locale: localeChinese,
		page:   "history",
		commits: []gitrepo.Commit{
			{SHA: "aaaa", Subject: "first"},
			{SHA: "bbbb", Subject: "second"},
		},
		status: gitrepo.Status{
			Branch: "main",
			Entries: []gitrepo.Entry{
				{Path: "a.go", Index: ' ', Worktree: 'M'},
			},
		},
	}
}

// The patch header keeps the guarded stage action and identifies the original
// path and comparison even when the active file tab is truncated.
func TestTheWorkingTreePatchKeepsItsStageAction(t *testing.T) {
	a := worktreePageApp()
	a.worktreeSelected = true
	a.selectedPath = "dir/a.go"
	a.status.Entries[0].Path = "dir/a.go"
	a.detailTabs().Open(detailFileKey{Path: a.detailFilePath(), Staged: a.selectedStaged})
	tt := pageTester(a, func(c *ui.Context) { a.historyDiffFile(c, theme.TokensOf(c)) })
	if _, ok := tt.Find(a.t("stageFile")); !ok {
		t.Fatal("the working tree's patch has no stage action")
	}
	if _, ok := tt.Find("dir/a.go"); !ok {
		t.Error("the header does not identify the original path")
	}
	if !tt.HasText(a.t("indexToWorktree")) {
		t.Error("the header does not identify the comparison baseline")
	}
}

// The working tree's row is the list's first row, and choosing it points the
// page at the working tree instead of a commit.
func TestTheWorkingTreeIsTheListsFirstRow(t *testing.T) {
	a := worktreePageApp()
	a.selectedSHA = "bbbb"
	tt := pageTester(a, a.historyPage)
	row, ok := tt.AccessByRole(ui.RoleOption, a.changesLabel(1))
	if !ok {
		t.Fatal("the working tree's row is not in the list")
	}
	if row.PosInSet != 1 || row.SetSize != 3 {
		t.Fatalf("the working tree reads place %d of %d; want 1 of 3", row.PosInSet, row.SetSize)
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if !a.worktreeSelected {
		t.Fatal("clicking the working tree's row did not select it")
	}
	if a.selectedSHA != "" {
		t.Fatalf("the working tree is selected but commit %q still is", a.selectedSHA)
	}

	// A commit row takes the selection back.
	a.worktreeSelected = false
	a.selectedSHA = ""
	tt.Frame()
	commit, ok := tt.AccessByRole(ui.RoleOption, "first")
	if !ok {
		t.Fatal("the first commit is not in the list")
	}
	tt.ClickAt(commit.Bounds.X+commit.Bounds.W/2, commit.Bounds.Y+commit.Bounds.H/2)
	if a.worktreeSelected {
		t.Fatal("clicking a commit left the working tree selected")
	}
	if a.selectedSHA != "aaaa" {
		t.Fatalf("selected commit = %q; want the one that was clicked", a.selectedSHA)
	}
}

// The arrows walk the same list the pointer does: up from the first commit
// lands on the working tree's row, and down from there returns.
func TestArrowsWalkIntoTheWorkingTreeRow(t *testing.T) {
	a := worktreePageApp()
	a.selectedSHA = "aaaa"
	a.moveHistorySelection(-1)
	if !a.worktreeSelected {
		t.Fatal("the arrow up from the first commit did not reach the working tree")
	}
	a.moveHistorySelection(1)
	if a.worktreeSelected {
		t.Fatal("the working tree is still selected after the arrow down")
	}
	if a.selectedSHA != "aaaa" {
		t.Fatalf("selected commit = %q; want aaaa", a.selectedSHA)
	}
	// And the bottom of the list stops there instead of running past it.
	a.moveHistorySelection(1)
	a.moveHistorySelection(1)
	if a.selectedSHA != "bbbb" {
		t.Fatalf("the end of the list selected %q; want the last commit", a.selectedSHA)
	}
}

// The working tree's Summary tab holds the commit box: that is what the row
// was chosen for.
func TestTheWorktreeSummaryHoldsTheCommitBox(t *testing.T) {
	a := worktreePageApp()
	a.worktreeSelected = true
	tt := pageTester(a, a.historyPage)
	if _, ok := tt.Find(a.t("commit")); !ok {
		t.Fatal("the working tree's details have no Summary tab")
	}
	box := false
	for _, info := range tt.AccessAll() {
		if info.Placeholder == a.t("commitMessage") {
			box = true
			break
		}
	}
	if !box {
		t.Fatal("the working tree's Summary does not hold the commit box")
	}
}

// Hiding the patch in Settings takes the file tab away rather than opening
// the pane on an empty column.
func TestHidingThePatchLeavesNoFileTab(t *testing.T) {
	a := worktreePageApp()
	a.selectedSHA = "aaaa"
	a.commitPath = "a.go"
	a.detailTabs().Open(detailFileKey{Path: a.detailFilePath(), Staged: a.selectedStaged})
	if got := a.detailTabShown(); got != detailTabFile {
		t.Fatalf("with the patch shown the pane shows %q; want the file", got)
	}
	a.detailsDiffHidden = true
	if got := a.detailTabShown(); got != detailTabSummary {
		t.Fatalf("with the patch hidden the pane shows %q; want the Summary", got)
	}
	if a.detailFileShown() {
		t.Fatal("the file tab is still offered while the patch is hidden")
	}
}

// The working tree keeps its Summary even when Settings hides the commit
// message: the commit box lives there.
func TestTheWorktreeKeepsItsSummaryWhenInfoIsHidden(t *testing.T) {
	a := worktreePageApp()
	a.worktreeSelected = true
	a.detailsInfoHidden = true
	if got := a.detailTabShown(); got != detailTabSummary {
		t.Fatalf("the working tree shows %q; want the Summary", got)
	}
	if !a.detailSummaryShown() {
		t.Fatal("the working tree's Summary tab is hidden")
	}
}

// The files column follows the selection: a commit's files, then the working
// tree's once its row is chosen.
func TestTheFilesColumnFollowsTheSelection(t *testing.T) {
	a := worktreePageApp()
	a.selectedSHA = "aaaa"
	a.commitFiles = []gitrepo.ChangedFile{{Path: "committed.go", Status: "M"}}
	tt := pageTester(a, a.historyPage)
	if _, ok := tt.Find("committed.go"); !ok {
		t.Fatal("the commit's files are not in the files column")
	}
	if _, ok := tt.Find("a.go"); ok {
		t.Fatal("the working tree's file shows while a commit is selected")
	}
	a.selectWorktree()
	tt.Frame()
	if _, ok := tt.Find("a.go"); !ok {
		t.Fatal("the working tree's file is not in the files column after choosing its row")
	}
	if _, ok := tt.Find("committed.go"); ok {
		t.Fatal("the commit's file still shows while the working tree is selected")
	}
}

// A reload keeps the working tree's row selected: auto-picking the first
// commit would mark two rows at once and hand the arrows a commit the reader
// never chose.
func TestAReloadKeepsTheWorkingTreeSelected(t *testing.T) {
	a := worktreePageApp()
	a.worktreeSelected = true
	a.reconcileSelection()
	if a.selectedSHA != "" {
		t.Fatalf("a reload selected commit %q behind the working tree", a.selectedSHA)
	}
	if !a.worktreeSelected {
		t.Fatal("a reload dropped the working tree's selection")
	}
}

// Choosing the working tree's row does not move the file the workspace has
// open: only a file that is gone from the working tree is replaced.
func TestChoosingTheWorktreeKeepsTheOpenFile(t *testing.T) {
	a := worktreePageApp()
	a.status.Entries = append(a.status.Entries, gitrepo.Entry{Path: "b.go", Index: ' ', Worktree: 'M'})
	a.selectedPath, a.selectedStaged = "b.go", false
	a.selectWorktree()
	if a.selectedPath != "b.go" {
		t.Fatalf("the working tree replaced the open file with %q", a.selectedPath)
	}

	a.selectedPath = "gone.go"
	a.worktreeSelected = false
	a.selectWorktree()
	if a.selectedPath == "gone.go" || a.selectedPath == "" {
		t.Fatalf("a file that left the working tree stayed selected: %q", a.selectedPath)
	}
}

// A search that matches no commit still leaves the working tree's row on
// screen: the commit box is reached through it.
func TestTheWorktreeRowSurvivesAnEmptySearch(t *testing.T) {
	a := worktreePageApp()
	a.historyQuery = "no such commit"
	tt := pageTester(a, a.historyPage)
	if _, ok := tt.AccessByRole(ui.RoleOption, a.changesLabel(1)); !ok {
		t.Fatal("an empty search took the working tree's row away")
	}
	if _, ok := tt.Find(a.t("searchNoMatch")); !ok {
		t.Fatal("the empty search shows no placeholder")
	}
}

// ⌘↩ commits from the history page while the working tree's docked commit box
// is on screen, and stays quiet while that column is hidden or covered.
func TestCommitShortcutReachesTheHistoryCommitBox(t *testing.T) {
	a := worktreePageApp()
	a.worktreeSelected = true
	if a.canCommitShortcut() {
		t.Fatal("commit shortcut accepts a missing repository")
	}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.busy = true
	if a.canCommitShortcut() {
		t.Fatal("commit shortcut accepts a busy repository")
	}
	a.busy = false
	a.loading = true
	if a.canCommitShortcut() {
		t.Fatal("commit shortcut accepts a repository reload")
	}
	a.loading = false
	a.detailTabs().Activate(detailTabKey{Summary: true})
	if !a.canCommitShortcut() {
		t.Fatal("⌘↩ does nothing on the working tree's Summary")
	}
	a.selectedPath = "a.go"
	a.detailTabs().Open(detailFileKey{Path: a.selectedPath})
	if !a.canCommitShortcut() {
		t.Fatal("⌘↩ stops working while a file tab is in front of the docked commit box")
	}
	a.detailTabs().Activate(detailTabKey{Summary: true})
	a.historyRefOpen = true
	if a.canCommitShortcut() {
		t.Fatal("⌘↩ commits behind the ref picker, which covers the commit box")
	}
	a.historyRefOpen = false
	a.historyDetailsHidden = true
	if a.canCommitShortcut() {
		t.Fatal("⌘↩ commits while the details pane, and the commit box with it, is hidden")
	}
	a.historyDetailsHidden = false
	if !a.canCommitShortcut() {
		t.Fatal("the commit box is back in front but ⌘↩ still refuses")
	}
	a.worktreeSelected = false
	a.page = "history"
	if a.canCommitShortcut() {
		t.Fatal("⌘↩ commits from a commit's Summary, which has no commit box")
	}
}

// The files column gives width back to the patch when the details are narrow,
// and caps at the default width when there is room.
func TestTheFilesColumnYieldsToThePatch(t *testing.T) {
	a := &app{}
	a.historySplit.rowWidth = 1500
	if got := a.historyFilesWidth(); got < historyFilesPaneWidth-0.01 || got > historyFilesPaneWidth+0.01 {
		t.Fatalf("wide details give the files %v; want %v", got, historyFilesPaneWidth)
	}
	a.historySplit.rowWidth = 700
	if got := a.historyFilesWidth(); got >= historyFilesPaneWidth {
		t.Fatalf("narrow details still give the files %v", got)
	}
	if got := a.historyFilesWidth(); got < historyFilesPaneMin {
		t.Fatalf("the files column shrank to %v, under %v", got, historyFilesPaneMin)
	}
}

// The file tab shows only the base name, so it has to carry the full path
// somewhere: the tooltip is what tells two same-named files apart.
func TestTheFileTabNamesItsFullPath(t *testing.T) {
	a := &app{selectedPath: "src/nested/this-file-name-is-long-enough-to-overflow-a-narrow-strip.go"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Width(300).Height(38).Children(func() {
			a.detailTabButton(c, theme.TokensOf(c), detailTabKey{File: detailFileKey{Path: a.selectedPath}})
		})
	}, 300, 38)
	tab, ok := tt.AccessByRole(ui.RoleTab, "this-file-name-is-long-enough-to-overflow-a-narrow-strip.go")
	if !ok {
		t.Fatal("the file tab is not in the accessibility tree")
	}
	if tab.Description != "src/nested/this-file-name-is-long-enough-to-overflow-a-narrow-strip.go" {
		t.Errorf("the file tab describes %q; want the full path", tab.Description)
	}
	if tab.Bounds.W > 300 {
		t.Errorf("the file tab is %v wide inside a 300-wide strip", tab.Bounds.W)
	}
	if tab.Bounds.H != 36 {
		t.Errorf("the tab is %v high; want the full 36-point content row", tab.Bounds.H)
	}
}

// A file row clips its folder when the column is narrow, so the full path
// lives in a tooltip, as the diff tab's does.
func TestACommitFileRowTooltipsItsFullPath(t *testing.T) {
	a := &app{commitFiles: []gitrepo.ChangedFile{{Path: "src/nested/a.go", Status: "M"}}}
	tt := pageTester(a, a.commitFileList)
	row, ok := tt.AccessByRole(ui.RoleOption, "src/nested/a.go")
	if !ok {
		t.Fatal("the commit file row is not in the accessibility tree")
	}
	if row.Description != "src/nested/a.go" {
		t.Errorf("the file row describes %q; want the full path", row.Description)
	}
}

// The patch keeps its own 300 at the smallest supported window. The details
// minimum used to exceed the room left by the list minimum, so the split fell
// back to an even share and left the patch about 207.
func TestThePatchKeepsItsWidthAtTheSmallestWindow(t *testing.T) {
	for _, window := range []float32{980, 1031, 1320} {
		a := &app{}
		row := window - sidebarWidth - splitterWidth
		a.historySplit.rowWidth = row
		share := a.historyShare()
		list := (row - splitterWidth) * share
		details := (row - splitterWidth) * (1 - share)
		diff := details - splitterWidth - a.historyFilesWidth()
		if list < historySplitMinList {
			t.Errorf("window %v gives the commit list %v, under %v", window, list, historySplitMinList)
		}
		// Layout shares are float32; a sub-pixel shortfall is not the bug this
		// guards (the old split left it more than 90 points short).
		if diff < historyDiffPaneMin-0.5 {
			t.Errorf("window %v gives the patch %v, under %v", window, diff, historyDiffPaneMin)
		}
	}
}

func TestNarrowHistoryKeepsSpaceForCommitSubjectsAndPatch(t *testing.T) {
	a := &app{page: "history"}
	for _, window := range []float32{980, 1320, 1600, 980} {
		a.sidebarSplit.rowWidth = window
		a.historySplit.rowWidth = window - a.workbenchSidebarWidth() - splitterWidth
		list := a.commitListWidth()
		a.filesSplit.rowWidth = a.historySplit.rowWidth - splitterWidth - list
		files := a.historyFilesWidth()
		patch := a.filesSplit.rowWidth - splitterWidth - files
		if window == 980 && (list < 368 || files < 100 || files > 116) {
			t.Errorf("window %.0f: commits need at least 368 points and Files a compact usable rail, got commits %.1f and Files %.1f", window, list, files)
		}
		if window >= 1320 && (files < historyFilesPaneMin-0.5 || list < 0.44*(a.historySplit.rowWidth-splitterWidth)) {
			t.Errorf("window %.0f: the wide layout did not expand normally: commits %.1f, Files %.1f", window, list, files)
		}
		if patch < a.historyPatchMinWidth()-0.5 {
			t.Errorf("window %.0f: patch fell to %.1f below its %.1f minimum", window, patch, a.historyPatchMinWidth())
		}
		if a.historySplit.share != 0 || a.filesSplit.share != 0 {
			t.Fatal("temporary resize overwrote preferred pane widths")
		}
	}
}

// A saved setting that hid both panes the files column opens into is repaired
// on load: the files list needs somewhere to open a file.
func TestHiddenSummaryAndPatchAreRepairedOnLoad(t *testing.T) {
	hidden := false
	a := &app{}
	a.applySettings(persistedSettings{DetailsInfo: &hidden, DetailsDiff: &hidden})
	if a.detailsInfoHidden && a.detailsDiffHidden {
		t.Fatal("both panes stayed hidden after load")
	}
}

// At 980 the commit column is about 309 wide. A row with HEAD and three refs
// at the natural lane gap and three full capsules used most of it, leaving the
// subject a few characters. The row now budgets the graph, the capsules and
// the chips by width.
func TestTheNarrowCommitRowKeepsRoomForTheSubject(t *testing.T) {
	const window = 980
	// Seven pending parents keep seven auxiliary lanes live beside the first
	// one, the eight-lane case the design review measured.
	parents := []string{"main"}
	for i := 0; i < 7; i++ {
		parents = append(parents, "side-"+itoa(i))
	}
	head := gitrepo.Commit{
		SHA:     "head",
		Subject: "fix the narrow commit row width budget",
		Refs:    "HEAD -> main, tag: v1, tag: v2, tag: v3",
		Parents: parents,
	}
	commits := []gitrepo.Commit{head}
	for _, parent := range parents {
		commits = append(commits, gitrepo.Commit{SHA: parent, Subject: "side " + parent})
	}
	a := &app{locale: localeChinese, page: "history", commits: commits, selectedSHA: head.SHA, showGraph: true}
	a.historySplit.rowWidth = window - sidebarWidth
	content := float32(window - sidebarWidth)
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(content).Height(700).Children(func() {
			a.historyPage(c)
		})
	}, int(content), 700)
	// The page reads the row's width from the previous frame, so the second
	// frame is the one drawn with the real column width.
	tt.Frame()
	list := a.commitListWidth()
	if list < 300 {
		t.Fatalf("the commit list is %v wide; want about 309", list)
	}
	rect, ok := tt.Find(head.Subject)
	if !ok {
		t.Fatal("the crowded row does not show its subject")
	}
	if rect.W < 96 {
		t.Errorf("the subject gets %v of the row; want at least 96", rect.W)
	}
	if a.historyGraph.Width > list*0.26+1 {
		t.Errorf("the graph takes %v of the %v list; want at most a quarter", a.historyGraph.Width, list)
	}
	if a.historyGraph.Lanes != 8 {
		t.Fatalf("the fixture has %d lanes; want the crowded eight", a.historyGraph.Lanes)
	}
	row, ok := tt.AccessByRole(ui.RoleOption, head.Subject)
	if !ok {
		t.Fatal("the crowded row is not in the list")
	}
	for _, label := range []string{"main", "+3"} {
		pill, ok := tt.Find(label)
		if !ok {
			t.Fatalf("the row does not show %q", label)
		}
		if pill.X+pill.W > row.Bounds.X+row.Bounds.W+0.5 {
			t.Errorf("the %q capsule ends at %v, past the row's %v", label, pill.X+pill.W, row.Bounds.X+row.Bounds.W)
		}
	}
	if head, ok := tt.Find("HEAD"); ok && head.X < row.Bounds.X+row.Bounds.W {
		// The details pane has room for the word; the row does not.
		t.Error("the narrow row still spends a chip on the word HEAD")
	}
	graphWidth := a.historyGraph.Width
	a.historyDetailsHidden = true
	tt.Frame()
	tt.Frame()
	if expanded := a.commitListWidth(); expanded < content-1 || a.historyGraph.Width <= graphWidth || refSlotsForWidth(expanded) != 3 {
		t.Fatalf("hidden details retained the old width budget: list=%v graph=%v", expanded, a.historyGraph.Width)
	}
	if expanded, ok := tt.Find(head.Subject); !ok || expanded.W < rect.W {
		t.Fatalf("expanded row made the subject less readable: %+v", expanded)
	}
	a.historyDetailsHidden = false
	tt.Frame()
	tt.Frame()
	if a.commitListWidth() != list || a.historyGraph.Width != graphWidth {
		t.Fatal("restoring details discarded the split width")
	}
}
