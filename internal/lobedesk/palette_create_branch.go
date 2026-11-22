package lobedesk

func createBranchPaletteCommand() appShortcut {
	return appShortcut{
		ID: "branch.create", Label: "createBranchCommand",
		Available: func(a *app) bool { return a.canCreateBranchCommand() },
		Run:       func(a *app) { a.createBranchFromCommand() },
	}
}

func (a *app) createBranchUnavailable() string {
	if a.status.Head == "" {
		return a.t("commandPaletteNeedHead")
	}
	if a.operation.Kind != "" {
		return a.t("quickCheckoutOperation")
	}
	if a.operationConflicts() {
		return a.t("quickCheckoutConflict")
	}
	return ""
}

func (a *app) canCreateBranchCommand() bool {
	return a.commandPaletteUnavailable("branch.create") == "" && !a.hasPendingDialog(toastMain)
}

func (a *app) createBranchFromCommand() {
	if !a.canCreateBranchCommand() {
		return
	}
	root, epoch := a.paletteRoot(), a.loadSeq
	sha, branch := a.status.Head, a.status.Branch
	a.promptNewBranchAt(sha, true)
	if !a.prompt.open || a.prompt.action == nil {
		return
	}
	a.prompt.title = a.t("createBranchSwitch")
	a.prompt.description = a.tf("createBranchStart", shortSHA(sha))
	a.prompt.actionName = a.t("createBranchSwitch")
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
		if a.loadSeq != epoch || a.status.Head != sha || a.status.Branch != branch || !a.canCreateBranchCommand() {
			reason := a.commandPaletteUnavailable("branch.create")
			if a.loadSeq != epoch || a.status.Head != sha || a.status.Branch != branch || reason == "" {
				reason = a.t("createBranchChanged")
			}
			a.queue("warning", reason, "")
			a.invalidate()
			return
		}
		action(name)
	}
}
