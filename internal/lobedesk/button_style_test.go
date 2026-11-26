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
			buttonNormal:     {"#F6F8FA", "#F3F4F6", "#EBECF0"},
			buttonPrimary:    {"#1F883D", "#1A7F37", "#187733"},
			buttonDangerIcon: {"#FFEBE9", "#FFE0DD", "#FFD4D0"},
			buttonDanger:     {"#CF222E", "#B61D28", "#A41923"},
		} {
			if dark {
				switch role {
				case buttonNormal:
					colors = []string{"#21262D", "#30363D", "#282E33"}
				case buttonPrimary:
					colors = []string{"#1F7E33", "#238636", "#1A6D2C"}
				case buttonDangerIcon:
					colors = []string{"#321F24", "#41262C", "#4F2C33"}
				case buttonDanger:
					colors = []string{"#C9312E", "#DA3633", "#B22927"}
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
	for _, control := range []struct {
		label string
		size  float32
	}{
		{a.t("saveStashAction"), 24}, // Arrow of the compact titlebar split button.
		{a.t("syncMenu"), 29},
	} {
		info, ok := tt.AccessByRole(ui.RoleMenuButton, control.label)
		if !ok || info.Bounds.W < control.size || info.Bounds.H != control.size {
			t.Errorf("%s bounds=%v; want a target at least %v points wide and %v points tall", control.label, info.Bounds, control.size, control.size)
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
			ui.Row(c).FillWidth().Height(36).Background(tk.BgLayout).Children(func() {
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
					want = tk.BgContainer
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
