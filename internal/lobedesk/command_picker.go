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

// Keep the query at a fixed position as the result list grows or shrinks.
// DialogBase continues to own Escape, backdrop dismissal and opener restore.
func commandPickerDialog(c *ui.Context, open *bool, content func(ui.Element, ui.Element)) {
	_, height := c.Size()
	top := min(float32(76), max(float32(16), height/8))
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		content(backdrop, panel)
		backdrop.Column().AlignItems(ui.Center).Justify(ui.Start).Padding(top, 0, 16, 0)
		panel.MaxHeight(max(0, height-top-16))
	})
}

// Text inputs consume Backspace before ordinary shortcuts. OnTextKey reads
// the live editor text, including earlier events in the same frame, so deleting
// the last character remains normal editing. No Element escapes this build.
func (a *app) commandPickerBackspace(input ui.Element, query string, cancel func()) {
	input.OnTextKey(func(k ui.TextKey) bool {
		if k.Key != ui.KeyBackspace || k.Mods != 0 || k.Text() != "" {
			return false
		}
		cancel()
		a.openCommandPalette()
		return true
	})
}
