package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscardPreparedPreservesIndexAndUnrelatedFiles(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "base.txt", "index version\n")
	git(t, dir, "add", "--", "base.txt")
	write(t, dir, "base.txt", "working version\n")
	write(t, dir, "-特殊\nfile.txt", "untracked\n")
	r := openTest(t, dir)
	ctx := context.Background()
	snapshot, err := r.PrepareDiscard(ctx, []DiscardTarget{{Path: "base.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	index, head := git(t, dir, "ls-files", "--stage"), git(t, dir, "rev-parse", "HEAD")
	if err := r.DiscardPrepared(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "base.txt")); err != nil || string(got) != "index version\n" {
		t.Fatalf("working file = %q, %v", got, err)
	}
	if git(t, dir, "ls-files", "--stage") != index || git(t, dir, "rev-parse", "HEAD") != head {
		t.Fatal("discard wrote Index or HEAD")
	}
	if got, err := os.ReadFile(filepath.Join(dir, "-特殊\nfile.txt")); err != nil || string(got) != "untracked\n" {
		t.Fatal("tracked discard deleted untracked file")
	}
}

func TestDiscardPreparedRejectsChangedFilesBeforeAnyWrite(t *testing.T) {
	for _, change := range []string{"worktree", "index", "root", "untracked became tracked", "binary"} {
		t.Run(change, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "base.txt", "working version\n")
			write(t, dir, "second.txt", "second original\n")
			git(t, dir, "add", "--", "second.txt")
			write(t, dir, "second.txt", "second working\n")
			targets := []DiscardTarget{{Path: "base.txt"}, {Path: "second.txt"}}
			if change == "untracked became tracked" {
				write(t, dir, "new.txt", "untracked\n")
				targets = append(targets, DiscardTarget{Path: "new.txt", Untracked: true})
			}
			if change == "binary" {
				write(t, dir, "second.txt", "before\x00binary")
			}
			r := openTest(t, dir)
			ctx := context.Background()
			snapshot, err := r.PrepareDiscard(ctx, targets)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "worktree":
				write(t, dir, "second.txt", "external edit\n")
			case "index":
				git(t, dir, "add", "--", "second.txt")
			case "untracked became tracked":
				git(t, dir, "add", "--", "new.txt")
			case "binary":
				write(t, dir, "second.txt", "after\x00binary")
			case "root":
				r = openTest(t, fixture(t))
			}
			if err := r.DiscardPrepared(ctx, snapshot); err == nil {
				t.Fatal("stale confirmation accepted")
			}
			if got, _ := os.ReadFile(filepath.Join(dir, "base.txt")); string(got) != "working version\n" {
				t.Fatal("first path was discarded before checking later targets")
			}
		})
	}
}

func TestDiscardPreparedRejectsUnresolvedConflicts(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "checkout", "-b", "source")
	write(t, dir, "base.txt", "source version\n")
	git(t, dir, "commit", "-am", "source")
	git(t, dir, "checkout", "main")
	write(t, dir, "base.txt", "main version\n")
	git(t, dir, "commit", "-am", "main")
	ctx := context.Background()
	if _, err := run(ctx, dir, "", "merge", "source"); err == nil {
		t.Fatal("fixture did not conflict")
	}
	before, err := os.ReadFile(filepath.Join(dir, "base.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := openTest(t, dir).PrepareDiscard(ctx, []DiscardTarget{{Path: "base.txt"}}); err == nil {
		t.Fatal("unresolved conflict accepted as discard target")
	}
	after, err := os.ReadFile(filepath.Join(dir, "base.txt"))
	if err != nil || string(after) != string(before) {
		t.Fatal("rejected conflict changed file")
	}
}

func TestDiscardPreparedDeletesOnlyExplicitUntrackedFile(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	name := "-空 格\nnew.txt"
	write(t, dir, name, "temporary\n")
	write(t, dir, "keep.txt", "keep\n")
	r := openTest(t, dir)
	ctx := context.Background()
	snapshot, err := r.PrepareDiscard(ctx, []DiscardTarget{{Path: name, Untracked: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.DiscardPrepared(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
		t.Fatal("explicit deletion did not remove target")
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "keep.txt")); string(got) != "keep\n" {
		t.Fatal("delete touched unrelated untracked file")
	}
	if _, err := r.PrepareDiscard(ctx, []DiscardTarget{{Path: "base.txt", Untracked: true}}); err == nil {
		t.Fatal("tracked file accepted as an untracked deletion")
	}
}

func TestDiscardPreparedRejectsSymlinkEscapeAndProtectedTargets(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "outside")); err != nil {
		t.Fatal(err)
	}
	r := openTest(t, dir)
	if _, err := r.PrepareDiscard(context.Background(), []DiscardTarget{{Path: "outside/secret", Untracked: true}}); err == nil {
		t.Fatal("escaping path accepted")
	}
	// A large binary must be compared by bytes, without relying on a textual
	// 'Binary files differ' diff or the patch renderer's output limit.
	write(t, dir, "large.bin", "\x00"+strings.Repeat("x", 5<<20))
	snapshot, err := r.PrepareDiscard(context.Background(), []DiscardTarget{{Path: "large.bin", Untracked: true}})
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, "large.bin", "\x00"+strings.Repeat("y", 5<<20))
	if err := r.DiscardPrepared(context.Background(), snapshot); err == nil {
		t.Fatal("changed large binary accepted")
	}
}
