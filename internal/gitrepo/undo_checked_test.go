package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedUndoRejectsChangedHeadOrBranch(t *testing.T) {
	for _, change := range []string{"head", "branch", "detached"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			git(t, dir, "commit", "--allow-empty", "-m", "selected")
			repo := openTest(t, dir)
			expected, err := repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "head":
				git(t, dir, "commit", "--allow-empty", "-m", "newer")
			case "branch":
				git(t, dir, "switch", "-c", "other")
			case "detached":
				git(t, dir, "switch", "--detach")
			}
			// Undo must also leave existing staged and unstaged edits alone on refusal.
			write(t, dir, "base.txt", "staged\n")
			git(t, dir, "add", "--", "base.txt")
			write(t, dir, "base.txt", "working\n")
			head, index, refs := git(t, dir, "rev-parse", "HEAD"), git(t, dir, "ls-files", "--stage"), git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)")
			if err := repo.UndoCommitChecked(ctx, expected); !errors.Is(err, ErrUndoTargetChanged) {
				t.Fatalf("stale Undo requires changed-target refusal, got %v", err)
			}
			if git(t, dir, "rev-parse", "HEAD") != head || git(t, dir, "ls-files", "--stage") != index || git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)") != refs {
				t.Fatal("refused Undo changed HEAD, Index or refs")
			}
			if b, err := os.ReadFile(filepath.Join(dir, "base.txt")); err != nil || string(b) != "working\n" {
				t.Fatal("refused Undo changed working bytes")
			}
		})
	}
}

func TestCheckedUndoPreservesIndexAndWorkingTree(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	parent := git(t, dir, "rev-parse", "HEAD")
	write(t, dir, "committed.txt", "keep committed changes\n")
	git(t, dir, "add", "--", "committed.txt")
	git(t, dir, "commit", "-m", "selected")
	repo := openTest(t, dir)
	expected, err := repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, "base.txt", "staged\n")
	git(t, dir, "add", "--", "base.txt")
	write(t, dir, "base.txt", "working\n")
	index := git(t, dir, "ls-files", "--stage")
	if err := repo.UndoCommitChecked(ctx, expected); err != nil {
		t.Fatal(err)
	}
	if git(t, dir, "rev-parse", "HEAD") != parent || git(t, dir, "ls-files", "--stage") != index {
		t.Fatal("Undo failed to move to parent while preserving Index")
	}
	if b, err := os.ReadFile(filepath.Join(dir, "base.txt")); err != nil || string(b) != "working\n" {
		t.Fatal("Undo changed working bytes")
	}
	expected, err = repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UndoCommitChecked(ctx, expected); err == nil {
		t.Fatal("Undo accepted root commit")
	}
}

func TestCheckedUndoRefusesPendingOperation(t *testing.T) {
	for _, kind := range []string{"merge", "cherry-pick"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			dir := conflictFixture(t)
			repo := openTest(t, dir)
			if err := repo.Integrate(ctx, kind, "feature"); err == nil {
				t.Fatal("fixture lacks conflict")
			}
			write(t, dir, "base.txt", "resolved\n")
			git(t, dir, "add", "--", "base.txt")
			expected, err := repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			before, err := repo.Operation(ctx)
			if err != nil {
				t.Fatal(err)
			}
			index := git(t, dir, "ls-files", "--stage")
			if err := repo.UndoCommitChecked(ctx, expected); err == nil {
				t.Fatal("Undo changed HEAD during pending operation")
			}
			after, err := repo.Operation(ctx)
			if err != nil || before != after || git(t, dir, "ls-files", "--stage") != index {
				t.Fatal("refused Undo changed pending operation or Index")
			}
		})
	}
}
