package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStashesPropagatesCancellation(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := openTest(t, dir).Stashes(ctx); err == nil {
		t.Fatal("cancelled stash read reported success")
	}
}
func TestConflictDeletionRequiresExplicitFileChoice(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "checkout", "-b", "feature")
	git(t, dir, "rm", "base.txt")
	git(t, dir, "commit", "-m", "delete file")
	git(t, dir, "checkout", "main")
	write(t, dir, "base.txt", "ours changed\n")
	git(t, dir, "commit", "-am", "edit file")
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.Integrate(ctx, "merge", "feature"); err == nil {
		t.Fatal("expected delete/modify conflict")
	}
	c, err := r.Conflict(ctx, "base.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ResolveConflict(ctx, c, ""); err == nil {
		t.Fatal("deleted side silently converted to empty file")
	}
	if err := r.ResolveConflictDeletion(ctx, c); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "base.txt")); !os.IsNotExist(err) {
		t.Fatal("deleted file remains")
	}
	if err := r.Continue(ctx, "merge"); err != nil {
		t.Fatal(err)
	}
}
func TestCherryPickRevertAndRebase(t *testing.T) {
	for _, kind := range []string{"cherry-pick", "rebase"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			git(t, dir, "checkout", "-b", "feature")
			write(t, dir, "feature.txt", "feature\n")
			git(t, dir, "add", "feature.txt")
			git(t, dir, "commit", "-m", "feature")
			git(t, dir, "checkout", "main")
			write(t, dir, "main.txt", "main\n")
			git(t, dir, "add", "main.txt")
			git(t, dir, "commit", "-m", "main")
			r := openTest(t, dir)
			ctx := context.Background()
			if err := r.Integrate(ctx, kind, "feature"); err != nil {
				t.Fatal(err)
			}
			if b, err := os.ReadFile(filepath.Join(dir, "feature.txt")); err != nil || string(b) != "feature\n" {
				t.Fatal("integration missing feature")
			}
			if kind == "cherry-pick" {
				if err := r.Integrate(ctx, "revert", "HEAD"); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(filepath.Join(dir, "feature.txt")); !os.IsNotExist(err) {
					t.Fatal("revert did not undo feature")
				}
			}
		})
	}
}
