package gitrepo

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedForcePushKeepsSourceDestinationAndExplicitLease(t *testing.T) {
	for _, change := range []string{"source", "remote-url", "push-url", "remote-tip", "remote-tip-and-fetch"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			base := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
			git(t, dir, "branch", "topic")
			git(t, dir, "commit", "--allow-empty", "-m", "remote baseline")
			lease := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
			first, second := filepath.Join(t.TempDir(), "first.git"), filepath.Join(t.TempDir(), "second.git")
			git(t, dir, "clone", "--bare", dir, first)
			git(t, dir, "clone", "--bare", dir, second)
			git(t, dir, "remote", "add", "origin", first)
			git(t, dir, "fetch", "origin")
			repo := openTest(t, dir)
			remotes, err := repo.Remotes(ctx)
			if err != nil {
				t.Fatal(err)
			}
			source := Branch{Name: "topic", SHA: base}
			switch change {
			case "source":
				git(t, dir, "branch", "-f", "topic", "HEAD")
			case "remote-url":
				git(t, dir, "remote", "set-url", "origin", second)
			case "push-url":
				git(t, dir, "remote", "set-url", "--push", "origin", second)
			case "remote-tip", "remote-tip-and-fetch":
				git(t, dir, "commit", "--allow-empty", "-m", "remote later")
				git(t, dir, "push", first, "HEAD:refs/heads/main")
				if change == "remote-tip-and-fetch" {
					git(t, dir, "fetch", "origin")
				}
			}
			firstRefs := git(t, first, "for-each-ref", "--format=%(refname) %(objectname)")
			secondRefs := git(t, second, "for-each-ref", "--format=%(refname) %(objectname)")
			if err := repo.PushForceChecked(ctx, remotes[0], source, "main", lease); err == nil {
				t.Fatal("old force confirmation published changed source/destination or overwrote newer remote tip")
			}
			if git(t, first, "for-each-ref", "--format=%(refname) %(objectname)") != firstRefs || git(t, second, "for-each-ref", "--format=%(refname) %(objectname)") != secondRefs {
				t.Fatal("refused force push changed a remote")
			}
		})
	}
}

func TestCheckedForcePushAcceptsOriginalLeaseAndMissingDestination(t *testing.T) {
	for _, missing := range []bool{false, true} {
		name := "existing"
		if missing {
			name = "missing"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			base := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
			git(t, dir, "branch", "topic")
			git(t, dir, "commit", "--allow-empty", "-m", "remote baseline")
			bare := filepath.Join(t.TempDir(), "remote.git")
			git(t, dir, "clone", "--bare", dir, bare)
			git(t, dir, "remote", "add", "origin", bare)
			git(t, dir, "fetch", "origin")
			destination, lease := "main", strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
			if missing {
				destination, lease = "new-destination", ""
			}
			repo := openTest(t, dir)
			remotes, err := repo.Remotes(ctx)
			if err != nil {
				t.Fatal(err)
			}
			head := git(t, dir, "rev-parse", "HEAD")
			if missing {
				if err := repo.PushForceChecked(ctx, remotes[0], Branch{Name: "topic", SHA: base}, "main", ""); err == nil {
					t.Fatal("absent-target lease overwrote existing remote branch")
				}
				if git(t, bare, "rev-parse", "refs/heads/main") != head {
					t.Fatal("absent lease changed existing branch")
				}
			}
			if err := repo.PushForceChecked(ctx, remotes[0], Branch{Name: "topic", SHA: base}, destination, lease); err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(git(t, bare, "rev-parse", "refs/heads/"+destination)) != base || git(t, dir, "rev-parse", "HEAD") != head {
				t.Fatal("force push lost selected object or changed checkout")
			}
		})
	}
}
