package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFilePartialStagePreservesExecutableMode(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "config", "core.fileMode", "true")
	write(t, dir, "script.sh", "first\nsecond\n")
	if err := os.Chmod(filepath.Join(dir, "script.sh"), 0755); err != nil {
		t.Fatal(err)
	}
	r, ctx := openTest(t, dir), context.Background()
	d, err := r.Diff(ctx, "script.sh", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyLines(ctx, "script.sh", d, selectedText(d, "+first", "+second"), false); err != nil {
		t.Fatal(err)
	}
	d, err = r.Diff(ctx, "script.sh", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyLines(ctx, "script.sh", d, selectedText(d, "+first"), true); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "ls-files", "--stage", "--", "script.sh"); !strings.HasPrefix(got, "100755 ") {
		t.Fatalf("lost executable mode: %q", got)
	}
	if got := git(t, dir, "show", ":script.sh"); got != "second\n" {
		t.Fatalf("Index = %q", got)
	}
}

func TestNewFilePartialOperationsRejectUnsupportedAndStaleContent(t *testing.T) {
	for _, kind := range []string{"binary", "empty", "symlink", "large", "no-newline", "stale-worktree", "stale-index"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			r, ctx := openTest(t, dir), context.Background()
			path := "new.txt"
			body := "first\nsecond\n"
			switch kind {
			case "binary":
				body = "first\x00\nsecond\n"
			case "empty":
				body = ""
			case "large":
				body = strings.Repeat("first\n", 200000)
			case "no-newline":
				body = "first\nsecond"
			}
			if kind == "symlink" {
				if err := os.Symlink("base.txt", filepath.Join(dir, path)); err != nil {
					t.Fatal(err)
				}
			} else {
				write(t, dir, path, body)
			}
			d, err := r.Diff(ctx, path, false)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "stale-worktree" {
				body = "external\nsecond\n"
				write(t, dir, path, body)
			}
			if kind == "stale-index" {
				git(t, dir, "add", "--", path)
				d, err = r.Diff(ctx, path, true)
				if err != nil {
					t.Fatal(err)
				}
				body = "external Index\nsecond\n"
				write(t, dir, path, body)
				git(t, dir, "add", "--", path)
			}
			before := git(t, dir, "ls-files", "--stage")
			if err := r.ApplyLines(ctx, path, d, selectedText(d, "+first"), kind == "stale-index"); err == nil {
				t.Fatal("unsupported or stale selection accepted")
			}
			if got := git(t, dir, "ls-files", "--stage"); got != before {
				t.Fatal("rejected operation changed Index")
			}
			if kind == "symlink" {
				if got, err := os.Readlink(filepath.Join(dir, path)); err != nil || got != "base.txt" {
					t.Fatalf("working link = %q, %v", got, err)
				}
			} else {
				got, err := os.ReadFile(filepath.Join(dir, path))
				if err != nil || string(got) != body {
					t.Fatalf("working bytes changed: %v", err)
				}
			}
		})
	}
}

func TestNewFilePartialStageRespectsTextAttributes(t *testing.T) {
	for _, encoding := range []string{"crlf", "utf16le"} {
		t.Run(encoding, func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			attributes := "*.txt text eol=crlf\n"
			working := []byte("first\r\nsecond\r\n")
			if encoding == "utf16le" {
				attributes = "*.txt text working-tree-encoding=UTF-16LE\n"
				working = nil
				for _, ch := range []byte("first\nsecond\n") {
					working = append(working, ch, 0)
				}
			}
			write(t, dir, ".gitattributes", attributes)
			if err := os.WriteFile(filepath.Join(dir, "new.txt"), working, 0644); err != nil {
				t.Fatal(err)
			}
			r, ctx := openTest(t, dir), context.Background()
			d, err := r.Diff(ctx, "new.txt", false)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.ApplyLines(ctx, "new.txt", d, selectedText(d, "+first"), false); err != nil {
				t.Fatal(err)
			}
			if got := git(t, dir, "show", ":new.txt"); got != "first\n" {
				t.Fatalf("canonical Index = %q", got)
			}
			if body, err := os.ReadFile(filepath.Join(dir, "new.txt")); err != nil || string(body) != string(working) {
				t.Fatalf("working encoding changed: %v", err)
			}
		})
	}
}

func TestNewFileNoNewlineWholeHunkAndDiscardProtection(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "new.txt", "first\nsecond")
	r, ctx := openTest(t, dir), context.Background()
	d, err := r.Diff(ctx, "new.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.DiscardLines(ctx, "new.txt", d, selectedText(d, "+first", "+second")); err == nil {
		t.Fatal("new-file partial stage enabled destructive discard")
	}
	if !d.CanStageHunks || len(d.Hunks) != 1 {
		t.Fatal("ordinary new-file hunk missing")
	}
	if err := r.ApplyHunk(ctx, d.Hunks[0].Patch, false); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "show", ":new.txt"); got != "first\nsecond" {
		t.Fatalf("no-newline Index = %q", got)
	}
	d, err = r.Diff(ctx, "new.txt", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.ApplyHunk(ctx, d.Hunks[0].Patch, true); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "ls-files", "--", "new.txt"); got != "" {
		t.Fatalf("remaining new-file Index = %q", got)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "new.txt")); err != nil || string(got) != "first\nsecond" {
		t.Fatalf("working bytes = %q %v", got, err)
	}
}
