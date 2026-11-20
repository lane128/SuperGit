package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanUntrackedChecksContentsAndProtectsIgnoredNestedAndSymlinkTargets(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	ctx := context.Background()
	r := openTest(t, dir)
	write(t, dir, ".gitignore", "*.keep\n")
	git(t, dir, "add", ".gitignore")
	write(t, dir, "mixed/foo.keep", "keep")
	write(t, dir, "mixed/delete", "remove")
	write(t, dir, "sub/line\nbreak", "remove")
	write(t, dir, "-dash", "remove")
	nested := filepath.Join(dir, "nested")
	git(t, dir, "init", nested)
	write(t, nested, "readme", "untouched")
	// A nested repository needs a commit for the parent to treat it as
	// an embedded checkout on every supported Git version.
	git(t, nested, "config", "user.name", "Test")
	git(t, nested, "config", "user.email", "test@example.invalid")
	seed(t, nested)
	outside := t.TempDir()
	write(t, outside, "secret", "untouched")
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "empty"), 0755); err != nil {
		t.Fatal(err)
	}
	preview, err := r.PrepareCleanUntracked(ctx)
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, "mixed/delete", "changed after preview")
	if err := r.CleanUntrackedChecked(ctx, preview); err == nil {
		t.Fatal("clean accepted modified files")
	}
	preview, err = r.PrepareCleanUntracked(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.CleanUntrackedChecked(ctx, preview); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"base.txt", ".gitignore", "mixed/foo.keep", "nested/.git", "nested/readme"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("lost protected %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(outside, "secret")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"empty", "-dash", "sub", "mixed/delete", "link"} {
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed: %v", name, err)
		}
	}
}

func TestCleanUntrackedRefusesRedirectedParent(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "tracked/file", "preserve")
	git(t, dir, "add", "tracked/file")
	write(t, dir, "pending/file", "remove")
	preview, err := openTest(t, dir).PrepareCleanUntracked(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "pending"), filepath.Join(dir, "moved")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("tracked", filepath.Join(dir, "pending")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := preview.removeEntry(root, "pending/file"); err == nil {
		t.Fatal("clean followed the replaced directory")
	}
	if data, err := os.ReadFile(filepath.Join(dir, "tracked/file")); err != nil || string(data) != "preserve" {
		t.Fatalf("tracked file changed: %q %v", data, err)
	}
}

func TestClearStashesRejectsChangedReflog(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	add := func(message string) {
		t.Helper()
		write(t, dir, "base.txt", message)
		git(t, dir, "stash", "push", "-m", message)
	}
	add("first")
	first, err := r.Stashes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	add("second")
	if err := r.ClearStashesChecked(ctx, first); err == nil {
		t.Fatal("cleared newly added stash")
	}
	all, err := r.Stashes(ctx)
	if err != nil || len(all) != 2 {
		t.Fatalf("%v %v", all, err)
	}
	if err := r.ClearStashesChecked(ctx, all); err != nil {
		t.Fatal(err)
	}
	if list, err := r.Stashes(ctx); err != nil || len(list) != 0 {
		t.Fatalf("%v %v", list, err)
	}
}
