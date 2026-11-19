package desktop

import (
	"sort"
	"supergit/internal/gitrepo"
)

func splitDiffRows(d gitrepo.Diff) []gitrepo.SplitDiffRow {
	return gitrepo.SplitDiffRows(d.Lines, nil)
}

type lineTarget struct {
	Path      string
	Diff      gitrepo.Diff
	Selection []int
	Reverse   bool
}

func (a *app) linesAllowed(history bool) bool {
	return !history && !a.busy() && !a.diffLoading && a.diff.CanStageHunks && a.hunksAllowed()
}
func (a *app) captureLines(history bool) (lineTarget, bool) {
	if !a.linesAllowed(history) {
		return lineTarget{}, false
	}
	var selection []int
	for i, on := range a.selectedLines {
		if on && i >= 0 && i < len(a.diff.Lines) && (a.diff.Lines[i].Kind == "add" || a.diff.Lines[i].Kind == "del") {
			selection = append(selection, i)
		}
	}
	if len(selection) == 0 {
		return lineTarget{}, false
	}
	sort.Ints(selection)
	return lineTarget{a.path, a.diff, selection, a.staged}, true
}
func (a *app) clearLineSelection() { a.selectedLines = nil }
