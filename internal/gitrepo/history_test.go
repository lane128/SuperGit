package gitrepo

import (
	"context"
	"testing"
)

// A file's history reaches past a rename: the commits that changed it under
// its old path are part of the same story, and `--follow` finds them. The
// path from an older commit works too, even when that name is gone from HEAD,
// so opening a file's history from a commit before the rename still shows it.
func TestFileHistoryFollowsRenames(t *testing.T) {
	dir := fixture(t)
	seed(t, dir) // initial: base.txt "original\n"
	write(t, dir, "base.txt", "original\nsecond\n")
	git(t, dir, "commit", "-am", "edit base")
	git(t, dir, "mv", "base.txt", "renamed.txt")
	git(t, dir, "commit", "-m", "rename base")
	r := openTest(t, dir)

	subjects := func(path string) []string {
		t.Helper()
		commits, err := r.FileHistory(context.Background(), path, 50)
		if err != nil {
			t.Fatalf("FileHistory(%q): %v", path, err)
		}
		result := make([]string, 0, len(commits))
		for _, commit := range commits {
			result = append(result, commit.Subject)
		}
		return result
	}

	want := []string{"rename base", "edit base", "initial"}
	for _, path := range []string{"renamed.txt", "base.txt"} {
		got := subjects(path)
		if len(got) != len(want) {
			t.Fatalf("history of %q = %v; want %v", path, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("history of %q at %d = %q; want %q", path, i, got[i], want[i])
			}
		}
	}
}

// Before the first commit the branch has no HEAD to walk from, and `git log`
// exits with a fatal error. The file's history is empty, not broken.
func TestFileHistoryOfAnUnbornBranchIsEmpty(t *testing.T) {
	dir := fixture(t)
	write(t, dir, "new.txt", "one\n")
	r := openTest(t, dir)
	commits, err := r.FileHistory(context.Background(), "new.txt", 20)
	if err != nil {
		t.Fatalf("FileHistory on an empty repository: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("an empty repository returned %d commits", len(commits))
	}
}

// Deleting a file and adding it back is one path again: the history has to keep
// the deletion and what came before it, not start over at the re-add.
func TestFileHistoryShowsADeletedThenReAddedFile(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "base.txt", "original\nsecond\n")
	git(t, dir, "commit", "-am", "edit base")
	git(t, dir, "rm", "base.txt")
	git(t, dir, "commit", "-m", "delete base")
	write(t, dir, "base.txt", "re-added\n")
	git(t, dir, "add", "base.txt")
	git(t, dir, "commit", "-m", "re-add base")
	r := openTest(t, dir)

	commits, err := r.FileHistory(context.Background(), "base.txt", 50)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	want := []string{"re-add base", "delete base", "edit base", "initial"}
	if len(commits) != len(want) {
		t.Fatalf("history = %d commits; want %d", len(commits), len(want))
	}
	for i := range want {
		if commits[i].Subject != want[i] {
			t.Errorf("commit %d = %q; want %q", i, commits[i].Subject, want[i])
		}
	}
}

// The limit is clamped instead of trusted: zero, a negative number or an
// absurdly large one must still return a usable page, not nothing and not a
// query git chokes on.
func TestFileHistoryClampsTheLimit(t *testing.T) {
	dir := fixture(t)
	seed(t, dir) // initial
	write(t, dir, "base.txt", "original\nsecond\n")
	git(t, dir, "commit", "-am", "edit base")
	write(t, dir, "base.txt", "original\nsecond\nthird\n")
	git(t, dir, "commit", "-am", "edit base again")
	r := openTest(t, dir)

	for _, test := range []struct {
		limit int
		want  int
	}{
		{0, 3},
		{-5, 3},
		{9999, 3},
		{1, 1},
	} {
		commits, err := r.FileHistory(context.Background(), "base.txt", test.limit)
		if err != nil {
			t.Fatalf("limit %d: %v", test.limit, err)
		}
		if len(commits) != test.want {
			t.Errorf("limit %d returned %d commits; want %d", test.limit, len(commits), test.want)
		}
	}
}
