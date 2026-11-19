package lobedesk

// F7 enters the existing fast-forward-only action; availability is checked
// again at dispatch so retained callbacks cannot pass through a new dialog.
func (a *app) canPullShortcut() bool {
	if a.repo == nil || a.busy || a.loading || a.historyReloading || a.repositoryChooserOpen || a.historyRefOpen || a.hasPendingDialog(toastMain) || a.status.Branch == "" || a.status.Upstream == "" {
		return false
	}
	for _, branch := range a.branches {
		if branch.Current && branch.Name == a.status.Branch && branch.UpstreamGone {
			return false
		}
	}
	return true
}

func (a *app) pullFromShortcut() {
	if a.canPullShortcut() {
		a.pullFF()
	}
}
