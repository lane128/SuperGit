package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStageTrackedPathsMatchesUpdateOnlyAndPreservesWorktree(t *testing.T) {
	for _, unborn := range []bool{false, true} {
		t.Run(map[bool]string{false: "committed", true: "unborn"}[unborn], func(t *testing.T) {
			dir := fixture(t)
			names := []string{"modified.txt", "deleted.txt", "moved.txt", "replaced.txt"}
			for _, name := range names {
				write(t, dir, name, "baseline\n")
			}
			git(t, dir, "add", "--all")
			if !unborn {
				git(t, dir, "commit", "--no-gpg-sign", "-m", "base")
			}
			write(t, dir, "added.txt", "index addition\n")
			git(t, dir, "add", "--", "added.txt")
			before := git(t, dir, "write-tree")
			write(t, dir, "modified.txt", "latest tracked\n")
			write(t, dir, "added.txt", "latest addition\n")
			if err := os.Remove(filepath.Join(dir, "deleted.txt")); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(filepath.Join(dir, "moved.txt"), filepath.Join(dir, "untracked destination.txt")); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(dir, "replaced.txt")); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(dir, "replaced.txt"), 0700); err != nil {
				t.Fatal(err)
			}
			write(t, dir, "replaced.txt/child.txt", "do not add\n")
			write(t, dir, "sentinel.txt", "untracked\n")
			git(t, dir, "add", "-u", "--")
			want := git(t, dir, "write-tree")
			git(t, dir, "read-tree", strings.TrimSpace(before))
			r := openTest(t, dir)
			if err := r.StageTrackedPaths(context.Background(), append(names, "added.txt")); err != nil {
				t.Fatal(err)
			}
			if got := git(t, dir, "write-tree"); got != want {
				t.Fatalf("Index differs from git add -u: %s != %s", got, want)
			}
			for name, want := range map[string]string{"modified.txt": "latest tracked\n", "added.txt": "latest addition\n", "untracked destination.txt": "baseline\n", "replaced.txt/child.txt": "do not add\n", "sentinel.txt": "untracked\n"} {
				got, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil || string(got) != want {
					t.Fatalf("working bytes changed for %s", name)
				}
			}
		})
	}
}
