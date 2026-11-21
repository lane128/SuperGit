package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func (a *app) sidebarTokens(c *ui.Context) *theme.Tokens {
	tk := theme.TokensOf(c)
	if a.palette != themePaletteSuperGit {
		return tk
	}
	if !tk.Dark {
		tk.BgContainer = tk.BgContainerSecondary
		return tk
	}
	// Unified Precision Graphite dark sidebar — matches the chrome and main
	// canvas color temperature without introducing a separate navy tint.
	tk.BgContainer, tk.PrimaryBg = ui.Hex("#12161D"), ui.Hex("#1B2B42")
	tk.Text, tk.TextSecondary, tk.BorderSecondary = ui.Hex("#E6EDF3"), ui.Hex("#9BA6B2"), ui.Hex("#252B33")
	tk.BgElevated, tk.FillTertiary = ui.Hex("#1C2128"), ui.Hex("#21262D")
	tk.TextTertiary = tk.TextSecondary
	tk.TextDisabled = tk.TextSecondary.Alpha(0.5)
	return tk
}

func (a *app) sidebarSelectionColors(tk *theme.Tokens) (background, text ui.Color) {
	if a.palette != themePaletteSuperGit {
		return selectionBg(tk, tk.BgContainer), tk.Text
	}
	if tk.Dark {
		return tk.PrimaryBg, tk.Text
	}
	return tk.PrimaryBg, tk.PrimaryText
}

func sidebarGroupHeaderColors(tk *theme.Tokens, hovered bool) (bg, divider, badgeBg ui.Color) {
	return foldBandColors(tk, hovered)
}

func foldBandColors(tk *theme.Tokens, hovered bool) (bg, divider, badgeBg ui.Color) {
	if tk.Dark {
		mix := float32(0.045)
		if hovered {
			mix = 0.085
		}
		bg = tk.BgContainer.Mix(tk.Text, mix)
		divider = bg.Mix(tk.Border, 0.35)
		return bg, divider, tk.BgContainer
	}
	mix := float32(0.036)
	if hovered {
		mix = 0.068
	}
	bg = tk.BgContainer.Mix(tk.Text, mix)
	divider = bg.Mix(tk.Border, 0.30)
	return bg, divider, tk.BgContainer
}
