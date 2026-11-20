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

// History is the window: the title bar names the branch and keeps the
// repository actions, and no Changes/History/Stash tab splits it up.
func TestTitleBarKeepsRepositoryActionsWithoutPageTabs(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
			setUIFontScale(scale)
			a := &app{locale: language, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()},
				status:   gitrepo.Status{Branch: "feature/a-long-current-branch", Upstream: "origin/main", Ahead: 12, Behind: 34},
				branches: []gitrepo.Branch{{Name: "feature/a-long-current-branch", Current: true}},
				remotes:  []gitrepo.Remote{{Name: "origin"}}, busy: true,
				operation: gitrepo.Operation{Kind: "cherry-pick"}}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().Children(func() { a.workbenchToolbar(c, false) })
			}, 900, 44)
			branch, ok := tt.AccessByRole(ui.RoleButton, a.t("branches"))
			if center := branch.Bounds.X + branch.Bounds.W/2; !ok || center < 449.5 || center > 450.5 {
				t.Errorf("%s/%v: branch selector is not centered: %+v", language, scale, branch.Bounds)
			}
			status := "(" + a.tf("aheadBehind", 12, 34) + ")"
			statusBounds, visible := tt.Find(status)
			if !visible || statusBounds.X < branch.Bounds.X || statusBounds.X+statusBounds.W > branch.Bounds.X+branch.Bounds.W {
				t.Errorf("%s/%v: tracking text must fit inside the branch button: %q %+v; branch %+v", language, scale, status, statusBounds, branch.Bounds)
			}
			for _, page := range []string{"changes", "history", "stash"} {
				if tab, ok := tt.AccessByRole(ui.RoleTab, a.t(page)); ok {
					t.Fatalf("%s/%v: the title bar still offers a %q page tab: %+v", language, scale, page, tab.Bounds)
				}
			}
			var end float32
			for _, control := range []struct {
				role  ui.Role
				label string
			}{
				{ui.RoleButton, a.t("commandPalette")},
				{ui.RoleButton, a.tf("operation", "cherry-pick")},
				{ui.RoleText, a.progressText()},
				{ui.RoleButton, a.t("cancel")},
				{ui.RoleButton, a.t("branches")},
				{ui.RoleMenuButton, a.t("saveStashAction")},
				{ui.RoleMenuButton, a.t("syncMenu")},
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
	tt := pageTester(a, a.titleBar)
	switchRow, ok := tt.AccessByRole(ui.RoleButton, a.t("branches"))
	if !ok {
		t.Fatal("the title bar has no branch switch")
	}
	if switchRow.Description != "main" {
		t.Errorf("the branch switch describes itself as %q; want the checked-out branch", switchRow.Description)
	}
	for _, title := range []string{a.t("fetch"), a.t("pull"), a.t("pushButton"), a.t("refresh")} {
		if _, ok := tt.AccessByRole(ui.RoleButton, title); ok {
			t.Errorf("the title bar still shows the %q button", title)
		}
	}
	clickRepositoryTabsControl(t, tt, ui.RoleMenuButton, a.t("syncMenu"))
	for _, title := range []string{a.t("fetch"), a.t("pull"), a.t("pushBranchAction"), a.t("fetchPrune"), a.t("forcePushAction")} {
		item, ok := tt.AccessByRole(ui.RoleMenuItem, title)
		if !ok || item.Disabled {
			t.Errorf("no enabled %q in Sync: %+v", title, item)
		}
	}
	// A checkout is the one action the switch cannot undo in place, so the
	// selector still names the branch it would leave.
	if _, ok := tt.Find("feat"); ok {
		t.Error("the title bar lists every branch before the menu opens")
	}
}
