package lobedesk

import (
	"runtime"
	"slices"
	"sort"
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
	entries := append(shortcutTable(), fetchPaletteCommands()...)
	entries = append(entries, mergePaletteCommand(), stashPaletteCommand(), createBranchPaletteCommand(), historyRefPaletteCommand(), createTagPaletteCommand(), clonePaletteCommand(), copySHAPaletteCommand())
	for i := range entries {
		if entries[i].ID == "window.settings" {
			entries[i].Label = "palettePreferences"
		}
	}
	return append(entries, paletteActionShortcuts...)
}

type commandPaletteState struct {
	scroll       ui.ScrollState
	viewport     ui.Rect
	step         *paletteStep
	fetchParent  *fetchPaletteParent
	actionID     string
	stage        paletteStage
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
	case "repo.open", "repo.reopenClosedTab", "repo.clone", "window.settings", "page.history", "page.branches", "page.stash", "page.worktrees", "repo.reload", "history.head", "workspace.focusCommitMessage", "branch.quickCheckout", "branch.merge", "repo.quickSwitch", "stash.saveTracked", "branch.create", "history.chooseRef", "tag.create", "commit.copySHA":
		return true
	}
	if _, ok := fetchPaletteMetadata[id]; ok {
		return true
	}
	_, ok := paletteActionsByID[id]
	return ok
}

// fetchSafePaletteCommand lists commands that neither write Git nor depend on
// a repository reload, so they stay usable while a fetch runs in the
// background. A few of them persist UI settings; that is still safe.
func fetchSafePaletteCommand(id string) bool {
	for _, prefix := range []string{"page.", "history.show", "history.hide", "history.navigate", "view."} {
		if strings.HasPrefix(id, prefix) {
			return true
		}
	}
	switch id {
	case "history.head", "history.chooseRef", "commit.copySHA", "commit.viewTree", "file.history", "file.blame",
		"diff.ignoreWhitespace", "window.settings", "window.about", "workspace.focusCommitMessage",
		"repo.containingFolder", "repo.terminal", "repo.sublimeText":
		return true
	}
	return false
}

func (a *app) commandPaletteEntries() []appShortcut {
	var entries []appShortcut
	scores := map[string]int{}
	query := paletteCaptionStem(a.commandPalette.query)
	fetchQuery := strings.Join(strings.Fields(query), " ")
	plainFetch := isPlainFetchQuery(fetchQuery)
	for _, s := range commandPaletteTable() {
		if !paletteCommand(s.ID) {
			continue
		}
		if plainFetch {
			// An exact Fetch search shows the entry to the mode picker.
			if s.ID == "repo.fetch" {
				entries = append(entries, s)
			}
			continue
		}
		if query == "" && isFetchVariant(s.ID) {
			continue
		}
		en, zh := messages[localeEnglish][s.Label], messages[localeChinese][s.Label]
		aliases := fetchPaletteMetadata[s.ID].aliases
		names := append([]string{a.t(s.Label), en, zh, s.ID}, referenceCommandSearchNames(s.ID)...)
		names = append(names, aliases...)
		score := commandMatchScore(a.commandPalette.query, names...)
		if query != "" && (query == paletteCaptionStem(s.ID) || query == paletteCaptionStem(en) || query == paletteCaptionStem(zh) || slices.Contains(aliases, fetchQuery)) {
			// An exact command must beat a longer command with the same prefix.
			score = 1 << 30
		}
		if score >= 0 {
			scores[s.ID] = score
			entries = append(entries, s)
		}
	}
	if strings.TrimSpace(a.commandPalette.query) == "" {
		sort.SliceStable(entries, func(i, j int) bool { return a.t(entries[i].Label) < a.t(entries[j].Label) })
	} else {
		sort.SliceStable(entries, func(i, j int) bool { return scores[entries[i].ID] > scores[entries[j].ID] })
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
	if !a.commandPalette.open {
		return
	}
	if a.commandPalette.root != a.paletteRoot() {
		a.commandPalette = commandPaletteState{}
		return
	}
	// The command list recomputes availability every frame, so a background
	// reload in the same repository must not discard a query being typed.
	// Bump the generation instead: callbacks captured before the reload are
	// stale, while Enter on the open palette uses the current generation.
	if a.commandPalette.epoch != a.loadSeq {
		if a.commandPalette.stage != paletteCommands {
			a.commandPalette = commandPaletteState{}
			return
		}
		a.commandPaletteGeneration++
		a.commandPalette.epoch = a.loadSeq
		a.commandPalette.generation = a.commandPaletteGeneration
	}
}

// This predicate deliberately ignores the palette's own modal ownership.
// Existing action guards run normally once the palette has closed.
func (a *app) commandPaletteUnavailable(id string) string {
	if !paletteCommand(id) {
		return a.t("commandPaletteUnavailable")
	}
	if a.exclusiveBusy() || a.loading || a.historyReloading {
		return a.t("commandPaletteWait")
	}
	if a.fetching && !fetchSafePaletteCommand(id) {
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
	if action, ok := paletteActionsByID[id]; ok {
		if !action.Global {
			if a.repo == nil {
				return a.t("commandPaletteNeedRepo")
			}
			if a.repositoryOpenPending() || a.tabSnapshotUnavailable() {
				return a.t("commandPaletteWait")
			}
		}
		if action.Unavailable != nil {
			return action.Unavailable(a)
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
	_, fetchCommand := fetchPaletteMetadata[id]
	if (id == "branch.merge" || fetchCommand) && (a.repositoryOpenPending() || a.tabSnapshotUnavailable()) {
		return a.t("commandPaletteWait")
	}
	if id == "branch.merge" {
		return a.paletteMergeUnavailable()
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
	if fetchCommand && len(a.remotes) == 0 {
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
	return !a.commandPalette.open && !a.exclusiveBusy() && !a.loading && !a.historyReloading &&
		!a.repositoryChooserOpen && !a.historyRefOpen && !a.hasPendingDialogWithoutPalette(toastMain)
}

func (a *app) openCommandPalette() {
	if !a.canOpenCommandPalette() {
		return
	}
	a.setPaletteStage(paletteCommands)
}

func (a *app) runPaletteCommand(id string, generation uint64) {
	a.discardStaleCommandPalette()
	if !a.commandPalette.open || a.commandPalette.stage != paletteCommands || a.commandPalette.generation != generation || a.commandPaletteUnavailable(id) != "" {
		return
	}
	a.dispatchPaletteCommand(id)
}

// The caller must validate the generation and, for a parameter page, the row.
func (a *app) dispatchPaletteCommand(id string) {
	previous := a.commandPalette
	for _, s := range commandPaletteTable() {
		if s.ID != id || !paletteCommand(id) || s.Run == nil {
			continue
		}
		a.commandPalette = commandPaletteState{}
		// Closing first lets the existing action see the real remaining dialogs.
		if a.commandPaletteUnavailable(id) == "" && (s.Available == nil || s.Available(a)) {
			a.commandPalette.actionID = id
			if previous.stage == paletteFetchOptions && id == "repo.fetch" {
				a.fetchAll()
			} else {
				s.Run(a)
			}
			if !a.commandPalette.open {
				a.commandPalette = commandPaletteState{}
			} else if a.commandPalette.stage == paletteFetchOptions || previous.stage == paletteFetchOptions && a.commandPalette.stage == paletteFetchRemote {
				a.commandPalette.fetchParent = &fetchPaletteParent{
					stage: previous.stage, query: previous.query, cursor: previous.cursor, parent: previous.fetchParent,
				}
			}
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
	s := &a.commandPalette
	rows := a.paletteChoices()
	selectable := func(i int) bool { return rows[i].reason == "" }
	first := func() int { return pickerCursorAt(len(rows), -1, selectable) }
	if placePicker(&s.cursor, &s.cursorQuery, &s.cursorPlaced, s.query, first) ||
		repairPicker(&s.cursor, len(rows), first) || s.cursor >= 0 && !selectable(s.cursor) {
		s.cursor = pickerCursorAt(len(rows), s.cursor, selectable)
		s.reveal = true
	}
	title, chip, placeholder := a.paletteInputLabels()
	generation := s.generation
	var search ui.Element
	var chosen *paletteChoice
	var submitted func()
	textStep := s.step != nil && s.step.submit != nil
	c.Key(commandPickerOverlayKey)
	commandPickerDialog(c, &s.open, func(backdrop, panel ui.Element) {
		commandPanel(c, backdrop, panel, title).Children(func() {
			search = a.paletteInput(c.Key(generation), chip, placeholder, &s.query, func() {
				if textStep && (s.step.allowEmpty || strings.TrimSpace(s.query) != "") {
					step, value := s.step, s.query
					submitted = func() { step.submit(value) }
					return
				}
				if s.cursor >= 0 && s.cursor < len(rows) && selectable(s.cursor) {
					chosen = &rows[s.cursor]
				}
			}, func(delta int) {
				s.cursor = stepPickerCursor(len(rows), s.cursor, delta, selectable)
				s.reveal = true
			})
			if s.stage != paletteCommands {
				search.OnTextKey(func(k ui.TextKey) bool {
					if k.Mods != 0 || k.Key != ui.KeyBackspace || k.Text() != "" || generation != s.generation {
						return false
					}
					a.paletteStepBack()
					return true
				})
			}
			_, height := c.Size()
			tk := theme.TokensOf(c)
			scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &s.scroll}, func() {
				ui.Column(c).FillWidth().Gap(1).Role(ui.RoleList).Children(func() {
					draw := func(rc *ui.Context, i int) {
						item := rows[i]
						selected := i == s.cursor && selectable(i)
						row := pickerRowBase(rc.Key(item.id), item.label, i+1, len(rows), selected)
						if _, ok := fetchPaletteMetadata[item.id]; ok {
							row.Description(item.detail)
						} else if item.prefix != "" {
							row.Description(item.id)
						}
						if selected {
							row.Background(selectionBg(tk, tk.BgElevated))
							search.ActiveDescendant(row)
						} else if item.reason != "" {
							row.AccessDisabled(true).Description(item.reason)
						} else if row.Hovered() {
							row.Background(tk.FillTertiary.Over(tk.BgElevated))
						}
						row.Children(func() { drawPaletteChoice(rc, item, s.query) })
						pickerReveal(row, selected, &s.reveal)
						if row.Clicked() && selectable(i) {
							chosen = &rows[i]
						}
					}
					if len(rows) > 128 {
						heights := make([]float32, len(rows))
						for i, row := range rows {
							heights[i] = max(30, scaledFont(28))
							if row.reason != "" || row.detail != "" {
								heights[i] = max(heights[i], scaledFont(40))
							}
						}
						virtualPatchRows(c, s.viewport, &s.scroll, heights, s.cursor, func(rc *ui.Context, i int) {
							ui.Column(rc).FillWidth().Height(heights[i]).Shrink(0).Children(func() { draw(rc, i) })
						})
					} else {
						for i := range rows {
							draw(c, i)
						}
					}
					if len(rows) == 0 && !textStep {
						display.Text(c, a.t("commandPaletteNoResults"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).Padding(6, 8)
					}
				})
			}).MaxHeight(min(float32(570), max(0, height-170)))
			s.viewport = scroll.Bounds()
			if s.stage == paletteMergeRef && s.cursor >= 0 {
				preview := a.tf("paletteMergePreview", rows[s.cursor].id, a.status.Branch)
				display.Text(c.Key("preview"), preview, display.TextProps{FontSize: scaledFont(12), Rows: 2}).Padding(6, 8).Tooltip(preview)
			}
		})
	})
	if s.open && s.searchFocus {
		search.Focus()
		s.searchFocus = false
	}
	if submitted != nil {
		a.finishPaletteStep(generation, submitted)
	} else if chosen != nil {
		a.acceptPaletteChoice(*chosen, generation)
	} else if !s.open {
		a.commandPalette = commandPaletteState{}
	}
}

// A callback retained from a pre-modal frame must not reach background actions.
func (a *app) dispatchWindowShortcut(run func(*app)) {
	if a.comparison.view == nil && !a.commandPalette.open && !a.historyRefPalette.active && !a.quickCheckout.open && !a.childNavigation.open && !a.quickRepositorySwitch.open {
		run(a)
	}
}
