package lobedesk

import (
	_ "embed"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Status letters use one bundled face instead of the system UI or code font:
// macOS resolves SF Pro, other platforms resolve their own UI font, and the
// letter must read the same everywhere. Inter Bold is SIL OFL, so it can ship
// inside the app. The family stays out of the interface and code font settings.
const statusMarkerFontFamily = "SuperGit Status Text"

//go:embed fonts/Inter-Bold.ttf
var statusMarkerBoldTTF []byte

var statusMarkerFontOnce sync.Once

func ensureStatusMarkerFont() {
	statusMarkerFontOnce.Do(func() {
		_ = ui.RegisterFont(statusMarkerBoldTTF, statusMarkerFontFamily)
	})
}

func statusMarkerFont() string {
	ensureStatusMarkerFont()
	return statusMarkerFontFamily
}

// statusInkColors returns the local light/dark ink and the contrast floor for
// a status letter. A/M/D carry the most weight, so they aim higher.
func statusInkColors(code string) (light, dark string, minimum float64) {
	light, dark, minimum = "#7040cc", "#bc8cff", 4.5
	switch code {
	case "A":
		light, dark, minimum = "#06501f", "#70e27e", 7
	case "M", "T":
		light, dark, minimum = "#603b00", "#ffd166", 7
	case "D", "!":
		light, dark, minimum = "#8a1020", "#ffaaa2", 7
	case "?":
		light, dark = "#085bbd", "#58a6ff"
	}
	return light, dark, minimum
}

// statusMarkerInk keeps the letter readable on every ground a file row can
// show: idle, hover/press and the selected tint.
func statusMarkerInk(tk *theme.Tokens, code string) ui.Color {
	light, dark, minimum := statusInkColors(code)
	ink := ui.Hex(light)
	if tk.Dark {
		ink = ui.Hex(dark)
	}
	for _, ground := range []ui.Color{tk.BgContainer, tk.BgLayout, selectionBg(tk, tk.BgContainer)} {
		for _, bg := range []ui.Color{ground, tk.FillTertiary.Over(ground), tk.Fill.Over(ground)} {
			ink = capsuleInk(ink, bg, minimum)
		}
	}
	return ink
}

// statusBadge paints the hollow 18-DIP badge shared by file rows and detail
// tabs. A non-empty meaning exposes it as an image for screen readers; the
// detail tab passes none because its tab description already carries it.
func statusBadge(c *ui.Context, ink ui.Color, code, meaning string) ui.Element {
	badge := ui.Box(c).Width(scaledFont(18)).Height(scaledFont(18)).Shrink(0).Center().
		Border(1, ink).Radius(scaledFont(4))
	if meaning != "" {
		badge.Role(ui.RoleImage).Label(meaning)
	} else {
		badge.AccessHidden(true)
	}
	return badge.Children(func() {
		ui.Text(c, code).Font(statusMarkerFont()).FontSize(scaledFont(11)).FontWeight(800).
			TextColor(ink).SingleLine().AccessHidden(true)
	})
}

// fileStatusMarker places the change-type badge ahead of a file row's name.
func (a *app) fileStatusMarker(c *ui.Context, tk *theme.Tokens, code string) {
	if code == "" || code == " " {
		return
	}
	statusBadge(c, statusMarkerInk(tk, code), code, a.detailStatusMeaning(code))
}

// statusMarkerCode normalizes a raw porcelain or commit status to one letter.
func statusMarkerCode(code string) string {
	if code == "" || code == " " {
		return "?"
	}
	return code[:1]
}
