package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

func sampleCommitDiffs() []gitrepo.FileDiff {
	return []gitrepo.FileDiff{
		{Path: "a.go", Diff: gitrepo.Diff{
			Lines: []gitrepo.DiffLine{
				{Kind: "meta", Text: "diff --git a/a.go b/a.go"},
				{Kind: "hunk", Text: "@@ -1 +1 @@"},
				{Kind: "del", Text: "-old"},
				{Kind: "add", Text: "+new"},
			},
			Hunks: []gitrepo.Hunk{{Header: "@@ -1 +1 @@"}},
		}},
		{Path: "bin.dat", Diff: gitrepo.Diff{
			Lines: []gitrepo.DiffLine{{Kind: "meta", Text: "Binary files a/bin.dat and b/bin.dat differ"}},
		}},
		{Path: "c.go", Diff: gitrepo.Diff{
			Lines: []gitrepo.DiffLine{
				{Kind: "meta", Text: "diff --git a/c.go b/c.go"},
				{Kind: "hunk", Text: "@@ -5 +5 @@"},
				{Kind: "add", Text: "+more"},
			},
			Hunks: []gitrepo.Hunk{{Header: "@@ -5 +5 @@"}},
		}},
	}
}

// A section points at its own lines in the combined diff and draws the patch
// without the header rows the section header already carries. A file with no
// hunks keeps its meta rows, because they are all a binary change has to say.
func TestBuildCommitDiffSections(t *testing.T) {
	diff, sections := buildCommitDiff(sampleCommitDiffs())
	if len(diff.Lines) != 8 {
		t.Fatalf("combined diff has %d lines; want 8", len(diff.Lines))
	}
	if len(sections) != 3 {
		t.Fatalf("built %d sections; want 3", len(sections))
	}
	wantStarts := []int{0, 4, 5}
	wantRows := [][]int{{1, 2, 3}, {4}, {6, 7}}
	for i, sec := range sections {
		if sec.Start != wantStarts[i] {
			t.Errorf("section %d starts at %d; want %d", i, sec.Start, wantStarts[i])
		}
		if len(sec.Rows) != len(wantRows[i]) {
			t.Fatalf("section %d rows = %v; want %v", i, sec.Rows, wantRows[i])
		}
		for j, index := range sec.Rows {
			if index != wantRows[i][j] {
				t.Errorf("section %d row %d is %d; want %d", i, j, index, wantRows[i][j])
			}
		}
	}
}

// The file list scrolls the diff to a file by summing what is above it, so a
// collapsed section costs only its header.
func TestCommitDiffOffsetSkipsCollapsedSections(t *testing.T) {
	diff, sections := buildCommitDiff(sampleCommitDiffs())
	a := &app{commitDiff: diff, commitSections: sections}
	if y := a.commitDiffOffset("a.go"); y != 0 {
		t.Errorf("first file sits at %v; want 0", y)
	}
	open := diffFileHeaderH + diffHunkH + 2*diffRowH + diffFileHeaderH + diffRowH
	if y := a.commitDiffOffset("c.go"); y != open {
		t.Errorf("third file sits at %v; want %v", y, open)
	}
	a.toggleCommitSection("a.go")
	collapsed := diffFileHeaderH + diffFileHeaderH + diffRowH
	if y := a.commitDiffOffset("c.go"); y != collapsed {
		t.Errorf("with the first file closed the third sits at %v; want %v", y, collapsed)
	}
	if y := a.commitDiffOffset("missing.go"); y != -1 {
		t.Errorf("a file the commit does not touch is at %v; want -1", y)
	}
}

// The file list opens a closed section when it scrolls to it, which has to
// bring the selection back exactly like clicking the header does.
func TestScrollToFileReopensWithTheSelection(t *testing.T) {
	diff, sections := buildCommitDiff(sampleCommitDiffs())
	a := &app{commitDiff: diff, commitSections: sections,
		commitSel: map[int]bool{2: true}, commitCursor: 2}
	a.toggleCommitSection("a.go")
	if a.commitSel[2] {
		t.Fatal("closing left row 2 selected")
	}
	a.scrollCommitDiffToFile("a.go")
	if a.commitCollapsed["a.go"] {
		t.Error("the section is still closed; want it opened")
	}
	if !a.commitSel[2] || a.commitCursor != 2 {
		t.Errorf("selection = %v, cursor = %d after scrolling to the file; want row 2 and cursor 2", a.commitSel, a.commitCursor)
	}
}

// Closing a section drops its rows from the line selection, so a copy can
// never take code the reader has just hidden; reopening returns them.
func TestToggleCommitSectionHidesAndRestoresItsSelection(t *testing.T) {
	diff, sections := buildCommitDiff(sampleCommitDiffs())
	a := &app{commitDiff: diff, commitSections: sections,
		commitSel: map[int]bool{2: true, 7: true}, commitCursor: 2}
	a.toggleCommitSection("a.go")
	if len(a.commitSel) != 1 || !a.commitSel[7] {
		t.Errorf("selection after closing = %v; want only row 7", a.commitSel)
	}
	if a.commitCursor != -1 {
		t.Errorf("cursor = %d after its row was hidden; want -1", a.commitCursor)
	}
	a.toggleCommitSection("a.go")
	if a.commitCollapsed["a.go"] {
		t.Error("a second click left the section closed; want it opened")
	}
	// Opening the section again puts the hidden rows back, so closing a file
	// to peek at another one does not cost the reader their selection.
	if len(a.commitSel) != 2 || !a.commitSel[2] || !a.commitSel[7] {
		t.Errorf("selection after reopening = %v; want rows 2 and 7", a.commitSel)
	}
	if a.commitCursor != 2 {
		t.Errorf("cursor = %d after reopening; want 2", a.commitCursor)
	}
}
