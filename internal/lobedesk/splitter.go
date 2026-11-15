package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// splitterWidth is the hit area of a pane divider. The line it draws is one
// pixel until the pointer arrives, so the gap reads as an ordinary border.
const splitterWidth float32 = 7

// splitDrag is one draggable divider between two panes sharing a row: the left
// pane's share of the room, the row's width from the last layout, and the
// press state while the reader drags it. A share of zero means the page has
// not been dragged yet and its own default stands in, so a reset is just zero.
type splitDrag struct {
	share    float32
	rowWidth float32
	held     bool
	moved    bool
}

// fraction is the left pane's share, clamped so neither pane drops below its
// minimum width. def is the share used until the reader drags the divider.
func (s *splitDrag) fraction(def, minLeft, minRight float32) float32 {
	share := s.share
	if share <= 0 {
		share = def
	}
	return clampSplit(share, s.rowWidth, minLeft, minRight)
}

// clampSplit keeps both panes usable on a row of the given width. A row too
// narrow for the two minimums falls back to an even split, which is the least
// lopsided choice left.
func clampSplit(share, rowWidth, minLeft, minRight float32) float32 {
	if rowWidth <= 0 {
		return share
	}
	room := rowWidth - splitterWidth
	if room <= 0 {
		return share
	}
	lo := minLeft / room
	hi := 1 - minRight/room
	if lo > hi {
		return 0.5
	}
	if share < lo {
		return lo
	}
	if share > hi {
		return hi
	}
	return share
}

// shareForWidth turns a pane's opening width in points into a share of the
// row, for pages whose default is a pixel width rather than a proportion.
func shareForWidth(rowWidth, defaultWidth float32) float32 {
	if rowWidth <= 0 {
		return 0
	}
	return defaultWidth / rowWidth
}

// splitHandle draws a divider and runs its drag: dragging changes the share,
// a double click returns to the page's default, and both are saved when the
// gesture ends. The line stays quiet until the pointer is on it, when it takes
// the accent color and the resize cursor, the way a split view's divider does.
func (a *app) splitHandle(c *ui.Context, s *splitDrag, def, minLeft, minRight float32, tip string) {
	tk := theme.TokensOf(c)
	divider := ui.Box(c).Width(splitterWidth).Shrink(0).FillHeight().
		Cursor(ui.CursorResizeEW).Tooltip(tip)
	button, pressed := divider.PressedButton()
	if pressed && button == 0 {
		if !s.held {
			s.held = true
			s.moved = false
		}
		if dx, _, dragging := divider.Dragged(); dragging && dx != 0 && s.rowWidth > splitterWidth {
			room := s.rowWidth - splitterWidth
			s.share = clampSplit(s.fraction(def, minLeft, minRight)+dx/room, s.rowWidth, minLeft, minRight)
			s.moved = true
			a.invalidate()
		}
		// Tracking the pointer asks for a frame on every move, which a still
		// drag otherwise would not get.
		divider.PointerPosition()
	} else if s.held {
		s.held = false
		// The width matters across restarts, so a drag is saved when it ends
		// rather than on every frame of it.
		if s.moved {
			s.moved = false
			a.persistSettings()
		}
	}
	if divider.DoubleClicked() {
		s.share = 0
		a.persistSettings()
		a.invalidate()
	}
	hovered := divider.Hovered()
	divider.Draw(func(p *ui.Painter, r ui.Rect) {
		ink, width := tk.BorderSecondary, float32(1)
		if hovered || s.held {
			ink, width = tk.Blue, 2
		}
		x := r.X + r.W/2
		p.Line(x, r.Y, x, r.Y+r.H, width, ink)
	})
}
