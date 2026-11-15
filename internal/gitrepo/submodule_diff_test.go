package gitrepo

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// A submodule change is a pointer, and the diff has to show it as one:
// `--submodule=short` prints the usual patch with the "160000" mode, which
// also tells the commit details that this file cannot be blamed. The log
// form git prints by default puts no `diff --git` header in the output, so
// the details lost the file entirely.
func TestSubmoduleDiffsShowThePointer(t *testing.T) {
	ctx := context.Background()
	child := fixture(t)
	seed(t, child)
	first := strings.TrimSpace(git(t, child, "rev-parse", "HEAD"))
	write(t, child, "base.txt", "changed\n")
	git(t, child, "commit", "-am", "child change")
	second := strings.TrimSpace(git(t, child, "rev-parse", "HEAD"))

	parent := fixture(t)
	seed(t, parent)
	// A user who prefers the log form must not decide what the details read:
	// without --submodule=short, git's log summary has no per-file sections.
	git(t, parent, "config", "diff.submodule", "log")
	git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "sub")
	git(t, parent, "commit", "-m", "add submodule")
	added := strings.TrimSpace(git(t, parent, "rev-parse", "HEAD"))
	git(t, filepath.Join(parent, "sub"), "checkout", "-q", first)
	git(t, parent, "add", "sub")
	git(t, parent, "commit", "-m", "update submodule")
	updated := strings.TrimSpace(git(t, parent, "rev-parse", "HEAD"))
	r := openTest(t, parent)

	for _, test := range []struct {
		name string
		sha  string
	}{
		{"adding a submodule", added},
		{"moving its pointer", updated},
	} {
		diffs, err := r.ShowAll(ctx, test.sha)
		if err != nil {
			t.Fatalf("%s: ShowAll: %v", test.name, err)
		}
		var found *FileDiff
		for i := range diffs {
			if diffs[i].Path == "sub" {
				found = &diffs[i]
				break
			}
		}
		if found == nil {
			t.Fatalf("%s: the commit's submodule change is not in the diff at all: %+v", test.name, diffs)
		}
		if !strings.Contains(found.Diff.Text, "160000") {
			t.Errorf("%s: the patch does not carry the gitlink mode:\n%s", test.name, found.Diff.Text)
		}
		if !strings.Contains(found.Diff.Text, "Subproject commit") {
			t.Errorf("%s: the patch does not show the pointer:\n%s", test.name, found.Diff.Text)
		}
		if found.Diff.CanStageHunks {
			t.Errorf("%s: submodule pointer lines counted as stageable hunks", test.name)
		}
	}

	// The worktree diff reads the same format while the pointer is unstaged.
	git(t, filepath.Join(parent, "sub"), "checkout", "-q", second)
	diff, err := r.Diff(ctx, "sub", false)
	if err != nil {
		t.Fatalf("worktree diff: %v", err)
	}
	if !strings.Contains(diff.Text, "160000") || !strings.Contains(diff.Text, "Subproject commit") {
		t.Errorf("worktree diff lost the pointer:\n%s", diff.Text)
	}
}

// A gitlink can be replaced by an ordinary file at the same path. The commit
// then carries two sections for that path -- the gitlink deleted and the file
// created -- and both have to survive the split, since only the pair says what
// the revision holds now.
func TestReplacingAGitlinkWithAFileShowsBothSides(t *testing.T) {
	ctx := context.Background()
	child := fixture(t)
	seed(t, child)
	parent := fixture(t)
	seed(t, parent)
	git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "sub")
	git(t, parent, "commit", "-m", "add submodule")
	git(t, parent, "rm", "-f", "sub")
	write(t, parent, "sub", "now a file\n")
	git(t, parent, "add", "sub")
	git(t, parent, "commit", "-m", "gitlink becomes a file")
	sha := strings.TrimSpace(git(t, parent, "rev-parse", "HEAD"))
	r := openTest(t, parent)

	diffs, err := r.ShowAll(ctx, sha)
	if err != nil {
		t.Fatalf("ShowAll: %v", err)
	}
	var sides []string
	for _, diff := range diffs {
		if diff.Path == "sub" {
			sides = append(sides, diff.Diff.Text)
		}
	}
	if len(sides) != 2 {
		t.Fatalf("the type change produced %d sections for sub; want 2:\n%s", len(sides), strings.Join(sides, "\n---\n"))
	}
	if !strings.Contains(sides[0]+sides[1], "deleted file mode 160000") {
		t.Errorf("the deleted gitlink is missing from the sections:\n%s", strings.Join(sides, "\n---\n"))
	}
	if !strings.Contains(sides[0]+sides[1], "new file mode 100644") {
		t.Errorf("the file created in its place is missing from the sections:\n%s", strings.Join(sides, "\n---\n"))
	}
}
