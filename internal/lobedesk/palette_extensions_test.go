package lobedesk

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

func TestPaletteExtensionAsyncPatternPicker(t *testing.T) {
	for _, stale := range []bool{false, true} {
		a, callbacks := overviewFixture(t)
		a.openCommandPalette()
		a.runPaletteCommand("lfs_edit_pattern", a.commandPalette.generation)
		if !a.busy {
			t.Fatal("pattern listing did not start")
		}
		if stale {
			a.loadSeq++
		}
		select {
		case fn := <-callbacks:
			fn()
		case <-time.After(5 * time.Second):
			t.Fatal("listing did not complete")
		}
		if stale {
			if a.commandPalette.open || a.busy {
				t.Fatal("stale listing took ownership of the palette")
			}
			continue
		}
		if !a.commandPalette.open || a.commandPalette.actionID != "lfs_edit_pattern" || a.commandPalette.step == nil {
			t.Fatalf("async picker lost its command scope: %+v", a.commandPalette)
		}
		rows := a.paletteChoices()
		if len(rows) != 1 {
			t.Fatalf("expected the first pattern action, got %+v", rows)
		}
		a.acceptPaletteChoice(rows[0], a.commandPalette.generation)
		if a.commandPalette.step == nil || a.commandPalette.step.submit == nil || a.commandPalette.step.parent == nil {
			t.Fatal("missing pattern input or Backspace parent")
		}
		a.paletteStepBack()
		if got := a.paletteChoices(); len(got) != 1 || got[0] != rows[0] {
			t.Fatalf("Backspace lost the original row: %+v", got)
		}
	}
}

func TestPaletteExtensionNativePatchChooserDropsStaleRepository(t *testing.T) {
	a, _ := overviewFixture(t)
	patch := filepath.Join(t.TempDir(), "patch.diff")
	if err := os.WriteFile(patch, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	a.openDirectoryDialog = func(opts mygo.OpenDialogOptions) ([]string, error) {
		calls++
		if opts.Directory || !a.repositoryChooserOpen {
			t.Fatal("expected an owned file chooser")
		}
		a.loadSeq++
		return []string{patch}, nil
	}
	a.openCommandPalette()
	a.runPaletteCommand("apply_patch", a.commandPalette.generation)
	if calls != 1 || a.busy || a.repositoryChooserOpen || a.commandPalette.open {
		t.Fatalf("stale file chooser launched an operation: %d %t", calls, a.busy)
	}
}
