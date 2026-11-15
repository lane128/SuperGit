package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnbornUnstageAfterFurtherEditsPreservesWorktree(t *testing.T) {
	dir := fixture(t)
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "new.txt", "staged version\n")
	if err := r.Stage(ctx, "new.txt"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "new.txt", "newer working version\n")
	if err := r.Unstage(ctx, "new.txt"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "new.txt"))
	if err != nil || string(b) != "newer working version\n" {
		t.Fatalf("worktree changed: %q %v", b, err)
	}
	if got := git(t, dir, "ls-files"); got != "" {
		t.Fatalf("index not emptied: %s", got)
	}
}
func TestStageFurtherEditsToStagedRename(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	destination := "new\nname.txt"
	git(t, dir, "mv", "base.txt", destination)
	write(t, dir, destination, "edited after rename\n")
	r := openTest(t, dir)
	ctx := context.Background()
	s, err := r.Status(ctx)
	if err != nil || len(s.Entries) != 1 {
		t.Fatalf("status %v %v", s, err)
	}
	e := s.Entries[0]
	if e.OriginalPath != "base.txt" || !e.Staged() || !e.Unstaged() {
		t.Fatalf("rename fixture wrong: %+v", e)
	}
	if err := r.StageRename(ctx, e.Path, e.OriginalPath, false); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "show", ":"+destination); got != "edited after rename\n" {
		t.Fatalf("new content not staged: %q", got)
	}
	if got := git(t, dir, "diff", "--name-only"); got != "" {
		t.Fatalf("unstaged edits remain: %s", got)
	}
}
func TestHunkStagingOverridesDiffPrefixConfiguration(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "config", "diff.noprefix", "true")
	git(t, dir, "config", "diff.srcPrefix", "custom-before/")
	git(t, dir, "config", "diff.dstPrefix", "custom-after/")
	write(t, dir, "base.txt", "modified\n")
	r := openTest(t, dir)
	ctx := context.Background()
	d, err := r.Diff(ctx, "base.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Hunks) != 1 {
		t.Fatalf("hunks: %s", d.Text)
	}
	if err := r.ApplyHunk(ctx, d.Hunks[0].Patch, false); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "show", ":base.txt"); !strings.Contains(got, "modified") {
		t.Fatalf("not staged: %q", got)
	}
	staged, err := r.Diff(ctx, "base.txt", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyHunk(ctx, staged.Hunks[0].Patch, true); err != nil {
		t.Fatal(err)
	}
}
