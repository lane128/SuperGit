package gitrepo

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedTagPushRefusesChangedObjectOrRemote(t *testing.T) {
	for _, change := range []string{"annotation", "tip", "missing", "remote-url", "push-url", "missing-remote"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			first, second := filepath.Join(t.TempDir(), "first.git"), filepath.Join(t.TempDir(), "second.git")
			git(t, dir, "clone", "--bare", dir, first)
			git(t, dir, "clone", "--bare", dir, second)
			git(t, dir, "remote", "add", "origin", first)
			if change == "tip" {
				git(t, dir, "tag", "release")
			} else {
				git(t, dir, "tag", "-a", "--no-sign", "-m", "original", "release")
			}
			repo := openTest(t, dir)
			remotes, err := repo.Remotes(ctx)
			if err != nil || len(remotes) != 1 {
				t.Fatalf("fixture remote: %v", err)
			}
			tags, err := repo.Tags(ctx)
			if err != nil || len(tags) != 1 {
				t.Fatalf("fixture tag: %v", err)
			}
			switch change {
			case "annotation":
				git(t, dir, "tag", "-f", "-a", "--no-sign", "-m", "replacement", "release")
			case "tip":
				git(t, dir, "commit", "--allow-empty", "-m", "advance")
				git(t, dir, "tag", "-f", "release")
			case "missing":
				git(t, dir, "tag", "-d", "release")
			case "remote-url":
				git(t, dir, "remote", "set-url", "origin", second)
			case "push-url":
				git(t, dir, "remote", "set-url", "--push", "origin", second)
			case "missing-remote":
				git(t, dir, "remote", "remove", "origin")
			}
			firstRefs, secondRefs := git(t, first, "for-each-ref", "--format=%(refname) %(objectname)"), git(t, second, "for-each-ref", "--format=%(refname) %(objectname)")
			want := ErrTagPushTargetChanged
			if change == "remote-url" || change == "push-url" || change == "missing-remote" {
				want = ErrRemoteTargetChanged
			}
			if err := repo.PushTagChecked(ctx, remotes[0], tags[0]); !errors.Is(err, want) {
				t.Fatalf("old confirmation should refuse changed target: %v", err)
			}
			if git(t, first, "for-each-ref", "--format=%(refname) %(objectname)") != firstRefs || git(t, second, "for-each-ref", "--format=%(refname) %(objectname)") != secondRefs {
				t.Fatal("refused tag push changed a remote")
			}
		})
	}
}

func TestCheckedTagPushPreservesOriginalObjectAndNoForce(t *testing.T) {
	for _, annotated := range []bool{false, true} {
		name := "lightweight"
		if annotated {
			name = "annotated"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			bare := filepath.Join(t.TempDir(), "remote.git")
			git(t, dir, "clone", "--bare", dir, bare)
			git(t, dir, "remote", "add", "origin", bare)
			if annotated {
				git(t, dir, "tag", "-a", "--no-sign", "-m", "original", "release")
			} else {
				git(t, dir, "tag", "release")
			}
			repo := openTest(t, dir)
			remotes, err := repo.Remotes(ctx)
			if err != nil {
				t.Fatal(err)
			}
			tags, err := repo.Tags(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.PushTagChecked(ctx, remotes[0], tags[0]); err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(git(t, bare, "rev-parse", "refs/tags/release")); got != tags[0].ObjectSHA {
				t.Fatal("tag push lost original object")
			}
			if annotated {
				git(t, dir, "tag", "-f", "-a", "--no-sign", "-m", "new local annotation", "release")
			} else {
				git(t, dir, "commit", "--allow-empty", "-m", "advance")
				git(t, dir, "tag", "-f", "release")
			}
			newer, err := repo.Tags(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.PushTagChecked(ctx, remotes[0], newer[0]); err == nil {
				t.Fatal("tag push overwrote existing remote tag")
			}
			if got := strings.TrimSpace(git(t, bare, "rev-parse", "refs/tags/release")); got != tags[0].ObjectSHA {
				t.Fatal("failed no-force push changed remote tag")
			}
		})
	}
}
