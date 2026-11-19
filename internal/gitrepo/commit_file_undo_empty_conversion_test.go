package gitrepo

import (
	"context"
	"strings"
	"testing"
)

func TestPrepareCommitFileUndoRejectsConversionEvenWhenCurrentFileIsEmpty(t *testing.T) {
	root := fixture(t)
	write(t, root, "selected.txt", "old\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "base")
	write(t, root, "selected.txt", "")
	git(t, root, "commit", "-am", "empty source")
	source := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	git(t, root, "config", "core.autocrlf", "true")
	r := openTest(t, root)
	status, err := r.Status(context.Background())
	if err != nil || len(status.Entries) != 0 {
		t.Fatal("empty conversion fixture must be clean")
	}
	p, err := r.PrepareCommitFileUndo(context.Background(), source, "selected.txt", status)
	if p != nil {
		defer p.Close()
	}
	if err == nil {
		t.Fatal("empty current content hid a required result conversion")
	}
}
