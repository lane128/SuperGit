package lobedesk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestCommitFileUndoUIRejectsStaleConfirmation(t *testing.T) {
	for _, kind := range []string{"generation", "head", "branch", "root", "metadata"} {
		t.Run(kind, func(t *testing.T) {
			a, callbacks, tt := commitUndoUIFixture(t)
			root := a.repo.Root
			requestUndoUI(t, a, callbacks, tt)
			switch kind {
			case "generation":
				a.loadSeq++
			case "head":
				if err := os.WriteFile(filepath.Join(root, "later.txt"), []byte("later independent\n"), 0600); err != nil {
					t.Fatal(err)
				}
				selectionGit(t, root, "add", "--all")
				selectionGit(t, root, "commit", "--no-gpg-sign", "-m", "later")
			case "branch":
				selectionGit(t, root, "checkout", "-b", "other")
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "metadata":
				a.commitFiles[0].OriginalPath = "different.go"
			}
			head, index := selectionGit(t, root, "rev-parse", "HEAD"), selectionGit(t, root, "ls-files", "--stage", "-z")
			if kind == "root" {
				tt.Frame()
				if a.confirm.open {
					t.Fatal("stale repository kept confirmation")
				}
			} else {
				clickPatchScopeButton(t, tt, "Undo File Changes")
				waitSelectionOperation(t, a, callbacks)
			}
			got, err := os.ReadFile(filepath.Join(root, "same.go"))
			if err != nil || string(got) != "source change\n" || selectionGit(t, root, "rev-parse", "HEAD") != head || selectionGit(t, root, "ls-files", "--stage", "-z") != index || selectionGit(t, root, "status", "--porcelain=v1", "-z") != "" {
				t.Fatal("stale confirmation changed original")
			}
		})
	}
}
func TestCommitFileUndoUITabCanReachActionWithoutFocusBeingReclaimed(t *testing.T) {
	a, callbacks, tt := commitUndoUIFixture(t)
	requestUndoUI(t, a, callbacks, tt)
	tt.Key(0, ui.KeyTab)
	for i := 0; i < 3; i++ {
		tt.Frame()
	}
	tt.Key(0, ui.KeyEnter)
	waitSelectionOperation(t, a, callbacks)
	got, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
	if err != nil || string(got) != "original\n" {
		t.Fatalf("Cancel stole focus after Tab or action did not activate: %q %v", got, err)
	}
}
func TestCommitFileUndoUIRejectsStaleOpenMenu(t *testing.T) {
	a, _, tt := commitUndoUIFixture(t)
	item := openUndoUIMenu(t, a, tt)
	a.loadSeq++
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	if a.busy || a.confirm.open {
		t.Fatal("stale open menu started an undo request")
	}
	if selectionGit(t, a.repo.Root, "status", "--porcelain=v1", "-z") != "" {
		t.Fatal("stale menu changed worktree")
	}
}
