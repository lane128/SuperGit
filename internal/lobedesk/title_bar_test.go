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

func TestTitleBarPagesStayBesideRepositoryActions(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
			setUIFontScale(scale)
			a := &app{locale: language, page: "changes", repo: &gitrepo.Repository{Root: t.TempDir()},
				status:   gitrepo.Status{Branch: "feature/a-long-current-branch", Ahead: 12, Behind: 34},
				branches: []gitrepo.Branch{{Name: "feature/a-long-current-branch", Current: true}},
				remotes:  []gitrepo.Remote{{Name: "origin"}}, busy: true,
				operation: gitrepo.Operation{Kind: "cherry-pick"}}
			tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); ui.Column(c).Fill().Children(func() { a.titleBar(c) }) }, 900, 44) // Reserve 80 DIP for native window controls at 980 DIP.
			var end float32
			for _, page := range []string{"changes", "history", "stash"} {
				tab, ok := tt.AccessByRole(ui.RoleTab, a.t(page))
				if !ok || tab.Bounds.W <= 0 || tab.Bounds.X < end || tab.Bounds.Y+tab.Bounds.H > 44 {
					t.Fatalf("%s/%v: page tab unavailable in title bar: %s %+v", language, scale, page, tab)
				}
				end = tab.Bounds.X + tab.Bounds.W
			}
			for _, control := range []struct {
				role  ui.Role
				label string
			}{
				{ui.RoleButton, a.t("branches")},
				{ui.RoleText, "↑12 ↓34"},
				{ui.RoleButton, a.tf("operation", "cherry-pick")},
				{ui.RoleText, a.progressText()},
				{ui.RoleButton, a.t("cancel")},
				{ui.RoleButton, a.t("fetch")},
				{ui.RoleButton, a.t("pull")},
				{ui.RoleButton, a.t("pushButton")},
				{ui.RoleButton, a.t("refresh")},
			} {
				item, ok := tt.AccessByRole(control.role, control.label)
				if !ok || item.Bounds.X < end-0.5 || item.Bounds.X+item.Bounds.W > 900.5 || item.Bounds.W <= 0 {
					t.Errorf("%s/%v: control clipped or overlaps previous control (right=%v): %s %+v", language, scale, end, control.label, item.Bounds)
				}
				end = item.Bounds.X + item.Bounds.W
			}
		}
	}
	setUIFontScale(fontScaleStandard)
}

func TestSidebarNamesRepositoryAndPreservesFullPath(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: t.TempDir() + "/a-very-long-repository-checkout-name"}}
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); ui.Column(c).Fill().Children(func() { a.sidebar(c) }) }, 184, 600)
	// Repository identity is now the keyboard-accessible menu trigger.
	name, ok := tt.AccessByRole(ui.RoleButton, a.repoName())
	if !ok || name.Description != a.repo.Root || name.Bounds.W <= 0 || name.Bounds.X+name.Bounds.W > 184 {
		t.Fatalf("sidebar repository identity missing or clipped: %+v", name)
	}
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
