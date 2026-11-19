package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileHunkRejectsExternalReplacement(t *testing.T) {
	for _, replacement := range []string{"text", "symlink"} {
		t.Run(replacement, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, "new.txt", "first\nsecond\n")
			r, ctx := openTest(t, dir), context.Background()
			d, err := r.Diff(ctx, "new.txt", false)
			if err != nil || len(d.Hunks) != 1 {
				t.Fatalf("new file Diff: %v %+v", err, d)
			}
			if replacement == "symlink" {
				if err := os.Remove(filepath.Join(dir, "new.txt")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("base.txt", filepath.Join(dir, "new.txt")); err != nil {
					t.Fatal(err)
				}
			} else {
				write(t, dir, "new.txt", "external replacement\n")
			}
			before := git(t, dir, "ls-files", "--stage")
			if err := r.ApplyHunk(ctx, d.Hunks[0].Patch, false); err == nil {
				t.Fatal("stale new-file hunk applied")
			}
			if got := git(t, dir, "ls-files", "--stage"); got != before {
				t.Fatal("stale hunk changed Index")
			}
		})
	}
}

func TestNewFilePartialStageIgnoresUserDiffPrefixes(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "config", "diff.noprefix", "true")
	write(t, dir, "dir/new.txt", "first\nsecond\n")
	r, ctx := openTest(t, dir), context.Background()
	d, err := r.Diff(ctx, "dir/new.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyLines(ctx, "dir/new.txt", d, selectedText(d, "+first"), false); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "show", ":dir/new.txt"); got != "first\n" {
		t.Fatalf("Index = %q", got)
	}
	if got := git(t, dir, "ls-files", "--", "new.txt"); got != "" {
		t.Fatal("prefix config redirected patch")
	}
}
