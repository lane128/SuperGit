package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func repositoryTabColors(dark bool, preset string, selected bool) (background, ink, line ui.Color) {
	colors := map[string][2]string{
		"default": {"#EDF1F5", "#202B39"}, "blue": {"#E7EEF8", "#233249"},
		"green": {"#E7F0ED", "#273A35"}, "amber": {"#F5EFDF", "#3B3529"},
		"rose": {"#F3E9EE", "#3C2D38"}, "violet": {"#EEEBF5", "#332F45"}, "gray": {"#E9EDF2", "#303743"},
	}
	index := 0
	ink, line = ui.Hex("#526175"), ui.Hex("#1764C0")
	if selected {
		ink = ui.Hex("#202B38")
	}
	if dark {
		index = 1
		ink, line = ui.Hex("#B7C5D6"), ui.Hex("#86B7F9")
		if selected {
			ink = ui.Hex("#E9EFF7")
		}
	}
	background = ui.Hex(colors[normalizedRepositoryColor(preset)][index])
	return
}

func repositoryCardWidth(viewport float32, count int) float32 {
	if viewport <= 0 || count < 1 {
		return 192
	}
	return max(168, min(192, (viewport-6-float32(count-1)*6)/float32(count)))
}

func repositoryCaptionWidth(card float32) float32 {
	return card - 12 - scaledFont(15) - 8 - 8 - 24 - 5
}

func paintRepositoryTab(tab ui.Element, tk *theme.Tokens, preset string, selected bool) ui.Color {
	bg, ink, line := repositoryTabColors(tk.Dark, preset, selected)
	shadow, sheen := float32(.10), float32(.55)
	offset, blur := float32(1), float32(1.5)
	if tk.Dark {
		shadow, sheen = .28, .06
	}
	if selected {
		offset, blur = 2, 3
		shadow, sheen = .20, .85
		if tk.Dark {
			shadow, sheen = .55, .14
		}
	}
	// Depth belongs to the whole card, never the close button. Keep its
	// surface color and hit boxes stationary when selection changes.
	tab.Shadow(0, offset, blur, 0, ui.Hex("#000000").Alpha(shadow)).
		InsetShadow(0, 1, 0, 0, ui.Hex("#FFFFFF").Alpha(sheen))
	tab.Background(bg).Radius(6).Draw(func(p *ui.Painter, r ui.Rect) {
		if selected {
			p.Fill(ui.Rect{X: r.X + 12, Y: r.Y + r.H - 2, W: r.W - 24, H: 2}, line, 1)
		}
	})
	return ink
}

func styleRepositoryTabTarget(target ui.Element, ink ui.Color) {
	target.FocusRing(false).Draw(func(p *ui.Painter, r ui.Rect) {
		r.H -= 2 // Leave the fixed selection line unobscured.
		if !target.IsDisabled() {
			if target.Pressed() {
				p.Fill(r, ink.Alpha(.10), 6)
			} else if target.Hovered() {
				p.Fill(r, ink.Alpha(.05), 6)
			}
		}
		if target.FocusVisible() {
			r.X += 4
			r.Y += 4
			r.W -= 8
			r.H -= 8
			p.StrokeDashed(r, ink, 3, 1)
		}
	})
}

func styleRepositoryClose(target ui.Element, ink ui.Color) {
	target.Size(24, 24).Padding(0).Border(0, ui.Transparent).NoShadow().Radius(5).
		Background(ui.Transparent).TextColor(ink).FocusRing(false).DimDisabled(false)
	target.Draw(func(p *ui.Painter, r ui.Rect) {
		if !target.IsDisabled() {
			if target.Pressed() {
				p.Fill(r, ink.Alpha(.23), 5)
			} else if target.Hovered() {
				p.Fill(r, ink.Alpha(.14), 5)
			}
		}
		if target.FocusVisible() {
			r.X++
			r.Y++
			r.W -= 2
			r.H -= 2
			p.StrokeDashed(r, ink, 4, 1)
		}
	})
}
