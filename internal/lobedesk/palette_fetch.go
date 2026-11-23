package lobedesk

import "slices"

var fetchPaletteMetadata = map[string]struct {
	detail  string
	aliases []string
}{
	"repo.fetch":       {"paletteFetchSelectDetail", []string{"fetch all remotes", "获取全部远端"}},
	"repo.fetchPrune":  {"paletteFetchPruneDetail", []string{"fetch prune", "fetch all remotes with prune", "获取并清理", "获取全部远端并清理"}},
	"repo.fetchTags":   {"paletteFetchTagsDetail", []string{"fetch tags", "fetch all remotes with tags", "获取标签", "获取全部远端和标签"}},
	"repo.fetchRemote": {"paletteFetchRemoteDetail", []string{"fetch remote", "获取指定远端"}},
}

func isFetchVariant(id string) bool {
	return id == "repo.fetchPrune" || id == "repo.fetchTags" || id == "repo.fetchRemote"
}

func isPlainFetchQuery(query string) bool {
	return query == "fetch" || query == "获取" || slices.Contains(fetchPaletteMetadata["repo.fetch"].aliases, query)
}

func fetchPaletteCommands() []appShortcut {
	commands := []struct {
		id, label string
		run       func(*app)
	}{
		{"repo.fetch", "paletteFetchCommand", func(a *app) { a.setPaletteStage(paletteFetchOptions) }},
		{"repo.fetchPrune", "paletteFetchPruneCommand", (*app).fetchPruneAll},
		{"repo.fetchTags", "paletteFetchTagsCommand", (*app).fetchTagsAll},
		{"repo.fetchRemote", "paletteFetchRemoteCommand", (*app).fetchRemoteFromCommand},
	}
	entries := make([]appShortcut, 0, len(commands))
	for _, command := range commands {
		id, run := command.id, command.run
		entries = append(entries, appShortcut{
			ID: id, Label: command.label,
			Available: func(a *app) bool { return a.canPaletteFetch(id) },
			Run: func(a *app) {
				if a.canPaletteFetch(id) {
					run(a)
				}
			},
		})
	}
	return entries
}

// Preserve only the Fetch navigation context, not rows or callbacks from an
// old frame. Backspace always rebuilds the parent from current repository data.
type fetchPaletteParent struct {
	stage  paletteStage
	query  string
	cursor int
	parent *fetchPaletteParent
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
