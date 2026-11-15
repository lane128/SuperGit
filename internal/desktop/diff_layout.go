package desktop

import (
	"sort"
	"supergit/internal/gitrepo"
)

// Indices always address Diff.Lines. -1 denotes an absent cell, not line zero.
type splitRow struct{ Shared, Left, Right int }

func splitDiffRows(d gitrepo.Diff) []splitRow {
	var rows []splitRow
	for i := 0; i < len(d.Lines); {
		switch d.Lines[i].Kind {
		case "context":
			rows = append(rows, splitRow{-1, i, i})
			i++
		case "add", "del":
			var left, right []int
			j := i
			for j < len(d.Lines) && (d.Lines[j].Kind == "add" || d.Lines[j].Kind == "del") {
				if d.Lines[j].Kind == "del" {
					left = append(left, j)
				} else {
					right = append(right, j)
				}
				j++
			}
			for k := 0; k < max(len(left), len(right)); k++ {
				row := splitRow{-1, -1, -1}
				if k < len(left) {
					row.Left = left[k]
				}
				if k < len(right) {
					row.Right = right[k]
				}
				rows = append(rows, row)
			}
			i = j
		default:
			rows = append(rows, splitRow{i, -1, -1})
			i++
		}
	}
	return rows
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
