package lobedesk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestWorkspaceArrowThenShiftClickKeepsRangeAnchor(t *testing.T) {
	a, _, tt := workspaceSelectionFixture(t)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	tt.Key(0, ui.KeyDown)
	if a.selectedPath != "c.go" {
		t.Fatalf("arrow did not preview c.go: %q", a.selectedPath)
	}
	clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Shift)
	if len(a.workspaceSelection.Files) != 2 || !a.workspaceSelection.Files[fileRef{"c.go", false}] || !a.workspaceSelection.Files[fileRef{"same.go", false}] {
		t.Fatalf("arrow then Shift lost current range anchor: %#v", a.workspaceSelection)
	}
}

func TestWorkspaceWindowListMoveEstablishesRangeAnchor(t *testing.T) {
	a, _, tt := workspaceSelectionFixture(t)
	a.page = "history"
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	// The files column is its own focus group in the History window, so its
	// list move is the file move rather than the commit list's.
	a.moveFileSelection(1)
	tt.Frame()
	if a.selectedPath != "c.go" {
		t.Fatalf("page navigation did not preview c.go: %q", a.selectedPath)
	}
	clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Shift)
	if len(a.workspaceSelection.Files) != 2 || !a.workspaceSelection.Files[fileRef{"c.go", false}] {
		t.Fatalf("window list move lost range anchor: %#v", a.workspaceSelection)
	}
}

func TestWorkspaceFoldRestoresOnlySinglePreview(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "multiple"}[multiple], func(t *testing.T) {
			a, _, tt := workspaceSelectionFixture(t)
			clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
			if multiple {
				clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
			}
			original := a.selectedPath
			// Exercise the fold state transitions directly: persistence and header
			// click dispatch are outside this owned selection-state regression.
			a.filesCollapsed = map[string]bool{}
			a.foldFilesSection(filesSectionUnstaged)
			if a.selectedPath == original || len(a.workspaceSelection.Files) != 0 {
				t.Fatal("fold retained hidden multi-selection")
			}
			a.expandFilesSection(filesSectionUnstaged)
			if a.selectedPath != original || a.selectedStaged {
				t.Fatalf("fold did not restore original single preview: %q", a.selectedPath)
			}
			if len(a.workspaceSelection.Files) > 1 {
				t.Fatal("expand resurrected multi-selection")
			}
			tt.Frame()
			clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, ui.Shift)
			if len(a.workspaceSelection.Files) != 1 || !a.workspaceSelection.Files[fileRef{"c.go", false}] {
				t.Fatal("fold resurrected old range anchor")
			}
		})
	}
}

func TestWorkspaceBusyArrowKeepsFocusWithPreview(t *testing.T) {
	a, _, tt := workspaceSelectionFixture(t)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	if !selectionRow(t, tt, "b.go", filesSectionUnstaged).Focused {
		t.Fatal("fixture row did not receive focus")
	}
	a.busy = true
	tt.Key(0, ui.KeyDown)
	tt.Frame()
	tt.Frame()
	if a.selectedPath != "b.go" || !selectionRow(t, tt, "b.go", filesSectionUnstaged).Focused {
		t.Fatal("busy arrow moved focus away from unchanged preview")
	}
}

func TestWorkspaceInputOwnsEnterSpaceAndSelectAllAfterMultiselect(t *testing.T) {
	a, _, tt := workspaceSelectionFixture(t)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
	if err := tt.Click("selection-input"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeySpace)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("replacement")
	if a.busy || len(a.workspaceSelection.Files) != 2 {
		t.Fatal("text field keys mutated files or selection")
	}
	if node, ok := tt.Access("selection-input"); !ok || node.Value != "replacement" {
		t.Fatalf("input did not own editing keys: %#v", node)
	}
	if got := selectionGit(t, a.repo.Root, "show", ":same.go"); got != "index version\n" {
		t.Fatalf("input changed index: %q", got)
	}
	if got := selectionGit(t, a.repo.Root, "show", ":b.go"); got != "original\n" {
		t.Fatalf("input staged control: %q", got)
	}
	for _, path := range []string{"same.go", "b.go"} {
		data, err := os.ReadFile(filepath.Join(a.repo.Root, path))
		if err != nil || string(data) != "working version\n" {
			t.Fatalf("input changed worktree: %q %v", data, err)
		}
	}
}
