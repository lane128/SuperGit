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
	share     float32
	rowWidth  float32
	dragWidth float32
	held      bool
	moved     bool
}

// fraction is the left pane's share, clamped so neither pane drops below its
// minimum width. def is the share used until the reader drags the divider.
func (s *splitDrag) fraction(def, minLeft, minRight float32) float32 {
	share := s.share
	if share <= 0 {
		share = def
	}
	if room := s.rowWidth - splitterWidth; s.held && s.moved && room > 0 {
		share = s.dragWidth / room
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
	// Nested dividers can leave the exact minimum a few float32 ULPs short.
	// Do not turn a rounding error into a 50/50 split that breaks both panes.
	if lo > hi && lo-hi > 0.00001 {
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

// finishDrag commits only a visible change from the preferred width at the
// current window size. An immovable divider must not erase a wider preference.
func (s *splitDrag) finishDrag(def, minLeft, minRight float32) bool {
	final := s.fraction(def, minLeft, minRight)
	s.held = false
	s.moved = false
	preferred := s.fraction(def, minLeft, minRight)
	const shareTolerance = 0.000001
	if final >= preferred-shareTolerance && final <= preferred+shareTolerance {
		return false
	}
	s.share = final
	return true
}

// splitHandle previews a drag and saves the changed share on release.
// A double click restores and saves the page's default.
func (a *app) splitHandle(c *ui.Context, s *splitDrag, def, minLeft, minRight float32, tip string) {
	tk := theme.TokensOf(c)
	divider := ui.Box(c).Width(splitterWidth).Shrink(0).FillHeight()
	room := s.rowWidth - splitterWidth
	// Ignore float32 rounding at the exact minimum. At a one-sided limit
	// with room left, the divider is still draggable in the other direction.
	movable := room-minLeft-minRight > .01
	if movable {
		divider.Cursor(ui.CursorResizeEW).Tooltip(tip)
	}
	hovered := movable && divider.Hovered()
	divider.Draw(func(p *ui.Painter, r ui.Rect) {
		ink, width := tk.BorderSecondary, float32(1)
		if hovered || movable && s.held {
			ink, width = tk.Blue, 2
		}
		x := r.X + r.W/2
		p.Line(x, r.Y, x, r.Y+r.H, width, ink)
	})
	if !movable {
		s.held, s.moved = false, false
		return
	}
	button, pressed := divider.PressedButton()
	if pressed && button == 0 {
		if !s.held {
			s.dragWidth = s.fraction(def, minLeft, minRight) * room
			s.held = true
			s.moved = false
		}
		if dx, _, dragging := divider.Dragged(); dragging && dx != 0 && room > 0 {
			// Keep pointer travel past either limit so reversing only moves
			// the divider once the pointer returns to its usable range.
			s.dragWidth += dx
			s.moved = true
			a.invalidate()
		}
		// Tracking the pointer asks for a frame on every move, which a still
		// drag otherwise would not get.
		divider.PointerPosition()
	} else if s.held && s.finishDrag(def, minLeft, minRight) {
		a.persistSettings()
	}
	if divider.DoubleClicked() {
		s.share = 0
		s.moved = false
		// A double click is reported on press. A drag that follows while
		// still holding must start at the newly restored default width.
		s.dragWidth = s.fraction(def, minLeft, minRight) * (s.rowWidth - splitterWidth)
		a.persistSettings()
		a.invalidate()
	}
}
