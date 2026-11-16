package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clearIdentityEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL"} {
		old, present := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if present {
				_ = os.Setenv(key, old)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}

func TestAuthorIdentityUsesEffectiveGitConfiguration(t *testing.T) {
	clearIdentityEnvironment(t)
	dir := fixture(t)
	r := openTest(t, dir)
	ctx := context.Background()
	assertIdentity := func(want Identity) {
		t.Helper()
		got, err := r.AuthorIdentity(ctx)
		if err != nil || got != want {
			t.Fatalf("author = %+v, %v; want %+v", got, err, want)
		}
	}
	assertIdentity(Identity{Name: "Test Developer", Email: "test@example.invalid"})
	// A conditional include is resolved by Git, in the same repository as commit.
	included := filepath.Join(t.TempDir(), "included.gitconfig")
	if err := os.WriteFile(included, []byte("[user]\n name = Included Author\n email = included@example.invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	git(t, dir, "config", "includeIf.gitdir:"+filepath.ToSlash(realDir)+"/.git.path", included)
	assertIdentity(Identity{Name: "Included Author", Email: "included@example.invalid"})
	t.Setenv("GIT_AUTHOR_NAME", "Environment Author")
	t.Setenv("GIT_AUTHOR_EMAIL", "environment@example.invalid")
	assertIdentity(Identity{Name: "Environment Author", Email: "environment@example.invalid"})
	write(t, dir, "new.txt", "content\n")
	git(t, dir, "add", "--", "new.txt")
	message := "subject\n\nfirst paragraph\n\nsecond paragraph"
	if err := r.Commit(ctx, message); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "show", "-s", "--format=%an <%ae>%n%B", "HEAD"); !strings.HasPrefix(got, "Environment Author <environment@example.invalid>\n"+message) {
		t.Fatalf("actual commit differs: %q", got)
	}
}

func TestAuthorIdentityFinishingCherryPick(t *testing.T) {
	clearIdentityEnvironment(t)
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "checkout", "-b", "source")
	write(t, dir, "base.txt", "source change\n")
	git(t, dir, "-c", "user.name=Source Author", "-c", "user.email=source@example.invalid", "commit", "-am", "source change")
	source := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "checkout", "main")
	write(t, dir, "base.txt", "main change\n")
	git(t, dir, "commit", "-am", "main change")
	r := openTest(t, dir)
	ctx := context.Background()
	if _, err := run(ctx, dir, "", "cherry-pick", source); err == nil {
		t.Fatal("fixture did not stop on conflict")
	}
	preview, err := r.AuthorIdentity(ctx)
	want := Identity{Name: "Source Author", Email: "source@example.invalid"}
	if err != nil || preview != want {
		t.Fatalf("pending cherry-pick author = %+v, %v; want %+v", preview, err, want)
	}
	write(t, dir, "base.txt", "resolved\n")
	git(t, dir, "add", "--", "base.txt")
	if err := r.Commit(ctx, "finish cherry-pick"); err != nil {
		t.Fatal(err)
	}
	actual, err := r.HeadAuthorIdentity(ctx)
	if err != nil || actual != preview {
		t.Fatalf("actual author = %+v, %v; preview %+v", actual, err, preview)
	}
}

func TestAuthorIdentityUsesWorktreeConfiguration(t *testing.T) {
	clearIdentityEnvironment(t)
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "config", "extensions.worktreeConfig", "true")
	worktree := filepath.Join(t.TempDir(), "other checkout")
	git(t, dir, "worktree", "add", "--detach", worktree, "HEAD")
	git(t, worktree, "config", "--worktree", "user.name", "Worktree Author")
	git(t, worktree, "config", "--worktree", "user.email", "worktree@example.invalid")
	identity, err := openTest(t, worktree).AuthorIdentity(context.Background())
	if err != nil || identity != (Identity{Name: "Worktree Author", Email: "worktree@example.invalid"}) {
		t.Fatalf("worktree identity = %+v, %v", identity, err)
	}
}

func TestAuthorIdentityReportsMissingConfiguration(t *testing.T) {
	clearIdentityEnvironment(t)
	dir := fixture(t)
	git(t, dir, "config", "--unset", "user.email")
	git(t, dir, "config", "user.useConfigOnly", "true")
	if got, err := openTest(t, dir).AuthorIdentity(context.Background()); err == nil || got != (Identity{}) {
		t.Fatalf("missing author = %+v, %v", got, err)
	}
}

func TestAmendIdentityPreservesHeadAuthorAndRunsHooks(t *testing.T) {
	clearIdentityEnvironment(t)
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	t.Setenv("GIT_AUTHOR_NAME", "Next Author")
	t.Setenv("GIT_AUTHOR_EMAIL", "next@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Current Committer")
	t.Setenv("GIT_COMMITTER_EMAIL", "committer@example.invalid")
	before, err := r.HeadAuthorIdentity(ctx)
	if err != nil || before.Name != "Test Developer" {
		t.Fatalf("HEAD author = %+v, %v", before, err)
	}
	committer, err := r.CommitterIdentity(ctx)
	if err != nil || committer != (Identity{Name: "Current Committer", Email: "committer@example.invalid"}) {
		t.Fatalf("committer = %+v, %v", committer, err)
	}
	hook := filepath.Join(dir, ".git", "hooks", "commit-msg")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	oldHead := git(t, dir, "rev-parse", "HEAD")
	if err := r.AmendCommit(ctx, "rejected"); err == nil {
		t.Fatal("amend bypassed commit-msg hook")
	}
	if git(t, dir, "rev-parse", "HEAD") != oldHead {
		t.Fatal("failed hook changed HEAD")
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := r.AmendCommit(ctx, "amended\n\ncomplete body"); err != nil {
		t.Fatal(err)
	}
	after, err := r.HeadAuthorIdentity(ctx)
	if err != nil || after != before {
		t.Fatalf("amend author = %+v, %v; want %+v", after, err, before)
	}
	if got := git(t, dir, "show", "-s", "--format=%cn <%ce>", "HEAD"); strings.TrimSpace(got) != "Current Committer <committer@example.invalid>" {
		t.Fatalf("actual committer = %q", got)
	}
}
