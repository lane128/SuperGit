package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// filledTagInk is the label color for a filled status tag on a light surface.
// Lobe paints both the label and the tag's 10% tint from the base accent,
// which leaves the label below AA, and the tint over a selected row darkens
// the ground further. The ink is darkened toward black, not toward body text,
// so the family's hue survives; the 5.2 target over the row leaves headroom
// for the tag's own tint. Dark surfaces keep Lobe's rendering untouched.
func filledTagInk(tk *theme.Tokens, family string) (ui.Color, bool) {
	if surfaceLuminance(tk.BgContainer) < 0.5 {
		return ui.Color{}, false
	}
	base, ok := refColor(tk, family)
	if !ok {
		return ui.Color{}, false
	}
	surface := selectionBg(tk, tk.BgContainer)
	return deepInk(base, surface, 5.2), true
}

// statusTag draws a Lobe tag and keeps its label readable; see filledTagInk.
func (a *app) statusTag(c *ui.Context, label string, p display.TagProps) ui.Element {
	tag := display.Tag(c, label, p)
	if p.Color == "" {
		return tag
	}
	if ink, ok := filledTagInk(theme.TokensOf(c), string(p.Color)); ok {
		tag.TextColor(ink)
	}
	return tag
}
