package gitrepo

import (
	"context"
	"os"
	"testing"
)

// Opt-in fixture supplied by scripts/create-history-demo.py. No repository
// writes are performed by these benchmarks; timings include Git subprocesses.
func BenchmarkLargeRepository(b *testing.B) {
	path := os.Getenv("SUPERGIT_BENCH_REPO")
	if path == "" {
		b.Skip("set SUPERGIT_BENCH_REPO to a create-history-demo.py workspace")
	}
	r, err := Open(context.Background(), path)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	for _, limit := range []struct {
		name  string
		count int
	}{{"History300", 300}, {"History2000", 2000}} {
		b.Run(limit.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				commits, e := r.History(ctx, limit.count)
				if e != nil {
					b.Fatal(e)
				}
				if len(commits) != limit.count {
					b.Fatalf("need at least %d commits", limit.count)
				}
			}
		})
	}
	b.Run("Status", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, e := r.Status(ctx); e != nil {
				b.Fatal(e)
			}
		}
	})
	b.Run("ChangeStamp", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, e := r.ChangeStamp(ctx); e != nil {
				b.Fatal(e)
			}
		}
	})
}
