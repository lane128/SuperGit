package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"slices"
)

// virtualPatchRows retains the full layout extent but creates controls only
// around the reading position. Original indexes are also widget keys, so a
// scrolling row cannot inherit another row's focus or context menu.
func virtualPatchRows(c *ui.Context, viewport ui.Rect, scroll *ui.ScrollState, heights []float32, reveal int, draw func(*ui.Context, int), retained ...int) {
	total := float32(0)
	for _, height := range heights {
		total += height
	}
	body := ui.Column(c).FillWidth().Height(total).Shrink(0)
	bounds := body.Bounds()
	lastScroll := ui.Local(body, "patch-scroll", func() float32 { return scroll.Y })
	top := float32(0)
	if bounds.H > 0 {
		top = viewport.Y - bounds.Y + scroll.Y - *lastScroll
	}
	*lastScroll = scroll.Y
	viewHeight := viewport.H
	if viewHeight <= 0 {
		viewHeight = 640
	}
	// One viewport of overscan avoids empty edges during wheel/drag scrolling.
	lo, hi := top-viewHeight, top+2*viewHeight
	body.Children(func() {
		y, pending := float32(0), float32(0)
		spacer := func() {
			if pending > 0 {
				ui.Box(c).Height(pending).Shrink(0)
				pending = 0
			}
		}
		for index, height := range heights {
			if (y+height >= lo && y <= hi) || index == reveal || slices.Contains(retained, index) {
				spacer()
				draw(c.Key(index), index)
			} else {
				pending += height
			}
			y += height
		}
		spacer()
	})
}
