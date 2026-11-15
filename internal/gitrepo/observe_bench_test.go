package gitrepo

import (
	"context"
	"fmt"
	"testing"
)

// Measures complete local probes, including child process wall time. It is a
// small/medium fixture baseline, not evidence for monorepo-scale performance.
func BenchmarkChangeStampWithSubmodule(b *testing.B) {
	dir := fixture(b)
	seed(b, dir)
	child := fixture(b)
	seed(b, child)
	git(b, dir, "-c", "protocol.file.allow=always", "submodule", "add", child, "child")
	git(b, dir, "commit", "-am", "child")
	r := openTest(b, dir)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.ChangeStamp(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkChangeStamp(b *testing.B) {
	for _, files := range []int{1, 1000} {
		b.Run(fmt.Sprintf("tracked-%d", files), func(b *testing.B) {
			dir := fixture(b)
			for i := 0; i < files; i++ {
				write(b, dir, fmt.Sprintf("files/%04d.txt", i), "tracked contents\n")
			}
			git(b, dir, "add", "--", "files")
			git(b, dir, "commit", "-m", "benchmark fixture")
			r := openTest(b, dir)
			ctx := context.Background()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := r.ChangeStamp(ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
