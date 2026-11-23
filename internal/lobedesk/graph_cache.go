package lobedesk

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// History slices are immutable snapshots: readers append pages or replace the
// slice on reload/filter changes. Retain the slice to keep its identity alive.
type commitGraphCache struct {
	commits     []gitrepo.Commit
	palette     []ui.Color
	graph       *commitGraph
	head        string
	dirty       bool
	headPending bool
}

type reachableHistoryCache struct {
	source []gitrepo.Commit
	tip    string
	rows   []gitrepo.Commit
	valid  bool
}

func (a *app) cachedReachableHistory(tip string) []gitrepo.Commit {
	cache := &a.reachableHistory
	if !cache.valid || !sameHistorySnapshot(cache.source, a.commits) || cache.tip != tip {
		*cache = reachableHistoryCache{source: a.commits, tip: tip, rows: reachableCommits(a.commits, tip), valid: true}
	}
	return cache.rows
}

type filteredHistoryCache struct {
	source []gitrepo.Commit
	query  string
	rows   []gitrepo.Commit
	valid  bool
}

func sameHistorySnapshot(a, b []gitrepo.Commit) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

func (a *app) filteredHistoryRows() []gitrepo.Commit {
	source := a.historySource()
	cache := &a.filteredHistory
	if !cache.valid || !sameHistorySnapshot(cache.source, source) || cache.query != a.historyQuery {
		*cache = filteredHistoryCache{source: source, query: a.historyQuery, rows: filterCommits(source, a.historyQuery), valid: true}
	}
	return cache.rows
}

func (a *app) cachedCommitGraph(commits []gitrepo.Commit, palette []ui.Color, width float32) *commitGraph {
	cache := &a.commitGraphCache
	changed := cache.graph == nil || cache.head != a.status.Head || !sameHistorySnapshot(cache.commits, commits) || !slices.Equal(cache.palette, palette)
	if changed {
		graphRows := commits
		folds := &a.historyMergeFolds
		if folds.active && len(commits) < len(folds.source) {
			graphRows = slices.Clone(commits)
			for i := range graphRows {
				sha := graphRows[i].SHA
				if folds.branches[sha] && !folds.isExpanded(sha) {
					graphRows[i].Parents = graphRows[i].Parents[:1]
				}
			}
		}
		*cache = commitGraphCache{commits: commits, palette: slices.Clone(palette), head: a.status.Head, graph: newCommitGraph(a.status.Head, graphRows, palette)}
	}
	graph := cache.graph
	previousWidth, previousGap := graph.Width, graph.LaneGap()
	previousX := a.historyGraphScroll.X
	dirty := a.status.HasWorkspaceChanges()
	// An unfiltered walk pins the checked-out tip first. If its row is on
	// a later page, keep the dashed lane running to the end of this page.
	headPending := a.historyMore && a.fileFilter == "" && a.historyQuery == "" && a.historyRef.Kind == "" &&
		len(a.historyTips) > 0 && a.historyTips[0] == a.status.Head
	if changed || cache.dirty != dirty || cache.headPending != headPending {
		graph.Worktree = nil
		if link, ok := a.worktreeGraphConnection(commits, graph); ok {
			if link.HeadRow < 0 && link.HasHead && headPending {
				link.HeadRow = len(commits)
				link.Color = laneColorIndex(a.status.Branch, len(graph.Palette))
			}
			graph.Worktree = &link
		}
		cache.dirty, cache.headPending = dirty, headPending
	}
	graph.Width = graphWidth(graph.Lanes)
	graph.Gap = graphLaneGap
	if width > 0 {
		folds := &a.historyMergeFolds
		if folds.active && len(folds.branches) > 0 {
			if sameHistorySnapshot(commits, folds.source) {
				folds.fullLanes = graph.Lanes
			} else if folds.fullLanes == 0 {
				folds.fullLanes = newCommitGraph(a.status.Head, folds.source, palette).Lanes
			}
			// Fit the same viewport and lane spacing before and after a fold.
			// The projected graph still determines the actual scroll range.
			layout := commitGraph{Lanes: folds.fullLanes}
			layout.FitWidth(width)
			graph.Width, graph.Gap = layout.Width, layout.Gap
		} else {
			graph.FitWidth(width)
		}
	}
	limit := max(float32(0), graph.contentWidth()-graph.Width)
	a.historyGraphScroll.X = min(limit, max(float32(0), a.historyGraphScroll.X))
	gap := graph.LaneGap()
	geometryChanged := previousWidth != graph.Width || previousGap != gap
	revealNode := a.revealSelection || a.revealGraphSelection
	a.revealGraphSelection = false
	target := a.selectedSHA
	if a.selectedStash != "" {
		target = a.selectedStash
	}
	if anchor := a.historyMergeFolds.anchor; anchor != "" {
		a.historyMergeFolds.anchor = ""
		if !revealNode {
			target, revealNode = anchor, true
		}
	}
	if target != "" && (changed || revealNode || geometryChanged) {
		line, node := a.graphStyle()
		_, halo := graphNodeRing(node, line)
		nodeMargin := halo + 1 // Painting rounds down the graph origin.
		for i, commit := range commits {
			if commit.SHA != target {
				continue
			}
			margin := nodeMargin
			if folds := &a.historyMergeFolds; folds.active && folds.branches[target] {
				margin = max(margin, historyMergeDisclosureInset)
			}
			lane := graph.Rows[i].Lane
			previousNode := laneXGap(0, lane, previousGap)
			wasVisible := previousNode-margin >= previousX && previousNode+margin <= previousX+previousWidth
			// A resize keeps a visible node in view while respecting a pan that
			// already took the selected node out of the viewport.
			if changed || revealNode || wasVisible {
				x := laneXGap(0, lane, gap)
				if x-margin < a.historyGraphScroll.X {
					a.historyGraphScroll.X = max(float32(0), x-margin)
				}
				if x+margin > a.historyGraphScroll.X+graph.Width {
					a.historyGraphScroll.X = min(limit, x+margin-graph.Width)
				}
			}
			break
		}
	}
	return graph
}
