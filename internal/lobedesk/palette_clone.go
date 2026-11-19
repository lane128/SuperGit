package lobedesk

func clonePaletteCommand() appShortcut {
	return appShortcut{
		ID: "repo.clone", Label: "cloneRepoTitle",
		Available: func(a *app) bool { return a.canShowCloneRepository() },
		Run: func(a *app) {
			if !a.canShowCloneRepository() {
				return
			}
			a.showCloneRepository()
			a.cloneForm.paletteHandoff, a.cloneForm.paletteFocus = true, true
		},
	}
}
