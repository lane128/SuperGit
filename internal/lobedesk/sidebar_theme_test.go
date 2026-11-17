package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestSidebarThemeKeepsReadableInkAndOtherPalettes(t *testing.T) {
	for _, palette := range []string{themePaletteSuperGit, themePaletteCatppuccin, themePaletteDefault} {
		for _, mode := range []string{"light", "dark"} {
			a := &app{palette: palette, themeMode: mode}
			ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				base := *theme.TokensOf(c)
				tk := a.sidebarTokens(c)
				if palette != themePaletteSuperGit {
					if tk.BgContainer != base.BgContainer || tk.Text != base.Text || tk.TextSecondary != base.TextSecondary || a.sidebarSelectionBg(tk) != selectionBg(&base, base.BgContainer) {
						t.Errorf("%s/%s sidebar changed existing palette", palette, mode)
					}
					return
				}
				for _, ground := range []ui.Color{tk.BgContainer, tk.PrimaryBg, tk.FillTertiary, tk.BgElevated} {
					for _, ink := range []ui.Color{tk.Text, tk.TextSecondary, tertiaryTextInk(tk, ground), tk.Green} {
						if ratio := contrastRatio(ink, ground); ratio < 4.5 {
							t.Errorf("%s sidebar ink %v on %v contrast %.2f", mode, ink, ground, ratio)
						}
					}
				}
				if *theme.TokensOf(c) != base {
					t.Error("sidebar token lookup mutated window theme")
				}
			}, 200, 200)
		}
	}
}

func TestSuperGitSidebarUsesItsOwnSurfaceAndSelection(t *testing.T) {
	for _, mode := range []string{"light", "dark"} {
		t.Run(mode, func(t *testing.T) {
			a := &app{palette: themePaletteSuperGit, themeMode: mode, locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}}
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				ui.Row(c).Fill().Children(func() {
					a.sidebar(c)
					ui.Box(c).Grow(1).Height(620).Background(theme.TokensOf(c).BgContainer)
				})
			}, 500, 620)
			ground, chosen := ui.Hex("#1D2D49"), ui.Hex("#344D79")
			if mode == "dark" {
				ground, chosen = ui.Hex("#111D34"), ui.Hex("#22385F")
			}
			pixel := func(x, y int) ui.Color {
				p := tt.Image().RGBAAt(x, y)
				return ui.RGB(p.R, p.G, p.B)
			}
			if got := pixel(1, 400); got != ground {
				t.Errorf("sidebar surface=%v want %v", got, ground)
			}
			row, ok := tt.AccessByRole(ui.RoleButton, a.t("history"))
			if !ok {
				t.Fatal("history entry missing")
			}
			if got := pixel(int(row.Bounds.X+3), int(row.Bounds.Y+row.Bounds.H/2)); got != chosen {
				t.Errorf("sidebar selection=%v want %v", got, chosen)
			}
			if got := pixel(450, 400); got != supergitTokens(mode == "dark").BgContainer {
				t.Errorf("sidebar colors leaked into content: %v", got)
			}
		})
	}
}
