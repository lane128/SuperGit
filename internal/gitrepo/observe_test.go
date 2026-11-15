package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestChangeStampTracksRepositoryStateAndIsStable(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	stamp := func() string {
		t.Helper()
		s, err := r.ChangeStamp(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	previous := stamp()
	if s := stamp(); s != previous {
		t.Fatal("idle probe changed stamp")
	}
	changed := func(name string) {
		t.Helper()
		s := stamp()
		if s == previous {
			t.Fatal(name, "was not detected")
		}
		previous = s
		if stamp() != s {
			t.Fatal(name, "unstable stamp")
		}
	}
	write(t, dir, "base.txt", "external\n")
	changed("working file")
	git(t, dir, "add", "--", "base.txt")
	changed("external stage")
	git(t, dir, "commit", "-m", "external")
	changed("external commit")
	git(t, dir, "branch", "external")
	changed("external ref")
	git(t, dir, "remote", "add", "local", filepath.Join(t.TempDir(), "local.git"))
	changed("config")
	write(t, dir, ".gitignore", "cache/\n")
	git(t, dir, "add", ".gitignore")
	git(t, dir, "commit", "-m", "ignore")
	previous = stamp()
	if err := os.Mkdir(filepath.Join(dir, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "cache/build.bin", "ignored output")
	if stamp() != previous {
		t.Fatal("ignored build output caused refresh")
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "MERGE_HEAD"), []byte(git(t, dir, "rev-parse", "HEAD")), 0600); err != nil {
		t.Fatal(err)
	}
	changed("operation marker")
}
func TestChangeStampTracksLinkedWorktreeAndChildHead(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	before, err := r.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(t.TempDir(), "linked")
	git(t, dir, "worktree", "add", "-b", "linked", linked)
	after, err := r.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("linked worktree not detected")
	}
	child := fixture(t)
	seed(t, child)
	git(t, dir, "-c", "protocol.file.allow=always", "submodule", "add", child, "child")
	git(t, dir, "commit", "-am", "child")
	before, err = r.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	git(t, filepath.Join(dir, "child"), "-c", "user.name=Preview", "-c", "user.email=preview@example.invalid", "commit", "--allow-empty", "-m", "new child HEAD")
	after, err = r.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("submodule child HEAD not detected")
	}
	// A second child move keeps porcelain-v2's SC.. state and recorded SHA unchanged.
	before = after
	rawBefore := git(t, dir, "status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all", "--ignore-submodules=none")
	git(t, filepath.Join(dir, "child"), "-c", "user.name=Preview", "-c", "user.email=preview@example.invalid", "commit", "--allow-empty", "-m", "second child HEAD")
	rawAfter := git(t, dir, "status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all", "--ignore-submodules=none")
	if rawBefore != rawAfter {
		t.Fatal("fixture must retain identical parent status")
	}
	after, err = r.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Error("second submodule HEAD movement was invisible")
	}
	linkedRepo := openTest(t, linked)
	before, err = linkedRepo.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	git(t, dir, "branch", "another-ref")
	after, err = linkedRepo.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("linked worktree missed shared refs")
	}
}
func TestChangeStampTracksRestagingAndIndexModeWithSameFlags(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "base.txt", "first staged\n")
	git(t, dir, "add", "--", "base.txt")
	first, err := r.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, "base.txt", "second staged\n")
	git(t, dir, "add", "--", "base.txt")
	second, err := r.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Error("restaging changed blob with same M. flags was invisible")
	}
	if err := os.Chmod(filepath.Join(dir, "base.txt"), 0755); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "update-index", "--chmod=+x", "--", "base.txt")
	third, err := r.ChangeStamp(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if second == third {
		t.Error("changed Index mode with same status flags was invisible")
	}
}
func TestChangeStampHonorsCancellation(t *testing.T) {
	r := openTest(t, fixture(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.ChangeStamp(ctx); err == nil {
		t.Fatal("cancelled probe succeeded")
	}
}
