package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// supergitTokens maps the redesign's semantic roles onto the existing theme
// system. Controls and both windows resolve the same light/dark token pair.
// Button fills are separate from semantic success/error text and selection.
func supergitTokens(dark bool) theme.Tokens {
	p := catppuccinLatte
	base := *theme.Light()
	if dark {
		p, base = catppuccinMocha, *theme.Dark()
	}
	if dark {
		p.Crust, p.Base, p.Mantle = ui.Hex("#13161B"), ui.Hex("#191C21"), ui.Hex("#1B1E24")
		p.Surface0, p.Surface1, p.Surface2 = ui.Hex("#232831"), ui.Hex("#303641"), ui.Hex("#8996AA")
		p.Text, p.Subtext1, p.Subtext0 = ui.Hex("#ECF0F7"), ui.Hex("#A8B4C7"), ui.Hex("#93A1B8")
		p.Overlay0, p.Overlay1 = p.Subtext0, p.Subtext0
		p.Blue, p.Green, p.Red, p.Yellow = ui.Hex("#8CB4FF"), ui.Hex("#8AD5AA"), ui.Hex("#F2A1AE"), ui.Hex("#EEC783")
	} else {
		p.Crust, p.Base, p.Mantle = ui.Hex("#F3F5F8"), ui.Hex("#FFFFFF"), ui.Hex("#F9FAFC")
		p.Surface0, p.Surface1, p.Surface2 = ui.Hex("#EAEFF6"), ui.Hex("#DCE3ED"), ui.Hex("#627084")
		p.Text, p.Subtext1, p.Subtext0 = ui.Hex("#1B2535"), ui.Hex("#4D5B70"), ui.Hex("#5B6A80")
		p.Overlay0, p.Overlay1 = p.Subtext0, p.Subtext0
		p.Blue, p.Green, p.Red, p.Yellow = ui.Hex("#2456BD"), ui.Hex("#16603A"), ui.Hex("#9A3346"), ui.Hex("#7D521A")
	}
	tk := p.tokens(base)
	tk.BorderSecondary, tk.Split = p.Surface1, p.Surface1
	tk.TextLightSolid = ui.Hex("#FFFFFF")
	tk.PrimaryForeground = ui.Hex("#FFFFFF")
	if dark {
		tk.PrimaryForeground = p.Crust
	}
	tk.PrimaryText, tk.SuccessText, tk.ErrorText, tk.WarningText = p.Blue, p.Green, p.Red, p.Yellow
	if dark {
		tk.BgElevated = ui.Hex("#262D38")
		tk.BgContainerDisabled = ui.Hex("#252A33")
		tk.PrimaryBg, tk.SuccessBg, tk.ErrorBg, tk.WarningBg = ui.Hex("#213653"), ui.Hex("#1A3428"), ui.Hex("#39252D"), ui.Hex("#3C3321")
	} else {
		tk.BgContainerDisabled = ui.Hex("#EFF2F6")
		tk.PrimaryBg, tk.SuccessBg, tk.ErrorBg, tk.WarningBg = ui.Hex("#E8F0FF"), ui.Hex("#EAF6EE"), ui.Hex("#FAECEF"), ui.Hex("#FFF2DC")
	}
	return tk
}
