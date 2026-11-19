package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTextFilePartialStageAndUnstage(t *testing.T) {
	for _, unborn := range []bool{false, true} {
		for _, path := range []string{"new.txt", "dir/空 格\n*.txt", "-option.txt"} {
			t.Run(strings.ReplaceAll(path, "/", "_")+map[bool]string{false: "_existing", true: "_unborn"}[unborn], func(t *testing.T) {
				dir := fixture(t)
				if !unborn {
					seed(t, dir)
				}
				write(t, dir, "control.txt", "keep staged\n")
				git(t, dir, "add", "--", "control.txt")
				control := git(t, dir, "ls-files", "--stage", "--", "control.txt")
				write(t, dir, path, "first\nsecond\nthird\n")
				r := openTest(t, dir)
				ctx := context.Background()
				d, err := r.Diff(ctx, path, false)
				if err != nil {
					t.Fatal(err)
				}
				if err := r.ApplyLines(ctx, path, d, selectedText(d, "+first", "+third"), false); err != nil {
					t.Fatalf("stage selected new-file lines: %v", err)
				}
				if got := git(t, dir, "show", ":"+path); got != "first\nthird\n" {
					t.Fatalf("Index = %q", got)
				}
				d, err = r.Diff(ctx, path, true)
				if err != nil {
					t.Fatal(err)
				}
				if err := r.ApplyLines(ctx, path, d, selectedText(d, "+first"), true); err != nil {
					t.Fatalf("partial unstage: %v", err)
				}
				if got := git(t, dir, "show", ":"+path); got != "third\n" {
					t.Fatalf("remaining Index = %q", got)
				}
				d, err = r.Diff(ctx, path, true)
				if err != nil {
					t.Fatal(err)
				}
				if err := r.ApplyLines(ctx, path, d, selectedText(d, "+third"), true); err != nil {
					t.Fatalf("unstage remaining: %v", err)
				}
				if got := git(t, dir, "ls-files", "--", path); got != "" {
					t.Fatalf("new path still in Index: %q", got)
				}
				if got := git(t, dir, "ls-files", "--stage", "--", "control.txt"); got != control {
					t.Fatalf("control Index changed: %q", got)
				}
				body, err := os.ReadFile(filepath.Join(dir, path))
				if err != nil || string(body) != "first\nsecond\nthird\n" {
					t.Fatalf("working bytes = %q, %v", body, err)
				}
			})
		}
	}
}
