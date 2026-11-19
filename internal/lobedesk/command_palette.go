package lobedesk

import (
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func commandPaletteShortcutBindings() []appShortcut {
	keys, shiftKeys := "Ctrl+P", "Ctrl+Shift+P"
	if runtime.GOOS == "darwin" {
		keys, shiftKeys = "⌘P", "⌘⇧P"
	}
	return []appShortcut{
		{ID: "palette.open", Mod: ui.Cmd, Key: ui.KeyP, Keys: keys, Group: shortcutGroupGlobal, Label: "commandPalette", Available: func(a *app) bool { return a.canOpenCommandPalette() }, Run: func(a *app) { a.openCommandPalette() }},
		{ID: "palette.openShift", Mod: ui.Cmd | ui.Shift, Key: ui.KeyP, Keys: shiftKeys, Group: shortcutGroupGlobal, Label: "commandPalette", Available: func(a *app) bool { return a.canOpenCommandPalette() }, Run: func(a *app) { a.openCommandPalette() }},
	}
}

// commandPaletteTable adds actions without a key to the existing shortcut
// metadata. Only the palette consumes these additions; window registration and
// the Settings shortcut card still read the actual shortcut table.
func commandPaletteTable() []appShortcut {
	return append(shortcutTable(), fetchPaletteCommand(), stashPaletteCommand(), createBranchPaletteCommand(), historyRefPaletteCommand(), createTagPaletteCommand(), clonePaletteCommand(), copySHAPaletteCommand())
}

type commandPaletteState struct {
	open         bool
	root         string
	epoch        uint64
	generation   uint64
	query        string
	cursor       int
	cursorQuery  string
	cursorPlaced bool
	reveal       bool
	searchFocus  bool
}

// Only actions with a meaningful palette context belong here. The command
// table supplies labels, execution functions and keys where one exists.
func paletteCommand(id string) bool {
	switch id {
	case "repo.open", "repo.reopenClosedTab", "repo.clone", "window.settings", "page.history", "page.branches", "page.stash", "page.worktrees", "repo.reload", "history.head", "workspace.focusCommitMessage", "branch.quickCheckout", "repo.quickSwitch", "repo.fetch", "stash.saveTracked", "branch.create", "history.chooseRef", "tag.create", "commit.copySHA":
		return true
	}
	return false
}

func (a *app) commandPaletteEntries() []appShortcut {
	var entries []appShortcut
	words := strings.Fields(strings.ToLower(a.commandPalette.query))
	for _, s := range commandPaletteTable() {
		if !paletteCommand(s.ID) {
			continue
		}
		text := strings.ToLower(a.t(s.Label) + " " + s.ID)
		matches := true
		for _, word := range words {
			if !strings.Contains(text, word) {
				matches = false
				break
			}
		}
		if matches {
			entries = append(entries, s)
		}
	}
	return entries
}

func (a *app) paletteRoot() string {
	if a.repo != nil {
		return a.repo.Root
	}
	return ""
}

func (a *app) discardStaleCommandPalette() {
	if a.commandPalette.open && (a.commandPalette.root != a.paletteRoot() || a.commandPalette.epoch != a.loadSeq) {
		a.commandPalette = commandPaletteState{}
	}
}

// This predicate deliberately ignores the palette's own modal ownership.
// Existing action guards run normally once the palette has closed.
func (a *app) commandPaletteUnavailable(id string) string {
	if !paletteCommand(id) {
		return a.t("commandPaletteUnavailable")
	}
	if a.busy || a.loading || a.historyReloading {
		return a.t("commandPaletteWait")
	}
	if a.repositoryChooserOpen || a.historyRefOpen || a.hasPendingDialogWithoutPalette(toastMain) {
		return a.t("commandPaletteCloseDialog")
	}
	if id == "repo.reopenClosedTab" {
		if a.repositoryTabs == nil || len(a.repositoryTabs.closed) == 0 {
			return a.t("noClosedRepositoryTab")
		}
		if a.repositoryOpenPending() || a.repositoryTabs.active != a || a.localWatch.closed {
			return a.t("commandPaletteWait")
		}
		if a.hasPendingDialog(toastSettings) {
			return a.t("commandPaletteCloseDialog")
		}
		return ""
	}
	if a.repo == nil {
		switch id {
		case "repo.open", "repo.clone", "window.settings":
		case "repo.quickSwitch":
			if len(a.quickRepositorySwitchCandidates()) == 0 {
				return a.t("commandPaletteNeedRepo")
			}
		default:
			return a.t("commandPaletteNeedRepo")
		}
	}
	if id == "tag.create" {
		return a.createTagUnavailable()
	}
	if id == "history.chooseRef" && a.fileFilter != "" {
		return a.t("historyRefExitFile")
	}
	if id == "branch.create" {
		return a.createBranchUnavailable()
	}
	if id == "stash.saveTracked" {
		return a.trackedStashUnavailable()
	}
	if id == "repo.fetch" && len(a.remotes) == 0 {
		return a.t("noRemote")
	}
	if id == "history.head" && a.status.Head == "" {
		return a.t("commandPaletteNeedHead")
	}
	if id == "commit.copySHA" && !a.canCopySelectedCommitSHA() {
		return a.t("commandPaletteNeedCommit")
	}
	return ""
}

func (a *app) canOpenCommandPalette() bool {
	a.discardStaleCommandPalette()
	return !a.commandPalette.open && a.commandPaletteUnavailable("repo.open") == ""
}

func (a *app) openCommandPalette() {
	if !a.canOpenCommandPalette() {
		return
	}
	a.commandPaletteGeneration++
	a.commandPalette = commandPaletteState{open: true, root: a.paletteRoot(), epoch: a.loadSeq,
		generation: a.commandPaletteGeneration, cursor: -1, reveal: true, searchFocus: true}
	a.invalidate()
}

func (a *app) runPaletteCommand(id string, generation uint64) {
	a.discardStaleCommandPalette()
	if !a.commandPalette.open || a.commandPalette.generation != generation || a.commandPaletteUnavailable(id) != "" {
		return
	}
	for _, s := range commandPaletteTable() {
		if s.ID != id || !paletteCommand(id) || s.Run == nil {
			continue
		}
		a.commandPalette = commandPaletteState{}
		// Closing first lets the existing action see the real remaining dialogs.
		if a.commandPaletteUnavailable(id) == "" && (s.Available == nil || s.Available(a)) {
			s.Run(a)
		}
		a.invalidate()
		return
	}
}

func (a *app) commandPaletteDialog(c *ui.Context) {
	a.discardStaleCommandPalette()
	if !a.commandPalette.open {
		return
	}
	tk := theme.TokensOf(c)
	matches := a.commandPaletteEntries()
	selectable := func(index int) bool { return a.commandPaletteUnavailable(matches[index].ID) == "" }
	first := func() int { return pickerCursorAt(len(matches), -1, selectable) }
	if placePicker(&a.commandPalette.cursor, &a.commandPalette.cursorQuery, &a.commandPalette.cursorPlaced, a.commandPalette.query, first) ||
		repairPicker(&a.commandPalette.cursor, len(matches), first) {
		a.commandPalette.reveal = true
	}
	if a.commandPalette.cursor >= 0 && !selectable(a.commandPalette.cursor) {
		a.commandPalette.cursor = first()
	}
	width, height := c.Size()
	generation := a.commandPalette.generation
	chosen := ""
	var search ui.Element
	c.Key(commandPickerOverlayKey)
	ui.DialogBase(c, &a.commandPalette.open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Column().Label(a.t("commandPalette")).Width(min(520, max(0, width-32))).MaxHeight(max(0, height-32)).Padding(16).Gap(10).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).Children(func() {
			display.Text(c, a.t("commandPalette"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
			search = a.pickerSearchBar(c, &a.commandPalette.query, a.t("commandPaletteSearch"), func() {
				// Never substitute another action if the highlighted row became invalid.
				index := a.commandPalette.cursor
				if index >= 0 && index < len(matches) && selectable(index) {
					chosen = matches[index].ID
				}
			}, func(delta int) {
				a.commandPalette.cursor = stepPickerCursor(len(matches), a.commandPalette.cursor, delta, selectable)
				a.commandPalette.reveal = true
			}).Label(a.t("commandPaletteSearch"))
			var active ui.Element
			hasActive := false
			layout.ScrollArea(c, layout.ScrollAreaProps{}, func() {
				ui.Column(c).FillWidth().Gap(1).Role(ui.RoleList).Children(func() {
					for index, s := range matches {
						reason := a.commandPaletteUnavailable(s.ID)
						highlighted := index == a.commandPalette.cursor && reason == ""
						row := pickerRowBase(c, a.t(s.Label), index+1, len(matches), highlighted)
						switch {
						case reason != "":
							row.AccessDisabled(true).Description(reason).Cursor(ui.CursorDefault)
						case highlighted:
							row.Background(selectionBg(tk, tk.BgElevated))
						case row.Hovered():
							row.Background(tk.FillTertiary.Over(tk.BgElevated))
						}
						row.Children(func() {
							ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
								display.Text(c, a.t(s.Label), display.TextProps{FontSize: scaledFont(12)}).SingleLine()
								if reason != "" {
									display.Text(c, reason, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).SingleLine()
								}
							})
							display.Text(c, s.Keys, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).Shrink(0)
						})
						pickerReveal(row, highlighted, &a.commandPalette.reveal)
						if highlighted {
							active, hasActive = row, true
						}
						if reason == "" && row.Clicked() {
							chosen = s.ID
						}
					}
					if len(matches) == 0 {
						display.Text(c, a.t("commandPaletteNoResults"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).Padding(6, 8)
					}
				})
			}).MaxHeight(min(float32(pickerListMaxHeight), max(0, height-150)))
			if hasActive {
				search.ActiveDescendant(active)
			}
		})
	})
	// Overlay captures its opener after building its children. Focus only after
	// that capture, or cancellation would try to restore the search itself.
	if a.commandPalette.open && a.commandPalette.searchFocus {
		search.Focus()
		a.commandPalette.searchFocus = false
	}
	if chosen != "" {
		a.runPaletteCommand(chosen, generation)
		return
	}
	if !a.commandPalette.open {
		a.commandPalette = commandPaletteState{}
	}
}

// A callback retained from a pre-modal frame must not reach background actions.
func (a *app) dispatchWindowShortcut(run func(*app)) {
	if !a.commandPalette.open && !a.historyRefPalette.active && !a.quickCheckout.open && !a.childNavigation.open && !a.quickRepositorySwitch.open {
		run(a)
	}
}
