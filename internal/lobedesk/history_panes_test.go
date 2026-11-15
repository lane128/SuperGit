package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

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

// The patch's own header keeps the stage action and the file menu that used
// to sit in the workspace's diff pane. The file tab above already names the
// file, so the header adds the folder instead of repeating the name.
func TestTheWorkingTreePatchKeepsItsStageAction(t *testing.T) {
	a := worktreePageApp()
	a.worktreeSelected = true
	a.selectedPath = "dir/a.go"
	a.status.Entries[0].Path = "dir/a.go"
	a.detailTab = detailTabFile
	tt := pageTester(a, func(c *ui.Context) { a.historyDiffFile(c, theme.TokensOf(c)) })
	if _, ok := tt.Find(a.t("stageFile")); !ok {
		t.Fatal("the working tree's patch has no stage action")
	}
	if _, ok := tt.Find("dir"); !ok {
		t.Error("the header does not say which folder the file is in")
	}
	if _, ok := tt.Find("a.go"); ok {
		t.Error("the header repeats the file name the tab already shows")
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
	a.moveCommitSelection(-1)
	if !a.worktreeSelected {
		t.Fatal("the arrow up from the first commit did not reach the working tree")
	}
	a.moveCommitSelection(1)
	if a.worktreeSelected {
		t.Fatal("the working tree is still selected after the arrow down")
	}
	if a.selectedSHA != "aaaa" {
		t.Fatalf("selected commit = %q; want aaaa", a.selectedSHA)
	}
	// And the bottom of the list stops there instead of running past it.
	a.moveCommitSelection(1)
	a.moveCommitSelection(1)
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
	if _, ok := tt.Find(a.t("summaryTab")); !ok {
		t.Fatal("the working tree's details have no Summary tab")
	}
	box := false
	for _, info := range tt.AccessAll() {
		if info.Placeholder == a.t("commitPlaceholder") {
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
	a.detailTab = detailTabFile
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

// ⌘↩ commits from the history page while the working tree's Summary holds the
// commit box, and stays quiet while another tab covers it.
func TestCommitShortcutReachesTheHistoryCommitBox(t *testing.T) {
	a := worktreePageApp()
	a.worktreeSelected = true
	a.detailTab = detailTabSummary
	if !a.canCommitShortcut() {
		t.Fatal("⌘↩ does nothing on the working tree's Summary")
	}
	a.detailTab = detailTabFile
	a.selectedPath = "a.go"
	if a.canCommitShortcut() {
		t.Fatal("⌘↩ commits while the file tab covers the commit box")
	}
	a.detailTab = detailTabSummary
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
// and never takes more than its own 300.
func TestTheFilesColumnYieldsToThePatch(t *testing.T) {
	a := &app{}
	a.historySplit.rowWidth = 1500
	if got := a.historyFilesWidth(); got != historyFilesPaneWidth {
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
	a := &app{worktreeSelected: true, selectedPath: "src/nested/this-file-name-is-long-enough-to-overflow-a-narrow-strip.go"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Width(300).Height(38).Children(func() {
			a.detailTabButton(c, theme.TokensOf(c), detailTabFile, "this-file-name-is-long-enough-to-overflow-a-narrow-strip.go", "src/nested/this-file-name-is-long-enough-to-overflow-a-narrow-strip.go")
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
	if tab.Bounds.H < 38 {
		t.Errorf("the tab is %v high; it should fill the strip", tab.Bounds.H)
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
		row := window - sidebarWidth
		a.historySplit.rowWidth = row
		share := a.historyShare()
		list := (row - splitterWidth) * share
		details := (row - splitterWidth) * (1 - share)
		diff := details - a.historyFilesWidth()
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
}
