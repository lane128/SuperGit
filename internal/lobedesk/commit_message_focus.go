package lobedesk

import "github.com/egoist/mygo/ui"

func (a *app) canFocusCommitMessage() bool {
	return a.repo != nil && !a.loading && !a.exclusiveBusy() && !a.historyReloading &&
		!a.repositoryChooserOpen && !a.historyRefOpen && !a.hasPendingDialog(toastMain)
}

func (a *app) focusCommitMessage() {
	if !a.canFocusCommitMessage() {
		return
	}
	a.openPage("changes")
	// The box lives in the details pane. A hidden pane has no editor to
	// focus, so the shortcut brings the pane back rather than doing nothing.
	if a.historyDetailsHidden {
		a.setHistoryDetailsHidden(false)
	}
	a.commitMessageFocusRoot = a.repo.Root
	a.commitMessageFocusSeq = a.loadSeq
	a.invalidate()
}

// Requests expire rather than surviving navigation or a newly blocking dialog.
func (a *app) discardStaleCommitMessageFocus() {
	if a.commitMessageFocusRoot != "" && (!a.canFocusCommitMessage() || a.page != "history" || !a.worktreeSelected ||
		a.repo.Root != a.commitMessageFocusRoot || a.loadSeq != a.commitMessageFocusSeq) {
		a.commitMessageFocusRoot = ""
	}
}

func (a *app) consumeCommitMessageFocus(summary ui.Element) {
	a.discardStaleCommitMessageFocus()
	if a.commitMessageFocusRoot == "" {
		return
	}
	a.commitMessageFocusRoot = ""
	summary.Focus().ScrollIntoView()
}
