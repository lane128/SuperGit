package gitrepo

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedRemoteDeleteKeepsSelectedTipAndDestination(t *testing.T) {
	for _, change := range []string{"tip", "tip-and-fetch", "remote-url", "push-url", "missing"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			git(t, dir, "branch", "topic")
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
			expected := strings.TrimSpace(git(t, dir, "rev-parse", "refs/remotes/origin/topic"))
			switch change {
			case "tip", "tip-and-fetch":
				git(t, dir, "commit", "--allow-empty", "-m", "later")
				git(t, dir, "push", "origin", "HEAD:refs/heads/topic")
				if change == "tip" {
					git(t, dir, "update-ref", "refs/remotes/origin/topic", expected)
				}
			case "remote-url":
				git(t, dir, "remote", "set-url", "origin", second)
			case "push-url":
				git(t, dir, "remote", "set-url", "--push", "origin", second)
			case "missing":
				git(t, first, "update-ref", "-d", "refs/heads/topic")
			}
			beforeFirst := git(t, first, "for-each-ref", "--format=%(refname) %(objectname)")
			beforeSecond := git(t, second, "for-each-ref", "--format=%(refname) %(objectname)")
			beforeLocal := git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)")
			if err := repo.DeleteRemoteBranchChecked(ctx, remotes[0], "topic", expected); err == nil {
				t.Fatal("old confirmation deleted changed remote branch or destination")
			}
			if git(t, first, "for-each-ref", "--format=%(refname) %(objectname)") != beforeFirst || git(t, second, "for-each-ref", "--format=%(refname) %(objectname)") != beforeSecond || git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)") != beforeLocal {
				t.Fatal("refused deletion changed remote or local refs")
			}
		})
	}
}

func TestCheckedRemoteDeleteRemovesOriginalTipAndTrackingRef(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "branch", "topic")
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, dir, "clone", "--bare", dir, bare)
	git(t, dir, "remote", "add", "origin", bare)
	git(t, dir, "fetch", "origin")
	repo := openTest(t, dir)
	remotes, err := repo.Remotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	expected := strings.TrimSpace(git(t, dir, "rev-parse", "refs/remotes/origin/topic"))
	head := git(t, dir, "rev-parse", "HEAD")
	if err := repo.DeleteRemoteBranchChecked(ctx, remotes[0], "topic", expected); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(git(t, bare, "for-each-ref", "--format=%(refname)", "refs/heads/topic")) != "" || strings.TrimSpace(git(t, dir, "for-each-ref", "--format=%(refname)", "refs/remotes/origin/topic")) != "" {
		t.Fatal("original remote or tracking branch remains")
	}
	if git(t, dir, "rev-parse", "HEAD") != head {
		t.Fatal("remote deletion changed local HEAD")
	}
}
