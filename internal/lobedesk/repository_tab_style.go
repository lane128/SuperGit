package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// repositoryFilledFolderIcon is a solid folder silhouette rendered in the
// repository's assigned tag color so colored checkouts stand out cleanly
// against the neutral Sublime-style tab chrome.
var repositoryFilledFolderIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M1.75 3.75A1.75 1.75 0 0 1 3.5 2h2.72c.46 0 .9.2 1.2.55l.83.95h4.25c.97 0 1.75.78 1.75 1.75v6.5A1.75 1.75 0 0 1 12.5 13.5h-9a1.75 1.75 0 0 1-1.75-1.75v-8Z"/></svg>`))

type repositoryChromePalette struct {
	track     ui.Color
	trackLine ui.Color
	deck      ui.Color
	tabEdge   ui.Color
	sep       ui.Color
	inkActive ui.Color
	inkIdle   ui.Color
	inkHover  ui.Color
}

func repositoryChrome(tk *theme.Tokens) repositoryChromePalette {
	if !tk.Dark && tk.BgLayout == ui.Hex("#F3F4F6") {
		return repositoryChromePalette{
			track:     ui.Hex("#E5E9EF"),
			trackLine: ui.Hex("#D4DBE3"),
			deck:      ui.Hex("#FFFFFF"),
			tabEdge:   ui.Hex("#D2D9E2"),
			sep:       ui.Hex("#1F2328").Alpha(.14),
			inkActive: ui.Hex("#1F2328"),
			inkIdle:   ui.Hex("#5C6672"),
			inkHover:  ui.Hex("#1F2328"),
		}
	}
	if tk.Dark && tk.BgLayout == ui.Hex("#0D1117") {
		return repositoryChromePalette{
			track:     ui.Hex("#0C1016"),
			trackLine: ui.Hex("#252C36"),
			deck:      ui.Hex("#1C2128"),
			tabEdge:   ui.Hex("#2E3642"),
			sep:       ui.Hex("#E6EDF3").Alpha(.13),
			inkActive: ui.Hex("#E6EDF3"),
			inkIdle:   ui.Hex("#8B949E"),
			inkHover:  ui.Hex("#DCE4ED"),
		}
	}
	sep := tk.Text.Alpha(.14)
	if tk.Dark {
		sep = tk.Text.Alpha(.13)
	}
	return repositoryChromePalette{
		track:     tk.BgLayout,
		trackLine: tk.BorderSecondary,
		deck:      tk.BgElevated,
		tabEdge:   tk.Border,
		sep:       sep,
		inkActive: tk.Text,
		inkIdle:   tk.TextSecondary,
		inkHover:  tk.Text,
	}
}

// repositoryTagColor returns the WCAG-AA calibrated foreground color for a
// non-default repository color preset. Default returns zero/false.
func repositoryTagColor(dark bool, preset string) (ui.Color, bool) {
	tags := map[string][2]string{
		"blue":   {"#0969DA", "#58A6FF"},
		"green":  {"#1A7F37", "#3FB950"},
		"amber":  {"#9A6700", "#E3B341"},
		"rose":   {"#BF3989", "#F778BA"},
		"violet": {"#7C3AED", "#BC8CFF"},
		"gray":   {"#57606A", "#9CA6B2"},
	}
	pair, ok := tags[normalizedRepositoryColor(preset)]
	if !ok {
		return ui.Transparent, false
	}
	if dark {
		return ui.Hex(pair[1]), true
	}
	return ui.Hex(pair[0]), true
}

func repositoryCardWidth(viewport float32, count int) float32 {
	if viewport <= 0 || count < 1 {
		return 192
	}
	return max(168, min(192, (viewport-4)/float32(count)))
}

func repositoryCaptionWidth(card float32) float32 {
	return card - 2 - 12 - scaledFont(15) - 8 - 8 - 24 - 5
}

// repositoryTabHeight leaves 3 DIPs above the tab top edge inside the 38 DIP
// title strip and runs flush to the strip bottom so the active tab covers the
// strip's 1px divider and merges seamlessly into the toolbar below.
const repositoryTabHeight = titleBarHeight - 3

// paintRepositoryTab styles one tab in the Sublime-inspired seamless deck:
//   - Selected tab matches the toolbar surface below with a 1px top/side edge
//     and no bottom border or accent line.
//   - Unselected tab stays transparent on the recessed strip track, with a
//     centered 1px vertical hairline before it when adjacent to another idle tab.
//   - Hover never paints a background fill; it gently brightens idle foreground ink.
func paintRepositoryTab(tab ui.Element, tk *theme.Tokens, preset string, selected, blocked, showLeadingSep bool) (ink, iconInk, closeInk ui.Color, filledIcon bool) {
	chrome := repositoryChrome(tk)
	hovered := !blocked && tab.Hovered()

	bg := ui.Transparent
	edge := ui.Transparent
	ink = chrome.inkIdle
	closeInk = chrome.inkIdle.Alpha(.72)
	if selected {
		bg = chrome.deck
		edge = chrome.tabEdge
		ink = chrome.inkActive
		closeInk = chrome.inkActive.Alpha(.78)
	} else if hovered {
		ink = chrome.inkHover
	}
	if hovered {
		closeInk = chrome.inkActive
	}

	iconInk = ink
	if tag, ok := repositoryTagColor(tk.Dark, preset); ok {
		filledIcon = true
		iconInk = tag
		if !selected && !hovered {
			iconInk = tag.Alpha(.88)
		}
	}

	tab.Background(bg).Radius(6, 6, 0, 0).
		BorderWidth(1, 1, 0, 1).BorderColor(edge).
		Draw(func(p *ui.Painter, r ui.Rect) {
			if showLeadingSep {
				p.Fill(ui.Rect{X: r.X, Y: r.Y + 9, W: 1, H: r.H - 18}, chrome.sep, 0)
			}
		})
	return ink, iconInk, closeInk, filledIcon
}

func styleRepositoryTabTarget(target ui.Element, ink ui.Color) {
	target.FocusRing(false).Draw(func(p *ui.Painter, r ui.Rect) {
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
	target.Size(24, 24).Padding(0).Border(0, ui.Transparent).NoShadow().Radius(4).
		Background(ui.Transparent).TextColor(ink).FocusRing(false).DimDisabled(false)
	target.Draw(func(p *ui.Painter, r ui.Rect) {
		if target.FocusVisible() {
			r.X++
			r.Y++
			r.W -= 2
			r.H -= 2
			p.StrokeDashed(r, ink, 4, 1)
		}
	})
}
