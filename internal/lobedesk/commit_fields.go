package lobedesk

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
)

// commitText remains authoritative. These are editor projections, refreshed on
// external draft changes and written back only after an actual edit. Rendering
// an old message must not normalize its separators or trailing blank lines.
type commitFieldsState struct {
	Source, Summary, Body string
	Pastes                []commitPasteEdit
	PasteIndex            int
	PendingPaste          bool
}

type commitPasteEdit struct{ Before, After string }

func splitCommitMessage(message string) (summary, body string) {
	summary, body, _ = strings.Cut(message, "\n")
	body = strings.TrimPrefix(body, "\n")
	return summary, body
}

func (a *app) projectCommitMessage(message string) {
	s := &a.commitFields
	s.Source = message
	s.Summary, s.Body = splitCommitMessage(message)
}

func (a *app) syncCommitFields() {
	if a.commitFields.Source != a.commitText {
		a.projectCommitMessage(a.commitText)
		a.commitFields.Pastes = nil
		a.commitFields.PasteIndex = 0
	}
}

func (a *app) storeCommitFields() {
	message := a.commitFields.Summary
	if a.commitFields.Body != "" {
		message += "\n\n" + a.commitFields.Body
	}
	a.commitText = message
	a.commitFields.Source = message
}

func (a *app) pasteCommitSummary(text string, start, end int) bool {
	if !strings.ContainsAny(text, "\r\n") {
		return false
	}
	// Clipboard line endings are normalized as native multiline fields do.
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	s := &a.commitFields
	before := a.commitText
	runes := []rune(s.Summary)
	start, end = min(max(0, start), len(runes)), min(max(0, end), len(runes))
	if end < start {
		start, end = end, start
	}
	s.Summary, text = splitCommitMessage(string(runes[:start]) + text + string(runes[end:]))
	if s.Body != "" && text != "" {
		text += "\n\n"
	}
	s.Body = text + s.Body
	a.storeCommitFields()
	s.Pastes = append(s.Pastes[:s.PasteIndex], commitPasteEdit{Before: before, After: a.commitText})
	s.PasteIndex = len(s.Pastes)
	// The native input publishes its unchanged buffer after OnPaste returns.
	// Reapply this projection after that publication, when building the field.
	s.PendingPaste = true
	return true
}

func (a *app) commitSummaryKey(k ui.TextKey) bool {
	if k.Key == ui.KeyEnter && k.Mods == 0 {
		return true
	}
	if k.Key != ui.KeyZ || k.Mods&ui.Cmd == 0 {
		return false
	}
	s := &a.commitFields
	var message string
	switch {
	case k.Mods == ui.Cmd && s.PasteIndex > 0 && a.commitText == s.Pastes[s.PasteIndex-1].After:
		s.PasteIndex--
		message = s.Pastes[s.PasteIndex].Before
	case k.Mods == ui.Cmd|ui.Shift && s.PasteIndex < len(s.Pastes) && a.commitText == s.Pastes[s.PasteIndex].Before:
		message = s.Pastes[s.PasteIndex].After
		s.PasteIndex++
	default:
		return false
	}
	a.commitText = message
	a.projectCommitMessage(message)
	k.SetText(s.Summary)
	return true
}

func (a *app) commitMessageFields(c *ui.Context) {
	a.syncCommitFields()
	s := &a.commitFields
	fontSize, maxRows := float32(13), 8
	if a.page == "changes" {
		fontSize, maxRows = 12, 3
	}
	ui.Column(c.Key("commit-message")).FillWidth().Gap(5).Children(func() {
		display.Text(c, a.t("commitSummary"), display.TextProps{FontSize: scaledFont(11)})
		summary := field.Input(c.Key("summary"), &s.Summary, field.InputProps{Variant: field.InputFilled, Placeholder: a.t("commitSummaryHint")}).
			FillWidth().FontSize(scaledFont(fontSize)).Label(a.t("commitSummary")).OnTextKey(a.commitSummaryKey)
		start, end := summary.TextSelection()
		summary.OnPaste(func(text string) bool { return a.pasteCommitSummary(text, start, end) })
		if s.PendingPaste {
			a.projectCommitMessage(a.commitText)
			s.PendingPaste = false
			c.Invalidate()
		} else if summary.Changed() {
			a.storeCommitFields()
		}
		display.Text(c, a.t("commitDescription"), display.TextProps{FontSize: scaledFont(11)})
		body := field.TextArea(c.Key("body"), &s.Body, field.TextAreaProps{Variant: field.InputFilled, Placeholder: a.t("commitDescriptionHint"), MinRows: 3, MaxRows: maxRows}).
			FillWidth().FontSize(scaledFont(fontSize)).LineHeight(1.45).Label(a.t("commitDescription"))
		if body.Changed() {
			a.storeCommitFields()
		}
	})
}
