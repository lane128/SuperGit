package lobedesk

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"slices"
	"testing"

	"supergit/internal/gitrepo"
)

// All tips share an old parent: edge spans grow quadratically even though
// the underlying DAG contains only n-1 edges. Keep this topology in coverage.
func distantParentHistory(n int) []gitrepo.Commit {
	commits := make([]gitrepo.Commit, n)
	for i := range commits {
		commits[i] = gitrepo.Commit{SHA: fmt.Sprintf("long-%d", i)}
		if i+1 < n {
			commits[i].Parents = []string{fmt.Sprintf("long-%d", n-1)}
		}
	}
	return commits
}

func TestCommitGraphLongEdgesUseLinearSpace(t *testing.T) {
	commits := distantParentHistory(1024)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	graph := newCommitGraph("", commits, nil)
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc
	// Generous allowance for maps and the edge index, but not copies of each
	// edge into hundreds of intervening rows (formerly about 79 MB).
	if allocated > 8<<20 {
		t.Fatalf("1024-commit graph allocated %d bytes; want less than 8 MiB", allocated)
	}
	runtime.KeepAlive(graph)
}

func graphRowSpans(graph *commitGraph, row int) []commitGraphEdge {
	return slices.Collect(graph.spans(row))
}

func TestCommitGraphSpanQueriesMatchDAG(t *testing.T) {
	rng := rand.New(rand.NewPCG(21, 42))
	for _, n := range []int{0, 1, 2, 31, 128} {
		commits := make([]gitrepo.Commit, n)
		for i := range commits {
			commits[i].SHA = fmt.Sprint(i)
			for j := i + 1; j < n; j++ {
				if rng.IntN(7) == 0 {
					commits[i].Parents = append(commits[i].Parents, fmt.Sprint(j))
				}
			}
			commits[i].Parents = append(commits[i].Parents, "not-loaded")
		}
		graph := newCommitGraph("", commits, nil)
		type endpoints struct{ start, end int }
		for row := -1; row <= n; row++ {
			var want []endpoints
			if row >= 0 && row < n {
				for start, commit := range commits {
					for _, parent := range commit.Parents {
						for end := start + 1; end < n; end++ {
							if commits[end].SHA == parent && start <= row && row <= end {
								want = append(want, endpoints{start, end})
							}
						}
					}
				}
			}
			var got []endpoints
			for edge := range graph.spans(row) {
				got = append(got, endpoints{edge.Start, edge.End})
			}
			if !slices.Equal(got, want) {
				t.Fatalf("n=%d row=%d: edges %v; want %v", n, row, got, want)
			}
			// Range termination must propagate through both tree branches.
			visited := 0
			for range graph.spans(row) {
				visited++
				break
			}
			if visited != min(1, len(want)) {
				t.Fatalf("early stop visited %d edges", visited)
			}
		}
	}
}

func TestManyLaneGraphStaysInsideViewport(t *testing.T) {
	graph := &commitGraph{Lanes: 64, Gap: graphLaneGap}
	graph.FitWidth(80.6)
	if graph.Width > 80.61 {
		t.Fatalf("64 lanes occupy %.1f DIP inside an 80.6-DIP budget", graph.Width)
	}
	if graph.LaneGap() < graphLaneGapMin || graph.contentWidth() <= graph.Width {
		t.Fatalf("dense graph must keep lanes separated and scrollable: gap %.1f, width %.1f in %.1f DIP", graph.LaneGap(), graph.contentWidth(), graph.Width)
	}
}

func BenchmarkCommitGraphLongEdges(b *testing.B) {
	for _, n := range []int{256, 512, 1024} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			commits := distantParentHistory(n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				runtime.KeepAlive(newCommitGraph("", commits, nil))
			}
		})
	}
}
