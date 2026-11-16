package gitrepo

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedBranchDeleteRejectsMovedOrRecreatedTarget(t *testing.T) {
	for _, change := range []string{"moved", "recreated", "missing"} {
		for _, force := range []bool{false, true} {
			name := change + "/normal"
			if force {
				name = change + "/force"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				dir := fixture(t)
				seed(t, dir)
				old := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
				git(t, dir, "branch", "topic")
				git(t, dir, "commit", "--allow-empty", "-m", "advance")
				current := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
				switch change {
				case "moved":
					git(t, dir, "branch", "-f", "topic", current)
				case "recreated":
					git(t, dir, "branch", "-D", "topic")
					git(t, dir, "branch", "topic", current)
				case "missing":
					git(t, dir, "branch", "-D", "topic")
				}
				before := git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads")
				repo := openTest(t, dir)
				if err := repo.DeleteBranchChecked(ctx, "topic", old, force); !errors.Is(err, ErrBranchTargetChanged) {
					t.Fatalf("stale delete requires changed-target refusal, got %v", err)
				}
				if git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads") != before {
					t.Fatal("refused delete changed branch refs")
				}
			})
		}
	}
}

func TestCheckedBranchDeleteKeepsGitSafetyRules(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "switch", "-c", "topic")
	git(t, dir, "commit", "--allow-empty", "-m", "unmerged")
	tip := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	repo := openTest(t, dir)
	if err := repo.DeleteBranchChecked(ctx, "topic", tip, true); err == nil {
		t.Fatal("checked force delete removed current branch")
	}
	git(t, dir, "switch", "main")
	if err := repo.DeleteBranchChecked(ctx, "topic", tip, false); err == nil {
		t.Fatal("normal checked delete removed unmerged branch")
	}
	if err := repo.DeleteBranchChecked(ctx, "topic", tip, true); err != nil {
		t.Fatal(err)
	}
	main := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "branch", "merged")
	if err := repo.DeleteBranchChecked(ctx, "merged", main, false); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "worktree", "add", "-b", "linked", filepath.Join(t.TempDir(), "linked"), "main")
	if err := repo.DeleteBranchChecked(ctx, "linked", main, true); err == nil {
		t.Fatal("checked force delete removed branch used by another worktree")
	}
	if strings.TrimSpace(git(t, dir, "rev-parse", "refs/heads/linked")) != main {
		t.Fatal("linked worktree branch changed")
	}
}
