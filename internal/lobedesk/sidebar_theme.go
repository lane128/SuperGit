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
		tk.BgContainer = tk.BgLayout
		return tk
	}
	tk.BgContainer, tk.PrimaryBg = ui.Hex("#111D34"), ui.Hex("#22385F")
	tk.Text, tk.TextSecondary, tk.BorderSecondary = ui.Hex("#EAF0FB"), ui.Hex("#A9B8D3"), ui.Hex("#2A3A56")
	tk.BgElevated, tk.FillTertiary = ui.Hex("#1D2C48"), ui.Hex("#263858")
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
