package gitrepo

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckedSubmoduleUpdateRejectsChangedIdentity(t *testing.T) {
	for _, change := range []string{"index", "checkout", "uninitialized-index", "removed"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			t.Setenv("GIT_ALLOW_PROTOCOL", "file")
			child := fixture(t)
			seed(t, child)
			base := strings.TrimSpace(git(t, child, "rev-parse", "HEAD"))
			git(t, child, "commit", "--allow-empty", "-m", "later child")
			later := strings.TrimSpace(git(t, child, "rev-parse", "HEAD"))
			parent := fixture(t)
			seed(t, parent)
			git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "module")
			git(t, parent, "commit", "-am", "module")
			module := filepath.Join(parent, "module")
			git(t, module, "checkout", "--detach", base)
			git(t, parent, "update-index", "--cacheinfo", "160000,"+base+",module")
			if change == "uninitialized-index" {
				git(t, parent, "submodule", "deinit", "-f", "module")
			}
			repo := openTest(t, parent)
			modules, err := repo.Submodules(ctx)
			if err != nil || len(modules) != 1 {
				t.Fatalf("fixture module: %v", err)
			}
			switch change {
			case "index", "uninitialized-index":
				git(t, parent, "update-index", "--cacheinfo", "160000,"+later+",module")
			case "checkout":
				git(t, module, "checkout", "--detach", later)
			case "removed":
				git(t, parent, "config", "--file", ".gitmodules", "--remove-section", "submodule.module")
			}
			beforeIndex := git(t, parent, "ls-files", "--stage")
			before, err := repo.Submodules(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.UpdateSubmoduleChecked(ctx, modules[0]); err == nil {
				t.Fatal("stale submodule action updated a changed target")
			}
			after, err := repo.Submodules(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if git(t, parent, "ls-files", "--stage") != beforeIndex || len(before) != len(after) {
				t.Fatal("refusal changed parent Index or modules")
			}
			if len(before) != 0 && (before[0].Current != after[0].Current || before[0].Initialized != after[0].Initialized) {
				t.Fatal("refusal changed child checkout or initialized a different target")
			}
		})
	}
}

func TestCheckedSubmoduleUpdateAcceptsOriginalTarget(t *testing.T) {
	for _, initialized := range []bool{false, true} {
		name := "uninitialized"
		if initialized {
			name = "initialized"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			t.Setenv("GIT_ALLOW_PROTOCOL", "file")
			child := fixture(t)
			seed(t, child)
			base := strings.TrimSpace(git(t, child, "rev-parse", "HEAD"))
			git(t, child, "commit", "--allow-empty", "-m", "later")
			parent := fixture(t)
			seed(t, parent)
			git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "module")
			git(t, parent, "commit", "-am", "module")
			git(t, parent, "update-index", "--cacheinfo", "160000,"+base+",module")
			if !initialized {
				git(t, parent, "submodule", "deinit", "-f", "module")
			}
			repo := openTest(t, parent)
			modules, err := repo.Submodules(ctx)
			if err != nil {
				t.Fatal(err)
			}
			beforeIndex := git(t, parent, "ls-files", "--stage")
			if err := repo.UpdateSubmoduleChecked(ctx, modules[0]); err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(git(t, filepath.Join(parent, "module"), "rev-parse", "HEAD")) != base || git(t, parent, "ls-files", "--stage") != beforeIndex {
				t.Fatal("update missed original Index target or rewrote parent Index")
			}
		})
	}
}
