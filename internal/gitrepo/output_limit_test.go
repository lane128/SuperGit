package gitrepo

import (
	"context"
	"errors"
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

func TestScopedCommitPreviewExcludesReplacementDirectory(t *testing.T) {
	root := fixture(t)
	seed(t, root)
	git(t, root, "rm", "--", "base.txt")
	write(t, root, "base.txt/large.txt", strings.Repeat("x", outputLimit+1024))
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "replace a small file with a directory")
	sha := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	ctx, repo := context.Background(), openTest(t, root)
	if _, err := repo.DisplayCommitDiff(ctx, sha, false); !errors.Is(err, ErrDiffPreviewLimit) {
		t.Fatalf("combined patch should exceed the limit: %v", err)
	}
	file := ChangedFile{Status: "D", Path: "base.txt"}
	diffs, err := repo.DisplayCommitFileDiff(ctx, sha, file, false)
	if err != nil || len(diffs) != 1 || diffs[0].Path != file.Path || !strings.Contains(diffs[0].Diff.Text, "-original") {
		t.Fatalf("deleted file preview included its replacement directory: files=%d err=%v", len(diffs), err)
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
