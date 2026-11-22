package lobedesk

import (
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type quickCheckoutState struct {
	open                        bool
	root                        string
	epoch, generation           uint64
	query                       string
	cursor                      int
	cursorQuery                 string
	cursorPlaced, reveal        bool
	step, remoteName, remoteSHA string
	localName, targetQuery      string
	nameFocus, searchFocus      bool
	savedCursor, targetCursor   int
}

func quickCheckoutShortcut() appShortcut {
	keys := "Ctrl+B"
	if runtime.GOOS == "darwin" {
		keys = "⌘B"
	}
	return appShortcut{ID: "branch.quickCheckout", Mod: ui.Cmd, Key: ui.KeyB, Keys: keys, Group: shortcutGroupGlobal, Label: "quickCheckoutCommand", Available: func(a *app) bool { return a.canOpenQuickCheckout() }, Run: func(a *app) { a.openQuickCheckout() }}
}

func (a *app) discardStaleQuickCheckout() {
	if a.quickCheckout.open && (a.repo == nil || a.repo.Root != a.quickCheckout.root || a.loadSeq != a.quickCheckout.epoch) {
		a.quickCheckout = quickCheckoutState{}
	}
}

func (a *app) quickCheckoutReady() bool {
	return a.repo != nil && !a.busy && !a.loading && !a.historyReloading && !a.repositoryChooserOpen && !a.historyRefOpen && !a.commandPalette.open && !a.hasPendingActionDialog(toastMain)
}

func (a *app) canOpenQuickCheckout() bool {
	a.discardStaleQuickCheckout()
	return !a.quickCheckout.open && a.quickCheckoutReady()
}

func (a *app) openQuickCheckout() {
	if !a.canOpenQuickCheckout() {
		return
	}
	a.quickCheckoutGeneration++
	a.quickCheckout = quickCheckoutState{open: true, root: a.repo.Root, epoch: a.loadSeq, generation: a.quickCheckoutGeneration, cursor: -1, reveal: true, searchFocus: true}
	a.invalidate()
}

func (a *app) quickCheckoutUnavailable(branch gitrepo.Branch) string {
	switch {
	case !a.quickCheckoutReady():
		return a.t("commandPaletteWait")
	case a.loadErr != "" || a.branchesErr != "" || a.worktreesErr != "":
		return a.t("quickCheckoutReadError")
	case branch.Current || branch.Name == a.status.Branch:
		return a.t("current")
	}
	for _, wt := range a.worktrees {
		if wt.Branch == branch.Name && filepath.Clean(wt.Path) != filepath.Clean(a.repo.Root) {
			return a.t("quickCheckoutOccupied")
		}
	}
	for _, entry := range a.status.Entries {
		if entry.Conflict {
			return a.t("quickCheckoutConflict")
		}
	}
	if a.operation.Kind != "" {
		return a.t("quickCheckoutOperation")
	}
	if branch.Name == "" || branch.SHA == "" {
		return a.t("commandPaletteUnavailable")
	}
	return ""
}

func (a *app) runQuickCheckout(name, sha string, generation uint64) {
	a.discardStaleQuickCheckout()
	if !a.quickCheckout.open || a.quickCheckout.generation != generation {
		return
	}
	for _, branch := range a.branches {
		if branch.Name != name || branch.SHA != sha || a.quickCheckoutUnavailable(branch) != "" {
			continue
		}
		a.quickCheckout = quickCheckoutState{}
		if a.quickCheckoutReady() {
			a.switchBranchAt(name, sha)
		}
		a.invalidate()
		return
	}
}

func (a *app) quickCheckoutDialog(c *ui.Context) {
	a.discardStaleQuickCheckout()
	if !a.quickCheckout.open {
		return
	}
	query := &a.quickCheckout.query
	if a.quickCheckout.step == "targets" {
		query = &a.quickCheckout.targetQuery
	}
	matches := a.quickCheckoutCandidates()
	selectable := func(index int) bool { return a.quickCheckoutCandidateReason(matches[index]) == "" }
	first := func() int { return pickerCursorAt(len(matches), -1, selectable) }
	if placePicker(&a.quickCheckout.cursor, &a.quickCheckout.cursorQuery, &a.quickCheckout.cursorPlaced, a.quickCheckout.step+*query, first) || repairPicker(&a.quickCheckout.cursor, len(matches), first) {
		a.quickCheckout.reveal = true
	}
	if a.quickCheckout.cursor >= 0 && !selectable(a.quickCheckout.cursor) {
		a.quickCheckout.cursor = -1
	}
	tk := theme.TokensOf(c)
	_, height := c.Size()
	generation := a.quickCheckout.generation
	var search ui.Element
	var chosen *quickCheckoutCandidate
	c.Key(commandPickerOverlayKey)
	commandPickerDialog(c, &a.quickCheckout.open, func(backdrop, panel ui.Element) {

		if a.quickCheckout.step != "" {
			panel.OnShortcut(0, ui.KeyEscape, func() { a.quickCheckoutBack() })
		}
		commandPanel(c, backdrop, panel, a.t("quickCheckout")).Children(func() {
			if a.quickCheckout.step == "name" {
				a.quickCheckoutNameForm(c, generation)
				return
			}
			if a.quickCheckout.step == "targets" {
				display.Text(c, a.quickCheckout.remoteName, display.TextProps{FontSize: scaledFont(12), Rows: 2})
			}
			search = a.paletteInput(c, a.t("quickCheckout"), a.t("filterBranches"), query, func() {
				index := a.quickCheckout.cursor
				if index >= 0 && index < len(matches) && selectable(index) {
					branch := matches[index]
					chosen = &branch
				}
			}, func(delta int) {
				a.quickCheckout.cursor = stepPickerCursor(len(matches), a.quickCheckout.cursor, delta, selectable)
				a.quickCheckout.reveal = true
			}).Label(a.t("filterBranches"))
			if a.quickCheckout.step == "" {
				a.commandPickerBackspace(search, *query, func() { a.quickCheckout = quickCheckoutState{} })
			}
			if a.quickCheckout.step == "targets" {
				search.OnShortcut(0, ui.KeyEscape, func() { a.quickCheckoutBack() })
			}
			var active ui.Element
			hasActive := false
			layout.ScrollArea(c, layout.ScrollAreaProps{}, func() {
				ui.Column(c).FillWidth().Gap(1).Role(ui.RoleList).Children(func() {
					for index, branch := range matches {
						reason := a.quickCheckoutCandidateReason(branch)
						highlighted := index == a.quickCheckout.cursor && reason == ""
						kindLabel := ""
						if branch.kind == "remote" {
							kindLabel = a.t("remoteBranches")
						} else if branch.kind != "create" {
							kindLabel = a.t("localBranches")
						}
						label := branch.Name
						if kindLabel != "" {
							label += " · " + kindLabel
						}
						row := pickerRowBase(c, label, index+1, len(matches), highlighted)
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
								nameText := display.Text(c, branch.Name, display.TextProps{FontSize: scaledFont(12)}).SingleLine()
								if reason != "" {
									nameText.TextColor(tk.TextDisabled)
								}
								if kindLabel != "" {
									kindText := display.Text(c, kindLabel, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).SingleLine()
									if reason != "" {
										kindText.TextColor(tk.TextDisabled)
									}
								}
								if reason != "" {
									display.Text(c, reason, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2})
								}
							})
						})
						pickerReveal(row, highlighted, &a.quickCheckout.reveal)
						if highlighted {
							active, hasActive = row, true
						}
						if reason == "" && row.Clicked() {
							b := branch
							chosen = &b
						}
					}
					if len(matches) == 0 {
						display.Text(c, a.t("searchNoBranches"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).Padding(6, 8)
					}
				})
			}).MaxHeight(min(float32(pickerListMaxHeight), max(0, height-150)))
			if len(matches) > 0 && first() < 0 {
				display.Text(c, a.t("quickCheckoutNoAvailable"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12), Rows: 2}).Padding(6, 8)
			}
			if hasActive {
				search.ActiveDescendant(active)
			}
			if a.quickCheckout.step == "targets" && workbenchButton(c, a.t("quickCheckoutBack"), buttonNormal, false).Clicked() {
				a.quickCheckoutBack()
			}
		})
	})
	if a.quickCheckout.open && a.quickCheckout.step != "name" && a.quickCheckout.searchFocus {
		search.Focus()
		a.quickCheckout.searchFocus = false
	}
	if chosen != nil {
		a.chooseQuickCheckout(*chosen, generation)
		return
	}
	if !a.quickCheckout.open {
		a.quickCheckout = quickCheckoutState{}
	}
}
