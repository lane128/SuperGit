package gitrepo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishCommitFileUndoReportsReplacementAndCleanupPermissionFailures(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can bypass the permission-rejection fixture")
	}
	r, _, source, status := undoCheckFixture(t, "selected.txt")
	p, err := r.PrepareCommitFileUndo(context.Background(), source, "selected.txt", status)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	rootInfo, err := os.Stat(r.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(r.Root, rootInfo.Mode().Perm())
	head, index := git(t, r.Root, "rev-parse", "HEAD"), git(t, r.Root, "ls-files", "--stage", "-z")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	tools := t.TempDir()
	script := `#!/bin/sh
case "$*" in
 *"config --type=bool-or-str"*)
   for candidate in "$UNDO_PERMISSION_ROOT"/.supergit-undo-*; do
     if [ -f "$candidate" ]; then
       chmod 0500 "$UNDO_PERMISSION_ROOT" || exit 19
       break
     fi
   done
   ;;
esac
exec "$UNDO_PERMISSION_REAL_GIT" "$@"
`
	if err := os.WriteFile(filepath.Join(tools, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("UNDO_PERMISSION_REAL_GIT", realGit)
	t.Setenv("UNDO_PERMISSION_ROOT", r.Root)
	t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	err = r.ApplyPreparedCommitFileUndo(context.Background(), p)
	// Restore only the owned fixture's directory permission before inspecting and
	// removing the owned temp. Production must report both actual failures.
	if restoreErr := os.Chmod(r.Root, rootInfo.Mode().Perm()); restoreErr != nil {
		t.Fatal(restoreErr)
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok || len(joined.Unwrap()) != 2 {
		t.Fatalf("replacement and cleanup errors were not both reported: %v", err)
	}
	failures := joined.Unwrap()
	for _, failure := range failures {
		if !errors.Is(failure, os.ErrPermission) {
			t.Fatalf("wrong failure instead of filesystem permission rejection: %v", failure)
		}
	}
	var replacement *os.LinkError
	var cleanup *os.PathError
	if !errors.As(failures[0], &replacement) || !strings.HasPrefix(replacement.Op, "rename") || replacement.New != "selected.txt" || !errors.As(failures[1], &cleanup) || !strings.HasPrefix(cleanup.Op, "remove") || cleanup.Path != replacement.Old {
		t.Fatalf("did not reach replacement and cleanup of the same owned file: %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(r.Root, "selected.txt"))
	if readErr != nil || string(got) != "source selected\n" || git(t, r.Root, "rev-parse", "HEAD") != head || git(t, r.Root, "ls-files", "--stage", "-z") != index {
		t.Fatal("failed replacement changed source")
	}
	temps, globErr := filepath.Glob(filepath.Join(r.Root, ".supergit-undo-*"))
	if globErr != nil || len(temps) != 1 {
		t.Fatalf("expected exactly the reported owned temp: %v %v", temps, globErr)
	}
	temp, readErr := os.ReadFile(temps[0])
	if readErr != nil || string(temp) != "original selected\n" {
		t.Fatal("failure fixture did not reach prepared adjacent output")
	}
	if filepath.Base(temps[0]) != cleanup.Path || !strings.Contains(err.Error(), cleanup.Path) {
		t.Fatal("returned failure lost residual temp path")
	}
	if err := os.Remove(temps[0]); err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Lstat(temps[0]); !os.IsNotExist(statErr) {
		t.Fatalf("owned cleanup left temp: %v", statErr)
	}
	if git(t, r.Root, "status", "--porcelain=v1", "-z") != "" {
		t.Fatal("owned fixture cleanup did not restore clean state")
	}
}
