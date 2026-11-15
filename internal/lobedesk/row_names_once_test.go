package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// A row's label already names it for assistive technology; the visible title
// inside the row is the same text, so leaving it in the tree made a screen
// reader announce every branch, commit and stash twice.
func TestRowNamesAreReadOnce(t *testing.T) {
	a := &app{locale: localeChinese, selectedSHA: "bbbb", branchSel: "local:feat"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(800).Height(600).Children(func() {
			a.branchRow(c, gitrepo.Branch{Name: "feat", SHA: "bbbb"}, 1, 1)
			a.remoteBranchRow(c, gitrepo.Branch{Name: "origin/topic"}, 1, 1)
			a.tagRow(c, gitrepo.Tag{Name: "v1.0"}, 1, 1)
			a.commitRow(c, gitrepo.Commit{SHA: "bbbb", Subject: "second"}, 0, 0, 1, &commitGraph{})
			a.stashRow(c, gitrepo.Stash{SHA: "aaaa", Ref: "stash@{0}", Message: "first"}, 1, 1)
			a.worktreeRow(c, gitrepo.Worktree{Path: "/repos/checkout", Branch: "main"}, 1, 1)
			a.submoduleRow(c, gitrepo.Submodule{Name: "lib", Path: "vendor/lib", Initialized: true}, 1, 1)
		})
	}, 800, 600)

	count := func(name string) int {
		n := 0
		for _, info := range tt.AccessAll() {
			if info.Name == name {
				n++
			}
		}
		return n
	}
	for _, name := range []string{"feat", "topic", "v1.0", "second", "first", "checkout", "lib"} {
		if got := count(name); got != 1 {
			t.Errorf("%q is read %d times; want once, on the row's label", name, got)
		}
	}
	// The worktree's branch column is different information, not a repeated
	// name, so it stays readable.
	if got := count("main"); got != 1 {
		t.Errorf("the worktree's branch is read %d times; want once", got)
	}
}
