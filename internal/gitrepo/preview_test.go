package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPreviewUsesSelectedVersion(t *testing.T) {
	dir := fixture(t)
	file := "assets/logo #1.png"
	write(t, dir, file, "original")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "base")
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	newName := "assets/new logo.png"
	git(t, dir, "mv", file, newName)
	write(t, dir, newName, "indexed")
	git(t, dir, "add", newName)
	write(t, dir, newName, "working")
	r := &Repository{Root: dir}
	for _, tc := range []struct{ rev, path, want string }{{sha, file, "original"}, {PreviewIndex, newName, "indexed"}, {PreviewWorktree, newName, "working"}} {
		data, err := r.ReadPreview(context.Background(), tc.rev, tc.path)
		if err != nil || string(data) != tc.want {
			t.Fatalf("%s: %q, %v", tc.rev, data, err)
		}
	}
	write(t, dir, "large.png", "")
	if err := os.Truncate(filepath.Join(dir, "large.png"), MaxPreviewBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadPreview(context.Background(), PreviewWorktree, "large.png"); !errors.Is(err, ErrPreviewTooLarge) {
		t.Fatal(err)
	}
	if err := os.Symlink(newName, filepath.Join(dir, "link.png")); err != nil {
		t.Skip(err)
	}
	if _, err := r.ReadPreview(context.Background(), PreviewWorktree, "link.png"); err == nil {
		t.Fatal("followed symlink")
	}
}
