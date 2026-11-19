package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestWorkspaceSideBySideKeepsCanonicalLineSelection(t *testing.T) {
	a := &app{locale: localeEnglish, selectedPath: "notes.txt", lineCursor: 3, linePressIndex: -1, lineSel: map[int]bool{3: true}}
	a.diff = gitrepo.Diff{Text: "patch", Lines: []gitrepo.DiffLine{
		{Kind: "hunk", Text: "@@ -1,2 +1,3 @@"},
		{Kind: "meta", Text: "diff --git a/dir-with-a-long-name/notes.txt b/dir-with-a-long-name/notes.txt"},
		{Kind: "context", Text: " common", Old: 1, New: 1},
		{Kind: "del", Text: "-old text", Old: 2},
		{Kind: "add", Text: "+replacement", New: 2},
		{Kind: "add", Text: "+extra line", New: 3},
	}}
	view := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() {
			a.worktreeDiffHeader(c, theme.TokensOf(c))
			a.diffBody(c)
		})
	}, 480, 280)
	clickRepositoryTabsControl(t, view, ui.RoleButton, "Show side-by-side diff")
	old, oldOK := view.Find("-old text")
	replacement, replacementOK := view.Find("+replacement")
	extra, extraOK := view.Find("+extra line")
	if !oldOK || !replacementOK || !extraOK || old.Y != replacement.Y || old.X >= replacement.X || extra.Y <= replacement.Y || extra.X != replacement.X || replacement.W <= 0 || replacement.X+replacement.W > 480 {
		t.Fatalf("replacement should align with the original, with the remaining addition on the right: old=%+v replacement=%+v extra=%+v", old, replacement, extra)
	}
	if !a.lineSel[3] || len(a.lineSel) != 1 || a.lineCursor != 3 {
		t.Fatal("rendering a paired row changed the selected source line")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, "Show unified diff")
	old, _ = view.Find("-old text")
	replacement, _ = view.Find("+replacement")
	if old.Y >= replacement.Y || old.X != replacement.X || !a.lineSel[3] {
		t.Fatal("returning to unified diff should restore consecutive rows and retain the selected source line")
	}
}

func TestWorkspaceLayoutToggleKeepsTheVisibleHunk(t *testing.T) {
	a := &app{locale: localeEnglish, selectedPath: "notes.txt", lineCursor: -1, linePressIndex: -1}
	lines := []gitrepo.DiffLine{{Kind: "hunk", Text: "@@ -1,14 +1,14 @@"}}
	for range 14 {
		lines = append(lines, gitrepo.DiffLine{Kind: "del", Text: "-old"})
	}
	for range 14 {
		lines = append(lines, gitrepo.DiffLine{Kind: "add", Text: "+new"})
	}
	lines = append(lines, gitrepo.DiffLine{Kind: "hunk", Hunk: 1, Text: "@@ -80 +80 @@"})
	for range 60 {
		lines = append(lines, gitrepo.DiffLine{Kind: "context", Hunk: 1, Text: " context"})
	}
	a.diff = gitrepo.Diff{Text: "patch", Lines: lines}
	view := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() {
			a.worktreeDiffHeader(c, theme.TokensOf(c))
			a.diffBody(c)
		})
	}, 480, 280)
	a.diffScroll.Y = 655
	view.Frame()
	before, ok := view.Find("@@ -80 +80 @@")
	if !ok || before.H <= 0 {
		t.Fatalf("the second hunk must be visible before switching: %+v", before)
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, "Show side-by-side diff")
	after, ok := view.Find("@@ -80 +80 @@")
	if !ok || after.H <= 0 || before.Y != after.Y {
		t.Fatalf("the second hunk moved when switching layouts: before=%+v after=%+v", before, after)
	}
}

func TestSideBySideCodeDragDoesNotCopyTheOtherColumn(t *testing.T) {
	a := &app{splitDiff: true, selectedPath: "notes.txt", linePressIndex: -1}
	a.diff = gitrepo.Diff{Text: "patch", Lines: []gitrepo.DiffLine{
		{Kind: "del", Text: "-old first", Old: 1},
		{Kind: "del", Text: "-old second", Old: 2},
		{Kind: "add", Text: "+new first", New: 1},
		{Kind: "add", Text: "+new second", New: 2},
	}}
	view := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() { a.diffBody(c) })
	}, 480, 280)
	first, okFirst := view.Find("-old first")
	second, okSecond := view.Find("-old second")
	if !okFirst || !okSecond {
		t.Fatal("missing old-side code")
	}
	view.Press(first.X+2, first.Y+first.H/2)
	view.Move(first.X+first.W-2, first.Y+first.H/2)
	view.Move(second.X+second.W-2, second.Y+second.H/2)
	view.Release(second.X+second.W-2, second.Y+second.H/2)
	view.Key(ui.Cmd, ui.KeyC)
	if got := view.Clipboard(); got != "-old first\n-old second" {
		t.Fatalf("copy from old-side code produced %q", got)
	}
}

func TestSideBySideShiftSelectsOnlyThePairedCells(t *testing.T) {
	lines := []gitrepo.DiffLine{
		{Kind: "hunk", Text: "@@ -1,2 +1,2 @@"},
		{Kind: "del", Text: "-old one", Old: 1},
		{Kind: "del", Text: "-old two", Old: 2},
		{Kind: "add", Text: "+new one", New: 1},
		{Kind: "add", Text: "+new two", New: 2},
	}
	diff := gitrepo.Diff{Lines: lines}
	t.Run("workspace", func(t *testing.T) {
		a := &app{splitDiff: true, diff: diff}
		a.clickLine(1, 0)
		a.clickLine(3, ui.Shift)
		if !a.lineSel[1] || !a.lineSel[3] || len(a.lineSel) != 2 {
			t.Fatalf("workspace selection = %v, want only the first paired row", a.lineSel)
		}
	})
	t.Run("history", func(t *testing.T) {
		a := &app{splitDiff: true, page: "history"}
		a.commitDiff, a.commitSections = buildCommitDiff([]gitrepo.FileDiff{{Path: "notes.txt", Diff: diff}})
		a.clickCommitLine(3, 0)
		a.clickCommitLine(1, ui.Shift)
		if !a.commitSel[1] || !a.commitSel[3] || len(a.commitSel) != 2 {
			t.Fatalf("history selection = %v, want only the first paired row", a.commitSel)
		}
	})
}

func TestSideBySideStagesAndUnstagesOnlyTheRightCell(t *testing.T) {
	a, callbacks, root := worktreeApp(t)
	a.page, a.locale, a.worktreeSelected = "changes", localeEnglish, true
	const path = "base.txt"
	const working = "replacement\nextra\n"
	if err := os.WriteFile(filepath.Join(root, path), []byte(working), 0644); err != nil {
		t.Fatal(err)
	}
	load := func(staged bool) {
		t.Helper()
		var err error
		a.status, err = a.repo.Status(a.ctx)
		if err != nil {
			t.Fatal(err)
		}
		a.selectedPath, a.selectedStaged = path, staged
		a.diff, err = a.repo.Diff(a.ctx, path, staged)
		if err != nil {
			t.Fatal(err)
		}
		a.detailTabs().Open(detailFileKey{Path: path, Staged: staged})
		a.diffLoading = false
		a.clearLineSelection()
	}
	a.splitDiff = true
	for _, step := range []struct {
		staged      bool
		label, want string
	}{
		{false, "Stage line", "base\nreplacement\n"},
		{true, "Unstage line", "base\n"},
	} {
		load(step.staged)
		view := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			ui.Column(c).Fill().Children(func() { a.diffBody(c) })
		}, 480, 300)
		gutter := "1 Added line"
		if step.staged {
			gutter = "2 Added line"
		}
		clickRepositoryTabsControl(t, view, ui.RoleButton, gutter)
		view.Key(0, ui.KeyContextMenu)
		clickRepositoryTabsControl(t, view, ui.RoleMenuItem, step.label)
		if !a.busy {
			t.Fatal("expected a Git write for the requested cell")
		}
		for a.busy || a.loading {
			runContinuation(t, callbacks)
		}
		if got := splitCommitGit(t, root, "show", ":base.txt"); got != step.want {
			t.Fatalf("Index after %s = %q, want %q", step.label, got, step.want)
		}
	}
	if got, err := os.ReadFile(filepath.Join(root, path)); err != nil || string(got) != working {
		t.Fatalf("working file = %q, %v", got, err)
	}
}

func TestHistorySideBySidePairsRowsAndKeepsActionsReadOnly(t *testing.T) {
	a := &app{locale: localeEnglish, page: "history", splitDiff: true, commitPressIndex: -1}
	a.commitDiff, a.commitSections = buildCommitDiff([]gitrepo.FileDiff{{Path: strings.Repeat("long-directory/", 8) + "notes.txt", Diff: gitrepo.Diff{Lines: []gitrepo.DiffLine{
		{Kind: "hunk", Text: "@@ -1 +1 @@ " + strings.Repeat("long hunk title ", 8)},
		{Kind: "del", Text: "-old version", Old: 1},
		{Kind: "add", Text: "+new version", New: 1},
	}}}})
	view := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() { a.commitDiffView(c) })
	}, 302, 280)
	left, okLeft := view.Find("-old version")
	right, okRight := view.Find("+new version")
	if !okLeft || !okRight || left.Y != right.Y || left.X >= right.X || right.W < left.W-.5 {
		t.Fatalf("history cells do not share a row: left=%+v right=%+v", left, right)
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, "1 Added line")
	view.Key(0, ui.KeyContextMenu)
	view.Frame()
	if !a.commitSel[2] || len(a.commitSel) != 1 {
		t.Fatalf("right cell selected a different source line: %v", a.commitSel)
	}
	if _, ok := view.AccessByRole(ui.RoleMenuItem, a.t("copyLine")); !ok {
		t.Fatal("missing copy action on the right cell")
	}
	if _, ok := view.AccessByRole(ui.RoleMenuItem, a.t("stageLine")); ok {
		t.Fatal("history exposed a mutating action")
	}
}
