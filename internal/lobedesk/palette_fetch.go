package lobedesk

func fetchPaletteCommand() appShortcut {
	return appShortcut{
		ID: "repo.fetch", Label: "paletteFetchCommand",
		Available: func(a *app) bool { return a.canFetchCommand() },
		Run:       func(a *app) { a.fetchFromCommand() },
	}
}

func (a *app) canFetchCommand() bool {
	return !a.commandPalette.open && a.commandPaletteUnavailable("repo.fetch") == ""
}

func (a *app) fetchFromCommand() {
	if a.canFetchCommand() {
		a.setPaletteStage(paletteFetchOptions, "")
	}
}
