package lobedesk

// Only the visible historical selection supplies a parent shortcut target.
// selectedCommit's fallback intentionally serves details hidden by a filter.
func (a *app) parentNavigationTarget() string {
	return a.parentNavigationTargetAt(0)
}

func (a *app) parentNavigationTargetAt(index int) string {
	if index < 0 || a.repo == nil || a.page != "history" || a.worktreeSelected || a.busy || a.loading || a.historyReloading || a.repositoryChooserOpen || a.historyRefOpen || a.hasPendingDialog(toastMain) {
		return ""
	}
	for _, commit := range a.historyRows() {
		if commit.SHA == a.selectedSHA && len(commit.Parents) > index {
			return commit.Parents[index]
		}
	}
	return ""
}

func (a *app) navigateToSecondParent() {
	if sha := a.parentNavigationTargetAt(1); sha != "" {
		a.jumpWithHistory(sha)
	}
}

func (a *app) navigateToParent() {
	if sha := a.parentNavigationTarget(); sha != "" {
		a.jumpWithHistory(sha)
	}
}
