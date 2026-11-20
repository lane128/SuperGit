package gitrepo

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func paletteRefFixture(t *testing.T) (context.Context, *Repository, Remote, string) {
	t.Helper()
	dir := fixture(t)
	seed(t, dir)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, dir, "clone", "--bare", dir, bare)
	git(t, dir, "remote", "add", "origin", bare)
	ctx, repo := context.Background(), openTest(t, dir)
	remotes, err := repo.Remotes(ctx)
	if err != nil || len(remotes) != 1 {
		t.Fatalf("remote fixture: %+v, %v", remotes, err)
	}
	return ctx, repo, remotes[0], bare
}

func TestPaletteRefsDeleteOnlyAdvertisedTagOnPushEndpoint(t *testing.T) {
	ctx, repo, original, fetch := paletteRefFixture(t)
	push := filepath.Join(t.TempDir(), "push.git")
	git(t, repo.Root, "clone", "--bare", repo.Root, push)
	git(t, fetch, "tag", "fetch-only")
	git(t, repo.Root, "tag", "-a", "release", "-m", "annotation")
	git(t, repo.Root, "push", push, "refs/tags/release")
	git(t, repo.Root, "remote", "set-url", "--push", "origin", push)
	remotes, err := repo.Remotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AdvertisedRefs(ctx, original, true); !errors.Is(err, ErrRemoteTargetChanged) {
		t.Fatalf("read accepted retargeted remote: %v", err)
	}
	remote := remotes[0]
	tags, err := repo.AdvertisedRefs(ctx, remote, true)
	object := strings.TrimSpace(git(t, repo.Root, "rev-parse", "release"))
	if err != nil || len(tags) != 1 || tags[0].Name != "release" || tags[0].SHA != object {
		t.Fatalf("wrong advertised tag objects: %+v, %v", tags, err)
	}
	git(t, push, "update-ref", "refs/tags/release", "HEAD")
	if err := repo.DeleteRemoteTagChecked(ctx, remote, "release", object); err == nil {
		t.Fatal("stale lease removed a replacement tag")
	}
	git(t, push, "update-ref", "refs/tags/release", object)
	if err := repo.DeleteRemoteTagChecked(ctx, remote, "release", object); err != nil {
		t.Fatal(err)
	}
	if got := git(t, push, "tag", "-l"); got != "" || strings.TrimSpace(git(t, repo.Root, "rev-parse", "release")) != object || strings.TrimSpace(git(t, fetch, "tag", "-l")) != "fetch-only" {
		t.Fatalf("wrong tags after deleting the remote object: %q", got)
	}
}

func TestPaletteRefsUpstreamAndRemoteEditsCheckSelection(t *testing.T) {
	ctx, repo, remote, _ := paletteRefFixture(t)
	git(t, repo.Root, "fetch", "origin")
	sha := strings.TrimSpace(git(t, repo.Root, "rev-parse", "HEAD"))
	local := Branch{Name: "main", SHA: sha}
	target := Branch{Name: "origin/main", SHA: sha}
	if err := repo.ChangeUpstreamChecked(ctx, local, &target); err != nil {
		t.Fatal(err)
	}
	if err := repo.ChangeUpstreamChecked(ctx, local, nil); !errors.Is(err, ErrUpstreamTargetChanged) {
		t.Fatalf("removed a newly configured upstream: %v", err)
	}
	local.Upstream = "origin/main"
	if err := repo.ChangeUpstreamChecked(ctx, local, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.RenameRemoteChecked(ctx, remote, "upstream"); err != nil {
		t.Fatal(err)
	}
	git(t, repo.Root, "remote", "add", "origin", filepath.Join(t.TempDir(), "different.git"))
	if err := repo.RemoveRemoteChecked(ctx, remote); !errors.Is(err, ErrRemoteTargetChanged) {
		t.Fatalf("removed replacement remote: %v", err)
	}
	remotes, err := repo.Remotes(ctx)
	if err != nil || len(remotes) != 2 {
		t.Fatal(remotes, err)
	}
	if err := repo.RemoveRemoteChecked(ctx, remotes[1]); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(git(t, repo.Root, "remote")); got != "origin" {
		t.Fatalf("remaining remote = %s", got)
	}
}

func TestPaletteRefsPullChecksHeadAndAdvertisedTip(t *testing.T) {
	for _, mode := range []string{"ff-only", "merge", "rebase"} {
		t.Run(mode, func(t *testing.T) {
			ctx, repo, remote, bare := paletteRefFixture(t)
			status, err := repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			git(t, repo.Root, "switch", "-c", "source")
			git(t, repo.Root, "commit", "--allow-empty", "-m", "new tip")
			git(t, repo.Root, "push", bare, "source:refs/heads/incoming")
			sha := strings.TrimSpace(git(t, repo.Root, "rev-parse", "source"))
			branch := Branch{Name: "incoming", SHA: sha}
			if err := repo.PullBranchChecked(ctx, remote, branch, status, mode); !errors.Is(err, ErrIntegrationTargetChanged) {
				t.Fatalf("pull accepted changed current branch: %v", err)
			}
			git(t, repo.Root, "switch", "main")
			branch.SHA = status.Head
			if err := repo.PullBranchChecked(ctx, remote, branch, status, mode); !errors.Is(err, ErrIntegrationTargetChanged) {
				t.Fatalf("pull accepted stale remote tip: %v", err)
			}
			branch.SHA = sha
			if err := repo.PullBranchChecked(ctx, remote, branch, status, mode); err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(git(t, repo.Root, "rev-parse", "main")); got != sha {
				t.Fatalf("main = %s, want %s", got, sha)
			}
		})
	}
}

func TestPaletteRefsBranchAndTagWithSameName(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "branch", "release")
	git(t, dir, "tag", "-a", "release", "-m", "original", "refs/heads/main")
	repo := openTest(t, dir)
	branches, err := repo.Branches(ctx)
	if err != nil || len(branches) != 2 || branches[1].Name != "release" {
		t.Fatalf("local picker lost the literal branch name: %+v, %v", branches, err)
	}
	tags, err := repo.Tags(ctx)
	if err != nil || len(tags) != 1 {
		t.Fatal(tags, err)
	}
	tag := tags[0]
	if err := repo.RenameBranchChecked(ctx, "release", "renamed", branches[1].SHA); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "tag", "-f", "-a", "release", "-m", "different annotation", "refs/heads/main")
	if err := repo.CreateBranchFromTagChecked(ctx, "from-tag", tag); !errors.Is(err, ErrSwitchTargetChanged) {
		t.Fatalf("stale tag selection created a branch: %v", err)
	}
	git(t, dir, "update-ref", "refs/tags/release", tag.ObjectSHA)
	if err := repo.CreateBranchFromTagChecked(ctx, "from-tag", tag); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(git(t, dir, "rev-parse", "refs/heads/from-tag")); got != tag.TargetSHA {
		t.Fatalf("created branch at %s, expected %s", got, tag.TargetSHA)
	}
}
