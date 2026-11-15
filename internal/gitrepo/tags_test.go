package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTagUsesExactCommitAndPreservesDirtyState(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	old := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "commit", "--allow-empty", "-m", "later")
	head := git(t, dir, "rev-parse", "HEAD")
	write(t, dir, "base.txt", "index version\n")
	git(t, dir, "add", "--", "base.txt")
	write(t, dir, "base.txt", "worktree version\n")
	index := git(t, dir, "ls-files", "--stage")
	for _, name := range []string{"release/v1.2.3", "版本/候选"} {
		if err := r.CreateTag(ctx, name, old); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(git(t, dir, "rev-parse", "refs/tags/"+name)) != old || strings.TrimSpace(git(t, dir, "cat-file", "-t", "refs/tags/"+name)) != "commit" {
			t.Fatal("tag did not directly reference exact selected commit")
		}
	}
	if git(t, dir, "rev-parse", "HEAD") != head || git(t, dir, "ls-files", "--stage") != index {
		t.Fatal("tag creation altered HEAD/Index")
	}
	b, err := os.ReadFile(filepath.Join(dir, "base.txt"))
	if err != nil || string(b) != "worktree version\n" {
		t.Fatal("worktree altered", err)
	}
}

func TestCreateTagRejectsInvalidTargetNamesAndCancellation(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	for _, name := range []string{"", "--force", "-a", "bad name", "bad..name", "bad@{1}", "a.lock", "a\nname", "a\x00name", "/leading", "trailing/"} {
		if err := r.CreateTag(ctx, name, sha); err == nil {
			t.Fatalf("invalid name %q accepted", name)
		}
	}
	for _, target := range []string{"", "HEAD", "--help", strings.Repeat("f", 40), strings.TrimSpace(git(t, dir, "rev-parse", "HEAD^{tree}")), strings.TrimSpace(git(t, dir, "rev-parse", "HEAD:base.txt"))} {
		if err := r.CreateTag(ctx, "invalid-target", target); err == nil {
			t.Fatalf("invalid target %q accepted", target)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := r.CreateTag(canceled, "canceled", sha); err == nil {
		t.Fatal("canceled write succeeded")
	}
	if git(t, dir, "tag", "--list") != "" {
		t.Fatal("rejected operation created ref")
	}
}

func TestCreateTagNeverOverwritesExistingTagsOrSymbolicRefs(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "tag", "existing", sha)
	git(t, dir, "tag", "-a", "annotated", "-m", "existing annotation", sha)
	annotated := git(t, dir, "rev-parse", "refs/tags/annotated")
	git(t, dir, "symbolic-ref", "refs/tags/symbolic", "refs/heads/main")
	git(t, dir, "symbolic-ref", "refs/tags/dangling", "refs/heads/missing")
	git(t, dir, "commit", "--allow-empty", "-m", "later")
	later := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	for _, name := range []string{"existing", "annotated", "symbolic", "dangling"} {
		if err := r.CreateTag(ctx, name, later); err == nil {
			t.Fatalf("overwrote %s", name)
		}
	}
	if strings.TrimSpace(git(t, dir, "rev-parse", "refs/tags/existing")) != sha || git(t, dir, "rev-parse", "refs/tags/annotated") != annotated || strings.TrimSpace(git(t, dir, "symbolic-ref", "refs/tags/symbolic")) != "refs/heads/main" || strings.TrimSpace(git(t, dir, "symbolic-ref", "refs/tags/dangling")) != "refs/heads/missing" {
		t.Fatal("existing reference altered")
	}
}

// Characterize system Git's reference hook boundary: a rejecting prepared
// transaction must surface its error and leave the proposed tag absent.
func TestCreateTagRespectsReferenceTransactionHookRejection(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	hook := "#!/bin/sh\nif [ \"$1\" = prepared ]; then\n  printf 'TAG REJECTED BY TEST HOOK\\n' >&2\n  exit 1\nfi\n"
	if err := os.WriteFile(filepath.Join(dir, ".git", "hooks", "reference-transaction"), []byte(hook), 0700); err != nil {
		t.Fatal(err)
	}
	err := r.CreateTag(context.Background(), "rejected-by-hook", sha)
	if err == nil || !strings.Contains(err.Error(), "TAG REJECTED BY TEST HOOK") {
		t.Fatalf("reference hook rejection was lost: %v", err)
	}
	if git(t, dir, "tag", "--list", "rejected-by-hook") != "" || strings.TrimSpace(git(t, dir, "rev-parse", "HEAD")) != sha {
		t.Fatal("rejected transaction wrote tag or moved HEAD")
	}
}

func TestCreateTagRemainsLightweightWithSigningConfigured(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "config", "tag.gpgSign", "true")
	git(t, dir, "config", "gpg.program", "/nonexistent-test-signer")
	if err := r.CreateTag(context.Background(), "lightweight", sha); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(git(t, dir, "cat-file", "-t", "refs/tags/lightweight")) != "commit" {
		t.Fatal("configured signing changed explicitly lightweight operation")
	}
}

func TestDeleteTagRemovesOnlyTheNamedTag(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	for _, name := range []string{"keep", "remove"} {
		if err := r.CreateTag(ctx, name, sha); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.DeleteTag(ctx, "remove"); err != nil {
		t.Fatal(err)
	}
	if git(t, dir, "tag", "--list", "remove") != "" {
		t.Fatal("deleted tag still exists")
	}
	if strings.TrimSpace(git(t, dir, "rev-parse", "refs/tags/keep")) != sha {
		t.Fatal("unrelated tag changed")
	}
	if err := r.DeleteTag(ctx, "missing"); err == nil {
		t.Fatal("missing tag accepted")
	}
	if err := r.DeleteTag(ctx, "--force"); err == nil {
		t.Fatal("invalid tag name accepted")
	}
}

func TestPushTagPublishesLightweightAndAnnotatedTags(t *testing.T) {
	ctx := context.Background()
	source := fixture(t)
	seed(t, source)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, source, "clone", "--bare", source, bare)
	r, err := Clone(ctx, bare, filepath.Join(t.TempDir(), "cloned"))
	if err != nil {
		t.Fatal(err)
	}
	git(t, r.Root, "config", "user.name", "Test Developer")
	git(t, r.Root, "config", "user.email", "test@example.invalid")
	sha := strings.TrimSpace(git(t, r.Root, "rev-parse", "HEAD"))
	if err := r.CreateTag(ctx, "lightweight", sha); err != nil {
		t.Fatal(err)
	}
	git(t, r.Root, "tag", "-a", "annotated", "-m", "annotated message")
	for _, name := range []string{"lightweight", "annotated"} {
		if err := r.PushTag(ctx, "origin", name); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(git(t, bare, "rev-parse", "refs/tags/"+name)) == "" {
			t.Fatalf("tag %s missing on remote", name)
		}
	}
	if got := strings.TrimSpace(git(t, bare, "cat-file", "-t", "refs/tags/annotated")); got != "tag" {
		t.Fatalf("annotated tag lost its object type: %q", got)
	}
	if err := r.PushTag(ctx, "origin", "missing"); err == nil {
		t.Fatal("missing tag pushed")
	}
	if err := r.PushTag(ctx, "--force", "lightweight"); err == nil {
		t.Fatal("option remote accepted")
	}
	if err := r.PushTag(ctx, "origin", "--exec=bad"); err == nil {
		t.Fatal("option tag accepted")
	}
}

func TestCreateAnnotatedTagWritesTagObject(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	if err := r.CreateAnnotatedTag(ctx, "v2.0", sha, "release notes"); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(git(t, dir, "cat-file", "-t", "refs/tags/v2.0")); got != "tag" {
		t.Fatalf("annotated tag object type = %q", got)
	}
	body := git(t, dir, "cat-file", "tag", "refs/tags/v2.0")
	if !strings.Contains(body, "release notes") || !strings.Contains(body, "tagger ") {
		t.Fatalf("annotated tag body: %q", body)
	}
	if got := strings.TrimSpace(git(t, dir, "rev-parse", "refs/tags/v2.0^{commit}")); got != sha {
		t.Fatalf("annotated tag points at %q", got)
	}
	if err := r.CreateAnnotatedTag(ctx, "v2.0", sha, "again"); err == nil {
		t.Fatal("duplicate tag accepted")
	}
	if err := r.CreateAnnotatedTag(ctx, "empty-message", sha, "   "); err == nil {
		t.Fatal("empty message accepted")
	}
	if err := r.CreateAnnotatedTag(ctx, "--force", sha, "message"); err == nil {
		t.Fatal("option name accepted")
	}
	if err := r.CreateAnnotatedTag(ctx, "tree", strings.TrimSpace(git(t, dir, "rev-parse", "HEAD^{tree}")), "message"); err == nil {
		t.Fatal("tree object accepted")
	}
}
