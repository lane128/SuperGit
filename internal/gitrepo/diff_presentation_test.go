package gitrepo

import (
	"slices"
	"testing"
)

func TestSplitDiffRowsPairsEOFReplacement(t *testing.T) {
	diff := parseDiff("@@ -1 +1 @@\n-old\n\\ No newline at end of file\n+new\n\\ No newline at end of file\n")
	for _, visible := range [][]int{nil, {0, 1, 3}} {
		want := []SplitDiffRow{{Shared: 0, Left: -1, Right: -1}, {Shared: -1, Left: 1, Right: 3}}
		if visible == nil {
			want = append(want, SplitDiffRow{Shared: 2, Left: -1, Right: -1}, SplitDiffRow{Shared: 4, Left: -1, Right: -1})
		}
		if got := SplitDiffRows(diff.Lines, visible); !slices.Equal(got, want) {
			t.Fatalf("visible %v: rows = %v, want %v", visible, got, want)
		}
	}
}

func TestSplitDiffRowsKeepsFilteredSourceIndexes(t *testing.T) {
	lines := []DiffLine{
		{Kind: "add", Hunk: 0},
		{Kind: "meta", Hunk: -1},
		{Kind: "hunk", Hunk: 0},
		{Kind: "del", Hunk: 0},
		{Kind: "del", Hunk: 0},
		{Kind: "add", Hunk: 0},
		{Kind: "hunk", Hunk: 1},
		{Kind: "add", Hunk: 1},
		{Kind: "context", Hunk: 1},
	}
	want := []SplitDiffRow{
		{Shared: 2, Left: -1, Right: -1},
		{Shared: -1, Left: 3, Right: 5},
		{Shared: -1, Left: 4, Right: -1},
		{Shared: -1, Left: -1, Right: 7},
		{Shared: -1, Left: 8, Right: 8},
	}
	got := SplitDiffRows(lines, []int{2, 3, 4, 5, 7, 8})
	if !slices.Equal(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
}
