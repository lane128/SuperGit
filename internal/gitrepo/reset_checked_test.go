package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedResetRejectsChangedHeadAndBranchWithoutWrites(t *testing.T) {
	for _, change := range []string{"head", "branch", "detached"} {
		for _, mode := range []string{"soft", "mixed", "hard"} {
			t.Run(change+"/"+mode, func(t *testing.T) {
				ctx := context.Background()
				dir := fixture(t)
				seed(t, dir)
				target := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
				write(t, dir, "base.txt", "second\n")
				git(t, dir, "commit", "-am", "second")
				repo := openTest(t, dir)
				expected, err := repo.Status(ctx)
				if err != nil {
					t.Fatal(err)
				}
				switch change {
				case "head":
					git(t, dir, "commit", "--allow-empty", "-m", "external commit")
				case "branch":
					git(t, dir, "switch", "-c", "other")
				case "detached":
					git(t, dir, "switch", "--detach")
				}
				write(t, dir, "base.txt", "staged edit\n")
				git(t, dir, "add", "--", "base.txt")
				write(t, dir, "base.txt", "working edit\n")
				beforeHead := git(t, dir, "rev-parse", "HEAD")
				beforeIndex := git(t, dir, "ls-files", "--stage", "-z")
				if err := repo.ResetToChecked(ctx, target, mode, expected); err == nil {
					t.Fatal("stale reset moved a changed HEAD or branch")
				}
				if git(t, dir, "rev-parse", "HEAD") != beforeHead || git(t, dir, "ls-files", "--stage", "-z") != beforeIndex {
					t.Fatal("refused reset modified HEAD or Index")
				}
				bytes, err := os.ReadFile(filepath.Join(dir, "base.txt"))
				if err != nil || string(bytes) != "working edit\n" {
					t.Fatal("refused reset modified working bytes")
				}
			})
		}
	}
}

func TestCheckedResetAcceptsOriginalHeadAndBranch(t *testing.T) {
	for _, mode := range []string{"soft", "mixed", "hard"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			target := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
			write(t, dir, "base.txt", "second\n")
			git(t, dir, "commit", "-am", "second")
			repo := openTest(t, dir)
			expected, err := repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.ResetToChecked(ctx, target, mode, expected); err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(git(t, dir, "rev-parse", "HEAD")) != target {
				t.Fatal("current reset did not move HEAD to selected SHA")
			}
		})
	}
}
