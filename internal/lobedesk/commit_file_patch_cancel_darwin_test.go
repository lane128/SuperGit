package lobedesk

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// A controlled Git-process fixture acknowledges a real write to the adjacent
// temporary file, then blocks until the app cancels it. Metadata calls still
// use real Git. No sleeps, filesystem polling or production hooks are needed.
func TestCommitFilePatchPartialWriteCancellation(t *testing.T) {
	a, callbacks, tt := commitPatchFixture(t)
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	tools, output := t.TempDir(), t.TempDir()
	ackPath, waitPath := filepath.Join(tools, "ack"), filepath.Join(tools, "wait")
	for _, path := range []string{ackPath, waitPath} {
		if err := syscall.Mkfifo(path, 0600); err != nil {
			t.Fatalf("create fixture pipe: %v", err)
		}
	}
	ack, err := os.OpenFile(ackPath, os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer ack.Close()
	script := `#!/bin/sh
case "$*" in
 *--binary*)
   printf 'owned partial patch bytes\n'
   printf 'R' > "$PATCH_FIXTURE_ACK"
   IFS= read -r ignored < "$PATCH_FIXTURE_WAIT"
   ;;
 *) exec "$PATCH_FIXTURE_REAL_GIT" "$@" ;;
esac
`
	if err := os.WriteFile(filepath.Join(tools, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATCH_FIXTURE_REAL_GIT", realGit)
	t.Setenv("PATCH_FIXTURE_ACK", ackPath)
	t.Setenv("PATCH_FIXTURE_WAIT", waitPath)
	t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	destination := filepath.Join(output, "existing.patch")
	original := []byte("preserve this destination\n")
	if err := os.WriteFile(destination, original, 0600); err != nil {
		t.Fatal(err)
	}
	a.saveFileDialog = func(mygo.SaveDialogOptions) (string, error) { return destination, nil }
	acknowledged := make(chan error, 1)
	go func() { var value [1]byte; _, err := io.ReadFull(ack, value[:]); acknowledged <- err }()
	activatePatchExport(t, a, tt)
	if !a.busy || a.opCancel == nil {
		t.Fatal("export did not enter cancellable operation")
	}
	// Cleanup prevents a failed assertion from leaving the controlled child alive.
	cancel := a.opCancel
	defer cancel()
	select {
	case err := <-acknowledged:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("partial write was not acknowledged")
	}
	temporary, err := filepath.Glob(filepath.Join(output, ".supergit-patch-*"))
	if err != nil || len(temporary) != 1 {
		t.Fatalf("missing active temporary file: %v %v", temporary, err)
	}
	partial, err := os.ReadFile(temporary[0])
	if err != nil || !bytes.Equal(partial, []byte("owned partial patch bytes\n")) {
		t.Fatalf("no proven partial write: %q %v", partial, err)
	}
	cancel()
	waitSelectionOperation(t, a, callbacks)
	got, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("mid-write cancellation replaced destination")
	}
	remaining, err := filepath.Glob(filepath.Join(output, ".supergit-patch-*"))
	if err != nil || len(remaining) != 0 {
		t.Fatal("mid-write cancellation left temporary files")
	}
	if a.busy || a.opCancel != nil || len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].title != a.t("operationCancelled") || a.notices[toastMain][0].kind != "info" {
		t.Fatal("cancellation left busy state or reported success/failure")
	}
}
