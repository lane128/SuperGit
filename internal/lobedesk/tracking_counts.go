package lobedesk

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// trackingInk is the one color pair for ahead (outgoing) and behind
// (incoming) counts, shared by the title bar, sync menu and branch views.
func trackingInk(tk *theme.Tokens, incoming bool) (*ui.SVG, ui.Color) {
	if incoming {
		return lucide.ArrowDown(), tk.Blue
	}
	return lucide.ArrowUp(), tk.SuccessText
}

// trackingCount draws one arrow and a tabular number in the code font, so
// counts line up and read the same wherever they appear.
func (a *app) trackingCount(c *ui.Context, tk *theme.Tokens, incoming bool, n int, size float32, surface ui.Color) {
	icon, ink := trackingInk(tk, incoming)
	ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(2).TextColor(capsuleInk(ink, surface, 4.5)).AccessHidden(true).Children(func() {
		ui.Icon(c, icon).Size(size, size)
		ui.Text(c, strconv.Itoa(n)).Font(a.patchFontFamily()).FontSize(size).SingleLine()
	})
}

// trackingCounts draws only the nonzero sides. A nonempty spoken label keeps
// the counts available to assistive technology; otherwise the caller already
// speaks them.
func (a *app) trackingCounts(c *ui.Context, tk *theme.Tokens, ahead, behind int, size float32, surface ui.Color, spoken string) {
	if ahead == 0 && behind == 0 {
		return
	}
	group := ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(6)
	if spoken != "" {
		group.Label(spoken)
	} else {
		group.AccessHidden(true)
	}
	group.Children(func() {
		if ahead > 0 {
			a.trackingCount(c, tk, false, ahead, size, surface)
		}
		if behind > 0 {
			a.trackingCount(c, tk, true, behind, size, surface)
		}
	})
}
