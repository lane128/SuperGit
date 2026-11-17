package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// sidebarTokens keeps the redesign's dark repository rail local: menus and
// the workbench continue using the window's normal light/dark tokens.
func (a *app) sidebarTokens(c *ui.Context) *theme.Tokens {
	tk := theme.TokensOf(c)
	if a.palette != themePaletteSuperGit {
		return tk
	}
	tk.BgContainer, tk.PrimaryBg = ui.Hex("#1D2D49"), ui.Hex("#344D79")
	tk.Text, tk.TextSecondary, tk.BorderSecondary = ui.Hex("#EFF4FF"), ui.Hex("#B4C3DF"), ui.Hex("#3A4D6D")
	tk.BgElevated, tk.FillTertiary = ui.Hex("#2C4268"), ui.Hex("#374F79")
	if tk.Dark {
		tk.BgContainer, tk.PrimaryBg = ui.Hex("#111D34"), ui.Hex("#22385F")
		tk.Text, tk.TextSecondary, tk.BorderSecondary = ui.Hex("#EAF0FB"), ui.Hex("#A9B8D3"), ui.Hex("#2A3A56")
		tk.BgElevated, tk.FillTertiary = ui.Hex("#1D2C48"), ui.Hex("#263858")
	}
	tk.TextTertiary = tk.TextSecondary
	tk.TextDisabled = tk.TextSecondary.Alpha(0.5)
	tk.Green = ui.Hex("#8AD5AA")
	return tk
}

func (a *app) sidebarSelectionBg(tk *theme.Tokens) ui.Color {
	if a.palette == themePaletteSuperGit {
		return tk.PrimaryBg
	}
	return selectionBg(tk, tk.BgContainer)
}
