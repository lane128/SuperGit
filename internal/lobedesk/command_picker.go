package lobedesk

import "github.com/egoist/mygo/ui"

// One view build constructs one picker. A command can change the session's
// content without constructing a second overlay or capturing a new opener.
func (a *app) commandPickerDialogs(c *ui.Context) {
	a.discardStaleCommandPalette()
	a.discardStaleQuickCheckout()
	a.discardStaleQuickRepositorySwitch()
	a.discardStalePrompt()
	a.discardStaleHistoryRefPalette()
	if a.commandPalette.open {
		a.commandPaletteDialog(c)
	} else if a.cloneForm.open && a.cloneForm.paletteHandoff {
		a.paletteCloneDialog(c)
	} else if a.historyRefPalette.active {
		a.historyRefPaletteDialog(c)
	} else if a.prompt.open && a.prompt.paletteHandoff {
		a.palettePromptDialog(c)
	} else if a.quickRepositorySwitch.open {
		a.quickRepositorySwitchDialog(c)
	} else {
		a.quickCheckoutDialog(c)
	}
}

const commandPickerOverlayKey = "workbench-command-picker"
