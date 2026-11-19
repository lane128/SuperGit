package lobedesk

func fetchPaletteCommand() appShortcut {
	return appShortcut{
		ID: "repo.fetch", Label: "fetchAllCommand",
		Available: func(a *app) bool { return a.canFetchCommand() },
		Run:       func(a *app) { a.fetchFromCommand() },
	}
}

func (a *app) canFetchCommand() bool {
	return a.repo != nil && len(a.remotes) > 0 && !a.busy && !a.loading &&
		!a.historyReloading && !a.repositoryChooserOpen && !a.historyRefOpen && !a.hasPendingDialog(toastMain)
}

func (a *app) fetchFromCommand() {
	if a.canFetchCommand() {
		a.fetchAll()
	}
}
