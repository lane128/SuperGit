package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareCommitFileUndoRejectsContentFilters(t *testing.T) {
	root := fixture(t)
	git(t, root, "config", "filter.undo-fixture.clean", "sed s/prod/@HOST@/g")
	git(t, root, "config", "filter.undo-fixture.smudge", "sed s/@HOST@/prod/g")
	write(t, root, ".gitattributes", "selected.txt filter=undo-fixture\n")
	write(t, root, "selected.txt", "url=prod\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "filtered base")
	write(t, root, "selected.txt", "url=static\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "source")
	source := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	r := openTest(t, root)
	status, err := r.Status(context.Background())
	if err != nil || len(status.Entries) != 0 {
		t.Fatal("filtered fixture must be clean")
	}
	index := git(t, root, "ls-files", "--stage", "-z")
	prepared, err := r.PrepareCommitFileUndo(context.Background(), source, "selected.txt", status)
	if prepared != nil {
		defer prepared.Close()
	}
	if err == nil {
		got, _ := os.ReadFile(prepared.result)
		t.Fatalf("unsupported filter accepted, prepared bytes %q instead of working-tree url=prod", got)
	}
	got, readErr := os.ReadFile(filepath.Join(root, "selected.txt"))
	if readErr != nil || string(got) != "url=static\n" || git(t, root, "ls-files", "--stage", "-z") != index || git(t, root, "status", "--porcelain=v1", "-z") != "" {
		t.Fatal("filter rejection changed source")
	}
}
