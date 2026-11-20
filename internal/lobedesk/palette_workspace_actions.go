package lobedesk

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"supergit/internal/gitrepo"
)

// Canonical reference names are additional search aliases for existing
// SuperGit commands that have more specific labels.
var referenceCommandNames = map[string]string{
	"repo.fetch":           "Fetch…",
	"branch.quickCheckout": "Checkout…",
	"branch.create":        "Create Branch…",
	"branch.merge":         "Merge Branch…",
	"window.settings":      "Preferences…",
	"repo.quickSwitch":     "Switch Repository…",
	"tag.create":           "Create Tag…",
}

// The default palette shows these Git equivalents to the right of the caption.
var paletteGitHints = map[string]string{
	"stash.pop": "git stash pop", "stash.drop": "git stash drop",
	"workspace.stageAllFiles": "git add -A", "workspace.stageAllModified": "git add -u",
	"workspace.unstageAllFiles": "git reset", "workspace.clean": "git clean -fd",
}

func init() {
	registerPaletteActions(workspacePaletteActions()...)
}

func workspacePaletteActions() []paletteAction {
	actions := []paletteAction{
		{ID: "window.about", English: "About", Chinese: "关于", Global: true, Run: func(a *app) {
			a.selectSettingsSection("about")
			a.openSettingsWindow()
		}},
		{ID: "stash.save", English: "Stash…", Chinese: "保存改动…", Unavailable: func(a *app) string {
			switch {
			case a.status.Head == "":
				return a.t("commandPaletteNeedHead")
			case len(a.status.Entries) == 0:
				return a.t("stashNothing")
			case a.operation.Kind != "":
				return a.t("quickCheckoutOperation")
			}
			for _, e := range a.status.Entries {
				if e.Conflict {
					return a.t("quickCheckoutConflict")
				}
			}
			return ""
		}, Run: (*app).chooseStashScope},
		{ID: "stash.pop", English: "Pop Stash", Chinese: "弹出最近的 Stash", Unavailable: needsStashes, Run: func(a *app) {
			a.popStash(a.stashes[0])
		}},
		{ID: "stash.drop", English: "Drop Stash…", Chinese: "删除 Stash…", Unavailable: needsStashes, Run: (*app).chooseStashToDrop},
		{ID: "stash.clear", English: "Clear All Stashes", Chinese: "清空所有 Stash", Unavailable: needsStashes, Run: (*app).confirmClearStashes},
		{ID: "workspace.clean", English: "Delete All Untracked Files…", Chinese: "删除所有未跟踪文件…", Run: (*app).prepareCleanUntracked},
		{ID: "repo.containingFolder", English: "Open Containing Folder…", Chinese: "打开仓库文件夹…", Run: func(a *app) { a.openPath(a.repo.Root, "open") }},
		{ID: "repo.terminal", English: "Open Repository in Terminal", Chinese: "在终端中打开仓库", Run: func(a *app) { a.openPath(a.repo.Root, "terminal") }},
		{ID: "repo.sublimeText", English: "Open Repository in Sublime Text", Chinese: "在 Sublime Text 中打开仓库", Run: func(a *app) { a.openPath(a.repo.Root, "sublimeText") }},
	}
	for _, op := range []struct{ id, en, zh, kind string }{
		{"workspace.stageAllFiles", "Stage All Modified and Untracked", "暂存所有已修改和未跟踪文件", "stage"},
		{"workspace.stageAllModified", "Stage All Modified", "暂存所有已修改文件", "stageTracked"},
		{"workspace.unstageAllFiles", "Unstage All", "取消暂存全部文件", "unstage"},
	} {
		actions = append(actions, paletteAction{ID: op.id, English: op.en, Chinese: op.zh,
			Unavailable: func(a *app) string { _, reason := a.batchFileTargets(op.kind); return reason },
			Run:         func(a *app) { a.runBatchStageKind(op.kind) }})
	}
	return actions
}

func needsStashes(a *app) string {
	if a.stashesErr != "" {
		return a.stashesErr
	}
	if len(a.stashes) == 0 {
		return a.t("noStashes")
	}
	return ""
}

func (a *app) chooseStashScope() {
	title := a.paletteText("Stash", "保存改动")
	rows := []paletteChoice{
		{id: "tracked", label: "stash push", detail: a.paletteText("Tracked changes", "仅已跟踪的改动"), reason: a.trackedStashUnavailable()},
		{id: "untracked", label: "stash push --include-untracked", detail: a.paletteText("Include untracked files", "包括未跟踪文件")},
	}
	a.choosePalette(title, a.paletteText("Stash options", "选择保存范围"), rows, func(row paletteChoice) {
		a.inputPalette(title, a.paletteText("Message (optional)", "说明（可选）"), "", func(message string) {
			a.performAs(busySaveStash, a.t("saveStashFailed"), a.t("saveStashSuccess"), func(ctx context.Context, repo *gitrepo.Repository) error {
				return repo.SaveStash(ctx, message, row.id == "untracked")
			})
		})
		a.commandPalette.step.allowEmpty = true
	})
}

func (a *app) chooseStashToDrop() {
	stashes := slices.Clone(a.stashes)
	var rows []paletteChoice
	for _, stash := range stashes {
		rows = append(rows, paletteChoice{id: stash.Ref, label: stash.Ref, detail: stashTitle(stash), sha: stash.SHA})
	}
	a.choosePalette(a.paletteText("Drop Stash", "删除 Stash"), a.paletteText("Choose a stash", "选择 Stash"), rows, func(row paletteChoice) {
		for _, stash := range stashes {
			if stash.Ref == row.id && stash.SHA == row.sha {
				a.confirmDropStash(stash)
				return
			}
		}
	})
}

func (a *app) confirmClearStashes() {
	expected := slices.Clone(a.stashes)
	title := a.paletteText("Clear All Stashes", "清空所有 Stash")
	message := fmt.Sprintf(a.paletteText("Permanently remove all %d stashes in this repository?", "永久删除此仓库的全部 %d 条 Stash？"), len(expected))
	a.askConfirm(title, message, a.t("delete"), true, func() {
		a.perform(title, "", func(ctx context.Context, repo *gitrepo.Repository) error {
			return repo.ClearStashesChecked(ctx, expected)
		})
	})
}

func (a *app) prepareCleanUntracked() {
	repo, epoch, generation := a.repo, a.loadSeq, a.commandPaletteGeneration
	ctx := a.ctx
	a.discardPrepareSeq++
	request := a.discardPrepareSeq
	go func() {
		preview, err := repo.PrepareCleanUntracked(ctx)
		a.hopToMain(func() {
			if a.repo != repo || a.loadSeq != epoch || a.commandPaletteGeneration != generation || a.activeRepositorySession() != a || a.discardPrepareSeq != request || a.busy || a.loading || a.hasPendingDialog(toastMain) {
				return
			}
			title := a.paletteText("Delete All Untracked Files", "删除所有未跟踪文件")
			if err != nil {
				a.queue("error", title, err.Error())
			} else if len(preview.Paths) == 0 {
				a.queue("info", a.paletteText("No untracked files or directories", "没有未跟踪的文件或目录"), "")
			} else {
				list := strings.Join(preview.Paths[:min(8, len(preview.Paths))], "\n")
				if len(preview.Paths) > 8 {
					list += "\n…"
				}
				message := fmt.Sprintf(a.paletteText("Permanently remove %d untracked paths?\n%s", "永久删除 %d 个未跟踪路径？\n%s"), len(preview.Paths), list)
				a.askConfirm(title, message, a.t("delete"), true, func() {
					a.perform(title, "", func(ctx context.Context, current *gitrepo.Repository) error {
						return current.CleanUntrackedChecked(ctx, preview)
					})
				})
			}
			a.invalidate()
		})
	}()
}
