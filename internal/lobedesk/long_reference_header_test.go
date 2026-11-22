package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestLongCurrentBranchKeepsTitleActionsWithinWindow(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
			t.Run(string(language)+"/"+fontScaleKey(scale), func(t *testing.T) {
				setUIFontScale(scale)
				t.Cleanup(func() { setUIFontScale(fontScaleStandard) })
				name := "stress/" + strings.Repeat("跨平台-descriptive-segment-", 12)
				a := &app{locale: language, repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Branch: name, Upstream: "origin/main", Ahead: 102}, branches: []gitrepo.Branch{{Name: name, Current: true}}, remotes: []gitrepo.Remote{{Name: "origin"}}}
				tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); ui.Column(c).Fill().Children(func() { a.titleBar(c) }) }, 900, 44)
				selector, ok := tt.AccessByRole(ui.RoleButton, a.t("branches"))
				if !ok || selector.Description != name+" ("+a.tf("aheadBehind", 102, 0)+")" {
					t.Fatal("branch selector lost complete reference identity or tracking status")
				}
				assertCommandControlBounds(t, tt, a, 900)
				tracking, shown := tt.Find("102")
				if !shown || tracking.X+tracking.W > selector.Bounds.X+selector.Bounds.W {
					t.Fatalf("long branch clipped its tracking status: %+v", tracking)
				}
				sync, ok := tt.AccessByRole(ui.RoleMenuButton, a.t("syncMenu"))
				if !ok || sync.Bounds.X+sync.Bounds.W > 900.5 || sync.Bounds.X < selector.Bounds.X+selector.Bounds.W {
					t.Fatalf("long branch obstructs Sync: %+v", sync.Bounds)
				}
			})
		}
	}
}

func TestLongCurrentBranchBeforeReferencesLoadKeepsActionsWithinWindow(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Branch: strings.Repeat("跨平台-descriptive-", 30)}, remotes: []gitrepo.Remote{{Name: "origin"}}}
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); ui.Column(c).Fill().Children(func() { a.titleBar(c) }) }, 980, 44)
	assertCommandControlBounds(t, tt, a, 980)
	sync, ok := tt.AccessByRole(ui.RoleMenuButton, a.t("syncMenu"))
	if !ok || sync.Bounds.X+sync.Bounds.W > 980.5 {
		t.Fatalf("Sync outside window before references load: %+v", sync.Bounds)
	}
}

func TestBranchHelpFitsAboveScrollableReferencesAtLargeFont(t *testing.T) {
	setUIFontScale(fontScaleLarge)
	t.Cleanup(func() { setUIFontScale(fontScaleStandard) })
	a := branchDetailFixture()
	a.locale = localeEnglish
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); ui.Column(c).Fill().Children(func() { a.branchesPage(c) }) }, 600, 596)
	help, ok := tt.Find(a.t("branchHelp"))
	if !ok {
		t.Fatal("branch help missing")
	}
	scroll, ok := tt.AccessByRole(ui.RoleScroll, "")
	if !ok {
		t.Fatal("reference scroll area missing")
	}
	if help.Y < 46 || help.Y+help.H > scroll.Bounds.Y {
		t.Fatalf("wrapped help escapes its header: help=%+v scroll=%+v", help, scroll.Bounds)
	}
}

func TestLongCurrentBranchKeepsPageHeadingAndSearchSeparate(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
			t.Run(string(language)+"/"+fontScaleKey(scale), func(t *testing.T) {
				setUIFontScale(scale)
				t.Cleanup(func() { setUIFontScale(fontScaleStandard) })
				a := branchDetailFixture()
				a.locale = language
				name := "stress/" + strings.Repeat("跨平台-descriptive-segment-", 12)
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
				a.status = gitrepo.Status{Branch: name, Head: strings.Repeat("1", 40)}
				a.branches[0].Name = name
				a.branchSel = "local:" + name
				tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); ui.Column(c).Fill().Children(func() { a.branchesPage(c) }) }, 790, 596)
				title, ok := tt.Find(a.t("branches"))
				if !ok {
					t.Fatal("heading missing")
				}
				search, ok := tt.AccessByRole(ui.RoleTextField, "")
				if !ok {
					t.Fatal("branch search missing")
				}
				if title.X+title.W+6 > search.Bounds.X {
					t.Fatalf("heading overlaps search: heading=%+v search=%+v", title, search.Bounds)
				}
				create, ok := tt.AccessByRole(ui.RoleButton, a.t("newBranchAction"))
				if !ok || create.Bounds.X+create.Bounds.W > 790.5 {
					t.Fatalf("new branch control outside page: %+v", create.Bounds)
				}
			})
		}
	}
}
