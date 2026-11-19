package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type commitFileUndoChecker interface {
	CheckCommitFileUndo(context.Context, string, string, Status) error
}

func undoChecker(t *testing.T, r *Repository) commitFileUndoChecker {
	t.Helper()
	check, ok := any(r).(commitFileUndoChecker)
	if !ok {
		t.Fatal("historical file undo eligibility is unavailable")
	}
	return check
}
func undoCheckFixture(t *testing.T, path string) (*Repository, string, string, Status) {
	t.Helper()
	root := fixture(t)
	write(t, root, path, "original selected\n")
	write(t, root, "other/control.txt", "unchanged control\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "base")
	base := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	write(t, root, path, "source selected\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "source")
	source := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	r := openTest(t, root)
	status, err := r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return r, base, source, status
}
func assertUndoCheckReadOnly(t *testing.T, r *Repository, sha, path string, expected Status, wantAllowed bool) {
	t.Helper()
	head, index, status := git(t, r.Root, "rev-parse", "HEAD"), git(t, r.Root, "ls-files", "--stage", "-z"), git(t, r.Root, "status", "--porcelain=v1", "-z")
	before, readErr := os.ReadFile(filepath.Join(r.Root, path))
	err := undoChecker(t, r).CheckCommitFileUndo(context.Background(), sha, path, expected)
	if (err == nil) != wantAllowed {
		t.Fatalf("allowed=%v, error=%v", wantAllowed, err)
	}
	if git(t, r.Root, "rev-parse", "HEAD") != head || git(t, r.Root, "ls-files", "--stage", "-z") != index || git(t, r.Root, "status", "--porcelain=v1", "-z") != status {
		t.Fatal("eligibility check changed Git state")
	}
	after, afterErr := os.ReadFile(filepath.Join(r.Root, path))
	if readErr == nil && (afterErr != nil || string(before) != string(after)) {
		t.Fatal("eligibility check changed worktree bytes")
	}
}
func TestCommitFileUndoEligibilityRejectsHiddenWorkingEdits(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		t.Run(flag, func(t *testing.T) {
			r, _, source, status := undoCheckFixture(t, "selected.txt")
			git(t, r.Root, "update-index", flag, "--", "selected.txt")
			write(t, r.Root, "selected.txt", "hidden working edit\n")
			hidden, err := r.Status(context.Background())
			if err != nil || len(hidden.Entries) != 0 {
				t.Fatal("fixture must demonstrate status-hidden edits")
			}
			assertUndoCheckReadOnly(t, r, source, "selected.txt", status, false)
		})
	}
}
func TestCommitFileUndoEligibilityLiteralTextAndLaterCommit(t *testing.T) {
	path := ":(glob)*.txt"
	r, _, source, _ := undoCheckFixture(t, path)
	write(t, r.Root, "later.txt", "later independent content\n")
	git(t, r.Root, "add", "--all")
	git(t, r.Root, "commit", "-m", "later")
	status, err := r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertUndoCheckReadOnly(t, r, source, path, status, true)
	assertUndoCheckReadOnly(t, r, source, "other/control.txt", status, false)
}
func TestCommitFileUndoEligibilityRejectsStaleDirtyAndNonAncestor(t *testing.T) {
	for _, kind := range []string{"head", "branch", "working", "index", "operation", "non-ancestor", "root", "invalid-sha", "invalid-path"} {
		t.Run(kind, func(t *testing.T) {
			r, base, source, status := undoCheckFixture(t, "selected.txt")
			path := "selected.txt"
			switch kind {
			case "head":
				write(t, r.Root, "later.txt", "new head\n")
				git(t, r.Root, "add", "--all")
				git(t, r.Root, "commit", "-m", "later")
			case "branch":
				git(t, r.Root, "checkout", "-b", "other")
			case "working":
				write(t, r.Root, path, "dirty working\n")
			case "index":
				write(t, r.Root, path, "dirty index\n")
				git(t, r.Root, "add", "--", path)
			case "operation":
				gitDir := strings.TrimSpace(git(t, r.Root, "rev-parse", "--absolute-git-dir"))
				if err := os.WriteFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD"), []byte(source+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "non-ancestor":
				git(t, r.Root, "checkout", "-b", "foreign", base)
				write(t, r.Root, path, "foreign source\n")
				git(t, r.Root, "commit", "-am", "foreign")
				source = strings.TrimSpace(git(t, r.Root, "rev-parse", "HEAD"))
				git(t, r.Root, "checkout", "main")
			case "root":
				source = base
			case "invalid-sha":
				source = "invalid"
			case "invalid-path":
				path = "../outside"
			}
			assertUndoCheckReadOnly(t, r, source, path, status, false)
		})
	}
}
func TestCommitFileUndoEligibilityRejectsAttributeBinaryAndCancellation(t *testing.T) {
	r, _, source, _ := undoCheckFixture(t, "selected.txt")
	write(t, r.Root, ".gitattributes", "selected.txt -diff\n")
	git(t, r.Root, "add", "--all")
	git(t, r.Root, "commit", "-m", "binary attribute")
	status, err := r.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertUndoCheckReadOnly(t, r, source, "selected.txt", status, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := undoChecker(t, r).CheckCommitFileUndo(ctx, source, "selected.txt", status); err == nil {
		t.Fatal("cancelled eligibility check succeeded")
	}
}
func TestCommitFileUndoEligibilityRejectsUnsupportedCurrentTarget(t *testing.T) {
	for _, kind := range []string{"deleted", "renamed", "binary", "mode"} {
		t.Run(kind, func(t *testing.T) {
			r, _, source, _ := undoCheckFixture(t, "selected.txt")
			switch kind {
			case "deleted":
				git(t, r.Root, "rm", "--", "selected.txt")
			case "renamed":
				git(t, r.Root, "mv", "--", "selected.txt", "moved.txt")
			case "binary":
				write(t, r.Root, "selected.txt", "source selected\nnow binary\x00payload\n")
				git(t, r.Root, "add", "--all")
			case "mode":
				if err := os.Chmod(filepath.Join(r.Root, "selected.txt"), 0755); err != nil {
					t.Fatal(err)
				}
				git(t, r.Root, "add", "--all")
				git(t, r.Root, "update-index", "--chmod=+x", "--", "selected.txt")
			}
			git(t, r.Root, "commit", "-m", "later target change")
			status, err := r.Status(context.Background())
			if err != nil || len(status.Entries) != 0 {
				t.Fatal("later target fixture must be clean")
			}
			assertUndoCheckReadOnly(t, r, source, "selected.txt", status, false)
		})
	}
}
func TestCommitFileUndoEligibilityRejectsModesBinaryAndRename(t *testing.T) {
	for _, kind := range []string{"chmod-only", "text-and-chmod", "binary", "rename", "addition", "deletion", "merge"} {
		t.Run(kind, func(t *testing.T) {
			r, _, _, _ := undoCheckFixture(t, "selected.txt")
			path := "selected.txt"
			switch kind {
			case "chmod-only", "text-and-chmod":
				if kind == "text-and-chmod" {
					write(t, r.Root, path, "changed with mode\n")
				}
				if err := os.Chmod(filepath.Join(r.Root, path), 0755); err != nil {
					t.Fatal(err)
				}
				git(t, r.Root, "add", "--", path)
				git(t, r.Root, "update-index", "--chmod=+x", "--", path)
			case "binary":
				write(t, r.Root, path, "binary\x00payload\n")
				git(t, r.Root, "add", "--", path)
			case "rename":
				git(t, r.Root, "mv", "--", path, "renamed.txt")
				path = "renamed.txt"
			case "addition":
				path = "added.txt"
				write(t, r.Root, path, "added\n")
				git(t, r.Root, "add", "--", path)
			case "deletion":
				git(t, r.Root, "rm", "--", path)
			case "merge":
				git(t, r.Root, "checkout", "-b", "side")
				write(t, r.Root, path, "side change\n")
				git(t, r.Root, "commit", "-am", "side")
				git(t, r.Root, "checkout", "main")
				write(t, r.Root, "separate.txt", "main change\n")
				git(t, r.Root, "add", "--all")
				git(t, r.Root, "commit", "-m", "main")
				git(t, r.Root, "merge", "--no-ff", "side", "-m", "merge")
			}
			if kind != "merge" {
				git(t, r.Root, "commit", "-m", kind)
			}
			source := strings.TrimSpace(git(t, r.Root, "rev-parse", "HEAD"))
			status, err := r.Status(context.Background())
			if err != nil || len(status.Entries) != 0 {
				t.Fatalf("unsupported fixture is not clean: %+v %v", status, err)
			}
			assertUndoCheckReadOnly(t, r, source, path, status, false)
		})
	}
}
