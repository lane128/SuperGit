package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
)

// The editor treats Cmd+Shift+A as Select All before window shortcuts.
// Route explicit batch and detail-tab keys through their shared bindings and
// guards; ordinary editing keys remain with the editor.
func (a *app) commitFieldBatchKey(k ui.TextKey) bool {
	if a.fileTabTextKey(k) {
		return true
	}
	for _, s := range shortcutTable() {
		if (s.ID == "workspace.stageModified" || s.ID == "workspace.unstageAll") &&
			s.Mod == k.Mods && s.Key == k.Key && s.Available != nil && s.Available(a) {
			s.Run(a)
			return true
		}
	}
	return false
}

func (a *app) commitMessageField(c *ui.Context) {
	message := field.TextArea(c.Key("commit-message"), &a.commitText, field.TextAreaProps{
		Variant: field.InputFilled, Placeholder: a.t("commitMessage"), MinRows: 5, MaxRows: 9,
	}).FillWidth().FontSize(scaledFont(12)).Label(a.t("commitMessage")).OnTextKey(a.commitFieldBatchKey)
	a.consumeCommitMessageFocus(message)
}
