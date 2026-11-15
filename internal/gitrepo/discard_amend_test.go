package gitrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDiscardTrackedRestoresIndexContent(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "base.txt", "working change\n")
	r := openTest(t, dir)
	if err := r.Discard(context.Background(), "base.txt", false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "base.txt"))
	if err != nil || string(data) != "original\n" {
		t.Fatalf("discard left %q, %v", data, err)
	}
	status, err := r.Status(context.Background())
	if err != nil || len(status.Entries) != 0 {
		t.Fatalf("status after discard: %#v %v", status, err)
	}
}

func TestDiscardUntrackedRemovesOnlyTheFile(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "draft.txt", "untracked\n")
	r := openTest(t, dir)
	if err := r.Discard(context.Background(), "draft.txt", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "draft.txt")); !os.IsNotExist(err) {
		t.Fatalf("draft still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "base.txt")); err != nil {
		t.Fatalf("tracked file was removed: %v", err)
	}
	if err := r.Discard(context.Background(), "../outside", true); err == nil {
		t.Fatal("path outside the repository accepted")
	}
}

func TestTakeConflictSideStagesResolvedFile(t *testing.T) {
	dir := fixture(t)
	write(t, dir, "base.txt", "base\n")
	git(t, dir, "add", "--", "base.txt")
	git(t, dir, "commit", "-m", "base")
	git(t, dir, "checkout", "-b", "side")
	write(t, dir, "base.txt", "their side\n")
	git(t, dir, "commit", "-am", "side")
	git(t, dir, "checkout", "main")
	write(t, dir, "base.txt", "our side\n")
	git(t, dir, "commit", "-am", "main")
	if cmd := exec.Command("git", "-C", dir, "merge", "side"); cmd.Run() == nil {
		t.Fatal("merge unexpectedly succeeded")
	}
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.TakeConflictSide(ctx, "base.txt", "ours"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "base.txt"))
	if err != nil || string(data) != "our side\n" {
		t.Fatalf("ours not written: %q %v", data, err)
	}
	status, err := r.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range status.Entries {
		if entry.Path == "base.txt" && !entry.Staged() {
			t.Fatalf("resolved file not staged: %#v", entry)
		}
	}
	if err := r.TakeConflictSide(ctx, "base.txt", "both"); err == nil {
		t.Fatal("invalid side accepted")
	}
}

func TestUndoCommitKeepsChangesStaged(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	write(t, dir, "extra.txt", "kept\n")
	if err := r.Stage(ctx, "extra.txt"); err != nil {
		t.Fatal(err)
	}
	if err := r.Commit(ctx, "second"); err != nil {
		t.Fatal(err)
	}
	if err := r.UndoCommit(ctx); err != nil {
		t.Fatal(err)
	}
	if got := git(t, dir, "log", "-1", "--format=%s"); got != "initial\n" {
		t.Fatalf("HEAD=%q", got)
	}
	status, err := r.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range status.Entries {
		if entry.Path == "extra.txt" && entry.Staged() {
			found = true
		}
	}
	if !found {
		t.Fatalf("undone commit not staged: %#v", status.Entries)
	}
	if err := r.UndoCommit(ctx); err == nil {
		t.Fatal("root commit was undone")
	}
}

func TestAmendCommitReplacesHeadAndKeepsStagedTree(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	before := git(t, dir, "rev-parse", "HEAD")
	write(t, dir, "extra.txt", "amended\n")
	if err := r.Stage(ctx, "extra.txt"); err != nil {
		t.Fatal(err)
	}
	if err := r.AmendCommit(ctx, "amended subject"); err != nil {
		t.Fatal(err)
	}
	after := git(t, dir, "rev-parse", "HEAD")
	if before == after {
		t.Fatal("amend kept the old SHA")
	}
	if got := git(t, dir, "log", "-1", "--format=%s"); got != "amended subject\n" {
		t.Fatalf("subject=%q", got)
	}
	if got := git(t, dir, "show", "HEAD:extra.txt"); got != "amended\n" {
		t.Fatalf("staged tree not amended: %q", got)
	}
	if got := git(t, dir, "rev-list", "--count", "HEAD"); got != "1\n" {
		t.Fatalf("commit count=%q", got)
	}
	if err := r.AmendCommit(ctx, "  "); err == nil {
		t.Fatal("empty amend message accepted")
	}
}
