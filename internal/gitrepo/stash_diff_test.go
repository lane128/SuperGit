package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStashDiffPreservesFileHeadersAndHunkOwnership(t *testing.T) {
	root := fixture(t)
	seed(t, root)
	write(t, root, "deleted.txt", "delete me\n")
	write(t, root, "renamed.txt", "rename me\n")
	write(t, root, "binary.bin", "base\x00binary\n")
	git(t, root, "add", "--", "deleted.txt", "renamed.txt", "binary.bin")
	git(t, root, "commit", "-m", "Shared file kinds")
	write(t, root, "base.txt", "++ literal content\n-- literal content\n")
	write(t, root, "scratch.txt", "untracked content\n")
	write(t, root, "空 格\nfile.txt", "unicode and control path\n")
	write(t, root, "binary.bin", "changed\x00binary\n")
	if err := os.Remove(filepath.Join(root, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	git(t, root, "mv", "--", "renamed.txt", "moved.txt")
	repo := openTest(t, root)
	ctx := context.Background()
	if err := repo.SaveStash(ctx, "Mixed read-only preview", true); err != nil {
		t.Fatal(err)
	}
	list, err := repo.Stashes(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("stash fixture: %v %v", list, err)
	}
	beforeIndex, beforeRefs := git(t, root, "ls-files", "--stage"), git(t, root, "show-ref")
	d, err := repo.StashDiff(ctx, list[0].SHA)
	if err != nil {
		t.Fatal(err)
	}
	raw := git(t, root, "-c", "core.quotepath=false", "stash", "show", "--include-untracked", "--patch", "--no-ext-diff", "--no-textconv", "--no-color", list[0].SHA)
	if d.Text != raw || d.CanStageHunks {
		t.Error("read-only preview changed raw content or enabled partial writes")
	}
	seenHeader, seenLiteral, seenBinary := false, false, false
	for _, row := range d.Lines {
		if row.Text == "--- /dev/null" || row.Text == "+++ b/scratch.txt" {
			seenHeader = true
			if row.Kind != "meta" || row.Hunk != -1 || row.Old != 0 || row.New != 0 {
				t.Errorf("file header became source change: %+v", row)
			}
		}
		if row.Text == "+++ literal content" {
			seenLiteral = true
			if row.Kind != "add" || row.New != 1 {
				t.Errorf("literal source misclassified: %+v", row)
			}
		}
		if strings.HasPrefix(row.Text, "Binary files ") {
			seenBinary = true
			if row.Kind != "meta" {
				t.Errorf("binary notice misclassified: %+v", row)
			}
		}
		if row.Kind == "hunk" && (row.Hunk < 0 || row.Hunk >= len(d.Hunks) || d.Hunks[row.Hunk].Header != row.Text) {
			t.Errorf("hunk row lost global identity: %+v", row)
		}
	}
	if !seenHeader || !seenLiteral || !seenBinary {
		t.Fatalf("missing expected content: headers=%v literal=%v binary=%v", seenHeader, seenLiteral, seenBinary)
	}
	if len(d.Hunks) < 4 {
		t.Fatalf("expected multiple file hunks, got %d", len(d.Hunks))
	}
	for i, hunk := range d.Hunks {
		if strings.Count(hunk.Patch, "diff --git ") != 1 {
			t.Errorf("hunk %d contains another file", i)
		}
		if !strings.Contains(hunk.Patch, hunk.Header+"\n") {
			t.Errorf("hunk %d header lost", i)
		}
	}
	if git(t, root, "ls-files", "--stage") != beforeIndex || git(t, root, "show-ref") != beforeRefs || git(t, root, "status", "--porcelain=v1") != "" {
		t.Fatal("reading stash changed Git state")
	}
}
