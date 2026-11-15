package gitrepo

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// A hunk is staged by applying its patch, and the patch has to name the file
// the same way git does. Git quotes names that contain quotes, backslashes or
// control characters, and a name can itself contain " b/", so reading the path
// off the end of the `diff --git` line is not enough: the `+++` line is the
// authoritative one.
func TestApplyHunkStagesFilesWithAwkwardNames(t *testing.T) {
	names := []string{
		"a b/c.txt",
		`quote"q.txt`,
		`back\slash.txt`,
		"line\nbreak.txt",
		"dir b/sub/f.txt",
		"trail ",
		" lead.txt",
	}
	for _, name := range names {
		name := name
		t.Run(fmt.Sprintf("%q", name), func(t *testing.T) {
			dir := fixture(t)
			seed(t, dir)
			write(t, dir, name, "before\n")
			git(t, dir, "add", "--", name)
			git(t, dir, "commit", "-m", "add the awkward name")
			write(t, dir, name, "after\n")
			r := openTest(t, dir)

			d, err := r.Diff(context.Background(), name, false)
			if err != nil {
				t.Fatalf("Diff: %v", err)
			}
			if !d.CanStageHunks || len(d.Hunks) == 0 {
				t.Fatalf("the edit reads as %d stageable hunks:\n%s", len(d.Hunks), d.Text)
			}
			if err := r.ApplyHunk(context.Background(), d.Hunks[0].Patch, false); err != nil {
				t.Fatalf("ApplyHunk: %v\n%s", err, d.Hunks[0].Patch)
			}
			staged := git(t, dir, "diff", "--cached", "--", name)
			if !strings.Contains(staged, "+after") {
				t.Errorf("the accepted hunk staged nothing:\n%s", staged)
			}
		})
	}
}

// A staged rename leaves an entry whose OriginalPath is the old name. Its
// unstaged edit draws a hunk patch that git apply would accept, but staging a
// hunk into an index that also carries the rename is the file row's job; the
// backend refuses it so a caller that skipped the UI gate cannot split the
// rename from its edit.
func TestApplyHunkRefusesAStagedRename(t *testing.T) {
	ctx := context.Background()
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "a.txt", "one\n")
	git(t, dir, "add", "--", "a.txt")
	git(t, dir, "commit", "-m", "add a")
	git(t, dir, "mv", "a.txt", "b.txt")
	write(t, dir, "b.txt", "two\n")
	r := openTest(t, dir)

	d, err := r.Diff(ctx, "b.txt", false)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !d.CanStageHunks || len(d.Hunks) == 0 {
		t.Fatalf("the unstaged edit reads as %d stageable hunks:\n%s", len(d.Hunks), d.Text)
	}
	if err := r.ApplyHunk(ctx, d.Hunks[0].Patch, false); err == nil {
		t.Fatal("ApplyHunk accepted a hunk for a renamed entry")
	}
	if staged := git(t, dir, "diff", "--cached", "--", "b.txt"); strings.Contains(staged, "+two") {
		t.Errorf("the rejected hunk still reached the index:\n%s", staged)
	}
}

// A binary patch has no +++/--- lines and no @@ body, so the diff header is
// the only place left to read the path from. A name can itself contain
// " b/", which a split at the last one truncates.
func TestPatchPathReadsABinaryHeadersAwkwardName(t *testing.T) {
	patch := "diff --git a/dir b/sub/f.txt b/dir b/sub/f.txt\n" +
		"Binary files a/dir b/sub/f.txt and b/dir b/sub/f.txt differ\n"
	if got := patchPath(patch); got != "dir b/sub/f.txt" {
		t.Errorf("patchPath read %q; want dir b/sub/f.txt", got)
	}
}

// A deleted file's +++ line is /dev/null; the path comes from its --- line.
func TestPatchPathReadsADeletedFilesOldName(t *testing.T) {
	patch := "diff --git a/gone.txt b/gone.txt\n" +
		"deleted file mode 100644\n" +
		"--- a/gone.txt\n" +
		"+++ /dev/null\n" +
		"@@ -1 +0,0 @@\n" +
		"-bye\n"
	if got := patchPath(patch); got != "gone.txt" {
		t.Errorf("patchPath read %q; want gone.txt", got)
	}
}

// A body line can start with "+++ " too; the scan stops at the first @@ so a
// later line never becomes the file's name.
func TestPatchPathStopsAtTheHunkBody(t *testing.T) {
	patch := "diff --git a/real.txt b/real.txt\n" +
		"--- a/real.txt\n" +
		"+++ b/real.txt\n" +
		"@@ -1 +1 @@\n" +
		"+++ b/fake.txt\n"
	if got := patchPath(patch); got != "real.txt" {
		t.Errorf("patchPath read %q; want real.txt", got)
	}
}

// Git ends a path with a tab when the name itself ends in whitespace; the
// tab is the terminator, the spaces belong to the name.
func TestPatchPathKeepsATrailingSpaceInTheName(t *testing.T) {
	patch := "diff --git a/trail  b/trail \n" +
		"--- a/trail \t\n" +
		"+++ b/trail \t\n" +
		"@@ -1 +1 @@\n" +
		"-before\n" +
		"+after\n"
	if got := patchPath(patch); got != "trail " {
		t.Errorf("patchPath read %q; want %q", got, "trail ")
	}
}

// A quoted header can contain " b/" inside a name; splitting the quoted text
// on that sequence truncates the path. Both names are parsed as quoted
// strings instead.
func TestPatchPathReadsAQuotedBinaryHeader(t *testing.T) {
	patch := "diff --git \"a/dir b/tab\\t.bin\" \"b/dir b/tab\\t.bin\"\n" +
		"Binary files a/dir b/tab\\t.bin and b/dir b/tab\\t.bin differ\n"
	want := "dir b/tab\t.bin"
	if got := patchPath(patch); got != want {
		t.Errorf("patchPath read %q; want %q", got, want)
	}
	files := splitFileDiffs(patch)
	if len(files) != 1 || files[0].Path != want {
		t.Errorf("splitFileDiffs read %+v; want one file at %q", files, want)
	}
}
