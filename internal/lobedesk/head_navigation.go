package lobedesk

// navigateToHead reuses commit navigation without changing the checkout or draft.
func (a *app) navigateToHead() {
	if a.repo == nil || a.status.Head == "" || a.busy || a.loading || a.historyReloading || a.repositoryChooserOpen || a.historyRefOpen || a.hasPendingDialog(toastMain) {
		return
	}
	loaded := false
	for _, commit := range a.historySource() {
		if commit.SHA == a.status.Head {
			loaded = true
			break
		}
	}
	// File history pauses the full traversal internally; jumpToCommit resumes
	// it when leaving that filter. Other pauses and read errors still need a
	// retry when HEAD is missing, rather than retaining an inert pending jump.
	if a.historyErr != "" || !loaded && (a.historyPaused && a.fileFilter == "" || a.historyPageErr != "") {
		a.pendingJumpSHA = ""
		a.queue("warning", a.t("jumpToHead"), a.t("headNavigationBlocked"))
		a.invalidate()
		return
	}
	record := a.page == "history" && a.historyNavigationVisible() && a.historyNavigationMember(a.status.Head)
	a.openPage("history")
	if record {
		a.jumpWithHistory(a.status.Head)
	} else {
		a.clearHistoryNavigation()
		a.showCommit(a.status.Head)
	}
}
