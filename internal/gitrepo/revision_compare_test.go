package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRevisionComparisonReadsTwoSnapshots(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "remove me", "removed only\n")
	write(t, dir, "[before].go", "renamed only\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "base")
	base := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "rm", "remove me")
	git(t, dir, "mv", "[before].go", "after.go")
	write(t, dir, "new\tfile", "intermediate only\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "intermediate")
	write(t, dir, "base.txt", "target version\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "target")
	target := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	write(t, dir, "base.txt", "staged only\n")
	git(t, dir, "add", "--", "base.txt")
	write(t, dir, "base.txt", "dirty only\n")
	index := git(t, dir, "ls-files", "--stage")
	repo := openTest(t, dir)
	ctx := context.Background()
	files, err := repo.CompareRevisionFiles(ctx, base, target)
	if err != nil || len(files) != 4 {
		t.Fatalf("changed files = %+v, error = %v", files, err)
	}
	for _, f := range files {
		want := map[string][]string{
			"remove me": {"D", "-removed only"},
			"after.go":  {"R100", "rename from [before].go", "rename to after.go"},
			"base.txt":  {"M", "-original", "+target version"},
			"new\tfile": {"A", "+intermediate only"},
		}[f.Path]
		if len(want) == 0 || f.Status != want[0] || f.Path == "after.go" && f.OriginalPath != "[before].go" {
			t.Fatalf("unexpected file %+v", f)
		}
		patch, err := repo.CompareRevisionFile(ctx, base, target, f)
		if err != nil || strings.Count(patch.Text, "diff --git ") != 1 || strings.Contains(patch.Text, "dirty only") || strings.Contains(patch.Text, "staged only") {
			t.Fatalf("file %q: %q (%v)", f.Path, patch.Text, err)
		}
		for _, part := range want[1:] {
			if !strings.Contains(patch.Text, part) {
				t.Errorf("file %q missing %q in patch %q", f.Path, part, patch.Text)
			}
		}
	}
	if empty, err := repo.CompareRevisionFiles(ctx, base, base); err != nil || len(empty) != 0 {
		t.Fatalf("identical trees: %+v, %v", empty, err)
	}
	for _, bad := range []string{"HEAD", "--stat", strings.Repeat("f", 40)} {
		if _, err := repo.CompareRevisionFiles(ctx, bad, target); err == nil {
			t.Errorf("accepted base %q", bad)
		}
		if _, err := repo.CompareRevisionFile(ctx, target, bad, files[0]); err == nil {
			t.Errorf("accepted target %q", bad)
		}
	}
	content, err := os.ReadFile(filepath.Join(dir, "base.txt"))
	if err != nil || string(content) != "dirty only\n" || git(t, dir, "ls-files", "--stage") != index || strings.TrimSpace(git(t, dir, "rev-parse", "HEAD")) != target {
		t.Fatal("comparison modified the checkout")
	}
}
