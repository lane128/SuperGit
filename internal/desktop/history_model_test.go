package desktop

import (
	"reflect"
	"supergit/internal/gitrepo"
	"testing"
)

func TestGraphIndexMatchesOriginalCrossingsAndOrder(t *testing.T) {
	for _, commits := range [][]gitrepo.Commit{
		performanceCommits(100),
		{{SHA: "merge", Parents: []string{"main", "feature"}}, {SHA: "feature", Parents: []string{"root"}}, {SHA: "main", Parents: []string{"root"}}, {SHA: "root", Parents: []string{"outside"}}},
	} {
		rows := layoutGraph(commits)
		index := indexGraph(rows)
		for row := -1; row <= len(rows); row++ {
			var expected, got []graphSpan
			for start, r := range rows {
				for _, edge := range r.Edges {
					if start <= row && row <= edge.Row {
						expected = append(expected, graphSpan{Start: start, End: edge.Row, SourceLane: r.Lane, Lane: edge.Lane, ViaLane: edge.ViaLane})
					}
				}
			}
			index.eachAt(row, func(span graphSpan) { got = append(got, span) })
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("row %d crossings %#v, want %#v", row, got, expected)
			}
		}
	}
}
func TestGraphIndexStoresLongEdgesOnce(t *testing.T) {
	rows := make([]graphRow, 2000)
	for i := 0; i < 1999; i++ {
		rows[i].Edges = []graphEdge{{Row: 1999, Lane: 0, ViaLane: 0}}
	}
	index := indexGraph(rows)
	if len(index.spans) != 1999 {
		t.Fatal("long spans were expanded per crossed row")
	}
	count := 0
	index.eachAt(1800, func(graphSpan) { count++ })
	if count != 1801 {
		t.Fatalf("long crossings %d", count)
	}
}
func TestHistoryPresentationCachesNormalizedQueryAndInvalidatesSnapshot(t *testing.T) {
	a := &app{historyRevision: 1, historyQuery: "Perf change"}
	a.data.history = performanceCommits(100)
	model := a.historyModel()
	commits, rows := &model.Commits[0], &model.Rows[0]
	a.historyQuery = "  PERF change  "
	next := a.historyModel()
	if &next.Commits[0] != commits || &next.Rows[0] != rows {
		t.Fatal("unchanged history rebuilt between frames")
	}
	a.data.history = performanceCommits(101)
	a.historyRevision++
	next = a.historyModel()
	if len(next.Commits) != 101 || &next.Rows[0] == rows {
		t.Fatal("new snapshot reused stale history")
	}
	a.guard.Switch()
	a.data.history = performanceCommits(5)
	next = a.historyModel()
	if len(next.Commits) != 5 {
		t.Fatal("new repository reused previous presentation")
	}
}
func TestHistoryPresentationKeepsOriginalSelectionAndBoundaries(t *testing.T) {
	a := &app{historyQuery: "needle"}
	a.data.history = performanceCommits(60)
	a.data.history[5].Subject = "needle first"
	a.data.history[33].Subject = "needle second"
	model := a.historyModel()
	if !reflect.DeepEqual(model.Indices, []int{5, 33}) || model.Commits[1].SHA != a.data.history[33].SHA {
		t.Fatal("filtered row targets wrong original commit")
	}
	if !model.Rows[0].Boundary || !model.Rows[1].Boundary {
		t.Fatal("filtered parent boundary disappeared")
	}
	a.historyQuery = "no matching subject"
	if len(a.historyModel().Commits) != 0 {
		t.Fatal("query change retained old results")
	}
}
