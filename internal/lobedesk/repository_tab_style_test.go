package lobedesk

import (
	"context"
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestRepositoryTabsSublimeSeamlessDeckAndToolbar(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tk := supergitTokens(dark)
		chrome := repositoryChrome(&tk)
		a := &app{locale: localeEnglish, palette: themePaletteSuperGit, repo: &gitrepo.Repository{Root: "/a"}}
		b := &app{locale: localeEnglish, palette: themePaletteSuperGit, repo: &gitrepo.Repository{Root: "/b"}}
		d := &app{locale: localeEnglish, palette: themePaletteSuperGit, repo: &gitrepo.Repository{Root: "/c"}}
		tabs := newRepositoryTabs(context.Background(), a)
		tabs.sessions = append(tabs.sessions, b, d)
		view := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(themeConfig(themePaletteSuperGit).Theme(dark))
			ui.Column(c).Fill().Children(func() {
				tabs.strip(c, tabs.active)
				tabs.active.workbenchToolbar(c, false)
			})
		}, 900, 120)
		view.Move(880, 110)
		view.Frame()

		tabA, okA := view.AccessByRole(ui.RoleTab, "a")
		tabB, okB := view.AccessByRole(ui.RoleTab, "b")
		tabC, okC := view.AccessByRole(ui.RoleTab, "c")
		if !okA || !okB || !okC {
			t.Fatalf("dark=%t: missing tabs", dark)
		}
		img := view.Image()
		sample := func(x, y float32) color.RGBA {
			return img.RGBAAt(int(math.Floor(float64(x))), int(math.Floor(float64(y))))
		}
		same := func(got color.RGBA, want ui.Color) bool {
			return got.R == want.R && got.G == want.G && got.B == want.B
		}

		// Selected tab fills with deck and seamlessly covers the 1px strip
		// divider at Y=titleBarHeight-1, matching the toolbar directly below.
		activeMidX := tabA.Bounds.X + tabA.Bounds.W/2
		if got := sample(activeMidX, tabA.Bounds.Y+4); !same(got, chrome.deck) {
			t.Fatalf("dark=%t: selected tab surface=%v want deck=%v", dark, got, chrome.deck)
		}
		if got := sample(activeMidX, float32(titleBarHeight)-1); !same(got, chrome.deck) {
			t.Fatalf("dark=%t: selected tab seam at bottom=%v want seamless deck=%v", dark, got, chrome.deck)
		}
		if got := sample(activeMidX, float32(titleBarHeight)+4); !same(got, chrome.deck) {
			t.Fatalf("dark=%t: toolbar surface=%v want deck=%v", dark, got, chrome.deck)
		}
		// Top border of selected tab uses tabEdge.
		if got := sample(activeMidX, tabA.Bounds.Y-1); !same(got, chrome.tabEdge) {
			t.Fatalf("dark=%t: selected tab top edge=%v want %v", dark, got, chrome.tabEdge)
		}

		// Unselected tab sits flush on track above the strip's bottom divider.
		idleMidX := tabB.Bounds.X + tabB.Bounds.W/2
		if got := sample(idleMidX, tabB.Bounds.Y+4); !same(got, chrome.track) {
			t.Fatalf("dark=%t: unselected tab surface=%v want track=%v", dark, got, chrome.track)
		}
		if got := sample(idleMidX, float32(titleBarHeight)-1); !same(got, chrome.trackLine) {
			t.Fatalf("dark=%t: strip bottom line under idle tab=%v want %v", dark, got, chrome.trackLine)
		}

		// Centered vertical hairline appears between consecutive unselected
		// tabs (b and c), not on the edge touching the selected tab (a and b).
		sepBetweenIdle := sample(tabC.Bounds.X-1, tabC.Bounds.Y+tabC.Bounds.H/2)
		if same(sepBetweenIdle, chrome.track) {
			t.Fatalf("dark=%t: missing hairline separator between unselected tabs b and c", dark)
		}
		// Left edge of selected tab is its crisp 1px tabEdge, not track.
		if got := sample(tabA.Bounds.X-1, tabA.Bounds.Y+tabA.Bounds.H/2); !same(got, chrome.tabEdge) {
			t.Fatalf("dark=%t: selected tab left edge=%v want %v", dark, got, chrome.tabEdge)
		}
	}
}

func TestRepositoryTabHoverDoesNotChangeBackground(t *testing.T) {
	for _, dark := range []bool{false, true} {
		a := &app{locale: localeEnglish, palette: themePaletteSuperGit, repo: &gitrepo.Repository{Root: "/a"}}
		b := &app{locale: localeEnglish, palette: themePaletteSuperGit, repo: &gitrepo.Repository{Root: "/b"}}
		tabs := newRepositoryTabs(context.Background(), a)
		tabs.sessions = append(tabs.sessions, b)
		view := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(themeConfig(themePaletteSuperGit).Theme(dark))
			tabs.strip(c, tabs.active)
		}, 900, 100)
		view.Frame()

		for _, tc := range []struct{ tabName, closeName string }{
			{"a", "Close /a"},
			{"b", "Close /b"},
		} {
			tab, ok := view.AccessByRole(ui.RoleTab, tc.tabName)
			if !ok {
				t.Fatalf("missing tab %s", tc.tabName)
			}
			closeBtn, ok := view.AccessByRole(ui.RoleButton, tc.closeName)
			if !ok {
				t.Fatalf("missing close %s", tc.closeName)
			}
			pixel := func(x, y float32) color.RGBA {
				return view.Image().RGBAAt(int(math.Floor(float64(x))), int(math.Floor(float64(y))))
			}
			labelX, labelY := tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+4
			closeX, closeY := closeBtn.Bounds.X+3, closeBtn.Bounds.Y+3

			view.Move(880, 95)
			view.Frame()
			idleLabel, idleClose := pixel(labelX, labelY), pixel(closeX, closeY)
			if idleLabel != idleClose {
				t.Fatalf("dark=%t tab=%s: idle card uneven: label=%v close=%v", dark, tc.tabName, idleLabel, idleClose)
			}

			view.Move(labelX, labelY)
			view.Frame()
			hoverLabel, hoverLabelClose := pixel(labelX, labelY), pixel(closeX, closeY)
			if hoverLabel != idleLabel || hoverLabelClose != idleClose {
				t.Fatalf("dark=%t tab=%s: hovering label changed background: idle=%v/%v hover=%v/%v",
					dark, tc.tabName, idleLabel, idleClose, hoverLabel, hoverLabelClose)
			}

			view.Move(closeBtn.Bounds.X+closeBtn.Bounds.W/2, closeBtn.Bounds.Y+closeBtn.Bounds.H/2)
			view.Frame()
			hoverClose, hoverCloseBg := pixel(labelX, labelY), pixel(closeX, closeY)
			if hoverClose != idleLabel || hoverCloseBg != idleClose {
				t.Fatalf("dark=%t tab=%s: hovering close painted background: idle=%v/%v hover=%v/%v",
					dark, tc.tabName, idleLabel, idleClose, hoverClose, hoverCloseBg)
			}
		}
	}
}

func TestRepositoryTabColorUsesFilledIconOnNeutralSurface(t *testing.T) {
	defer setUIFontScale(fontScaleStandard)
	for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
		setUIFontScale(scale)
		for _, dark := range []bool{false, true} {
			tk := supergitTokens(dark)
			chrome := repositoryChrome(&tk)
			for _, preset := range repositoryColorPresets {
				a := &app{locale: localeEnglish, palette: themePaletteSuperGit, repo: &gitrepo.Repository{Root: "/a"}, repositoryColor: preset}
				b := &app{locale: localeEnglish, palette: themePaletteSuperGit, repo: &gitrepo.Repository{Root: "/very-long-repository-name-with-中文"}, repositoryColor: preset}
				tabs := newRepositoryTabs(context.Background(), a)
				tabs.sessions = append(tabs.sessions, b)
				view := ui.NewTester(func(c *ui.Context) {
					c.SetTheme(themeConfig(themePaletteSuperGit).Theme(dark))
					tabs.strip(c, tabs.active)
				}, 900, 100)
				view.Move(850, 90)
				var previous ui.Rect
				for _, active := range []*app{a, b} {
					tabs.active = active
					view.Frame()
					tab, _ := view.AccessByRole(ui.RoleTab, "a")
					closeBtn, ok := view.AccessByRole(ui.RoleButton, "Close /a")
					if !ok {
						t.Fatal("missing close")
					}
					width := closeBtn.Bounds.X + closeBtn.Bounds.W + 5 - tab.Bounds.X
					if width < 166 || width > 193 || tab.Bounds.H != repositoryTabHeight-1 {
						t.Fatalf("card width/height = %.1f/%.1f", width, tab.Bounds.H)
					}
					if previous.W > 0 && previous != tab.Bounds {
						t.Fatalf("selection moved geometry: %v => %v", previous, tab.Bounds)
					}
					previous = tab.Bounds
					if closeBtn.Bounds.W != 24 || closeBtn.Bounds.H != 24 {
						t.Fatalf("close target = %v", closeBtn.Bounds)
					}

					img := view.Image()
					wantBg := chrome.track
					if active == a {
						wantBg = chrome.deck
					}
					check := func(x, y float32, want ui.Color) {
						t.Helper()
						got := img.RGBAAt(int(math.Floor(float64(x))), int(math.Floor(float64(y))))
						if got.R != want.R || got.G != want.G || got.B != want.B {
							t.Fatalf("%s dark=%t scale=%v active=%t at %.1f,%.1f got=%v want=%v",
								preset, dark, scale, active == a, x, y, got, want)
						}
					}
					// Card surface stays neutral regardless of repository color preset.
					check(tab.Bounds.X+8, tab.Bounds.Y+4, wantBg)
					check(closeBtn.Bounds.X+closeBtn.Bounds.W/2, closeBtn.Bounds.Y, wantBg)
					// Active tab has no blue bottom accent bar — seamless deck down to bottom.
					if active == a {
						check(tab.Bounds.X+width/2, tab.Bounds.Y+tab.Bounds.H-1, chrome.deck)
					}

					// When a non-default preset is active, the filled folder icon paints the tag hue.
					if tag, colored := repositoryTagColor(dark, preset); colored && active == a {
						iconMatches := 0
						x0 := int(math.Floor(float64(tab.Bounds.X + 12)))
						x1 := x0 + int(math.Ceil(float64(scaledFont(16))))
						y0 := int(math.Floor(float64(tab.Bounds.Y + 4)))
						y1 := int(math.Ceil(float64(tab.Bounds.Y + tab.Bounds.H - 4)))
						for y := y0; y < y1; y++ {
							for x := x0; x < x1; x++ {
								p := img.RGBAAt(x, y)
								if absDiff(p.R, tag.R) <= 4 && absDiff(p.G, tag.G) <= 4 && absDiff(p.B, tag.B) <= 4 {
									iconMatches++
								}
							}
						}
						if iconMatches < 20 {
							t.Fatalf("%s dark=%t scale=%v: filled folder icon painted only %d pixels matching tag %v",
								preset, dark, scale, iconMatches, tag)
						}
					}
				}
			}
		}
	}
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
