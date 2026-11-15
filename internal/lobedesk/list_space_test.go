package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// listSpaceTester renders the rows the caller gives it and returns the tester
// plus the bounds of the row whose label is first.
func listSpaceTester(t *testing.T, rows func(c *ui.Context), first string) (*ui.Tester, ui.Rect) {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(320).Children(func() { rows(c) })
	}, 600, 320)
	row, ok := tt.AccessByRole(ui.RoleOption, first)
	if !ok {
		t.Fatalf("the %q row is not an option", first)
	}
	return tt, row.Bounds
}

// Space is how a button activates, so the list rows had the same trap Enter
// had: after the arrows moved the selection, Space on the clicked row put the
// selection back on it. Each row claims the key.
func TestSpaceOnAHistoryRowKeepsTheArrowSelection(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "aaaa", Subject: "first", Parents: []string{"root"}},
		{SHA: "bbbb", Subject: "second", Parents: []string{"aaaa"}},
	}
	a := &app{locale: localeChinese, page: "history", commits: commits}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt, bounds := listSpaceTester(t, func(c *ui.Context) {
		for i, commit := range commits {
			a.commitRow(c, commit, i, 0, len(commits), &commitGraph{})
		}
	}, "first")
	tt.ClickAt(bounds.X+bounds.W/2, bounds.Y+bounds.H/2)
	a.moveSelection(1)
	if a.selectedSHA != "bbbb" {
		t.Fatalf("the arrow selected %q; want bbbb", a.selectedSHA)
	}
	tt.Key(0, ui.KeySpace)
	if a.selectedSHA != "bbbb" {
		t.Errorf("Space on the focused row selected %q; want the arrow selection bbbb", a.selectedSHA)
	}
}

func TestSpaceOnAStashRowKeepsTheArrowSelection(t *testing.T) {
	stashes := []gitrepo.Stash{
		{SHA: "aaaa", Ref: "stash@{0}", Message: "first"},
		{SHA: "bbbb", Ref: "stash@{1}", Message: "second"},
	}
	a := &app{locale: localeChinese, page: "stash", stashes: stashes}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	tt, bounds := listSpaceTester(t, func(c *ui.Context) {
		for i, stash := range stashes {
			a.stashRow(c, stash, i+1, len(stashes))
		}
	}, "first")
	tt.ClickAt(bounds.X+bounds.W/2, bounds.Y+bounds.H/2)
	a.moveStashSelection(1)
	if a.selectedStash != "bbbb" {
		t.Fatalf("the arrow selected %q; want bbbb", a.selectedStash)
	}
	tt.Key(0, ui.KeySpace)
	if a.selectedStash != "bbbb" {
		t.Errorf("Space on the focused row selected %q; want the arrow selection bbbb", a.selectedStash)
	}
}

// A branch row has an action for Space, so it mirrors Enter: the page's action
// runs for the arrow selection instead of the row reselecting itself.
func TestSpaceOnABranchRowKeepsTheArrowSelection(t *testing.T) {
	branches := []gitrepo.Branch{
		{Name: "main", SHA: "aaaa"},
		{Name: "dev", SHA: "bbbb"},
	}
	a := &app{locale: localeChinese, page: "branches", branches: branches, branchSel: "local:dev"}
	tt, bounds := listSpaceTester(t, func(c *ui.Context) {
		for i, branch := range branches {
			a.branchRow(c, branch, i+1, len(branches))
		}
	}, "main")
	tt.ClickAt(bounds.X+bounds.W/2, bounds.Y+bounds.H/2)
	// The click selected the row it landed on; the arrows would move the
	// selection to dev, which is where this test starts.
	if a.branchSel != "local:main" {
		t.Fatalf("clicking the first row selected %q", a.branchSel)
	}
	a.branchSel = "local:dev"
	tt.Key(0, ui.KeySpace)
	if a.branchSel != "local:dev" {
		t.Errorf("Space on the focused row selected %q; want the arrow selection local:dev", a.branchSel)
	}
}

func TestSpaceOnAWorktreeRowKeepsTheArrowSelection(t *testing.T) {
	worktrees := []gitrepo.Worktree{
		{Path: "/tmp/one", Branch: "main"},
		{Path: "/tmp/two", Branch: "dev"},
	}
	a := &app{locale: localeChinese, page: "worktrees", worktrees: worktrees, worktreeSel: "worktree:/tmp/two"}
	tt, bounds := listSpaceTester(t, func(c *ui.Context) {
		for i, worktree := range worktrees {
			a.worktreeRow(c, worktree, i+1, len(worktrees))
		}
	}, "one")
	tt.ClickAt(bounds.X+bounds.W/2, bounds.Y+bounds.H/2)
	if a.worktreeSel != "worktree:/tmp/one" {
		t.Fatalf("clicking the first row selected %q", a.worktreeSel)
	}
	a.worktreeSel = "worktree:/tmp/two"
	tt.Key(0, ui.KeySpace)
	if a.worktreeSel != "worktree:/tmp/two" {
		t.Errorf("Space on the focused row selected %q; want the arrow selection worktree:/tmp/two", a.worktreeSel)
	}
}
