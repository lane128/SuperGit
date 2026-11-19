package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func conflictEditorFixture(t *testing.T) (*app, <-chan func(), string) {
	t.Helper()
	return conflictEditorContentFixture(t, "current content\n", "incoming content\n")
}

func conflictEditorContentFixture(t *testing.T, current, incoming string) (*app, <-chan func(), string) {
	t.Helper()
	a, _, root := worktreeApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	t.Cleanup(cancel)
	done := make(chan func(), 128)
	a.runOnMain = func(fn func()) { done <- fn }
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	splitCommitGit(t, root, "checkout", "-b", "incoming")
	write(incoming)
	splitCommitGit(t, root, "commit", "-am", "Incoming change")
	splitCommitGit(t, root, "checkout", "main")
	write(current)
	splitCommitGit(t, root, "commit", "-am", "Current change")
	if out, err := exec.Command("git", "-C", root, "merge", "--no-edit", "incoming").CombinedOutput(); err == nil {
		t.Fatalf("expected conflict: %s", out)
	}
	var err error
	a.status, err = a.repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.selectedPath, a.worktreeSelected = "base.txt", true
	a.diff, err = a.repo.Diff(ctx, "base.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	a.locale = localeEnglish
	return a, done, root
}

func conflictEditorClick(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	item, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok || item.Offscreen || item.Bounds.W <= 0 || item.Bounds.H <= 0 || item.Disabled {
		t.Fatalf("missing usable button %q", label)
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
}

func TestConflictEditorSavesCombinedResultBeforeMergeCommit(t *testing.T) {
	a, done, root := conflictEditorFixture(t)
	tt := pageTester(a, a.diffBody)
	conflictEditorClick(t, tt, "Compare and edit result")
	runContinuation(t, done)
	tt.Frame()
	field, ok := tt.AccessByRole(ui.RoleTextField, "Resolved contents")
	if !ok || field.Offscreen {
		t.Fatal("missing editable result")
	}
	tt.ClickAt(field.Bounds.X+20, field.Bounds.Y+20)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("combined current and incoming\n")
	head := splitCommitGit(t, root, "rev-parse", "HEAD")
	incoming := splitCommitGit(t, root, "rev-parse", "MERGE_HEAD")
	conflictEditorClick(t, tt, "Save and mark resolved")
	runContinuation(t, done)
	if got := splitCommitGit(t, root, "show", ":base.txt"); got != "combined current and incoming\n" {
		t.Fatalf("Index = %q", got)
	}
	if splitCommitGit(t, root, "rev-parse", "HEAD") != head || splitCommitGit(t, root, "rev-parse", "MERGE_HEAD") != incoming {
		t.Fatal("saving unexpectedly completed merge")
	}
	// Drain the normal refresh before finishing through the existing commit path.
	deadline := time.After(10 * time.Second)
	for a.loading || a.diffLoading || a.commitFilesLoading || a.commitDiffLoading {
		select {
		case fn := <-done:
			fn()
		case <-deadline:
			t.Fatal("refresh timed out")
		}
	}
	a.commitText = "Combine changes\n\nKeep both intents."
	if !a.canCommit() {
		t.Fatalf("resolved merge cannot commit: loading=%v busy=%v status=%+v", a.loading, a.busy, a.status)
	}
	a.commit()
	for a.busy {
		runContinuation(t, done)
	}
	if got := splitCommitGit(t, root, "show", "HEAD:base.txt"); got != "combined current and incoming\n" {
		t.Fatalf("commit = %q", got)
	}
	if got := splitCommitGit(t, root, "show", "-s", "--format=%P", "HEAD"); got != head[:len(head)-1]+" "+incoming {
		t.Fatalf("parents = %q", got)
	}
}
