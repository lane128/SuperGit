package gitrepo

import (
	"context"
	"strings"
	"testing"
)

func TestFileHistoryFollowsOnlyCommitsTouchingPath(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "other.txt", "other\n")
	git(t, dir, "add", "--", "other.txt")
	git(t, dir, "commit", "-qm", "touch other")
	write(t, dir, "base.txt", "changed\n")
	git(t, dir, "commit", "-qam", "touch base")
	git(t, dir, "mv", "base.txt", "renamed.txt")
	git(t, dir, "commit", "-qm", "rename base")
	commits, err := r.FileHistory(ctx, "renamed.txt", 50)
	if err != nil {
		t.Fatal(err)
	}
	var subjects []string
	for _, commit := range commits {
		subjects = append(subjects, commit.Subject)
	}
	joined := strings.Join(subjects, "|")
	for _, want := range []string{"rename base", "touch base", "initial"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
	if strings.Contains(joined, "touch other") {
		t.Fatalf("unrelated commit in %q", joined)
	}
	if _, err := r.FileHistory(ctx, "../escape", 10); err == nil {
		t.Fatal("path outside repository accepted")
	}
}
