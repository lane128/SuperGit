package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// A worktree removed from the list leaves its key behind; the page then
// highlights nothing, and Enter on the stale key does nothing at all. The
// page drops the selection when the list no longer shows it.
func TestWorktreePageDropsASelectionTheListNoLongerShows(t *testing.T) {
	for _, stale := range []string{"worktree:/repos/gone", "submodule:vendor/gone"} {
		a := &app{
			locale: localeChinese,
			worktrees: []gitrepo.Worktree{
				{Path: "/repos/main", Branch: "main"},
			},
			submodules: []gitrepo.Submodule{
				{Name: "lib", Path: "vendor/lib", Initialized: true},
			},
			worktreeSel:     stale,
			revealSelection: true,
		}
		pageTester(a, a.worktreesPage)

		if a.worktreeSel != "" {
			t.Errorf("the page keeps %q although the list does not show it", a.worktreeSel)
		}
		if a.revealSelection {
			t.Errorf("dropping %q left a reveal request for a row that is gone", stale)
		}
	}
}

// A selection that is still on the page survives.
func TestWorktreePageKeepsAVisibleSelection(t *testing.T) {
	a := &app{
		locale: localeChinese,
		worktrees: []gitrepo.Worktree{
			{Path: "/repos/main", Branch: "main"},
			{Path: "/repos/feat", Branch: "feat"},
		},
		worktreeSel: "worktree:/repos/feat",
	}
	pageTester(a, a.worktreesPage)

	if a.worktreeSel != "worktree:/repos/feat" {
		t.Errorf("the page dropped the visible selection to %q", a.worktreeSel)
	}
}
