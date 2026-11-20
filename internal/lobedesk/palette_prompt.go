package lobedesk

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
)

// Palette prompt handoffs retain the command session's original opener. Other
// generic prompts keep their existing renderer and focus behavior.
func (a *app) palettePromptDialog(c *ui.Context) {
	confirmed := false
	var input ui.Element
	c.Key(commandPickerOverlayKey)
	commandPickerDialog(c, &a.prompt.open, func(backdrop, panel ui.Element) {
		commandPanel(c, backdrop, panel, a.prompt.title).Children(func() {
			input = a.paletteInput(c, a.prompt.title, a.prompt.placeholder, &a.prompt.value, nil, nil)
			if a.prompt.description != "" {
				display.Text(c, a.prompt.description, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2})
			}
			a.commandPickerBackspace(input, a.prompt.value, func() {
				if a.prompt.paletteCancel != nil {
					a.prompt.paletteCancel()
				}
				a.prompt = promptState{}
			})
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
