package gitrepo

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A controlled process acknowledges the init stage after the original file was
// copied to scratch. Cancellation then exercises real process and temp cleanup.
func TestPrepareCommitFileUndoCancelAfterScratchCopy(t *testing.T) {
	r, _, source, status := undoCheckFixture(t, "selected.txt")
	original, err := os.ReadFile(filepath.Join(r.Root, "selected.txt"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(r.Root, "selected.txt"))
	if err != nil {
		t.Fatal(err)
	}
	head, index := git(t, r.Root, "rev-parse", "HEAD"), git(t, r.Root, "ls-files", "--stage", "-z")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	tools, scratch := t.TempDir(), t.TempDir()
	ackPath, waitPath := filepath.Join(tools, "ack"), filepath.Join(tools, "wait")
	for _, path := range []string{ackPath, waitPath} {
		if err := syscall.Mkfifo(path, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ack, err := os.OpenFile(ackPath, os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer ack.Close()
	script := `#!/bin/sh
case "$*" in
 *"init --quiet --template="*)
   printf 'R' > "$UNDO_FIXTURE_ACK"
   IFS= read -r ignored < "$UNDO_FIXTURE_WAIT"
   ;;
 *) exec "$UNDO_FIXTURE_REAL_GIT" "$@" ;;
esac
`
	if err := os.WriteFile(filepath.Join(tools, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("UNDO_FIXTURE_REAL_GIT", realGit)
	t.Setenv("UNDO_FIXTURE_ACK", ackPath)
	t.Setenv("UNDO_FIXTURE_WAIT", waitPath)
	t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TMPDIR", scratch)
	t.Setenv("TMP", scratch)
	t.Setenv("TEMP", scratch)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type outcome struct {
		prepared *PreparedCommitFileUndo
		err      error
	}
	finished := make(chan outcome, 1)
	go func() {
		p, err := r.PrepareCommitFileUndo(ctx, source, "selected.txt", status)
		finished <- outcome{p, err}
	}()
	acknowledged := make(chan error, 1)
	go func() {
		var value [1]byte
		_, err := io.ReadFull(ack, value[:])
		if err == nil && value[0] != 'R' {
			err = errors.New("unexpected acknowledgement")
		}
		acknowledged <- err
	}()
	select {
	case err := <-acknowledged:
		if err != nil {
			t.Fatal(err)
		}
	case result := <-finished:
		if result.prepared != nil {
			result.prepared.Close()
		}
		t.Fatalf("preparation finished before scratch acknowledgement: %v", result.err)
	case <-time.After(10 * time.Second):
		t.Fatal("scratch stage not reached")
	}
	entries, err := os.ReadDir(scratch)
	if err != nil || len(entries) != 1 {
		t.Fatal("acknowledgement did not prove owned scratch exists")
	}
	copied, err := os.ReadFile(filepath.Join(scratch, entries[0].Name(), "worktree", "selected.txt"))
	if err != nil || string(copied) != string(original) {
		t.Fatal("scratch copy was not complete before cancellation")
	}
	copiedInfo, err := os.Stat(filepath.Join(scratch, entries[0].Name(), "worktree", "selected.txt"))
	if err != nil || copiedInfo.Mode().Perm() != info.Mode().Perm() {
		t.Fatal("scratch copy lost original permissions")
	}
	select {
	case result := <-finished:
		if result.prepared != nil {
			result.prepared.Close()
		}
		t.Fatalf("preparation finished before cancel: %v", result.err)
	default:
	}
	cancel()
	select {
	case result := <-finished:
		if result.prepared != nil {
			result.prepared.Close()
			t.Fatal("cancelled preparation returned a usable result")
		}
		if !errors.Is(result.err, context.Canceled) {
			t.Fatalf("wrong cancellation result: %v", result.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cancelled preparation did not finish")
	}
	entries, err = os.ReadDir(scratch)
	if err != nil || len(entries) != 0 {
		t.Fatal("post-copy cancellation leaked scratch")
	}
	after, err := os.ReadFile(filepath.Join(r.Root, "selected.txt"))
	if err != nil || string(after) != string(original) {
		t.Fatal("post-copy cancellation changed original bytes")
	}
	afterInfo, err := os.Stat(filepath.Join(r.Root, "selected.txt"))
	if err != nil || afterInfo.Mode() != info.Mode() || git(t, r.Root, "rev-parse", "HEAD") != head || git(t, r.Root, "ls-files", "--stage", "-z") != index || git(t, r.Root, "status", "--porcelain=v1", "-z") != "" {
		t.Fatal("post-copy cancellation changed source mode or Git state")
	}
}
