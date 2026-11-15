package gitrepo

import (
	"context"
	"strings"
	"testing"
)

func TestBlameAttributesCommittedAndUncommittedLines(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	head := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	write(t, dir, "base.txt", "original\nlocal edit\n")
	lines, err := r.Blame(ctx, "base.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("blame lines=%d", len(lines))
	}
	if lines[0].SHA != head || lines[0].Author == "" || lines[0].Date == "" || lines[0].Line != 1 {
		t.Fatalf("committed line not attributed: %#v", lines[0])
	}
	if lines[1].Text != "local edit" || lines[1].Line != 2 {
		t.Fatalf("uncommitted line: %#v", lines[1])
	}
	if lines[1].SHA != strings.Repeat("0", 40) {
		t.Fatalf("uncommitted SHA=%q", lines[1].SHA)
	}
	if _, err := r.Blame(ctx, "../escape.txt", false); err == nil {
		t.Fatal("path outside repository accepted")
	}
}

func TestBlameIgnoreWhitespaceSkipsReformatCommit(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "ws.txt", "base line\n")
	git(t, dir, "add", "--", "ws.txt")
	git(t, dir, "commit", "-qm", "add ws")
	added := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	write(t, dir, "ws.txt", "  base line\n")
	git(t, dir, "add", "--", "ws.txt")
	git(t, dir, "commit", "-qm", "reformat only")
	reformat := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))

	plain, err := r.Blame(ctx, "ws.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != 1 || plain[0].SHA != reformat {
		t.Fatalf("plain blame should credit the reformat: %#v", plain)
	}
	ignored, err := r.Blame(ctx, "ws.txt", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(ignored) != 1 || ignored[0].SHA != added {
		t.Fatalf("blame -w should skip whitespace-only changes: %#v", ignored)
	}
}

// The commit view's Blame means the version that commit held, not the file in
// the worktree: another commit may have touched the line since, and the
// worktree may hold uncommitted edits on top of that.
func TestBlameAtReadsTheCommitVersion(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "at.txt", "first\n")
	git(t, dir, "add", "--", "at.txt")
	git(t, dir, "commit", "-qm", "add at")
	first := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	write(t, dir, "at.txt", "second\n")
	git(t, dir, "add", "--", "at.txt")
	git(t, dir, "commit", "-qm", "rewrite at")
	write(t, dir, "at.txt", "uncommitted\n")

	atFirst, err := r.BlameAt(ctx, first, "at.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(atFirst) != 1 || atFirst[0].Text != "first" || atFirst[0].SHA != first {
		t.Fatalf("blame at the first commit read %#v; want its own line", atFirst)
	}
	worktree, err := r.Blame(ctx, "at.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(worktree) != 1 || worktree[0].Text != "uncommitted" {
		t.Fatalf("worktree blame read %#v; want the uncommitted line", worktree)
	}
	// A path the revision never held is an error, not an empty walk.
	if _, err := r.BlameAt(ctx, first, "later.txt", false); err == nil {
		t.Fatal("blame at a revision without the path succeeded")
	}
}
