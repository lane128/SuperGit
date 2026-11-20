package lobedesk

// copySHAPaletteCommand copies the selected commit's full hash. The Summary
// pane's copy button and the row menu already do this; the palette entry
// makes the same action reachable without aiming at a small icon or opening a
// context menu.
func copySHAPaletteCommand() appShortcut {
	return appShortcut{
		ID: "commit.copySHA", Label: "copySHA",
		Available: func(a *app) bool { return a.canCopySelectedCommitSHA() },
		Run:       func(a *app) { a.copySelectedCommitSHA() },
	}
}

func (a *app) canCopySelectedCommitSHA() bool {
	if a.repo == nil {
		return false
	}
	_, ok := a.selectedCommit()
	return ok
}

func (a *app) copySelectedCommitSHA() {
	if !a.canCopySelectedCommitSHA() {
		return
	}
	if commit, ok := a.selectedCommit(); ok {
		a.copyText(commit.SHA)
	}
}
