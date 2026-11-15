package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func titleBarTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Width(900).Height(44).Children(func() { a.titleBar(c) })
	}, 900, 44)
}

// The branch switch and the fetch, pull and push actions used to live in the
// workspace page header. That page is folded into history, whose first row is
// the working tree, so they moved to the title bar every page shares. No
// capability may disappear with the page.
func TestTheTitleBarKeepsTheBranchSwitchAndRemoteActions(t *testing.T) {
	a := &app{
		locale:   localeChinese,
		page:     "history",
		repo:     &gitrepo.Repository{Root: t.TempDir()},
		status:   gitrepo.Status{Branch: "main", Upstream: "origin/main"},
		branches: []gitrepo.Branch{{Name: "main", Current: true}, {Name: "feat"}},
		remotes:  []gitrepo.Remote{{Name: "origin"}},
	}
	tt := titleBarTester(a)
	switchRow, ok := tt.AccessByRole(ui.RoleButton, a.t("branches"))
	if !ok {
		t.Fatal("the title bar has no branch switch")
	}
	if switchRow.Description != "main" {
		t.Errorf("the branch switch describes itself as %q; want the checked-out branch", switchRow.Description)
	}
	for _, title := range []string{a.t("fetch"), a.t("pull"), a.t("pushButton")} {
		if _, ok := tt.AccessByRole(ui.RoleButton, title); !ok {
			t.Errorf("the title bar lost the %q action", title)
		}
	}
	// A checkout is the one action the switch cannot undo in place, so the
	// selector still names the branch it would leave.
	if _, ok := tt.Find("feat"); ok {
		t.Error("the title bar lists every branch before the menu opens")
	}
}
