package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareCommitFileUndoKeepsOriginalAndLaterChanges(t *testing.T) {
	root := fixture(t)
	lines := make([]string, 80)
	for i := range lines {
		lines[i] = strings.Repeat("x", i+1)
	}
	original := append([]string(nil), lines...)
	write(t, root, "script.sh", strings.Join(lines, "\n")+"\n")
	if err := os.Chmod(filepath.Join(root, "script.sh"), 0755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "control.txt", "original control\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "base")
	lines[3] = "source change"
	write(t, root, "script.sh", strings.Join(lines, "\n")+"\n")
	write(t, root, "control.txt", "source control\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "source")
	source := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	lines[60] = "later change"
	working := strings.Join(lines, "\n") + "\n"
	write(t, root, "script.sh", working)
	git(t, root, "commit", "-am", "later")
	r := openTest(t, root)
	status, err := r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	head, index := git(t, root, "rev-parse", "HEAD"), git(t, root, "ls-files", "--stage", "-z")
	prepared, err := r.PrepareCommitFileUndo(context.Background(), source, "script.sh", status)
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.Close()
	got, err := os.ReadFile(prepared.result)
	if err != nil {
		t.Fatal(err)
	}
	original[60] = "later change"
	if string(got) != strings.Join(original, "\n")+"\n" {
		t.Fatal("prepared result lost later content or did not undo source")
	}
	info, err := os.Stat(prepared.result)
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatal("prepared result lost executable permission")
	}
	current, err := os.ReadFile(filepath.Join(root, "script.sh"))
	if err != nil || string(current) != working || git(t, root, "rev-parse", "HEAD") != head || git(t, root, "ls-files", "--stage", "-z") != index || git(t, root, "status", "--porcelain=v1", "-z") != "" {
		t.Fatal("preparation changed real worktree, HEAD or index")
	}
	directory := prepared.directory
	if err := prepared.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatal("preparation cleanup left scratch files")
	}
}
func TestPrepareCommitFileUndoFailureCleansScratch(t *testing.T) {
	useDirectGitForTempAssertions(t)
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "conflict", true: "cancelled"}[cancelled], func(t *testing.T) {
			r, _, source, _ := undoCheckFixture(t, "selected.txt")
			write(t, r.Root, "selected.txt", "later conflicting content\n")
			git(t, r.Root, "commit", "-am", "later conflict")
			status, err := r.Status(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			scratch := t.TempDir()
			t.Setenv("TMPDIR", scratch)
			t.Setenv("TMP", scratch)
			t.Setenv("TEMP", scratch)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelled {
				cancel()
			}
			prepared, err := r.PrepareCommitFileUndo(ctx, source, "selected.txt", status)
			prepareErr := err
			if prepared != nil {
				prepared.Close()
			}
			if err == nil {
				t.Fatal("conflicting/cancelled preparation succeeded")
			}
			entries, err := os.ReadDir(scratch)
			if err != nil || len(entries) != 0 {
				t.Fatalf("failed preparation left scratch files: scratch=%s entries=%v readErr=%v prepareErr=%v", scratch, entries, err, prepareErr)
			}
			got, err := os.ReadFile(filepath.Join(r.Root, "selected.txt"))
			gitStatus := git(t, r.Root, "status", "--porcelain=v1", "-z")
			if err != nil || string(got) != "later conflicting content\n" || gitStatus != "" {
				t.Fatalf("failed preparation changed original worktree: root=%s bytes=%q readErr=%v status=%q prepareErr=%v", r.Root, got, err, gitStatus, prepareErr)
			}
		})
	}
}
