package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCheckedWorktreeRemovalRejectsChangedTarget(t *testing.T) {
	for _, change := range []string{"head", "branch", "replaced", "missing"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			path := filepath.Join(t.TempDir(), "linked")
			git(t, dir, "worktree", "add", "-b", "topic", path, "HEAD")
			repo := openTest(t, dir)
			list, err := repo.Worktrees(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var expected Worktree
			for _, w := range list {
				if canonical(w.Path) == canonical(path) {
					expected = w
				}
			}
			if expected.HEAD == "" {
				t.Fatal("fixture missing linked worktree")
			}
			switch change {
			case "head":
				git(t, path, "commit", "--allow-empty", "-m", "advance")
			case "branch":
				git(t, path, "switch", "-c", "replacement")
			case "replaced":
				git(t, dir, "worktree", "remove", path)
				git(t, dir, "worktree", "add", "-b", "replacement", path, "HEAD")
			case "missing":
				git(t, dir, "worktree", "remove", path)
			}
			before, err := repo.Worktrees(ctx)
			if err != nil {
				t.Fatal(err)
			}
			refs := git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)")
			if err := repo.RemoveWorktreeChecked(ctx, expected); !errors.Is(err, ErrWorktreeTargetChanged) {
				t.Fatalf("stale removal needs changed-target refusal, got %v", err)
			}
			after, err := repo.Worktrees(ctx)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("refusal changed worktree registration: %v", err)
			}
			if git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)") != refs {
				t.Fatal("refusal changed branch refs")
			}
			if change != "missing" {
				if b, err := os.ReadFile(filepath.Join(path, "base.txt")); err != nil || string(b) != "original\n" {
					t.Fatal("old confirmation deleted replacement worktree bytes")
				}
			}
		})
	}
}

func TestCheckedWorktreeRemovalKeepsExistingSafetyRules(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	path := filepath.Join(t.TempDir(), "linked")
	git(t, dir, "worktree", "add", "-b", "topic", path, "HEAD")
	repo := openTest(t, dir)
	list, err := repo.Worktrees(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var expected Worktree
	for _, w := range list {
		if canonical(w.Path) == canonical(path) {
			expected = w
		}
	}
	write(t, path, "base.txt", "dirty\n")
	if err := repo.RemoveWorktreeChecked(ctx, expected); err == nil {
		t.Fatal("removed dirty worktree")
	}
	git(t, path, "add", "--", "base.txt")
	if err := repo.RemoveWorktreeChecked(ctx, expected); err == nil {
		t.Fatal("removed staged worktree")
	}
	git(t, path, "restore", "--staged", "--worktree", "--", "base.txt")
	git(t, dir, "worktree", "lock", path)
	if err := repo.RemoveWorktreeChecked(ctx, expected); err == nil {
		t.Fatal("removed locked worktree")
	}
	git(t, dir, "worktree", "unlock", path)
	if err := repo.RemoveWorktreeChecked(ctx, expected); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("original clean worktree survived: %v", err)
	}
	git(t, dir, "rev-parse", "--verify", "refs/heads/topic")
	list, err = repo.Worktrees(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("unexpected remaining trees: %+v %v", list, err)
	}
	if err := repo.RemoveWorktreeChecked(ctx, list[0]); err == nil {
		t.Fatal("removed currently open worktree")
	}
}
