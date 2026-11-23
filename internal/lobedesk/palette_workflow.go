package lobedesk

import (
	"slices"
	"strings"
)

// Palette-only actions reuse the global dispatch guards without registering a
// window shortcut. The label key also supplies bilingual search terms.
type paletteAction struct {
	ID, English, Chinese string
	Global               bool
	Unavailable          func(*app) string
	Run                  func(*app)
}

var paletteActionsByID = map[string]paletteAction{}
var paletteActionShortcuts []appShortcut

func registerPaletteActions(actions ...paletteAction) {
	for _, action := range actions {
		if _, exists := paletteActionsByID[action.ID]; exists || action.ID == "" || action.Run == nil {
			panic("invalid palette action: " + action.ID)
		}
		paletteActionsByID[action.ID] = action
		key := "palette/" + action.ID
		messages[localeEnglish][key] = action.English
		messages[localeChinese][key] = action.Chinese
		paletteActionShortcuts = append(paletteActionShortcuts, appShortcut{ID: action.ID, Label: key, Run: action.Run})
	}
}

func (a *app) paletteText(en, zh string) string {
	if a.locale == localeChinese {
		return zh
	}
	return en
}

// Steps carry the exact choices (including refs and object IDs) shown to the
// user. Keeping the parent supports Backspace without replaying an action.
type paletteStep struct {
	title, placeholder, initial string
	options                     []paletteChoice
	filter                      string
	matches                     []paletteChoice
	choose                      func(paletteChoice)
	submit                      func(string)
	allowEmpty                  bool
	savedCursor                 int
	resume                      bool
	parent                      *paletteStep
}

func (a *app) choosePalette(caption, placeholder string, options []paletteChoice, choose func(paletteChoice)) {
	a.showPaletteStep(&paletteStep{title: caption, placeholder: placeholder, options: slices.Clone(options), choose: choose, parent: a.commandPalette.step})
}

func (a *app) inputPalette(caption, placeholder, initial string, accept func(string)) {
	a.showPaletteStep(&paletteStep{title: caption, placeholder: placeholder, initial: initial, submit: accept, parent: a.commandPalette.step})
}

func (a *app) showPaletteStep(step *paletteStep) {
	id := a.commandPalette.actionID
	a.setPaletteStage(paletteWorkflow)
	a.commandPalette.step, a.commandPalette.actionID = step, id
	a.commandPalette.query = step.initial
	if step.resume {
		a.commandPalette.cursor, a.commandPalette.cursorQuery, a.commandPalette.cursorPlaced = step.savedCursor, step.initial, true
	}
}

func (a *app) paletteStepBack() {
	s := a.commandPalette
	if s.step != nil && s.step.parent != nil {
		a.showPaletteStep(s.step.parent)
	} else {
		a.setPaletteStage(paletteCommands)
	}
}

func (a *app) finishPaletteStep(generation uint64, run func()) {
	a.discardStaleCommandPalette()
	s := a.commandPalette
	if !s.open || s.generation != generation || s.stage != paletteWorkflow || a.exclusiveBusy() || a.loading || a.repositoryOpenPending() {
		return
	}
	if s.actionID != "" && a.commandPaletteUnavailable(s.actionID) != "" {
		return
	}
	// Release modal ownership while handing off, but keep the step until
	// the callback has had the opportunity to open the next parameter.
	s.step.initial, s.step.savedCursor, s.step.resume = s.query, s.cursor, true
	a.commandPalette.open = false
	run()
	if !a.commandPalette.open {
		a.commandPalette = commandPaletteState{}
	}
}

func (a *app) stepChoices() []paletteChoice {
	s := a.commandPalette
	if s.step.submit != nil {
		return nil
	}
	if strings.TrimSpace(s.query) == "" {
		return s.step.options
	}
	if s.step.filter == s.query {
		return s.step.matches
	}
	var matches []paletteChoice
	for _, row := range s.step.options {
		if commandMatchScore(s.query, row.label, row.id, row.detail) >= 0 {
			matches = append(matches, row)
		}
	}
	s.step.filter, s.step.matches = s.query, matches
	return matches
}
