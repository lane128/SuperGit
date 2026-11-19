package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishCommitFileUndoPreservesLaterEditsAndExecutableMode(t *testing.T) {
	root := fixture(t)
	lines := make([]string, 80)
	for i := range lines {
		lines[i] = strings.Repeat("x", i+1)
	}
	base := append([]string(nil), lines...)
	write(t, root, "script.sh", strings.Join(lines, "\n")+"\n")
	if err := os.Chmod(filepath.Join(root, "script.sh"), 0755); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "base")
	lines[3] = "source change"
	write(t, root, "script.sh", strings.Join(lines, "\n")+"\n")
	git(t, root, "commit", "-am", "source")
	source := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	lines[60] = "later edit"
	write(t, root, "script.sh", strings.Join(lines, "\n")+"\n")
	git(t, root, "commit", "-am", "later")
	r := openTest(t, root)
	status, err := r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p, err := r.PrepareCommitFileUndo(context.Background(), source, "script.sh", status)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	index := git(t, root, "ls-files", "--stage", "-z")
	if err := publishUndo(t, r, context.Background(), p); err != nil {
		t.Fatal(err)
	}
	base[60] = "later edit"
	expected := strings.Join(base, "\n") + "\n"
	got, err := os.ReadFile(filepath.Join(root, "script.sh"))
	if err != nil || string(got) != expected {
		t.Fatal("published result lost later edit")
	}
	info, err := os.Stat(filepath.Join(root, "script.sh"))
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatal("publication lost executable mode")
	}
	if git(t, root, "ls-files", "--stage", "-z") != index || strings.TrimSpace(git(t, root, "rev-parse", "HEAD")) != status.Head {
		t.Fatal("publication changed history or index")
	}
	// A consumed result must not be usable a second time, even with its source
	// bytes restored by a separate user edit.
	write(t, root, "script.sh", strings.Join(lines, "\n")+"\n")
	if err := publishUndo(t, r, context.Background(), p); err == nil {
		t.Fatal("consumed preparation accepted twice")
	}
}

func TestPrepareCommitFileUndoRejectsCRLFConversion(t *testing.T) {
	r, _, source, _ := undoCheckFixture(t, "selected.txt")
	git(t, r.Root, "config", "core.autocrlf", "true")
	write(t, r.Root, "selected.txt", "source selected\r\n")
	git(t, r.Root, "add", "--renormalize", "--", "selected.txt")
	status, err := r.Status(context.Background())
	if err != nil || len(status.Entries) != 0 {
		t.Fatalf("CRLF fixture must be Git-clean: %+v %v", status, err)
	}
	p, err := r.PrepareCommitFileUndo(context.Background(), source, "selected.txt", status)
	if p != nil {
		defer p.Close()
	}
	if err == nil {
		t.Fatal("unsupported line-ending conversion accepted")
	}
	got, err := os.ReadFile(filepath.Join(r.Root, "selected.txt"))
	if err != nil || string(got) != "source selected\r\n" {
		t.Fatal("conversion rejection changed working bytes")
	}
}
