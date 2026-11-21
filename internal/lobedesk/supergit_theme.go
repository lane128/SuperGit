package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// supergitTokens maps the Plan A+ (Precision Native + GitHub Primer) semantic
// roles onto the existing theme system. Controls and both windows resolve the
// same light/dark token pair, and dark surfaces stay unified in neutral graphite.
func supergitTokens(dark bool) theme.Tokens {
	p := catppuccinLatte
	base := *theme.Light()
	if dark {
		p, base = catppuccinMocha, *theme.Dark()
	}
	if dark {
		p.Crust, p.Base, p.Mantle = ui.Hex("#0D1117"), ui.Hex("#161B22"), ui.Hex("#12161D")
		p.Surface0, p.Surface1, p.Surface2 = ui.Hex("#21262D"), ui.Hex("#30363D"), ui.Hex("#6E7681")
		p.Text, p.Subtext1, p.Subtext0 = ui.Hex("#E6EDF3"), ui.Hex("#8B949E"), ui.Hex("#8B949E")
		p.Overlay0, p.Overlay1 = p.Subtext0, p.Subtext0
		p.Blue, p.Green, p.Red, p.Yellow = ui.Hex("#58A6FF"), ui.Hex("#3FB950"), ui.Hex("#F85149"), ui.Hex("#D29922")
	} else {
		p.Crust, p.Base, p.Mantle = ui.Hex("#F3F4F6"), ui.Hex("#FFFFFF"), ui.Hex("#F7F8FA")
		p.Surface0, p.Surface1, p.Surface2 = ui.Hex("#E5E8EC"), ui.Hex("#D0D7DE"), ui.Hex("#656D76")
		p.Text, p.Subtext1, p.Subtext0 = ui.Hex("#1F2328"), ui.Hex("#59636E"), ui.Hex("#656D76")
		p.Overlay0, p.Overlay1 = p.Subtext0, p.Subtext0
		p.Blue, p.Green, p.Red, p.Yellow = ui.Hex("#0969DA"), ui.Hex("#167432"), ui.Hex("#CF222E"), ui.Hex("#8A5B00")
	}
	tk := p.tokens(base)
	tk.BorderSecondary, tk.Split = p.Surface0, p.Surface0
	tk.TextLightSolid = ui.Hex("#FFFFFF")
	tk.PrimaryForeground = ui.Hex("#FFFFFF")
	if dark {
		tk.PrimaryForeground = p.Crust
	}
	tk.PrimaryText, tk.SuccessText, tk.ErrorText, tk.WarningText = p.Blue, p.Green, p.Red, p.Yellow
	if dark {
		tk.BgElevated = ui.Hex("#1C2128")
		tk.BgContainerDisabled = ui.Hex("#1B2027")
		tk.PrimaryBg, tk.SuccessBg, tk.ErrorBg, tk.WarningBg = ui.Hex("#182940"), ui.Hex("#182E22"), ui.Hex("#331D22"), ui.Hex("#312817")
	} else {
		tk.BgElevated = ui.Hex("#FFFFFF")
		tk.BgContainerDisabled = ui.Hex("#F6F8FA")
		tk.PrimaryBg, tk.SuccessBg, tk.ErrorBg, tk.WarningBg = ui.Hex("#EDF4FF"), ui.Hex("#DAFBE1"), ui.Hex("#FFEBE9"), ui.Hex("#FFF8C5")
	}
	return tk
}
