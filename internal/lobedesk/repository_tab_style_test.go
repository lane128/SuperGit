package lobedesk

import (
	"context"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestRepositoryCardsHaveStrongerDepthWhenSelected(t *testing.T) {
	for _, dark := range []bool{false, true} {
		ground := ui.Hex("#E7ECF1")
		if dark {
			ground = ui.Hex("#151C25")
		}
		selected := false
		view := ui.NewTester(func(c *ui.Context) {
			config := themeConfig("supergit")
			c.SetTheme(config.Theme(dark))
			ui.Box(c).Size(240, 80).Background(ground).Padding(12).Children(func() {
				card := ui.Box(c).Size(192, 36).Role(ui.RoleTab).Label("card")
				paintRepositoryTab(card, config.Tokens(dark), "default", selected)
			})
		}, 240, 80)
		var idleShadow uint8
		for _, active := range []bool{false, true} {
			selected = active
			view.Frame()
			card, _ := view.AccessByRole(ui.RoleTab, "card")
			pixels := view.Image()
			x, y := int(card.Bounds.X+card.Bounds.W/2), int(card.Bounds.Y)
			top := pixels.RGBAAt(x, y)
			middle := pixels.RGBAAt(x, y+12)
			shadow := pixels.RGBAAt(x, y+37)
			if top.R <= middle.R || shadow.R >= ground.R {
				t.Fatalf("dark=%t selected=%t: expected top highlight and lower shadow, top=%v middle=%v shadow=%v ground=%v", dark, active, top, middle, shadow, ground)
			}
			if !active {
				idleShadow = shadow.R
			} else if shadow.R >= idleShadow {
				t.Fatalf("dark=%t: selected shadow %d is not stronger than idle %d", dark, shadow.R, idleShadow)
			}
		}
	}
}

func TestRepositoryCardsKeepGeometryAndBorderlessClose(t *testing.T) {
	defer setUIFontScale(fontScaleStandard)
	for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
		setUIFontScale(scale)
		for _, dark := range []bool{false, true} {
			for _, preset := range []struct{ id, light, dark string }{
				{"default", "#EDF1F5", "#202B39"}, {"blue", "#E7EEF8", "#233249"},
				{"green", "#E7F0ED", "#273A35"}, {"amber", "#F5EFDF", "#3B3529"},
				{"rose", "#F3E9EE", "#3C2D38"}, {"violet", "#EEEBF5", "#332F45"}, {"gray", "#E9EDF2", "#303743"},
			} {
				a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: "/a"}, repositoryColor: preset.id}
				b := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: "/very-long-repository-name-with-中文"}, repositoryColor: preset.id}
				tabs := newRepositoryTabs(context.Background(), a)
				tabs.sessions = append(tabs.sessions, b)
				view := ui.NewTester(func(c *ui.Context) {
					c.SetTheme(themeConfig("supergit").Theme(dark))
					tabs.strip(c, tabs.active)
				}, 900, 100)
				view.Move(850, 90)
				var previous ui.Rect
				for _, active := range []*app{a, b} {
					tabs.active = active
					view.Frame()
					tab, _ := view.AccessByRole(ui.RoleTab, "a")
					close, ok := view.AccessByRole(ui.RoleButton, "Close /a")
					if !ok {
						t.Fatal("missing close")
					}
					width := close.Bounds.X + close.Bounds.W + 5 - tab.Bounds.X
					if width < 168 || width > 193 || tab.Bounds.H != 36 {
						t.Fatalf("card width/height = %.1f/%.1f", width, tab.Bounds.H)
					}
					if previous.W > 0 && previous != tab.Bounds {
						t.Fatalf("selection moved geometry: %v => %v", previous, tab.Bounds)
					}
					previous = tab.Bounds
					if close.Bounds.W != 24 || close.Bounds.H != 24 {
						t.Fatalf("close target = %v", close.Bounds)
					}
					image := view.Image()
					bg, line := ui.Hex(preset.light), ui.Hex("#1764C0")
					if dark {
						bg, line = ui.Hex(preset.dark), ui.Hex("#86B7F9")
					}
					check := func(x, y float32, want ui.Color) {
						t.Helper()
						got := image.RGBAAt(int(math.Floor(float64(x))), int(math.Floor(float64(y))))
						if got.R != want.R || got.G != want.G || got.B != want.B {
							t.Fatalf("%s dark=%t scale=%v at %.1f,%.1f got=%v want=%v", preset.id, dark, scale, x, y, got, want)
						}
					}
					check(tab.Bounds.X+8, tab.Bounds.Y+4, bg)
					// This pixel was the inherited button border/inset sheen.
					check(close.Bounds.X+close.Bounds.W/2, close.Bounds.Y, bg)
					want := bg
					if active == a {
						want = line
					}
					check(tab.Bounds.X+width/2, tab.Bounds.Y+tab.Bounds.H-1, want)
				}
			}
		}
	}
}
