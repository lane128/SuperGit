package lobedesk

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// The same metrics drive the fitting budget and the rendered element.
func (a *app) historyRefLeading(icon, head bool) float32 {
	if historyRefStyleKey(a.historyRefStyle) == "segmented" {
		if head {
			return scaledFont(34)
		}
		if icon {
			return scaledFont(20)
		}
		return 0
	}
	if head {
		return scaledFont(29)
	}
	if icon {
		return scaledFont(16)
	}
	return 0
}

func (a *app) historyNamedRefWidth(c *ui.Context, label string, icon, head bool) float32 {
	weight := 500
	if head {
		weight = 600
	}
	w, _ := c.MeasureText(0, ui.Span{Text: label, Size: scaledFont(11), Weight: weight})
	return w + a.historyRefLeading(icon, head) + scaledFont(12) + 2
}

// Prefer the meaningful last component before shortening the component itself.
func historyRefCandidate(label string, n int) string {
	if utf8.RuneCountInString(label) <= n {
		return label
	}
	if i := strings.LastIndex(label, "/"); i >= 0 {
		leaf := label[i+1:]
		remaining := n - utf8.RuneCountInString(leaf)
		if remaining >= 2 {
			prefix := []rune(label[:i+1])
			take := min(len(prefix), remaining-2)
			return string(prefix[:take]) + "…/" + leaf
		}
		return elideMiddle(leaf, n)
	}
	return elideMiddle(label, n)
}

func (a *app) historyNamedRefLabel(c *ui.Context, label string, icon, head bool, limit float32) string {
	if a.historyNamedRefWidth(c, label, icon, head) <= limit {
		return label
	}
	// Widths grow with the rune budget, so find the longest label that fits
	// without measuring every prefix: a long branch name would otherwise cost
	// one text layout per rune on every frame it is drawn.
	lo, hi, best := 1, utf8.RuneCountInString(label)-1, ""
	for lo <= hi {
		n := (lo + hi) / 2
		candidate := historyRefCandidate(label, n)
		if a.historyNamedRefWidth(c, candidate, icon, head) <= limit {
			best, lo = candidate, n+1
		} else {
			hi = n - 1
		}
	}
	return best
}

type historyRefColors struct{ surface, ink, muted, mark, segment, border ui.Color }

func historyRefColorsFor(tk *theme.Tokens, accent, ground ui.Color, head bool) historyRefColors {
	surface := ground
	if head {
		surface = accent.Alpha(.08).Over(ground)
	}
	segment := accent.Alpha(.10).Over(surface)
	if head {
		segment = accent.Alpha(.18).Over(surface)
	}
	ink := capsuleInk(tk.Text, surface, 4.5)
	return historyRefColors{surface: surface, ink: ink, muted: capsuleInk(tk.TextSecondary, surface, 4.5), mark: capsuleInk(accent, segment, 4.5), segment: segment, border: capsuleBorder(accent.Alpha(.22).Over(surface), ink, ground)}
}

// Only history rows use these treatments; other views keep their existing chips.
func (a *app) historyNamedRefPill(c *ui.Context, label string, icon *ui.SVG, head bool, accent, ground ui.Color) ui.Element {
	tk := theme.TokensOf(c)
	colors := historyRefColorsFor(tk, accent, ground, head)
	segmented := historyRefStyleKey(a.historyRefStyle) == "segmented"
	h := scaledFont(20)
	leading := a.historyRefLeading(icon != nil, head)
	pill := ui.Row(c).Height(h).Width(a.historyNamedRefWidth(c, label, icon != nil, head)).Shrink(0).AlignItems(ui.Center).Radius(scaledFont(5)).Background(colors.surface).Border(1, colors.border).Clip()
	if !segmented {
		pill.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Clip(r, scaledFont(5), func() { p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 2, H: r.H}, accent, 0) })
		})
	}
	pill.Children(func() {
		if !segmented {
			ui.Box(c).Width(scaledFont(6)).Shrink(0)
		}
		if leading > 0 {
			mark := ui.Row(c).Width(leading).Height(h - 2).Shrink(0).AlignItems(ui.Center).Justify(ui.Center)
			if segmented {
				mark.Background(colors.segment)
			}
			mark.Children(func() {
				if head {
					ui.Text(c, "HEAD").FontSize(scaledFont(8)).FontWeight(600).TextColor(colors.mark).SingleLine()
				} else if icon != nil {
					ui.Icon(c, icon).FontSize(scaledFont(12)).TextColor(colors.mark)
				}
			})
			if segmented {
				mark.Draw(func(p *ui.Painter, r ui.Rect) { p.Line(r.X+r.W-0.5, r.Y, r.X+r.W-0.5, r.Y+r.H, 1, colors.border) })
			}
		}
		text := ui.Row(c).MinWidth(0).Grow(1).AlignItems(ui.Center)
		if segmented {
			text.PaddingX(scaledFont(6))
		} else {
			text.Padding(0, scaledFont(6), 0, 0)
		}
		text.Children(func() {
			weight := 500
			if head {
				weight = 600
			}
			// Plain texts keep the row's own line metrics, which centers the
			// label on the capsule. A rich-text box carries a taller line box
			// and sits visibly low in a 20-point pill.
			if i := strings.LastIndex(label, "/"); i >= 0 {
				ui.Text(c, label[:i+1]).FontSize(scaledFont(11)).FontWeight(weight).TextColor(colors.muted).SingleLine()
				ui.Text(c, label[i+1:]).FontSize(scaledFont(11)).FontWeight(weight).TextColor(colors.ink).SingleLine()
			} else {
				ui.Text(c, label).FontSize(scaledFont(11)).FontWeight(weight).TextColor(colors.ink).SingleLine()
			}
		})
	})
	return pill
}

func (a *app) historyStashBadge(c *ui.Context, stash gitrepo.Stash, ground ui.Color, limit float32) {
	tk := theme.TokensOf(c)
	label := a.historyNamedRefLabel(c, stash.Ref, true, false, limit)
	a.historyNamedRefPill(c, label, lucide.Archive(), false, tk.Gold, ground).MaxWidth(max(0, limit)).Tooltip(stash.Ref)
}
