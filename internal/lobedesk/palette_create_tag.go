package lobedesk

func createTagPaletteCommand() appShortcut {
	return appShortcut{ID: "tag.create", Label: "createTagCommand",
		Available: func(a *app) bool { return a.canCreateTagCommand() },
		Run:       func(a *app) { a.createTagFromCommand() },
	}
}

func (a *app) createTagUnavailable() string {
	if a.page != "history" || a.worktreeSelected || a.selectedSHA == "" || a.selectedCommitIndex() < 0 {
		return a.t("createTagNeedCommit")
	}
	return ""
}

func (a *app) canCreateTagCommand() bool {
	return a.commandPaletteUnavailable("tag.create") == "" && !a.hasPendingDialog(toastMain)
}

func (a *app) createTagFromCommand() {
	if !a.canCreateTagCommand() {
		return
	}
	root, epoch, sha := a.paletteRoot(), a.loadSeq, a.selectedSHA
	a.promptCreateTag(sha)
	if !a.prompt.open || a.prompt.action == nil {
		return
	}
	a.prompt.title = a.t("createLightweightTagTitle")
	a.prompt.description = a.tf("createTagStart", shortSHA(sha))
	a.prompt.paletteHandoff, a.prompt.paletteFocus = true, true
	action := a.prompt.action
	consumed := false
	a.prompt.paletteCancel = func() { consumed = true }
	a.prompt.action = func(name string) {
		if consumed {
			return
		}
		consumed = true
		if a.paletteRoot() != root {
			return
		}
		if a.loadSeq != epoch || !a.canCreateTagCommand() {
			reason := a.commandPaletteUnavailable("tag.create")
			if a.loadSeq != epoch || reason == "" {
				reason = a.t("createTagChanged")
			}
			a.queue("warning", reason, "")
			a.invalidate()
			return
		}
		action(name)
	}
}
