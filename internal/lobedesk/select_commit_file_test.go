package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// Reopening an existing file tab keeps its own position and line selection.
// It no longer scrolls an aggregate diff to that file's global offset.
func TestSelectCommitFileKeepsItsOwnReadingPosition(t *testing.T) {
	a := &app{
		repo:        &gitrepo.Repository{Root: t.TempDir()},
		selectedSHA: "aaaa",
		commitPath:  "b.go",
		commitDiff: gitrepo.Diff{Lines: []gitrepo.DiffLine{
			{Kind: "add", Text: "one", New: 1},
			{Kind: "add", Text: "two", New: 1},
		}},
		commitSections: []commitDiffSection{
			{Path: "a.go", Rows: []int{0}},
			{Path: "b.go", Rows: []int{1}},
		},
	}
	a.selectCommitFile("b.go")
	a.selectCommitLines([]int{1})
	want := float32(125)
	a.commitDiffScroll.Y = want
	a.selectCommitFile("b.go")
	if a.commitDiffScroll.Y != want {
		t.Errorf("second click scrolled to %v; want b.go's offset %v", a.commitDiffScroll.Y, want)
	}
	if a.commitPath != "b.go" {
		t.Errorf("second click moved the details to %q", a.commitPath)
	}
	// Scrolling is not a reselect: the lines the reader chose stay chosen.
	if len(a.commitSel) != 1 || !a.commitSel[1] {
		t.Errorf("second click cleared the line selection: %v", a.commitSel)
	}
}

// A different file starts its patch at the top: keeping the old offset would
// open the new patch halfway down, past the header the reader needs. Saving the
// same file again is not a change of target and keeps the reader's place.
func TestSelectFileStartsTheNewPatchAtTheTop(t *testing.T) {
	a := &app{
		selectedPath:   "a.go",
		diff:           gitrepo.Diff{Text: "a patch"},
		selectedStaged: false,
	}
	a.diffScroll.X, a.diffScroll.Y = 12, 300
	// No repository, so the load returns early; the scroll the selection owns
	// is already reset by then.
	a.selectFile("b.go", false)
	if a.diffScroll.X != 0 || a.diffScroll.Y != 0 {
		t.Errorf("the new file kept scroll (%v, %v); want the top", a.diffScroll.X, a.diffScroll.Y)
	}

	a.diffScroll.X, a.diffScroll.Y = 12, 300
	a.selectFile("b.go", false)
	if a.diffScroll.X != 12 || a.diffScroll.Y != 300 {
		t.Errorf("re-selecting the same file reset its scroll to (%v, %v)", a.diffScroll.X, a.diffScroll.Y)
	}
}
