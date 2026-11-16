package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
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
		button := action.ActionIcon(c.Key("discard-file"), icon, action.ActionIconProps{Title: label, Size: theme.Small}).
			Disabled(!ready || target.Entry.IsSubmodule()).Description(discardHint).Tooltip(label + "\n" + discardHint).Shrink(0)
		if button.Clicked() && a.fileActionTargetCurrent(target) {
			a.prepareDiscardTargets([]fileActionTarget{target}, false)
		}
	}
	label, icon := a.t("stageFile"), lucide.Plus()
	if target.Staged {
		label, icon = a.t("unstageFile"), lucide.Minus()
	}
	button := action.ActionIcon(c.Key("stage-file"), icon, action.ActionIconProps{Title: label, Size: theme.Small}).
		Disabled(!ready).Description(hint).Tooltip(label + "\n" + hint).Shrink(0)
	if button.Clicked() {
		a.runFileStage(target)
	}
}

func (a *app) overviewBatchActions(c *ui.Context) {
	ui.Row(c).FillWidth().Padding(0, 12, 8, 12).Gap(4).Children(func() {
		for _, item := range []struct{ Kind, Label, Scope string }{
			{"stage", "stageAll", "stageAllScope"},
			{"discard", "discardAll", "discardAllScope"},
			{"unstage", "unstageAll", "unstageAllScope"},
		} {
			_, reason := a.batchFileTargets(item.Kind)
			hint := a.t(item.Scope)
			if reason != "" {
				hint += "\n" + reason
			}
			button := action.Button(c.Key(item.Kind), a.t(item.Label), action.ButtonProps{Size: theme.Small}).
				Grow(1).Basis(0).MinWidth(0).Disabled(reason != "").Description(hint).Tooltip(hint)
			if button.Clicked() {
				switch item.Kind {
				case "stage":
					a.stageAll()
				case "unstage":
					a.unstageAll()
				case "discard":
					a.confirmDiscardAll()
				}
			}
		}
	})
}
