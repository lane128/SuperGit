package lobedesk

func stashPaletteCommand() appShortcut {
	return appShortcut{
		ID: "stash.saveTracked", Label: "stashTrackedCommand",
		Available: func(a *app) bool { return a.canSaveTrackedStashCommand() },
		Run:       func(a *app) { a.saveTrackedStashFromCommand() },
	}
}

func (a *app) trackedStashUnavailable() string {
	if a.status.Head == "" {
		return a.t("commandPaletteNeedHead")
	}
	if a.operation.Kind != "" {
		return a.t("quickCheckoutOperation")
	}
	tracked := false
	for _, entry := range a.status.Entries {
		if entry.Conflict {
			return a.t("quickCheckoutConflict")
		}
		if entry.Index != '?' && (entry.Staged() || entry.Unstaged()) {
			tracked = true
		}
	}
	if !tracked {
		return a.t("stashTrackedNothing")
	}
	return ""
}

func (a *app) canSaveTrackedStashCommand() bool {
	return a.commandPaletteUnavailable("stash.saveTracked") == "" && !a.hasPendingDialog(toastMain)
}

func (a *app) saveTrackedStashFromCommand() {
	if !a.canSaveTrackedStashCommand() {
		return
	}
	root, epoch := a.paletteRoot(), a.loadSeq
	a.promptSaveStash(false)
	if !a.prompt.open || a.prompt.action == nil {
		return
	}
	// The generic prompt already captures the repository root. This command
	// also rejects refreshed sessions and rechecks readiness when accepted.
	a.prompt.paletteHandoff, a.prompt.paletteFocus = true, true
	action := a.prompt.action
	cancelled := false
	a.prompt.paletteCancel = func() { cancelled = true }
	a.prompt.action = func(message string) {
		if cancelled || a.paletteRoot() != root {
			return
		}
		if a.loadSeq != epoch || !a.canSaveTrackedStashCommand() {
			reason := a.commandPaletteUnavailable("stash.saveTracked")
			if a.loadSeq != epoch || reason == "" {
				reason = a.t("stashTrackedChanged")
			}
			a.queue("warning", reason, "")
			a.invalidate()
			return
		}
		action(message)
	}
}
