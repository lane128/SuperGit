package lobedesk

import (
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func entryPaths(entries []gitrepo.Entry) string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	return strings.Join(paths, ",")
}

// Untracked files are Unstaged as far as Git is concerned, so the workspace
// listed them inside the same group as the tracked edits waiting in the tree.
// They get a section of their own now. A file with both staged and unstaged
// changes stays in two sections -- that is how Git GUIs have always shown it
// -- but an untracked file must not leak into either tracked group.
func TestWorkspaceListsUntrackedApartFromUnstaged(t *testing.T) {
	a := &app{status: gitrepo.Status{Entries: []gitrepo.Entry{
		{Path: "edited.go", Index: ' ', Worktree: 'M'},
		{Path: "staged.go", Index: 'M', Worktree: ' '},
		{Path: "new.md", Index: '?', Worktree: '?'},
		{Path: "new-then-edited.go", Index: 'A', Worktree: 'M'},
	}}}
	if got := entryPaths(a.unstagedEntries()); got != "edited.go,new-then-edited.go" {
		t.Errorf("unstaged = %q; want edited.go,new-then-edited.go", got)
	}
	if got := entryPaths(a.stagedEntries()); got != "staged.go,new-then-edited.go" {
		t.Errorf("staged = %q; want staged.go,new-then-edited.go", got)
	}
	if got := entryPaths(a.untrackedEntries()); got != "new.md" {
		t.Errorf("untracked = %q; want new.md", got)
	}
	for _, entry := range a.unstagedEntries() {
		if entry.Path == "new.md" {
			t.Error("the untracked file was listed in the unstaged group as well")
		}
	}
}

// A section the reader folded is not drawn, so it is not in the keyboard order
// either: the arrows must never land on a row that is not on screen. Folding
// the section that held the selection moves the selection to the first row
// still shown.
func TestFoldedSectionsLeaveTheKeyboardOrder(t *testing.T) {
	a := &app{status: gitrepo.Status{Entries: []gitrepo.Entry{
		{Path: "edited.go", Index: ' ', Worktree: 'M'},
		{Path: "staged.go", Index: 'M', Worktree: ' '},
		{Path: "new.md", Index: '?', Worktree: '?'},
	}}}
	if got, want := refsPaths(a.fileRefs()), "edited.go,new.md,staged.go"; got != want {
		t.Fatalf("fileRefs() = %q; want %q", got, want)
	}
	a.toggleFilesSection(filesSectionStaged)
	if got, want := refsPaths(a.fileRefs()), "edited.go,new.md"; got != want {
		t.Errorf("fileRefs(folded staged) = %q; want %q", got, want)
	}
	a.toggleFilesSection(filesSectionStaged)
	if got, want := refsPaths(a.fileRefs()), "edited.go,new.md,staged.go"; got != want {
		t.Errorf("fileRefs(unfolded) = %q; want %q", got, want)
	}

	// The selection was on the staged row; folding its section moves it to the
	// first row still on screen.
	a.selectFile("staged.go", true)
	a.toggleFilesSection(filesSectionStaged)
	if a.selectedPath != "edited.go" || a.selectedStaged {
		t.Errorf("selection after folding = (%q, %v); want (edited.go, false)", a.selectedPath, a.selectedStaged)
	}
	// Folding a section that does not hold the selection leaves it alone.
	a.toggleFilesSection(filesSectionUntracked)
	if a.selectedPath != "edited.go" {
		t.Errorf("selection after folding another section = %q; want edited.go", a.selectedPath)
	}
}

// Unfolding the section that held the selection puts it back, so folding a
// section to read the rest of the list costs nothing; a reader who picked
// another row in the meantime keeps that row.
func TestUnfoldingASectionBringsTheSelectionBack(t *testing.T) {
	a := &app{status: gitrepo.Status{Entries: []gitrepo.Entry{
		{Path: "edited.go", Index: ' ', Worktree: 'M'},
		{Path: "staged.go", Index: 'M', Worktree: ' '},
		{Path: "new.md", Index: '?', Worktree: '?'},
	}}}
	a.selectFile("staged.go", true)
	a.toggleFilesSection(filesSectionStaged)
	if a.selectedPath != "edited.go" {
		t.Fatalf("fold did not move the selection: %q", a.selectedPath)
	}
	a.toggleFilesSection(filesSectionStaged)
	if a.selectedPath != "staged.go" || !a.selectedStaged {
		t.Errorf("selection after unfolding = (%q, %v); want (staged.go, true)", a.selectedPath, a.selectedStaged)
	}

	// Picking another row while the section is folded wins over the memory.
	a.selectFile("staged.go", true)
	a.toggleFilesSection(filesSectionStaged)
	a.selectFile("new.md", false)
	a.toggleFilesSection(filesSectionStaged)
	if a.selectedPath != "new.md" {
		t.Errorf("selection after unfolding = %q; want new.md", a.selectedPath)
	}
}

// A reload keeps folded sections folded. When the selected file has moved into
// one of them (the last waiting edit was staged while the index was folded),
// the selection has to follow the file onto the screen rather than stay on a
// row that is no longer drawn.
func TestReconcileSelectionSkipsFoldedSections(t *testing.T) {
	a := &app{
		filesCollapsed: map[string]bool{filesSectionStaged: true},
		status: gitrepo.Status{Entries: []gitrepo.Entry{
			{Path: "edited.go", Index: ' ', Worktree: 'M'},
			{Path: "staged.go", Index: 'M', Worktree: ' '},
		}},
		selectedPath: "staged.go", selectedStaged: true,
	}
	a.reconcileSelection()
	if a.selectedPath != "edited.go" || a.selectedStaged {
		t.Errorf("selection = (%q, %v); want the visible row (edited.go, false)", a.selectedPath, a.selectedStaged)
	}
}

func refsPaths(refs []fileRef) string {
	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		paths = append(paths, ref.path)
	}
	return strings.Join(paths, ",")
}

// The first row the workspace selects follows the order the sections are
// drawn in: tracked edits, then new files, then the index.
func TestFirstWorkspaceEntryFollowsTheSectionOrder(t *testing.T) {
	onlyUntracked := []gitrepo.Entry{{Path: "new.md", Index: '?', Worktree: '?'}}
	if path, staged := firstWorkspaceEntry(onlyUntracked); path != "new.md" || staged {
		t.Errorf("firstWorkspaceEntry(untracked only) = (%q, %v); want (new.md, false)", path, staged)
	}
	mixed := []gitrepo.Entry{
		{Path: "staged.go", Index: 'M', Worktree: ' '},
		{Path: "new.md", Index: '?', Worktree: '?'},
	}
	if path, staged := firstWorkspaceEntry(mixed); path != "new.md" || staged {
		t.Errorf("firstWorkspaceEntry(mixed) = (%q, %v); want (new.md, false)", path, staged)
	}
}
