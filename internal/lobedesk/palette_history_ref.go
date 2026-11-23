package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type historyRefPaletteState struct {
	active, searchFocus bool
	root                string
}

func historyRefPaletteCommand() appShortcut {
	return appShortcut{ID: "history.chooseRef", Label: "historyRefCommand",
		Available: func(a *app) bool { return a.canChooseHistoryRefCommand() },
		Run:       func(a *app) { a.chooseHistoryRefFromCommand() },
	}
}

func (a *app) canChooseHistoryRefCommand() bool {
	return a.commandPaletteUnavailable("history.chooseRef") == "" && !a.hasPendingDialog(toastMain)
}

func (a *app) chooseHistoryRefFromCommand() {
	if !a.canChooseHistoryRefCommand() {
		return
	}
	a.initializeHistoryRefPicker()
	a.historyRefPalette = historyRefPaletteState{active: true, searchFocus: true, root: a.paletteRoot()}
	a.historyRefOpen = true
	a.invalidate()
}

func (a *app) discardStaleHistoryRefPalette() {
	s := a.historyRefPalette
	// A background reload in the same repository refreshes the rows; it must
	// not close the picker and discard the query being typed.
	if s.active && (!a.historyRefOpen || s.root != a.paletteRoot() || a.fileFilter != "") {
		a.historyRefPalette = historyRefPaletteState{}
		a.historyRefOpen = false
		a.historyRefQuery = ""
	}
}

func (a *app) historyRefPaletteDialog(c *ui.Context) {
	a.discardStaleHistoryRefPalette()
	if !a.historyRefPalette.active {
		return
	}
	tk := theme.TokensOf(c)
	width, height := c.Size()
	var search ui.Element
	var chosen *historyRef
	c.Key(commandPickerOverlayKey)
	commandPickerDialog(c, &a.historyRefOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, .4))
		panel.Column().Label(a.t("historyRefCommand")).Width(min(400, max(0, width-32))).MaxHeight(max(0, height-32)).Padding(16).Gap(10).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).Children(func() {
			display.Text(c, a.t("historyRefCommand"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
			display.Text(c, a.t("historyRefScope"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
			search = a.historyRefPickerContent(c, min(368, max(0, width-64)), min(float32(pickerListMaxHeight), max(0, height-210)), false, func(ref historyRef) {
				if a.exclusiveBusy() || a.loading || a.historyReloading || a.repositoryChooserOpen || a.hasPendingActionDialog(toastMain) {
					return
				}
				if ref.Kind != "" && a.historyRefTip(ref) == "" {
					return
				}
				chosen = &ref
				a.historyRefOpen = false
			})
			a.commandPickerBackspace(search, a.historyRefQuery, func() {
				a.historyRefPalette = historyRefPaletteState{}
				a.historyRefOpen, a.historyRefQuery = false, ""
			})
		})
	})
	// Capture the original opener before focusing a field from the next phase.
	if a.historyRefOpen && (a.historyRefPalette.searchFocus || a.historyRefErr() == "") {
		search.Focus()
		a.historyRefPalette.searchFocus = false
	}
	if !a.historyRefOpen {
		a.historyRefPalette = historyRefPaletteState{}
		a.historyRefQuery = ""
		if chosen != nil {
			a.applyHistoryRef(*chosen)
		}
	}
}
