package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookFailureIncludesStandardOutput(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "base.txt", "changed\n")
	r := openTest(t, dir)
	r.Stage(context.Background(), "base.txt")
	if err := os.WriteFile(filepath.Join(dir, ".git/hooks/pre-commit"), []byte("#!/bin/sh\nprintf 'hook diagnostic on stdout\\n'\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	err := r.Commit(context.Background(), "blocked")
	if err == nil || !strings.Contains(err.Error(), "hook diagnostic on stdout") {
		t.Fatalf("missing hook stdout: %v", err)
	}
}
func TestMalformedGitmodulesIsReported(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, ".gitmodules", "[invalid\n")
	r := openTest(t, dir)
	if _, err := r.Submodules(context.Background()); err == nil {
		t.Fatal("invalid submodule config silently treated as no modules")
	}
}
func TestSelectedRepositoryIgnoresInheritedIndexOverride(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	write(t, dir, "selected.txt", "selected\n")
	wrong := filepath.Join(t.TempDir(), "other-index")
	t.Setenv("GIT_INDEX_FILE", wrong)
	if err := r.Stage(context.Background(), "selected.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wrong); !os.IsNotExist(err) {
		t.Fatal("inherited index override redirected operation")
	}
	t.Setenv("GIT_INDEX_FILE", filepath.Join(dir, ".git/index"))
	if got := git(t, dir, "diff", "--cached", "--name-only"); got != "selected.txt\n" {
		t.Fatalf("selected repository index not updated: %q", got)
	}
}
