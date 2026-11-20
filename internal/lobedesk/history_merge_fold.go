package lobedesk

import (
	"strings"

	"supergit/internal/gitrepo"
)

// historyMergeFolds is view state. The projection keeps the original commit
// objects, including all parents for the details pane and commit navigation.
type historyMergeFolds struct {
	source          []gitrepo.Commit
	fullLanes       int
	head            string
	branches        map[string]bool
	mergeOf         map[string]string
	expanded        map[string]bool // Per-merge overrides for the current default.
	defaultExpanded bool
	projected       []gitrepo.Commit
	dirty           bool
	active          bool
	partialRows     bool
	anchor          string
}

func (a *app) projectHistoryMerges(rows []gitrepo.Commit) []gitrepo.Commit {
	folds := &a.historyMergeFolds
	if folds.defaultExpanded != a.expandMerges {
		folds.defaultExpanded = a.expandMerges
		folds.expanded = nil
		folds.dirty = true
	}
	canFold := a.showGraph && !a.historyReloading && a.historyErr == "" && a.fileFilter == "" &&
		a.historyRef.Kind == "" && strings.TrimSpace(a.historyQuery) == ""
	folds.active = canFold && !a.historyMore
	if !folds.active {
		folds.anchor = ""
		folds.partialRows = canFold && a.historyMore && len(rows) > 0
		return rows
	}
	preserveScroll := folds.partialRows && !a.revealSelection && a.historyScroll.Y > historyListPadding
	folds.partialRows = false
	if !sameHistorySnapshot(folds.source, rows) || folds.head != a.status.Head {
		folds.source, folds.head = rows, a.status.Head
		folds.fullLanes = 0
		folds.branches, folds.mergeOf = findHistoryMergeBranches(rows, folds.head)
		if len(folds.expanded) > 0 {
			// A new ref can temporarily prevent folding an existing merge.
			// Keep its explicit expansion for when that ref goes away.
			present := make(map[string]bool, len(folds.expanded))
			for _, commit := range rows {
				if state, ok := folds.expanded[commit.SHA]; ok {
					present[commit.SHA] = state
				}
			}
			folds.expanded = present
		}
		folds.dirty = true
	}
	// Selecting a hidden commit via a parent link, a search, or an outside
	// jump reveals the containing branch again.
	if merge := folds.mergeOf[a.selectedSHA]; merge != "" && !folds.isExpanded(merge) {
		folds.expand(merge)
	}
	if folds.dirty || preserveScroll {
		folds.projected = rows
		for merge := range folds.branches {
			if !folds.isExpanded(merge) {
				folds.projected = a.foldedHistoryRows(rows, preserveScroll)
				break
			}
		}
		folds.dirty = false
	}
	return folds.projected
}

// foldedHistoryRows preserves the first visible surviving row when folding
// begins after pagination. If the top row is folded, its next visible neighbor
// takes its place. A deliberate selection reveal is handled by the list.
func (a *app) foldedHistoryRows(rows []gitrepo.Commit, preserveScroll bool) []gitrepo.Commit {
	folds := &a.historyMergeFolds
	projected := make([]gitrepo.Commit, 0, len(rows))
	anchor, height := -1, a.historyRowHeight()
	if preserveScroll {
		anchor = int((a.historyScroll.Y - historyListPadding) / height)
	}
	for i, commit := range rows {
		if merge := folds.mergeOf[commit.SHA]; merge != "" && !folds.isExpanded(merge) {
			continue
		}
		if anchor >= 0 && i >= anchor {
			a.historyScroll.Y -= float32(anchor-len(projected)) * height
			anchor = -1
		}
		projected = append(projected, commit)
	}
	return projected
}

// findHistoryMergeBranches recognizes a linear side connected to the first
// parent's mainline. Every hidden commit must have exactly one child and no
// ref of its own. The history is complete and ordered child before parent.
func findHistoryMergeBranches(rows []gitrepo.Commit, head string) (map[string]bool, map[string]string) {
	positions := make(map[string]int, len(rows))
	children := make(map[string]int, len(rows))
	for i, commit := range rows {
		positions[commit.SHA] = i
		for _, parent := range commit.Parents {
			children[parent]++
		}
	}
	// Index the first-parent forest once. Its DFS intervals make ancestry
	// checks constant-time even when many merges branched from the same base.
	firstChildren := make([][]int, len(rows))
	var roots []int
	for i, commit := range rows {
		if len(commit.Parents) != 0 {
			if parent, ok := positions[commit.Parents[0]]; ok && parent > i {
				firstChildren[parent] = append(firstChildren[parent], i)
				continue
			}
		}
		roots = append(roots, i)
	}
	type interval struct{ start, end int }
	intervals := make([]interval, len(rows))
	clock := 0
	for _, root := range roots {
		stack := []int{root}
		for len(stack) > 0 {
			last := len(stack) - 1
			i := stack[last]
			stack = stack[:last]
			if i < 0 {
				intervals[^i].end = clock
				continue
			}
			intervals[i].start = clock
			clock++
			stack = append(stack, ^i) // Record the end after visiting the descendants.
			stack = append(stack, firstChildren[i]...)
		}
	}
	branches := make(map[string]bool)
	mergeOf := make(map[string]string)
	var side []string
	for i, merge := range rows {
		if len(merge.Parents) != 2 {
			continue
		}
		main, ok := positions[merge.Parents[0]]
		if !ok || main <= i {
			continue
		}
		side = side[:0]
		last, next := i, merge.Parents[1]
		for {
			sideRow, ok := positions[next]
			if !ok || sideRow <= last {
				break
			}
			base := intervals[sideRow]
			if base.start <= intervals[main].start && intervals[main].start < base.end {
				if len(side) > 0 {
					branches[merge.SHA] = true
					for _, sha := range side {
						mergeOf[sha] = merge.SHA
					}
				}
				break
			}
			commit := rows[sideRow]
			if len(commit.Parents) != 1 || children[next] != 1 || commit.Refs != "" || next == head {
				break
			}
			side = append(side, next)
			last, next = sideRow, commit.Parents[0]
		}
	}
	return branches, mergeOf
}

func (folds *historyMergeFolds) isExpanded(merge string) bool {
	if v, ok := folds.expanded[merge]; ok {
		return v
	}
	return folds.defaultExpanded
}

func (folds *historyMergeFolds) expand(merge string) {
	if folds.expanded == nil {
		folds.expanded = make(map[string]bool)
	}
	folds.expanded[merge] = true
	folds.dirty = true
}

func (a *app) toggleHistoryMerge(merge string) {
	folds := &a.historyMergeFolds
	if !folds.active || !folds.branches[merge] {
		return
	}
	folds.anchor = merge
	if !folds.isExpanded(merge) {
		folds.expand(merge)
		return
	}
	if folds.mergeOf[a.selectedSHA] == merge {
		a.selectCommit(merge)
	}
	if folds.expanded == nil {
		folds.expanded = make(map[string]bool)
	}
	folds.expanded[merge] = false
	folds.dirty = true
}
