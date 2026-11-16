package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedPushRejectsChangedSourceOrRemote(t *testing.T) {
	for _, change := range []string{"source", "remote-url", "push-url", "missing-source"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			first, second := filepath.Join(t.TempDir(), "first.git"), filepath.Join(t.TempDir(), "second.git")
			git(t, dir, "clone", "--bare", dir, first)
			git(t, dir, "clone", "--bare", dir, second)
			git(t, dir, "branch", "topic")
			git(t, dir, "remote", "add", "origin", first)
			repo := openTest(t, dir)
			remotes, err := repo.Remotes(ctx)
			if err != nil {
				t.Fatal(err)
			}
			branch := Branch{Name: "topic", SHA: strings.TrimSpace(git(t, dir, "rev-parse", "refs/heads/topic"))}
			switch change {
			case "source":
				git(t, dir, "commit", "--allow-empty", "-m", "later")
				git(t, dir, "branch", "-f", "topic", "HEAD")
			case "missing-source":
				git(t, dir, "branch", "-D", "topic")
			case "remote-url":
				git(t, dir, "remote", "set-url", "origin", second)
			case "push-url":
				git(t, dir, "remote", "set-url", "--push", "origin", second)
			}
			beforeFirst := git(t, first, "for-each-ref", "--format=%(refname) %(objectname)")
			beforeSecond := git(t, second, "for-each-ref", "--format=%(refname) %(objectname)")
			if err := repo.PushChecked(ctx, remotes[0], branch, "published", true); err == nil {
				t.Fatal("old confirmation pushed changed branch or destination")
			}
			if git(t, first, "for-each-ref", "--format=%(refname) %(objectname)") != beforeFirst || git(t, second, "for-each-ref", "--format=%(refname) %(objectname)") != beforeSecond {
				t.Fatal("refused push changed remote refs")
			}
			if got := git(t, dir, "for-each-ref", "--format=%(upstream)", "refs/heads/topic"); strings.TrimSpace(got) != "" {
				t.Fatalf("refused push set upstream: %s", got)
			}
		})
	}
}

func TestCheckedPushPublishesOriginalSourceAndSetsUpstreamWithoutCheckout(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, dir, "clone", "--bare", dir, bare)
	git(t, dir, "branch", "topic")
	git(t, dir, "remote", "add", "origin", bare)
	repo := openTest(t, dir)
	remotes, err := repo.Remotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	head := git(t, dir, "rev-parse", "HEAD")
	branch := Branch{Name: "topic", SHA: strings.TrimSpace(head)}
	if err := repo.PushChecked(ctx, remotes[0], branch, "published", true); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(git(t, bare, "rev-parse", "refs/heads/published")) != branch.SHA || git(t, dir, "rev-parse", "HEAD") != head || strings.TrimSpace(git(t, dir, "symbolic-ref", "--short", "HEAD")) != "main" {
		t.Fatal("push changed checkout or published wrong source")
	}
	if got := strings.TrimSpace(git(t, dir, "rev-parse", "--abbrev-ref", "topic@{upstream}")); got != "origin/published" {
		t.Fatalf("upstream = %s", got)
	}
	// A normal push must retain Git's non-fast-forward protection.
	git(t, dir, "commit", "--allow-empty", "-m", "remote advance")
	git(t, dir, "push", "origin", "HEAD:refs/heads/published")
	newer := git(t, bare, "rev-parse", "refs/heads/published")
	if err := repo.PushChecked(ctx, remotes[0], branch, "published", false); err == nil {
		t.Fatal("normal push rewound remote branch")
	}
	if git(t, bare, "rev-parse", "refs/heads/published") != newer {
		t.Fatal("non-fast-forward refusal changed remote")
	}
}

func TestCheckedPushSetsUpstreamWithCustomFetchMapping(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "branch", "topic")
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, dir, "init", "--bare", bare)
	git(t, dir, "remote", "add", "origin", bare)
	git(t, dir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/custom/*")
	repo := openTest(t, dir)
	remotes, err := repo.Remotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	if err := repo.PushChecked(ctx, remotes[0], Branch{Name: "topic", SHA: sha}, "published", true); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(git(t, dir, "rev-parse", "--symbolic-full-name", "topic@{upstream}")); got != "refs/remotes/custom/published" {
		t.Fatalf("custom upstream: %s", got)
	}
}

func TestCheckedPushPinsObjectThroughPrePushHook(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	original := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "branch", "topic")
	git(t, dir, "commit", "--allow-empty", "-m", "later source")
	later := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, dir, "init", "--bare", bare)
	git(t, dir, "remote", "add", "origin", bare)
	hook := "#!/bin/sh\ngit update-ref refs/heads/topic " + later + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".git", "hooks", "pre-push"), []byte(hook), 0o700); err != nil {
		t.Fatal(err)
	}
	repo := openTest(t, dir)
	remotes, err := repo.Remotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.PushChecked(ctx, remotes[0], Branch{Name: "topic", SHA: original}, "topic", false); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(git(t, bare, "rev-parse", "refs/heads/topic")) != original {
		t.Fatal("hook source change replaced authorized push object")
	}
}
