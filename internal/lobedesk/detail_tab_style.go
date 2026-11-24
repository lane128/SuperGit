package lobedesk

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Detail tabs have their own flat treatment; repository tabs retain theirs.
func paintDetailTab(tab ui.Element, tk *theme.Tokens, active bool) ui.Color {
	ground, ink := tk.BgLayout, tk.TextSecondary
	if active {
		ground, ink = tk.BgContainer, tk.Text
	}
	if tab.Pressed() {
		ground = tk.Fill.Over(ground)
	} else if tab.Hovered() {
		ground = tk.FillTertiary.Over(ground)
	}
	tab.Background(ground).Draw(func(p *ui.Painter, r ui.Rect) {
		if active {
			p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 2, W: r.W, H: 2}, tk.Blue, 0)
		} else {
			p.Fill(ui.Rect{X: r.X + r.W - 1, Y: r.Y + 8, W: 1, H: r.H - 16}, tk.BorderSecondary, 0)
		}
	})
	return ink
}

func detailStatusInk(tk *theme.Tokens, code string) ui.Color {
	light, dark, minimum := statusInkColors(code)
	ink := ui.Hex(light)
	if tk.Dark {
		ink = ui.Hex(dark)
	}
	// Include both selected and idle hover/press grounds. The existing capsule
	// helper preserves hue while deepening/lightening only failing colors.
	for _, ground := range []ui.Color{tk.BgContainer, tk.BgLayout} {
		for _, bg := range []ui.Color{ground, tk.FillTertiary.Over(ground), tk.Fill.Over(ground)} {
			ink = capsuleInk(ink, bg, minimum)
		}
	}
	return ink
}

func (a *app) detailStatusMeaning(code string) string {
	key := map[string]string{"A": "statusAdded", "M": "statusModified", "D": "statusDeleted", "?": "statusUntracked", "R": "statusRenamed", "C": "statusCopied", "T": "statusTypeChanged", "!": "statusConflict", "S": "statusSubmodule"}[code]
	if key == "" {
		return code
	}
	return code + " " + a.t(key)
}

func (a *app) detailStatusBadge(c *ui.Context, tk *theme.Tokens, code string) {
	statusBadge(c, detailStatusInk(tk, code), code, "")
}

func detailThumb(width, maxX, offset float32) (left, length float32) {
	if width <= 0 || maxX <= 0 {
		return 0, 0
	}
	length = min(width, max(24, width*width/(width+maxX)))
	left = (width - length) * max(0, min(maxX, offset)) / maxX
	return
}

func detailTrackAnchor(x, initialDelta, left, length float32) float32 {
	// PointerPosition is the latest move, even when down and moves arrive
	// before the first build. Recover the original press before anchoring.
	x -= initialDelta
	if x < left || x > left+length {
		return length / 2
	}
	return x - left
}

// The native scroll container still owns wheel input, clamping and reveal.
// Only its visual/pointer scrollbar moves into a separate six-DIP row.
func (a *app) detailTabTrack(c *ui.Context, tk *theme.Tokens) {
	track := ui.Box(c).FillWidth().Height(6).Shrink(0).Background(tk.BgLayout).KeepFocus()
	grabbed := ui.Local(track, "grabbed", func() bool { return false })
	anchor := ui.Local(track, "anchor", func() float32 { return 0 })
	width := track.Bounds().W
	left, length := detailThumb(width, a.detailTabStrip.MaxX, a.detailTabStrip.X)
	x, _, _ := track.PointerPosition()
	dx, _, held := track.Dragged()
	if track.PressedInside() && length > 0 {
		*grabbed = true
		*anchor = detailTrackAnchor(x, dx, left, length)
	}
	if *grabbed && held && width > length {
		next := max(0, min(a.detailTabStrip.MaxX, (x-*anchor)/(width-length)*a.detailTabStrip.MaxX))
		if math.Abs(float64(next-a.detailTabStrip.X)) > .01 {
			a.detailTabStrip.X = next
			c.Invalidate()
		}
	}
	if !held {
		*grabbed = false
	}
	expanded := track.Hovered() || held
	color := c.Theme().Scrollbar
	track.Draw(func(p *ui.Painter, r ui.Rect) {
		left, length := detailThumb(r.W, a.detailTabStrip.MaxX, a.detailTabStrip.X)
		if length == 0 {
			return
		}
		height := float32(3)
		if expanded {
			height = 6
		}
		p.Fill(ui.Rect{X: r.X + left, Y: r.Y + (r.H-height)/2, W: length, H: height}, color, height/2)
	})
}
