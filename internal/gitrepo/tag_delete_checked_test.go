package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedTagDeleteRejectsReplacementObject(t *testing.T) {
	for _, kind := range []string{"lightweight", "annotated-same-commit", "missing"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			if kind == "annotated-same-commit" {
				git(t, dir, "tag", "-a", "--no-sign", "-m", "original annotation", "--", "release")
			} else {
				git(t, dir, "tag", "release")
			}
			original := strings.TrimSpace(git(t, dir, "rev-parse", "refs/tags/release"))
			switch kind {
			case "lightweight":
				git(t, dir, "commit", "--allow-empty", "-m", "next")
				git(t, dir, "tag", "-f", "release")
			case "annotated-same-commit":
				git(t, dir, "tag", "-f", "-a", "--no-sign", "-m", "replacement annotation", "--", "release")
			case "missing":
				git(t, dir, "tag", "-d", "release")
			}
			before := git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)", "refs/tags")
			repo := openTest(t, dir)
			if err := repo.DeleteTagChecked(ctx, "release", original); err == nil {
				t.Fatal("old confirmation removed replacement tag object")
			}
			if after := git(t, dir, "for-each-ref", "--format=%(refname) %(objectname)", "refs/tags"); after != before {
				t.Fatal("refused tag deletion changed tag refs")
			}
		})
	}
}

func TestTagDeleteRemovesSymbolicNameOnly(t *testing.T) {
	for _, checked := range []bool{false, true} {
		name := "legacy"
		if checked {
			name = "checked"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			target := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
			git(t, dir, "tag", "keep")
			git(t, dir, "symbolic-ref", "refs/tags/alias", "refs/tags/keep")
			repo := openTest(t, dir)
			var err error
			if checked {
				err = repo.DeleteTagChecked(ctx, "alias", target)
			} else {
				err = repo.DeleteTag(ctx, "alias")
			}
			if err != nil {
				t.Fatal(err)
			}
			if refs := git(t, dir, "for-each-ref", "--format=%(refname)", "refs/tags"); strings.TrimSpace(refs) != "refs/tags/keep" {
				t.Fatalf("symbolic tag deletion changed referent: %q", refs)
			}
			if strings.TrimSpace(git(t, dir, "rev-parse", "refs/tags/keep")) != target {
				t.Fatal("symbolic referent changed")
			}
			// The selected symbolic ref must be gone, including its raw file.
			if _, err := os.Lstat(filepath.Join(dir, ".git", "refs", "tags", "alias")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("raw symbolic ref survived deletion: %v", err)
			}
		})
	}
}

func TestCheckedTagDeleteAcceptsOriginalLightweightAndAnnotatedObjects(t *testing.T) {
	for _, annotated := range []bool{false, true} {
		label := "lightweight"
		if annotated {
			label = "annotated"
		}
		t.Run(label, func(t *testing.T) {
			ctx := context.Background()
			dir := fixture(t)
			seed(t, dir)
			git(t, dir, "tag", "keep")
			if annotated {
				git(t, dir, "tag", "-a", "--no-sign", "-m", "annotation", "--", "release")
			} else {
				git(t, dir, "tag", "release")
			}
			original := strings.TrimSpace(git(t, dir, "rev-parse", "refs/tags/release"))
			if err := openTest(t, dir).DeleteTagChecked(ctx, "release", original); err != nil {
				t.Fatal(err)
			}
			if refs := strings.TrimSpace(git(t, dir, "tag", "--list")); refs != "keep" {
				t.Fatalf("unexpected remaining tags %q", refs)
			}
		})
	}
}
