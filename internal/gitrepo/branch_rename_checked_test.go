package gitrepo

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCheckedBranchRenameRejectsChangedTarget(t *testing.T) {
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
			refs, head := git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)"), git(t, dir, "symbolic-ref", "HEAD")
			if err := openTest(t, dir).RenameBranchChecked(context.Background(), "topic", "renamed", original); !errors.Is(err, ErrRenameTargetChanged) {
				t.Fatalf("stale rename needs changed-target refusal, got %v", err)
			}
			if git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)") != refs || git(t, dir, "symbolic-ref", "HEAD") != head {
				t.Fatal("refused rename changed branch refs or HEAD")
			}
		})
	}
}

func TestCheckedBranchRenameKeepsGitRenameSemantics(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	tip := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	repo := openTest(t, dir)
	git(t, dir, "branch", "occupied")
	refs := git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)")
	if err := repo.RenameBranchChecked(ctx, "main", "occupied", tip); err == nil {
		t.Fatal("rename replaced existing target name")
	}
	if git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)") != refs {
		t.Fatal("refused rename changed refs")
	}
	git(t, dir, "config", "branch.main.remote", "origin")
	git(t, dir, "config", "branch.main.merge", "refs/heads/main")
	if err := repo.RenameBranchChecked(ctx, "main", "renamed", tip); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(git(t, dir, "symbolic-ref", "HEAD")); got != "refs/heads/renamed" {
		t.Fatalf("current branch identity not renamed: %q", got)
	}
	if strings.TrimSpace(git(t, dir, "config", "branch.renamed.remote")) != "origin" || strings.TrimSpace(git(t, dir, "config", "branch.renamed.merge")) != "refs/heads/main" {
		t.Fatal("rename lost branch tracking configuration")
	}
}
