package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
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
			if !ok {
				t.Fatal("missing branch selector")
			}
			assertCommandControlBounds(t, tt, a, 900)
			if branch.Description != a.status.Branch+" ("+a.tf("aheadBehind", 12, 34)+")" {
				t.Fatal("missing localized tracking description")
			}
			ahead, hasAhead := tt.Find("12")
			behind, hasBehind := tt.Find("34")
			if !hasAhead || !hasBehind || ahead.X < branch.Bounds.X || ahead.X+ahead.W >= behind.X || behind.X+behind.W > branch.Bounds.X+branch.Bounds.W {
				t.Errorf("%s/%v: tracking counts must fit in order inside the branch button: %+v %+v; branch %+v", language, scale, ahead, behind, branch.Bounds)
			}
			for _, page := range []string{"changes", "history", "stash"} {
				if tab, ok := tt.AccessByRole(ui.RoleTab, a.t(page)); ok {
					t.Fatalf("%s/%v: the title bar still offers a %q page tab: %+v", language, scale, page, tab.Bounds)
				}
			}
			if _, ok := tt.AccessByRole(ui.RoleText, a.progressText()); ok {
				t.Errorf("%s/%v: left title bar still renders redundant progress text %q", language, scale, a.progressText())
			}
			if _, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel")); ok {
				t.Errorf("%s/%v: left title bar still renders redundant cancel button", language, scale)
			}
			if _, ok := tt.AccessByRole(ui.RoleProgress, a.progressText()); !ok {
				t.Errorf("%s/%v: branch selector missing thin progress line for %q", language, scale, a.progressText())
			}
			var end float32
			for _, control := range []struct {
				role  ui.Role
				label string
			}{
				{ui.RoleButton, a.tf("operation", "cherry-pick")},
				{ui.RoleButton, a.t("commandPalette")},
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

func TestBranchSelectorTracking(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   gitrepo.Status
		showUp   bool
		showDown bool
	}{
		{"ahead", gitrepo.Status{Branch: "main", Upstream: "origin/main", Ahead: 2}, true, false},
		{"behind", gitrepo.Status{Branch: "main", Upstream: "origin/main", Behind: 3}, false, true},
		{"diverged", gitrepo.Status{Branch: "main", Upstream: "origin/main", Ahead: 2, Behind: 3}, true, true},
		{"synced", gitrepo.Status{Branch: "main", Upstream: "origin/main"}, false, false},
		{"no upstream", gitrepo.Status{Branch: "main"}, false, false},
		{"upstream gone", gitrepo.Status{Branch: "main", Upstream: "origin/main", UpstreamGone: true}, false, false},
		{"detached", gitrepo.Status{}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{locale: localeEnglish, status: tc.status, repo: &gitrepo.Repository{Root: t.TempDir()}}
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				a.toolbarCommandControl(c, true)
			}, 360, 44)
			ahead, up := tt.Find("2")
			behind, down := tt.Find("3")
			_, zero := tt.Find("0")
			_, gone := tt.Find("(" + a.t("upstreamGone") + ")")
			if up != tc.showUp || down != tc.showDown || zero || gone != tc.status.UpstreamGone {
				t.Fatalf("unexpected tracking indicators: %v", tt.Texts())
			}
			label := tc.status.Branch
			if label == "" {
				label = a.t("detached")
			}
			name, found := tt.Find(label)
			selector, ok := tt.AccessByRole(ui.RoleButton, a.t("branches"))
			if !found || !ok || name.X-selector.Bounds.X < 8 || name.X-selector.Bounds.X > 12 {
				t.Fatalf("branch name lost its leading inset: %+v in %+v", name, selector.Bounds)
			}
			if up || down {
				tk := supergitTokens(false)
				items, _ := a.branchTrackingStatus(&tk)
				if up && (items[0].icon != lucide.ArrowUp() || items[0].ink != tk.SuccessText) {
					t.Fatal("outgoing commits lost their green up arrow")
				}
				if item := items[len(items)-1]; down && (item.icon != lucide.ArrowDown() || item.ink != tk.Blue) {
					t.Fatal("incoming commits lost their blue down arrow")
				}
				last := ahead
				if down {
					last = behind
				}
				if inset := selector.Bounds.X + selector.Bounds.W - last.X - last.W; inset < 8 || inset > 12 {
					t.Fatalf("counts lost their trailing inset: %+v in %+v", last, selector.Bounds)
				}
			}
		})
	}
}

func assertCommandControlBounds(t *testing.T, tt *ui.Tester, a *app, windowWidth float32) {
	t.Helper()
	icon, hasIcon := tt.AccessByRole(ui.RoleButton, a.t("commandPalette"))
	branch, hasBranch := tt.AccessByRole(ui.RoleButton, a.t("branches"))
	const wantedWidth = 360
	left := (windowWidth - wantedWidth) / 2
	withinPixel := func(x, y float32) bool { return x >= y-1 && x <= y+1 }
	if !hasIcon || !hasBranch || !withinPixel(icon.Bounds.X, left) ||
		!withinPixel(icon.Bounds.X+icon.Bounds.W, branch.Bounds.X) ||
		!withinPixel(branch.Bounds.X+branch.Bounds.W, left+wantedWidth) ||
		icon.Bounds.Y != branch.Bounds.Y || icon.Bounds.H != branch.Bounds.H {
		t.Fatalf("expected one centered %d pt control, got icon %+v and branch %+v", wantedWidth, icon.Bounds, branch.Bounds)
	}
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
