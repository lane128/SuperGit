package lobedesk

// Fetch commands share the palette's search and availability rules. Only the
// explicit remote command asks for a target; every other variant fetches all
// remotes in one step.
func fetchPaletteCommands() []appShortcut {
	commands := []struct {
		id, label, run string
	}{
		{"repo.fetch", "paletteFetchCommand", "fetchAll"},
		{"repo.fetchPrune", "paletteFetchPruneCommand", "fetchPruneAll"},
		{"repo.fetchTags", "paletteFetchTagsCommand", "fetchTagsAll"},
		{"repo.fetchRemote", "paletteFetchRemoteCommand", "fetchRemoteFromCommand"},
	}
	entries := make([]appShortcut, 0, len(commands))
	for _, command := range commands {
		id, run := command.id, command.run
		entries = append(entries, appShortcut{
			ID: id, Label: command.label,
			Available: func(a *app) bool { return a.canPaletteFetch(id) },
			Run: func(a *app) {
				// A direct call (context menu, stale callback) still has to
				// pass the same guards the palette row showed.
				if a.commandPalette.open || a.commandPaletteUnavailable(id) != "" {
					return
				}
				switch run {
				case "fetchAll":
					a.fetchAll()
				case "fetchPruneAll":
					a.fetchPruneAll()
				case "fetchTagsAll":
					a.fetchTagsAll()
				case "fetchRemoteFromCommand":
					a.fetchRemoteFromCommand()
				}
			},
		})
	}
	return entries
}

func (a *app) canPaletteFetch(id string) bool {
	return !a.commandPalette.open && a.commandPaletteUnavailable(id) == ""
}

func (a *app) fetchRemoteFromCommand() {
	if a.commandPalette.open || a.commandPaletteUnavailable("repo.fetchRemote") != "" {
		return
	}
	a.setPaletteStage(paletteFetchRemote)
}
