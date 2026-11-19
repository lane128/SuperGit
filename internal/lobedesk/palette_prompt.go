package lobedesk

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Palette prompt handoffs retain the command session's original opener. Other
// generic prompts keep their existing renderer and focus behavior.
func (a *app) palettePromptDialog(c *ui.Context) {
	tk := theme.TokensOf(c)
	width, height := c.Size()
	confirmed := false
	var input ui.Element
	c.Key(commandPickerOverlayKey)
	ui.DialogBase(c, &a.prompt.open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Column().Label(a.prompt.title).Width(min(400, max(0, width-32))).MaxHeight(max(0, height-32)).Padding(18).Gap(10).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).Children(func() {
			display.Text(c, a.prompt.title, display.TextProps{Strong: true, FontSize: scaledFont(14)})
			if a.prompt.description != "" {
				display.Text(c, a.prompt.description, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2})
			}
			input = field.Input(c, &a.prompt.value, field.InputProps{
				Variant: field.InputFilled, Placeholder: a.prompt.placeholder, Size: theme.Middle,
			}).Label(a.prompt.placeholder)
			ready := strings.TrimSpace(a.prompt.value) != ""
			if ready && input.Submitted() {
				confirmed = true
				a.prompt.open = false
			}
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.End).Children(func() {
				if workbenchButton(c, a.t("cancel"), buttonNormal, false).Clicked() {
					a.prompt.open = false
				}
				if workbenchButton(c, a.prompt.actionName, buttonPrimary, !ready).Clicked() && ready {
					confirmed = true
					a.prompt.open = false
				}
			})
		})
	})
	// DialogBase captures the opener after building children; focus once after
	// it returns so that Tab can leave the input on subsequent frames.
	if a.prompt.open && a.prompt.paletteFocus {
		input.Focus()
		a.prompt.paletteFocus = false
	}
	if confirmed {
		action, value := a.prompt.action, strings.TrimSpace(a.prompt.value)
		a.prompt = promptState{}
		if action != nil {
			action(value)
		}
	} else if !a.prompt.open {
		if a.prompt.paletteCancel != nil {
			a.prompt.paletteCancel()
		}
		a.prompt = promptState{}
	}
}
