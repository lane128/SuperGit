package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestFileSelectionBarCountsRangesCopiesAndClears(t *testing.T) {
	a := &app{page: "changes", locale: localeEnglish, worktreeSelected: true, repo: &gitrepo.Repository{Root: t.TempDir()}, selectedPath: "same.go",
		status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "same.go", Index: 'M', Worktree: 'M'}}},
		diff: gitrepo.Diff{Text: "selected patch", CanStageHunks: true, Lines: []gitrepo.DiffLine{
			{Kind: "hunk", Text: "@@ -3 +3 @@"}, {Kind: "del", Old: 3, Text: "-old"}, {Kind: "add", New: 3, Text: "+new"},
		}}, lineSel: map[int]bool{1: true, 2: true},
	}
	a.detailTabs().Open(detailFileKey{Path: "same.go"})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); ui.Column(c).Fill().Children(func() { a.historyDiffPane(c) }) }, 300, 480)
	if !tt.HasText("2 selected · Old 3 · New 3") {
		t.Fatal("visible selection count/range is missing")
	}
	stage, ok := tt.AccessByRole(ui.RoleButton, a.t("stageSelection"))
	if !ok || stage.Disabled || stage.Bounds.X < 0 || stage.Bounds.X+stage.Bounds.W > 300 {
		t.Fatal("selection stage action is missing, disabled or clipped")
	}
	copied := ""
	previous := writeClipboard
	writeClipboard = func(text string) error { copied = text; return nil }
	t.Cleanup(func() { writeClipboard = previous })
	copy, ok := tt.AccessByRole(ui.RoleButton, a.copyLinesLabel(2))
	if !ok {
		t.Fatal("selection has no copy action")
	}
	tt.ClickAt(copy.Bounds.X+copy.Bounds.W/2, copy.Bounds.Y+copy.Bounds.H/2)
	if copied != "old\nnew" {
		t.Fatalf("copied wrong selection: %q", copied)
	}
	a.busy = true
	tt.Frame()
	stage, _ = tt.AccessByRole(ui.RoleButton, a.t("stageSelection"))
	if !stage.Disabled {
		t.Fatal("selection write remains enabled while busy")
	}
	clear, ok := tt.AccessByRole(ui.RoleButton, a.t("shortcutClearSelection"))
	if !ok {
		t.Fatal("selection has no clear action")
	}
	tt.ClickAt(clear.Bounds.X+clear.Bounds.W/2, clear.Bounds.Y+clear.Bounds.H/2)
	tt.Frame()
	if len(a.lineSel) != 0 || tt.HasText("2 selected · Old 3 · New 3") {
		t.Fatal("clear left selection or action bar behind")
	}
}

func TestOverviewSelectionBarUsesOriginalVersionAndRejectsOldRoot(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page, a.locale = "changes", localeEnglish
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	var entry worktreeOverviewEntry
	for _, candidate := range a.overviewEntries {
		if candidate.Key.Staged {
			entry = candidate
			break
		}
	}
	target := overviewPatchTarget{Root: a.overviewRoot, Generation: a.overviewSeq, File: entry.Key, Diff: entry.Diff}
	indexes := changedLineIndexes(entry.Diff)
	a.selectOverviewLine(target, indexes[0], 0)
	// The background single-file cache describes a different version.
	a.selectedPath, a.selectedStaged = "background.go", false
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); ui.Column(c).Fill().Children(func() { a.historyDiffPane(c) }) }, 500, 480)
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("unstageSelection")); !ok {
		t.Fatal("overview bar followed background file instead of staged target")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("stageSelection")); ok {
		t.Fatal("overview bar offers wrong stage direction")
	}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("unstageSelection")); ok {
		t.Fatal("old repository retains selection action")
	}
}
