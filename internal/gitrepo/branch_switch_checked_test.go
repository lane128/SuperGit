package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedBranchSwitchRejectsMovedOrRecreatedTarget(t *testing.T) {
	for _, change := range []string{"moved", "recreated", "missing"} {
		t.Run(change, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			original := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
			git(t, dir, "branch", "topic")
			git(t, dir, "commit", "--allow-empty", "-m", "advance")
			switch change {
			case "moved":
				git(t, dir, "branch", "-f", "topic", "HEAD")
			case "recreated":
				git(t, dir, "branch", "-D", "topic")
				git(t, dir, "branch", "topic")
			case "missing":
				git(t, dir, "branch", "-D", "topic")
			}
			head, index, refs := git(t, dir, "symbolic-ref", "HEAD"), git(t, dir, "ls-files", "--stage"), git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)")
			if err := openTest(t, dir).SwitchBranchChecked(context.Background(), "topic", original); !errors.Is(err, ErrSwitchTargetChanged) {
				t.Fatalf("stale selection needs changed-target refusal, got %v", err)
			}
			if git(t, dir, "symbolic-ref", "HEAD") != head || git(t, dir, "ls-files", "--stage") != index || git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)") != refs {
				t.Fatal("refused switch changed HEAD, Index or refs")
			}
		})
	}
}

func TestCheckedBranchSwitchKeepsGitSafetyRules(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "branch", "topic")
	write(t, dir, "base.txt", "main change\n")
	git(t, dir, "commit", "-am", "advance main")
	tip := strings.TrimSpace(git(t, dir, "rev-parse", "topic"))
	repo := openTest(t, dir)

	// Conflicting local modification is rejected by git switch without losing bytes.
	write(t, dir, "base.txt", "conflicting local edit\n")
	if err := repo.SwitchBranchChecked(ctx, "topic", tip); !errors.Is(err, ErrDirtyWorkingTree) {
		t.Fatalf("checked switch ignored conflicting dirty edit: %v", err)
	}
	if contents, err := os.ReadFile(filepath.Join(dir, "base.txt")); err != nil || string(contents) != "conflicting local edit\n" {
		t.Fatal("dirty refusal changed bytes")
	}

	// Conflicting staged edit is also rejected safely.
	git(t, dir, "add", "--", "base.txt")
	if err := repo.SwitchBranchChecked(ctx, "topic", tip); !errors.Is(err, ErrDirtyWorkingTree) {
		t.Fatalf("checked switch ignored conflicting Index edits: %v", err)
	}
	git(t, dir, "restore", "--staged", "--worktree", "--", "base.txt")

	// Non-conflicting untracked file is carried across the branch switch.
	untracked := filepath.Join(dir, "untracked.txt")
	if err := os.WriteFile(untracked, []byte("keep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := repo.SwitchBranchChecked(ctx, "topic", tip); err != nil {
		t.Fatalf("non-conflicting dirty switch failed: %v", err)
	}
	if got := strings.TrimSpace(git(t, dir, "symbolic-ref", "HEAD")); got != "refs/heads/topic" {
		t.Fatalf("selected branch not checked out: %q", got)
	}
	if contents, err := os.ReadFile(untracked); err != nil || string(contents) != "keep\n" {
		t.Fatal("non-conflicting switch dropped untracked bytes")
	}

	git(t, dir, "switch", "main")
	git(t, dir, "worktree", "add", filepath.Join(t.TempDir(), "linked"), "topic")
	if err := repo.SwitchBranchChecked(ctx, "topic", tip); err == nil {
		t.Fatal("checked switch stole linked worktree branch")
	}
}
