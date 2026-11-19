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

func TestPublishCommitFileUndoInterruptionAfterAdjacentCopy(t *testing.T) {
	for _, kind := range []string{"cancel", "external-edit"} {
		t.Run(kind, func(t *testing.T) {
			r, _, source, status := undoCheckFixture(t, "selected.txt")
			p, err := r.PrepareCommitFileUndo(context.Background(), source, "selected.txt", status)
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			head, index := git(t, r.Root, "rev-parse", "HEAD"), git(t, r.Root, "ls-files", "--stage", "-z")
			realGit, err := exec.LookPath("git")
			if err != nil {
				t.Fatal(err)
			}
			tools := t.TempDir()
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
			release, err := os.OpenFile(waitPath, os.O_RDWR, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer release.Close()
			script := `#!/bin/sh
case "$*" in
 *"config --type=bool-or-str"*)
   for candidate in "$UNDO_PUBLISH_ROOT"/.supergit-undo-*; do
     if [ -f "$candidate" ]; then
       printf 'R' > "$UNDO_PUBLISH_ACK"
       IFS= read -r ignored < "$UNDO_PUBLISH_WAIT"
       break
     fi
   done
   ;;
esac
exec "$UNDO_PUBLISH_REAL_GIT" "$@"
`
			if err := os.WriteFile(filepath.Join(tools, "git"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("UNDO_PUBLISH_REAL_GIT", realGit)
			t.Setenv("UNDO_PUBLISH_ROOT", r.Root)
			t.Setenv("UNDO_PUBLISH_ACK", ackPath)
			t.Setenv("UNDO_PUBLISH_WAIT", waitPath)
			t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			finished := make(chan error, 1)
			go func() { finished <- r.ApplyPreparedCommitFileUndo(ctx, p) }()
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
			case err := <-finished:
				t.Fatalf("write finished before interruption: %v", err)
			case <-time.After(10 * time.Second):
				t.Fatal("adjacent-copy boundary not reached")
			}
			copies, err := filepath.Glob(filepath.Join(r.Root, ".supergit-undo-*"))
			if err != nil || len(copies) != 1 {
				t.Fatal("adjacent temp missing")
			}
			got, err := os.ReadFile(copies[0])
			if err != nil || string(got) != "original selected\n" {
				t.Fatal("adjacent result not complete")
			}
			select {
			case err := <-finished:
				t.Fatalf("write returned before interruption: %v", err)
			default:
			}
			expected := "source selected\n"
			if kind == "cancel" {
				cancel()
			} else {
				expected = "external edit after adjacent copy\n"
				write(t, r.Root, "selected.txt", expected)
				if _, err := release.WriteString("continue\n"); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case err := <-finished:
				if err == nil {
					t.Fatal("interrupted write succeeded")
				}
				if kind == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatalf("wrong cancel error: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("interrupted write did not return")
			}
			got, err = os.ReadFile(filepath.Join(r.Root, "selected.txt"))
			if err != nil || string(got) != expected {
				t.Fatal("interruption overwrote original or external edit")
			}
			copies, err = filepath.Glob(filepath.Join(r.Root, ".supergit-undo-*"))
			if err != nil || len(copies) != 0 {
				t.Fatal("interruption leaked adjacent temp")
			}
			if git(t, r.Root, "rev-parse", "HEAD") != head || git(t, r.Root, "ls-files", "--stage", "-z") != index {
				t.Fatal("interruption changed history or index")
			}
		})
	}
}
