package lobedesk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFileDiscardShortcutRestoresDeletedSelectedFile(t *testing.T) {
	a, callbacks, tt := discardShortcutFixture(t)
	index := selectionGit(t, a.repo.Root, "ls-files", "--stage")
	if err := os.Remove(filepath.Join(a.repo.Root, "b.go")); err != nil {
		t.Fatal(err)
	}
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, ui.Cmd)
	tt.Key(0, ui.KeyBackspace)
	waitShortcutDiscard(t, a, callbacks)
	tt.Frame()
	if err := tt.Click(a.confirm.actionName); err != nil {
		t.Fatal(err)
	}
	waitSelectionOperation(t, a, callbacks)
	assertDiscardBytes(t, a, "b.go", "original\n")
	assertDiscardBytes(t, a, "c.go", "original\n")
	assertDiscardBytes(t, a, "same.go", "working version\n")
	if selectionGit(t, a.repo.Root, "ls-files", "--stage") != index {
		t.Fatal("restoring deleted file changed Index")
	}
}
