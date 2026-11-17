package lobedesk

import "github.com/egoist/mygo/ui"

// markDiffGutter uses the existing right padding, so neither line numbers nor
// code move as a selection changes. The check also reads without its blue fill.
func markDiffGutter(numbers ui.Element, selected bool, ink ui.Color) {
	numbers.AccessSelected(selected)
	if !selected {
		return
	}
	numbers.DrawOver(func(p *ui.Painter, r ui.Rect) {
		x, y := r.X+r.W-7, r.Y+r.H/2
		p.StrokePolyline([][2]float32{{x, y}, {x + 2, y + 2}, {x + 6, y - 3}}, 1.5, ink)
	})
}
