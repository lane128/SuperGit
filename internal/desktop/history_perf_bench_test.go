package desktop

import (
	"context"
	"fmt"
	"os"
	"supergit/internal/gitrepo"
	"testing"
)

func performanceCommits(count int) []gitrepo.Commit {
	commits := make([]gitrepo.Commit, count)
	for i := range commits {
		commits[i] = gitrepo.Commit{SHA: fmt.Sprintf("%040x", count-i), Subject: fmt.Sprintf("Perf change %05d", count-i), Author: "Preview Developer"}
		if i+1 < count {
			commits[i].Parents = []string{fmt.Sprintf("%040x", count-i-1)}
		}
		if i%20 == 0 && i+17 < count {
			commits[i].Parents = append(commits[i].Parents, fmt.Sprintf("%040x", count-i-17))
		}
	}
	return commits
}

var frameEdgeCount int

func BenchmarkHistoryFrameCached(b *testing.B) {
	for _, query := range []string{"", "Perf change"} {
		b.Run(fmt.Sprintf("query-%q", query), func(b *testing.B) {
			a := &app{historyQuery: query}
			a.data.history = performanceCommits(2000)
			a.historyModel()
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				model := a.historyModel()
				count := 0
				for row := 800; row < 825; row++ {
					model.Edges.eachAt(row, func(graphSpan) { count++ })
					frameEdgeCount = count + int(model.Width)
				}
			}
		})
	}
}

func BenchmarkHistoryModelCold(b *testing.B) {
	a := &app{historyQuery: "Perf change"}
	a.data.history = performanceCommits(2000)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		a.historyRevision++
		frameEdgeCount = len(a.historyModel().Rows)
	}
}

func BenchmarkLargeSnapshot(b *testing.B) {
	path := os.Getenv("SUPERGIT_BENCH_REPO")
	if path == "" {
		b.Skip("set SUPERGIT_BENCH_REPO to a create-history-demo.py workspace")
	}
	r, err := gitrepo.Open(context.Background(), path)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		s, err := readSnapshot(context.Background(), r, 2000)
		if err != nil {
			b.Fatal(err)
		}
		if len(s.warnings) != 0 {
			b.Fatal(s.warnings)
		}
		frameEdgeCount = len(s.history)
	}
}

func BenchmarkHistoryFrameLegacy(b *testing.B) {
	commits := performanceCommits(2000)
	for _, query := range []string{"", "Perf change"} {
		b.Run(fmt.Sprintf("query-%q", query), func(b *testing.B) {
			allRows := layoutGraph(commits)
			b.ReportAllocs()
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				filtered, _ := filterHistory(commits, query)
				rows := allRows
				if query != "" {
					rows = layoutGraph(filtered)
				}
				count := 0
				for row := 800; row < 825; row++ {
					width := 90
					for _, r := range rows {
						width = max(width, (r.Lane+1)*18+14)
					}
					for start, r := range rows {
						for _, edge := range r.Edges {
							if start <= row && row <= edge.Row {
								count++
							}
						}
					}
					frameEdgeCount = count + width
				}
			}
		})
	}
}
