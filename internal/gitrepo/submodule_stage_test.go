package gitrepo

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// A submodule whose content is dirty but whose recorded object did not move
// produces a body that looks like a one-line patch: "-Subproject commit X" and
// "+Subproject commit X-dirty", with no index line. git apply accepts it and
// stages nothing, so neither the diff nor ApplyHunk may offer it as a hunk.
func TestDirtySubmoduleHunksAreNotStageable(t *testing.T) {
	ctx := context.Background()
	child := fixture(t)
	seed(t, child)
	parent := fixture(t)
	seed(t, parent)
	git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "sub")
	git(t, parent, "commit", "-m", "add submodule")
	// Modify a tracked file inside the submodule; its HEAD stays put.
	write(t, filepath.Join(parent, "sub"), "base.txt", "changed\n")
	r := openTest(t, parent)

	d, err := r.Diff(ctx, "sub", false)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(d.Text, "Subproject commit") || !strings.Contains(d.Text, "-dirty") {
		t.Fatalf("the fixture is not a dirty submodule diff:\n%s", d.Text)
	}
	if d.CanStageHunks {
		t.Errorf("a dirty submodule's pointer body counted as a stageable hunk:\n%s", d.Text)
	}
	if len(d.Hunks) != 1 {
		t.Fatalf("the dirty submodule diff has %d hunks; want the pointer's one", len(d.Hunks))
	}
	if err := r.ApplyHunk(ctx, d.Hunks[0].Patch, false); err == nil {
		t.Error("ApplyHunk accepted a dirty submodule's pseudo patch")
	}
	if staged := strings.TrimSpace(git(t, parent, "diff", "--cached", "--", "sub")); staged != "" {
		t.Errorf("the rejected hunk still staged something:\n%s", staged)
	}
}

// A moved pointer carries the gitlink mode, and the backend rejects it for the
// same reason even if a caller skipped the UI's gate.
func TestMovedSubmodulePointerHunkIsNotStageable(t *testing.T) {
	ctx := context.Background()
	child := fixture(t)
	seed(t, child)
	parent := fixture(t)
	seed(t, parent)
	git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "sub")
	git(t, parent, "commit", "-m", "add submodule")
	write(t, child, "base.txt", "child changed\n")
	git(t, child, "commit", "-am", "child change")
	second := strings.TrimSpace(git(t, child, "rev-parse", "HEAD"))
	// The submodule clone predates the new commit, so fetch it before moving
	// the pointer the way a real update would.
	git(t, filepath.Join(parent, "sub"), "-c", "protocol.file.allow=always", "fetch", "-q")
	git(t, filepath.Join(parent, "sub"), "checkout", "-q", second)
	r := openTest(t, parent)

	d, err := r.Diff(ctx, "sub", false)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if d.CanStageHunks || len(d.Hunks) == 0 {
		t.Fatalf("the moved pointer reads as %d stageable hunks; want none:\n%s", len(d.Hunks), d.Text)
	}
	if err := r.ApplyHunk(ctx, d.Hunks[0].Patch, false); err == nil {
		t.Error("ApplyHunk accepted a submodule pointer hunk")
	}
}

// A regular file can contain the words "Subproject commit" as text. It has an
// index line, so it stays stageable: the guard is the missing index, not the
// words in the body.
func TestAFileQuotingASubprojectLineStaysStageable(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "base.txt", "Subproject commit deadbeef\n")
	r := openTest(t, dir)

	d, err := r.Diff(ctx, "base.txt", false)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !d.CanStageHunks || len(d.Hunks) != 1 {
		t.Fatalf("a regular file quoting a subproject line is not stageable:\n%s", d.Text)
	}
	if err := r.ApplyHunk(ctx, d.Hunks[0].Patch, false); err != nil {
		t.Fatalf("staging the hunk: %v", err)
	}
	if staged := git(t, dir, "diff", "--cached", "--", "base.txt"); !strings.Contains(staged, "+Subproject commit deadbeef") {
		t.Errorf("the hunk did not stage the file:\n%s", staged)
	}
}
