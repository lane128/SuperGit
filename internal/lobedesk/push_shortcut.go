package lobedesk

// F8 opens the ordinary Push confirmation. Repeat and retained callbacks must
// respect the same modal and repository-read boundaries as initial dispatch.
func (a *app) canPushShortcut() bool {
	return a.repo != nil && a.status.Branch != "" && a.status.Head != "" && len(a.remotes) > 0 &&
		!a.busy && !a.loading && !a.historyReloading && !a.repositoryChooserOpen && !a.historyRefOpen && !a.hasPendingDialog(toastMain)
}

func (a *app) pushFromShortcut() {
	if a.canPushShortcut() {
		a.confirmPush()
	}
}
