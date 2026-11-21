package lobedesk

import (
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

func TestWorkbenchButtonColorsFollowSpecification(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for role, colors := range map[buttonRole][]string{
			buttonNormal:     {"#DEE4EF", "#D2DCEC", "#C6D3E7"},
			buttonPrimary:    {"#115534", "#196641", "#0D442B"},
			buttonDangerIcon: {"#F3E4E9", "#ECD6DD", "#E4C8D0"},
			buttonDanger:     {"#A53A47", "#AF3B4A", "#8D2D38"},
		} {
			if dark {
				switch role {
				case buttonNormal:
					colors = []string{"#333A4C", "#3D455B", "#464F67"}
				case buttonPrimary:
					colors = []string{"#2B5E40", "#356E4C", "#254F36"}
				case buttonDangerIcon:
					colors = []string{"#41333F", "#513A47", "#60414E"}
				case buttonDanger:
					colors = []string{"#A44451", "#AC4655", "#923642"}
				}
			}
			for state, want := range colors {
				tk := themeConfig("supergit").Tokens(dark)
				bg, ink := workbenchButtonColors(tk, role, state == 1, state == 2, false, false)
				if bg != ui.Hex(want) {
					t.Errorf("dark=%v role=%v state=%v fill=%v; want %s", dark, role, state, bg, want)
				}
				if contrastRatio(ink, bg) < 4.5 {
					t.Errorf("dark=%v role=%v state=%v text contrast below 4.5", dark, role, state)
				}
			}
		}
	}
}

func TestWorkbenchButtonPreservesNativeActivationAndDisabledState(t *testing.T) {
	clicks := 0
	disabled := false
	tt := ui.NewTester(func(c *ui.Context) {
		themeConfig("supergit").Apply(c)
		if workbenchButton(c, "Save", buttonPrimary, disabled).Clicked() {
			clicks++
		}
	}, 300, 80)
	tt.Click("Save")
	if clicks != 1 {
		t.Fatalf("pointer activation count=%d", clicks)
	}
	tt.Key(0, ui.KeyEnter)
	if clicks != 2 {
		t.Fatalf("keyboard activation count=%d", clicks)
	}
	disabled = true
	tt.Frame()
	info, ok := tt.AccessByRole(ui.RoleButton, "Save")
	if !ok || !info.Disabled {
		t.Fatal("disabled native state missing")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Key(0, ui.KeyEnter)
	if clicks != 2 {
		t.Fatalf("disabled button activated: %d", clicks)
	}
}

func TestWorkbenchToolbarButtonsUseStandardSize(t *testing.T) {
	setUIFontScale(fontScaleStandard)
	defer setUIFontScale(fontScaleStandard)
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: t.TempDir()}}
	tt := titleBarTester(a)
	for _, label := range []string{a.t("saveStashAction"), a.t("syncMenu")} {
		info, ok := tt.AccessByRole(ui.RoleMenuButton, label)
		if !ok || info.Bounds.W < 29 || info.Bounds.H != 29 {
			t.Errorf("%s bounds=%v; want a button at least 29 points wide and 29 points tall", label, info.Bounds)
		}
	}
}

// The close target has its own native button. At rest it should read as the
// trailing part of the selected file, including after switching the active tab.
func TestFileTabSelectionPaintsThroughCloseTarget(t *testing.T) {
	a := &app{locale: localeEnglish}
	first := detailFileKey{Path: "first.go"}
	second := detailFileKey{Path: "second.go"}
	a.detailTabs().Open(first)
	a.detailTabs().Open(second)
	for _, dark := range []bool{false, true} {
		config := themeConfig(themePaletteCatppuccinMacchiato)
		view := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(config.Theme(dark))
			tk := theme.TokensOf(c)
			ui.Row(c).FillWidth().Height(32).Background(tk.BgLayout).Children(func() {
				for _, f := range []detailFileKey{first, second} {
					a.detailTabButton(c, tk, detailTabKey{File: f})
				}
			})
		}, 360, 64)
		view.Move(330, 50)
		for _, active := range []detailFileKey{second, first} {
			a.detailTabs().Activate(detailTabKey{File: active})
			view.Frame()
			paint := view.Image()
			tk := config.Tokens(dark)
			for _, f := range []detailFileKey{first, second} {
				want := tk.BgLayout
				if f == active {
					want, _ = workbenchButtonColors(tk, buttonNormal, false, false, false, true)
				}
				for role, name := range map[ui.Role]string{ui.RoleTab: f.Path, ui.RoleButton: a.t("close") + " " + f.Path} {
					node, ok := view.AccessByRole(role, name)
					if !ok {
						t.Fatalf("missing %s", name)
					}
					got := paint.RGBAAt(int(node.Bounds.X+3), int(node.Bounds.Y+node.Bounds.H/2))
					if got.R != want.R || got.G != want.G || got.B != want.B {
						t.Errorf("dark=%t active=%s control=%s fill=%v want=%v", dark, active.Path, name, got, want)
					}
				}
			}
		}
	}
}
