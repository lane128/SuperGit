package gitrepo

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSubmoduleWorktreesUseCheckoutPathsFromBothContexts(t *testing.T) {
	source := fixture(t)
	seed(t, source)
	parent := fixture(t)
	seed(t, parent)
	module := filepath.Join(parent, "子 module")
	git(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", "--name", "different-module-name", source, "子 module")
	git(t, parent, "commit", "-am", "add module")
	linked := filepath.Join(t.TempDir(), "linked child")
	git(t, module, "worktree", "add", "-b", "child-review", linked)
	for _, root := range []string{module, linked} {
		t.Run(filepath.Base(root), func(t *testing.T) {
			worktrees, err := openTest(t, root).Worktrees(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(worktrees) != 2 {
				t.Fatalf("worktrees=%#v; want two checkouts", worktrees)
			}
			found := map[string]bool{}
			for _, w := range worktrees {
				found[canonical(w.Path)] = true
				checkout, err := Open(context.Background(), w.Path)
				if err != nil || checkout == nil || canonical(checkout.Root) != canonical(w.Path) {
					t.Errorf("listed path is not its actual checkout: %#v, repo=%#v, err=%v", w, checkout, err)
				}
			}
			if !found[canonical(module)] || !found[canonical(linked)] {
				t.Errorf("worktrees=%#v; want %q and %q", worktrees, module, linked)
			}
		})
	}
}
