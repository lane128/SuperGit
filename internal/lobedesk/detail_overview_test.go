package lobedesk

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestFileTabOnlyDrawsItsOwnPatch(t *testing.T) {
	files := sampleCommitDiffs()
	// Type changes can produce two independent sections for the same path.
	files = append(files, gitrepo.FileDiff{Path: "a.go", Diff: gitrepo.Diff{Lines: []gitrepo.DiffLine{{Kind: "meta", Text: "second shape of a.go"}}}})
	diff, sections := buildCommitDiff(files)
	a := &app{selectedSHA: "aaaa", commitDiff: diff, commitSections: sections, commitPath: "a.go"}
	a.detailTabs().Open(detailFileKey{Path: "a.go"})
	tt := pageTester(a, a.commitDiffView)
	for _, text := range []string{"-old", "+new", "second shape of a.go"} {
		if _, ok := tt.Find(text); !ok {
			t.Errorf("file tab omitted %q", text)
		}
	}
	for _, text := range []string{"+more", "Binary files a/bin.dat and b/bin.dat differ"} {
		if _, ok := tt.Find(text); ok {
			t.Errorf("file tab contains another file's patch: %q", text)
		}
	}
	if got := a.commitLineIndices(); !slices.Equal(got, []int{2, 3}) {
		t.Errorf("file tab selects invisible lines: %v", got)
	}
}

func TestFileTabSelectionCannotReachOtherFiles(t *testing.T) {
	diff, sections := buildCommitDiff(sampleCommitDiffs())
	a := &app{commitDiff: diff, commitSections: sections}
	a.detailTabs().Open(detailFileKey{Path: "a.go"})
	a.invertCommitLines()
	if len(a.commitSel) != 2 || !a.commitSel[2] || !a.commitSel[3] {
		t.Fatalf("invert selected another file: %v", a.commitSel)
	}
	a.selectCommitLines([]int{2, 7})
	if a.commitSel[7] {
		t.Fatal("selection accepted a hidden file's line")
	}
	a.detailTabs().Activate(detailTabKey{Summary: true})
	a.commitCollapsed = map[string]bool{"bin.dat": true, "a.go": true}
	a.commitSel = nil
	a.invertCommitLines()
	if len(a.commitSel) != 1 || !a.commitSel[7] {
		t.Fatalf("summary selected collapsed rows: %v", a.commitSel)
	}
}

func TestHistorySummaryDragScrollsItsOwnReadingPosition(t *testing.T) {
	a := &app{commitDiffViewport: ui.Rect{H: 100}, commitInfoScroll: ui.ScrollState{Y: 10, MaxY: 1000}}
	ui.NewTester(func(c *ui.Context) {
		body := ui.Box(c).Width(100).Height(100)
		a.autoScrollCommitDrag(c, body, 95)
	}, 100, 100)
	if a.commitInfoScroll.Y <= 10 {
		t.Fatal("summary drag did not scroll the overview")
	}
	if a.commitDiffScroll.Y != 0 {
		t.Fatal("summary drag moved the hidden file tab")
	}
}

func TestHistorySummaryPatchOwnsItsLineKeys(t *testing.T) {
	diff, sections := buildCommitDiff(sampleCommitDiffs())
	a := &app{commitDiff: diff, commitSections: sections}
	if !a.commitDiffShown() {
		t.Fatal("summary's visible patches do not own their line keys")
	}
	a.selectCommitLines([]int{2})
	if !a.moveCommitLineSelection(1, false) || a.commitCursor != 3 {
		t.Fatal("summary line navigation did not stay in the patch")
	}
}

func TestHistorySummaryIncludesMessageAndAllPatches(t *testing.T) {
	diff, sections := buildCommitDiff(sampleCommitDiffs())
	a := &app{selectedSHA: "aaaa", commits: []gitrepo.Commit{{SHA: "aaaa", Subject: "summary subject"}}, commitMessage: "summary subject\n\nfull message body", commitDiff: diff, commitSections: sections}
	tt := pageTester(a, func(c *ui.Context) { a.historyDiffBody(c, theme.TokensOf(c)) })
	for _, text := range []string{"full message body", "-old", "+new", "+more", "Binary files a/bin.dat and b/bin.dat differ"} {
		if _, ok := tt.Find(text); !ok {
			t.Errorf("summary omitted %q", text)
		}
	}
}
