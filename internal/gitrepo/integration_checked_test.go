package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedIntegrationRejectsChangedCurrentHeadOrBranch(t *testing.T) {
	for _, kind := range []string{"merge", "rebase", "cherry-pick", "revert"} {
		for _, change := range []string{"head", "branch", "detached"} {
			t.Run(kind+"/"+change, func(t *testing.T) {
				ctx := context.Background()
				dir := fixture(t)
				seed(t, dir)
				git(t, dir, "switch", "-c", "topic")
				write(t, dir, "topic.txt", "selected change\n")
				git(t, dir, "add", "--", "topic.txt")
				git(t, dir, "commit", "-m", "topic")
				selected := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
				git(t, dir, "switch", "main")
				if kind == "revert" {
					git(t, dir, "merge", "--ff-only", "topic")
				}
				repo := openTest(t, dir)
				expected, err := repo.Status(ctx)
				if err != nil {
					t.Fatal(err)
				}
				switch change {
				case "head":
					git(t, dir, "commit", "--allow-empty", "-m", "advance")
				case "branch":
					git(t, dir, "switch", "-c", "other")
				case "detached":
					git(t, dir, "switch", "--detach")
				}
				refs, head, index := git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)"), git(t, dir, "rev-parse", "HEAD"), git(t, dir, "ls-files", "--stage")
				before, err := repo.Status(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if err := repo.IntegrateChecked(ctx, kind, selected, expected); !errors.Is(err, ErrIntegrationTargetChanged) {
					t.Fatalf("stale current side needs refusal, got %v", err)
				}
				after, err := repo.Status(ctx)
				if err != nil || before.Head != after.Head || before.Branch != after.Branch {
					t.Fatalf("refusal changed current target: %+v %+v %v", before, after, err)
				}
				if git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)") != refs || git(t, dir, "rev-parse", "HEAD") != head || git(t, dir, "ls-files", "--stage") != index {
					t.Fatal("refusal changed refs, HEAD or Index")
				}
				if b, err := os.ReadFile(filepath.Join(dir, "base.txt")); err != nil || string(b) != "original\n" {
					t.Fatal("refusal changed working bytes")
				}
			})
		}
	}
}

func TestCheckedIntegrationAcceptsOriginalCurrentTarget(t *testing.T) {
	for _, kind := range []string{"merge", "rebase", "cherry-pick", "revert"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			git(t, dir, "switch", "-c", "topic")
			write(t, dir, "topic.txt", "selected change\n")
			git(t, dir, "add", "--", "topic.txt")
			git(t, dir, "commit", "-m", "topic")
			selected := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
			git(t, dir, "switch", "main")
			if kind == "revert" {
				git(t, dir, "merge", "--ff-only", "topic")
			}
			repo := openTest(t, dir)
			expected, err := repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.IntegrateChecked(ctx, kind, selected, expected); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(dir, "topic.txt")); kind == "revert" {
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("revert did not remove selected addition: %v", err)
				}
			} else if err != nil {
				t.Fatal("integration did not apply selected change")
			}
		})
	}
}
