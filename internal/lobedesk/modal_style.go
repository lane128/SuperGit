package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// workbenchModal strips ui.Modal's default gray padding frame so the inner
// dialog sheet owns a single elevated card, border, corner radius and shadow.
func workbenchModal(c *ui.Context, open *bool, fn func()) ui.Element {
	tk := theme.TokensOf(c)
	shadow := ui.RGBA(31, 35, 40, 0.24)
	if tk.Dark {
		shadow = ui.RGBA(1, 4, 9, 0.64)
	}
	return ui.Modal(c, open, fn).
		Padding(0).Gap(0).Background(ui.Transparent).Radius(12).
		Shadow(0, 16, 44, 0, shadow)
}
