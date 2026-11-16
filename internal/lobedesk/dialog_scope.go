package lobedesk

// hasPendingDialog checks the surfaces that share a window's modal input layer.
// Stale repository surfaces cannot block a new request in the current root.
func (a *app) hasPendingDialog(owner toastOwner) bool {
	a.discardStaleConfirmation()
	a.discardStalePrompt()
	a.discardStaleReset()
	a.discardStaleCheckoutBranch()
	a.discardStaleWorktreeForm()
	a.discardStaleBlame()
	a.discardStaleNoticeDetails(owner)
	if a.confirm.open && a.confirm.owner == owner {
		return true
	}
	if state := a.noticeDetails[owner]; state.open && state.notice != nil {
		return true
	}
	return owner == toastMain && (a.prompt.open || a.reset.open || a.checkoutBranch.open || a.worktreeForm.open || a.blameOpen)
}

func (a *app) rejectPendingDialog(owner toastOwner) bool {
	if !a.hasPendingDialog(owner) {
		return false
	}
	a.queueFor(owner, "warning", a.t("dialogPending"), "")
	a.invalidate()
	return true
}
