package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSwitchMatchesNativeGitOnIgnoredAndWorktreeRemovePreservesIgnored(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, ".gitignore", "ignored.txt\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "ignore")
	git(t, dir, "switch", "-c", "other")
	write(t, dir, "ignored.txt", "tracked replacement\n")
	git(t, dir, "add", "-f", "ignored.txt")
	git(t, dir, "commit", "-m", "track")
	git(t, dir, "switch", "main")
	write(t, dir, "ignored.txt", "valuable local content\n")
	r := openTest(t, dir)
	if err := r.SwitchBranch(ctx, "other"); err != nil {
		t.Fatalf("switch should match native git and allow overwriting ignored file: %v", err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "ignored.txt"))
	if string(b) != "tracked replacement\n" {
		t.Fatalf("switch did not check out tracked ignored.txt: %q", b)
	}
	git(t, dir, "switch", "main")
	wt := filepath.Join(t.TempDir(), "worktree")
	if err := r.AddWorktree(ctx, wt, "review", "main"); err != nil {
		t.Fatal(err)
	}
	write(t, wt, "ignored.txt", "valuable worktree content\n")
	if err := r.RemoveWorktree(ctx, wt); err == nil {
		t.Error("must refuse worktree containing ignored content")
	}
	b, err := os.ReadFile(filepath.Join(wt, "ignored.txt"))
	if err != nil || string(b) != "valuable worktree content\n" {
		t.Error("ignored worktree content lost")
	}
}
func TestConflictRejectsSymlinkAncestor(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	write(t, dir, "dir/file.txt", "base\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "base")
	git(t, dir, "switch", "-c", "feature")
	write(t, dir, "dir/file.txt", "theirs\n")
	git(t, dir, "commit", "-am", "theirs")
	git(t, dir, "switch", "main")
	write(t, dir, "dir/file.txt", "ours\n")
	git(t, dir, "commit", "-am", "ours")
	r := openTest(t, dir)
	if _, err := run(ctx, dir, "", "merge", "--no-edit", "feature"); err == nil {
		t.Fatal("expected conflict")
	}
	c, err := r.Conflict(ctx, "dir/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	git(t, nested, "init", "-b", "main")
	write(t, nested, "file.txt", c.Working)
	if err := os.Rename(filepath.Join(dir, "dir"), filepath.Join(dir, "saved-dir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nested", filepath.Join(dir, "dir")); err != nil {
		t.Fatal(err)
	}
	if err := r.ResolveConflict(ctx, c, "STALE RESULT\n"); err == nil {
		t.Error("symlink ancestor accepted")
	}
	b, err := os.ReadFile(filepath.Join(nested, "file.txt"))
	if err != nil || string(b) != c.Working {
		t.Fatal("nested repository overwritten through symlink")
	}
}
func TestConflictRejectsReplacedMergeSession(t *testing.T) {
	ctx := context.Background()
	dir := conflictFixture(t)
	r := openTest(t, dir)
	if _, err := run(ctx, dir, "", "merge", "--no-edit", "feature"); err == nil {
		t.Fatal("expected conflict")
	}
	c, err := r.Conflict(ctx, "base.txt")
	if err != nil {
		t.Fatal(err)
	}
	git(t, dir, "merge", "--abort")
	git(t, dir, "switch", "feature")
	write(t, dir, "second.txt", "new target\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "new target")
	git(t, dir, "switch", "main")
	if _, err := run(ctx, dir, "", "merge", "--no-edit", "feature"); err == nil {
		t.Fatal("expected conflict")
	}
	now, err := r.Conflict(ctx, "base.txt")
	if err != nil {
		t.Fatal(err)
	}
	if now.OIDs != c.OIDs || now.Working != c.Working {
		t.Fatal("fixture must preserve conflicted file stages")
	}
	if now.Operation == c.Operation {
		t.Error("different merge target has same operation identity")
	}
	if err := r.ResolveConflict(ctx, c, "resolution from prior operation\n"); err == nil {
		t.Error("stale conflict snapshot accepted")
	}
}
func TestPushChecksSubmoduleWhenParentUpToDateOrChildRemoteMissing(t *testing.T) {
	ctx := context.Background()
	child := fixture(t)
	seed(t, child)
	childBare := filepath.Join(t.TempDir(), "child.git")
	git(t, child, "clone", "--bare", child, childBare)
	parent := fixture(t)
	seed(t, parent)
	git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", childBare, "module")
	git(t, parent, "commit", "-am", "module")
	module := filepath.Join(parent, "module")
	git(t, module, "config", "user.name", "Test")
	git(t, module, "config", "user.email", "test@example.invalid")
	write(t, module, "base.txt", "unpublished\n")
	git(t, module, "commit", "-am", "unpublished")
	git(t, parent, "commit", "-am", "unpublished child pointer")
	parentBare := filepath.Join(t.TempDir(), "parent.git")
	git(t, parent, "init", "--bare", parentBare)
	git(t, parent, "remote", "add", "origin", parentBare)
	r := openTest(t, parent)
	if err := r.Push(ctx, "origin", "main", "main", false); err == nil {
		t.Error("unpublished child accepted")
	}
	git(t, parent, "push", "--no-recurse-submodules", "origin", "main")
	if err := r.Push(ctx, "origin", "main", "main", false); err == nil {
		t.Error("up-to-date parent skipped child reachability check")
	}
	git(t, module, "remote", "remove", "origin")
	write(t, parent, "next.txt", "next\n")
	git(t, parent, "add", ".")
	git(t, parent, "commit", "-m", "next")
	before := git(t, parentBare, "rev-parse", "refs/heads/main")
	if err := r.Push(ctx, "origin", "main", "main", false); err == nil {
		t.Error("missing child remote accepted")
	}
	if got := git(t, parentBare, "rev-parse", "refs/heads/main"); got != before {
		t.Error("parent advanced with unpublished child")
	}
	git(t, module, "remote", "add", "origin", childBare)
	git(t, module, "push", "origin", "HEAD:refs/heads/main")
	if err := r.Push(ctx, "origin", "main", "main", false); err != nil {
		t.Fatalf("published child refused: %v", err)
	}
}
func TestSubmoduleUpdatePreservesCleanUnfinishedMerge(t *testing.T) {
	ctx := context.Background()
	child := fixture(t)
	seed(t, child)
	parent := fixture(t)
	seed(t, parent)
	git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "module")
	git(t, parent, "commit", "-am", "module")
	module := filepath.Join(parent, "module")
	git(t, module, "config", "user.name", "Test")
	git(t, module, "config", "user.email", "test@example.invalid")
	git(t, module, "switch", "-c", "feature")
	write(t, module, "base.txt", "changed\n")
	git(t, module, "commit", "-am", "change")
	write(t, module, "base.txt", "original\n")
	git(t, module, "commit", "-am", "undo")
	git(t, module, "switch", "main")
	write(t, module, "ours.txt", "ours\n")
	git(t, module, "add", ".")
	git(t, module, "commit", "-m", "ours")
	git(t, module, "merge", "--no-ff", "--no-commit", "feature")
	m := openTest(t, module)
	op, err := m.Operation(ctx)
	if err != nil || op.Kind != "merge" {
		t.Fatal("missing active merge")
	}
	s, err := m.Status(ctx)
	if err != nil || len(s.Entries) != 0 {
		t.Fatal("fixture not clean")
	}
	if err := openTest(t, parent).UpdateSubmodule(ctx, "module"); err == nil {
		t.Error("unfinished child operation accepted")
	}
	after, err := m.Operation(ctx)
	if err != nil || after != op {
		t.Error("unfinished child operation changed")
	}
	if b, err := os.ReadFile(filepath.Join(module, "ours.txt")); err != nil || !strings.Contains(string(b), "ours") {
		t.Error("child file lost")
	}
}
