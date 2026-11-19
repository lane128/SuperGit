package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func historicalSelectionFixture(t *testing.T) (*app, *ui.Tester) {
	t.Helper()
	a := &app{ctx: canceledContext(), runOnMain: func(func()) {}, repo: &gitrepo.Repository{Root: t.TempDir()}, locale: localeEnglish, page: "history", selectedSHA: "selected-commit", commitFilesLoaded: true, commitFiles: []gitrepo.ChangedFile{{Path: "a.txt", Status: "A"}, {Path: "b.txt", Status: "A"}, {Path: "c.txt", Status: "A"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(500).Height(250).Children(func() { a.commitFileList(c) })
	}, 500, 250)
	return a, tt
}

func clickHistoricalFile(t *testing.T, tt *ui.Tester, path string, mods ui.Modifiers) {
	t.Helper()
	row, ok := tt.AccessByRole(ui.RoleOption, path)
	if !ok {
		t.Fatalf("missing file %s", path)
	}
	tt.ClickAtWith(mods, row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
}

func assertHistoricalFiles(t *testing.T, tt *ui.Tester, selected ...string) {
	t.Helper()
	want := map[string]bool{}
	for _, path := range selected {
		want[path] = true
	}
	for _, row := range tt.AccessAll() {
		if row.Role == ui.RoleOption && row.Selected != want[row.Name] {
			t.Errorf("historical %s selected=%t want=%t", row.Name, row.Selected, want[row.Name])
		}
	}
	for path := range want {
		if _, ok := tt.AccessByRole(ui.RoleOption, path); !ok {
			t.Errorf("selected file %s missing", path)
		}
	}
}

// Gestures reuse SuperGit workspace conventions; native reference modifier
// semantics remain a separate acceptance boundary.
func TestHistoricalFilesExtendSelection(t *testing.T) {
	for _, gesture := range []struct {
		name       string
		mods       ui.Modifiers
		wantMiddle bool
	}{{"toggle", ui.Cmd, false}, {"range", ui.Shift, true}} {
		t.Run(gesture.name, func(t *testing.T) {
			a, tt := historicalSelectionFixture(t)
			clickHistoricalFile(t, tt, "a.txt", 0)
			clickHistoricalFile(t, tt, "c.txt", gesture.mods)
			want := []string{"a.txt", "c.txt"}
			if gesture.wantMiddle {
				want = []string{"a.txt", "b.txt", "c.txt"}
			}
			assertHistoricalFiles(t, tt, want...)
			if a.commitPath != "c.txt" {
				t.Fatalf("primary preview=%q want c.txt", a.commitPath)
			}
		})
	}
}

func TestHistoricalFilesTogglePrimaryAndPlainReset(t *testing.T) {
	a, tt := historicalSelectionFixture(t)
	clickHistoricalFile(t, tt, "a.txt", 0)
	clickHistoricalFile(t, tt, "c.txt", ui.Cmd)
	clickHistoricalFile(t, tt, "c.txt", ui.Cmd)
	assertHistoricalFiles(t, tt, "a.txt")
	if a.commitPath != "a.txt" {
		t.Fatal("removing primary did not preview remaining file")
	}
	clickHistoricalFile(t, tt, "a.txt", ui.Cmd)
	assertHistoricalFiles(t, tt)
	if !a.detailTabs().Active.Summary {
		t.Fatal("empty membership retained file preview")
	}
	clickHistoricalFile(t, tt, "b.txt", ui.Shift)
	assertHistoricalFiles(t, tt, "b.txt")
	clickHistoricalFile(t, tt, "a.txt", 0)
	clickHistoricalFile(t, tt, "c.txt", ui.Shift)
	assertHistoricalFiles(t, tt, "a.txt", "b.txt", "c.txt")
	clickHistoricalFile(t, tt, "b.txt", 0)
	assertHistoricalFiles(t, tt, "b.txt")
	a.activateDetailTab(detailTabKey{Summary: true})
	tt.Frame()
	assertHistoricalFiles(t, tt, "b.txt")
	a.selectCommitFile("a.txt")
	tt.Frame()
	assertHistoricalFiles(t, tt, "a.txt")
}

func TestHistoricalFilesReconcileRefreshAndContext(t *testing.T) {
	a, tt := historicalSelectionFixture(t)
	clickHistoricalFile(t, tt, "a.txt", 0)
	clickHistoricalFile(t, tt, "c.txt", ui.Cmd)
	a.commitFilesLoading = true
	a.commitFiles = nil
	tt.Frame()
	a.commitFilesLoading = false
	a.commitFiles = []gitrepo.ChangedFile{{Path: "a.txt", Status: "A"}, {Path: "b.txt", Status: "A"}}
	tt.Frame()
	assertHistoricalFiles(t, tt, "a.txt")
	if a.commitPath != "a.txt" {
		t.Fatal("refresh kept removed primary")
	}
	files := a.commitFiles
	a.selectWorktree()
	a.selectCommit("selected-commit")
	a.commitFiles, a.commitFilesLoaded, a.commitFilesLoading = files, true, false
	tt.Frame()
	assertHistoricalFiles(t, tt, "a.txt")
	a.selectCommit("another-commit")
	a.commitFiles, a.commitFilesLoaded, a.commitFilesLoading = files, true, false
	tt.Frame()
	assertHistoricalFiles(t, tt)
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt.Frame()
	assertHistoricalFiles(t, tt)
}

func TestHistoricalFilesRightClickKeepsOnlyExplicitFileAction(t *testing.T) {
	a, tt := historicalSelectionFixture(t)
	clickHistoricalFile(t, tt, "a.txt", 0)
	clickHistoricalFile(t, tt, "c.txt", ui.Cmd)
	for _, path := range []string{"c.txt", "b.txt"} {
		row, ok := tt.AccessByRole(ui.RoleOption, path)
		if !ok {
			t.Fatal("row missing")
		}
		tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
		if path == "c.txt" {
			assertHistoricalFiles(t, tt, "a.txt", "c.txt")
		} else {
			assertHistoricalFiles(t, tt, "b.txt")
		}
		if a.commitPath != path {
			t.Fatal("context target did not become primary preview")
		}
		tt.Key(0, ui.KeyEscape)
	}
}
