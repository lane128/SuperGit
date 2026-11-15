package lobedesk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// Clicking the working-tree row is how the commit box is opened, and the row
// keeps the focus. Space and Enter are documented to stage and open the
// working tree's selected file, but the row's own handlers swallowed both, so
// the keys did nothing until the reader clicked somewhere else.
func TestSpaceOnTheFocusedWorktreeRowStagesItsFile(t *testing.T) {
	a, done, root := worktreeApp(t)
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.page = "history"
	a.status = gitrepo.Status{
		Branch:  "main",
		Entries: []gitrepo.Entry{{Path: "base.txt", Index: '.', Worktree: 'M'}},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(400).Height(80).Children(func() {
			a.historyWorktreeRow(c, 0, 1)
		})
	}, 400, 80)
	row, ok := tt.AccessByRole(ui.RoleOption, a.changesLabel(1))
	if !ok {
		t.Fatal("the working-tree row is not laid out")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if !a.worktreeSelected || a.selectedPath != "base.txt" {
		t.Fatalf("clicking the row selected worktree=%t path=%q", a.worktreeSelected, a.selectedPath)
	}
	if !tt.Focused(a.changesLabel(1)) {
		t.Fatal("the clicked row lost the focus; the test cannot reproduce the mismatch")
	}
	tt.Key(0, ui.KeySpace)
	runContinuation(t, done)
	repo, err := gitrepo.Open(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	status, err := repo.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Entries) != 1 || !status.Entries[0].Staged() {
		t.Errorf("Space left the working tree as %#v; want base.txt staged", status.Entries)
	}
}

// The same row keeps the focus while the arrows move the selection onto a
// commit. Space must not fall through to the working tree then: it would stage
// a file the reader is no longer looking at.
func TestSpaceOnTheFocusedRowLeavesACommitSelectionAlone(t *testing.T) {
	a, done, root := worktreeApp(t)
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.page = "history"
	a.status = gitrepo.Status{
		Branch:  "main",
		Entries: []gitrepo.Entry{{Path: "base.txt", Index: '.', Worktree: 'M'}},
	}
	a.worktreeSelected = true
	a.selectedPath = "base.txt"
	// The reader clicked the row, then pressed Down onto a commit.
	a.worktreeSelected = false
	a.commits = []gitrepo.Commit{{SHA: "aaaa", Subject: "first"}}
	a.selectedSHA = "aaaa"
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(400).Height(80).Children(func() {
			a.historyWorktreeRow(c, 0, 1)
		})
	}, 400, 80)
	row, ok := tt.AccessByRole(ui.RoleOption, a.changesLabel(1))
	if !ok {
		t.Fatal("the working-tree row is not laid out")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	// The click selected the working tree again; put the selection back on
	// the commit the arrows had reached.
	a.worktreeSelected = false
	a.selectedSHA = "aaaa"
	tt.Key(0, ui.KeySpace)
	select {
	case <-done:
		t.Fatal("Space started a staging operation while a commit row was selected")
	default:
	}
}
