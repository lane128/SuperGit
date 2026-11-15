package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A stage-all over thousands of files must not spend a process per file: the
// batched call stages every path, and paths that vanished since the status
// read do not keep the others out of the index.
func TestStagePathsStagesEveryPathAndKeepsGoingAfterAMissingOne(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		write(t, dir, name, name+"\n")
	}
	r := openTest(t, dir)
	ctx := context.Background()
	err := r.StagePaths(ctx, []string{"a.txt", "missing.txt", "b.txt", "c.txt"})
	if err == nil {
		t.Fatal("staging a missing path reported no error")
	}
	staged := strings.Split(git(t, dir, "diff", "--cached", "--name-only"), "\n")
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if !slices.Contains(staged, name) {
			t.Errorf("%s is not staged; staged files: %q", name, staged)
		}
	}
}

// Unstaging a batch takes both sides of a staged rename, and works before the
// first commit, where there is no HEAD to restore from.
func TestUnstagePathsHandlesRenamesAndAnUnbornHead(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "mv", "base.txt", "renamed.txt")
	r := openTest(t, dir)
	ctx := context.Background()
	// The rename sits in the worktree only until it is staged.
	if err := r.StagePaths(ctx, []string{"renamed.txt"}); err != nil {
		t.Fatal(err)
	}
	if err := r.UnstagePaths(ctx, []string{"base.txt", "renamed.txt"}); err != nil {
		t.Fatal(err)
	}
	if out := git(t, dir, "diff", "--cached", "--name-only"); out != "" {
		t.Fatalf("index still holds %q", out)
	}
	if got := git(t, dir, "status", "--porcelain"); got == "" {
		t.Fatal("unstaging the rename lost the worktree change")
	}

	unborn := fixture(t)
	write(t, unborn, "new.txt", "keep me\n")
	u := openTest(t, unborn)
	if err := u.StagePaths(ctx, []string{"new.txt"}); err != nil {
		t.Fatal(err)
	}
	if err := u.UnstagePaths(ctx, []string{"new.txt"}); err != nil {
		t.Fatal(err)
	}
	if out := git(t, unborn, "ls-files"); out != "" {
		t.Fatalf("index not empty: %q", out)
	}
	if b, err := os.ReadFile(filepath.Join(unborn, "new.txt")); err != nil || string(b) != "keep me\n" {
		t.Fatalf("worktree file lost: %q %v", b, err)
	}
}
