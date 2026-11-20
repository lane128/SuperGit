package lobedesk

// hasPendingDialog checks the surfaces that share a window's modal input layer.
// Stale repository surfaces cannot block a new request in the current root.
func (a *app) hasPendingDialog(owner toastOwner) bool {
	a.discardStaleCommandPalette()
	return owner == toastMain && a.commandPalette.open || a.hasPendingDialogWithoutPalette(owner)
}

// The palette needs to assess other modal owners while listing its own actions.
func (a *app) hasPendingDialogWithoutPalette(owner toastOwner) bool {
	a.discardStaleQuickCheckout()
	a.discardStaleHistoryRefPalette()
	return owner == toastMain && (a.quickCheckout.open || a.historyRefPalette.active) || a.hasPendingActionDialog(owner)
}

// Search pickers assess other dialog owners without blocking themselves.
func (a *app) hasPendingActionDialog(owner toastOwner) bool {
	a.discardStaleConfirmation()
	a.discardStalePrompt()
	a.discardStaleReset()
	a.discardStaleCheckoutBranch()
	a.discardStaleChildNavigation()
	a.discardStaleQuickRepositorySwitch()
	a.discardStaleWorktreeForm()
	a.discardStaleAddRemoteForm()
	a.discardStaleBlame()
	a.discardStaleComparison()
	a.discardStaleSnapshotFile()
	a.discardStaleNoticeDetails(owner)
	if a.confirm.open && a.confirm.owner == owner {
		return true
	}
	if state := a.noticeDetails[owner]; state.open && state.notice != nil {
		return true
	}
	return owner == toastMain && (a.prompt.open || a.reset.open || a.checkoutBranch.open || a.childNavigation.open || a.quickRepositorySwitch.open || a.worktreeForm.open || a.addRemoteForm.open || a.cloneForm.open || a.newRepoForm.open || a.blameOpen || a.snapshotView.open || a.comparison.view != nil)
}

func (a *app) rejectPendingDialog(owner toastOwner) bool {
	if !a.hasPendingDialog(owner) {
		return false
	}
	a.queueFor(owner, "warning", a.t("dialogPending"), "")
	a.invalidate()
	return true
}
