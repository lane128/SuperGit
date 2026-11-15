package gitrepo

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestOutputBufferCannotBypassLimitThroughReaderFrom(t *testing.T) {
	b := &limitedBuffer{limit: 16}
	input := strings.Repeat("x", 32)
	// A reader without WriterTo makes io.Copy try the destination's ReaderFrom.
	n, err := io.Copy(b, io.LimitReader(strings.NewReader(input), int64(len(input))))
	if err != nil || n != int64(len(input)) {
		t.Fatal("stream was not drained", n, err)
	}
	if len(b.String()) != 16 || !b.truncated {
		t.Fatalf("output limit bypassed: len=%d truncated=%v", len(b.String()), b.truncated)
	}
}
func TestGitCommandEnforcesOutputLimit(t *testing.T) {
	dir := fixture(t)
	write(t, dir, "oversized.txt", strings.Repeat("x", outputLimit+1024))
	git(t, dir, "add", "--", "oversized.txt")
	git(t, dir, "commit", "-m", "oversized output fixture")
	out, err := run(context.Background(), dir, "", "show", "HEAD:oversized.txt")
	if err == nil {
		t.Error("Git command accepted oversized output")
	}
	if len(out) != outputLimit {
		t.Fatalf("retained %d output bytes, want %d", len(out), outputLimit)
	}
}
