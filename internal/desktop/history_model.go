package desktop

import (
	"strings"
	"supergit/internal/gitrepo"
)

// Spans retain the original source-row/edge order, including both endpoints.
// A long edge is stored once, rather than copied into every crossed row.
type graphSpan struct{ Start, End, SourceLane, Lane, ViaLane int }
type graphIndex struct {
	spans  []graphSpan
	maxEnd []int
}

func indexGraph(rows []graphRow) graphIndex {
	count := 0
	for _, row := range rows {
		count += len(row.Edges)
	}
	index := graphIndex{spans: make([]graphSpan, 0, count)}
	for start, row := range rows {
		for _, edge := range row.Edges {
			index.spans = append(index.spans, graphSpan{Start: start, End: edge.Row, SourceLane: row.Lane, Lane: edge.Lane, ViaLane: edge.ViaLane})
		}
	}
	if count > 0 {
		index.maxEnd = make([]int, 4*count)
		index.build(1, 0, count)
	}
	return index
}
func (g *graphIndex) build(node, lo, hi int) int {
	if hi-lo == 1 {
		g.maxEnd[node] = g.spans[lo].End
	} else {
		mid := (lo + hi) / 2
		g.maxEnd[node] = max(g.build(node*2, lo, mid), g.build(node*2+1, mid, hi))
	}
	return g.maxEnd[node]
}
func (g *graphIndex) eachAt(row int, visit func(graphSpan)) {
	if len(g.spans) > 0 {
		g.visitAt(1, 0, len(g.spans), row, visit)
	}
}
func (g *graphIndex) visitAt(node, lo, hi, row int, visit func(graphSpan)) {
	// Starts are sorted. Whole subtrees can be rejected without visiting edges.
	if g.maxEnd[node] < row || g.spans[lo].Start > row {
		return
	}
	if hi-lo == 1 {
		visit(g.spans[lo])
		return
	}
	mid := (lo + hi) / 2
	g.visitAt(node*2, lo, mid, row, visit)
	g.visitAt(node*2+1, mid, hi, row, visit)
}

type historyPresentation struct {
	Commits []gitrepo.Commit
	Indices []int
	Rows    []graphRow
	Width   float32
	Edges   graphIndex
}
type historyCache struct {
	repository, revision uint64
	query, rawQuery      string
	model                *historyPresentation
}

// Main-thread only. Every accepted snapshot changes the revision; repository
// generation also invalidates the cache, even if snapshot revisions coincide.
// Models are immutable so retained draw callbacks keep their own frame's data.
func (a *app) historyModel() *historyPresentation {
	cached := &a.historyView
	current := cached.model != nil && cached.repository == a.guard.repository && cached.revision == a.historyRevision
	if current && cached.rawQuery == a.historyQuery {
		return cached.model
	}
	query := strings.ToLower(strings.TrimSpace(a.historyQuery))
	if current && cached.query == query {
		cached.rawQuery = a.historyQuery
		return cached.model
	}
	commits, indices := filterHistory(a.data.history, query)
	rows := layoutGraph(commits)
	width := float32(90)
	for _, row := range rows {
		width = max(width, float32(row.Lane+1)*18+14)
	}
	model := &historyPresentation{Commits: commits, Indices: indices, Rows: rows, Width: min(width, 210), Edges: indexGraph(rows)}
	*cached = historyCache{repository: a.guard.repository, revision: a.historyRevision, query: query, rawQuery: a.historyQuery, model: model}
	return model
}
