package lobedesk

import (
	"github.com/egoist/mygo/ui"

	"github.com/lobehub/lobe-mygo/pkg/lucide"
)

func (a *app) fileActionButtons(c *ui.Context, target fileActionTarget) {
	ready := a.fileActionTargetCurrent(target)
	hint := target.Entry.Path
	if !fileStageSupported(target.Entry, target.Staged) {
		hint += "\n" + a.t("fileActionsProtected")
	}
	if !target.Staged {
		label, icon := a.t("discard"), lucide.Undo2()
		if isUntracked(target.Entry) {
			label, icon = a.t("deleteFile"), lucide.Trash2()
		}
		discardHint := hint
		if target.Entry.IsSubmodule() {
			discardHint += "\n" + a.t("fileActionsProtected")
		}
		button := compactActionIcon(c.Key("discard-file"), icon, label, buttonDangerIcon, !ready || target.Entry.IsSubmodule()).
			Description(discardHint).Tooltip(label + "\n" + discardHint).Shrink(0)
		if button.Clicked() && a.fileActionTargetCurrent(target) {
			a.prepareDiscardTargets([]fileActionTarget{target}, false)
		}
	}
	label := a.t("stageFile")
	if target.Staged {
		label = a.t("unstageFile")
	}
	button := workbenchButton(c.Key("stage-file"), label, buttonNormal, !ready).
		Label(label).Description(hint).Tooltip(label + "\n" + hint).Shrink(0)
	if button.Clicked() {
		a.runFileStage(target)
	}
}
