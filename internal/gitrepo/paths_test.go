package gitrepo

import (
	"context"
	"strings"
	"testing"
)

func TestLiteralFilenameDoesNotStageOtherMatchingFiles(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "*.txt", "literal\n")
	write(t, dir, "other.txt", "other\n")
	r := openTest(t, dir)
	if err := r.Stage(context.Background(), "*.txt"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "diff", "--cached", "--name-only"); strings.Contains(got, "other.txt") {
		t.Fatalf("glob expanded: %s", got)
	}
}
func TestUnstageRenameRestoresBothIndexPaths(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "mv", "base.txt", "new.txt")
	r := openTest(t, dir)
	if err := r.StageRename(context.Background(), "new.txt", "base.txt", true); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "diff", "--cached"); got != "" {
		t.Fatalf("rename left deletion staged: %s", got)
	}
}
