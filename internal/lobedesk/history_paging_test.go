package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// Home, End and PageDown have to reach the ends of a list even before anything
// is selected. Starting every move at row 0 threw the distance away, so End
// landed on the first row and PageDown on the second.
func TestListMovesFromNothingReachTheEnds(t *testing.T) {
	for _, test := range []struct {
		name  string
		delta int
		rows  int
		want  int
	}{
		{"step down starts at the first row", 1, 5, 0},
		{"step up stays at the first row", -1, 5, 0},
		{"Home", -1 << 30, 5, 0},
		{"End", 1 << 30, 5, 4},
		{"PageDown with fewer rows than a page", pageStep, 5, 4},
		{"PageDown with a long list", pageStep, 100, pageStep - 1},
	} {
		if got := listStartIndex(test.delta, test.rows); got != test.want {
			t.Errorf("%s: listStartIndex(%d, %d) = %d; want %d", test.name, test.delta, test.rows, got, test.want)
		}
	}
}

// A file that only exists in the index has no HEAD version for blame to walk;
// offering the action anyway ends in a permanent error and a retry button.
func TestBlameSkipsFilesThatHEADDoesNotHave(t *testing.T) {
	for _, test := range []struct {
		name  string
		entry gitrepo.Entry
		want  bool
	}{
		{"modified file", gitrepo.Entry{Index: ' ', Worktree: 'M'}, true},
		{"staged modification", gitrepo.Entry{Index: 'M', Worktree: ' '}, true},
		{"staged new file", gitrepo.Entry{Index: 'A', Worktree: ' '}, false},
		{"staged then edited", gitrepo.Entry{Index: 'A', Worktree: 'M'}, false},
		{"untracked", gitrepo.Entry{Index: '?', Worktree: '?'}, false},
		{"deleted", gitrepo.Entry{Index: ' ', Worktree: 'D'}, false},
	} {
		if got := canBlame(test.entry); got != test.want {
			t.Errorf("%s: canBlame = %v; want %v", test.name, got, test.want)
		}
	}
}

// A page that comes back after the history was walked again from another tip
// belongs to the list that is no longer shown.
func TestHistoryPageStalenessCoversRepoAndRefWalks(t *testing.T) {
	repo := &gitrepo.Repository{}
	other := &gitrepo.Repository{}
	a := &app{repo: repo, loadSeq: 5, historyReload: 2}
	if a.historyPageStale(5, repo, 2) {
		t.Error("a page for the current repository and walk was called stale")
	}
	if !a.historyPageStale(4, repo, 2) {
		t.Error("a page from the previous repository load was not called stale")
	}
	if !a.historyPageStale(5, other, 2) {
		t.Error("a page from another repository was not called stale")
	}
	if !a.historyPageStale(5, repo, 1) {
		t.Error("a page from the previous ref walk was not called stale")
	}
}

// Switching the ref filter drops the file filter; a file-history load that was
// still running has to be abandoned with it, or the list keeps its loading
// placeholder with no callback left to clear it.
func TestApplyingARefAbandonsTheFileHistoryLoad(t *testing.T) {
	a := &app{fileFilter: "pkg/x.go", fileHistoryLoading: true, fileHistorySeq: 3}
	a.applyHistoryRef(historyRef{})
	if a.fileFilter != "" {
		t.Errorf("fileFilter = %q after choosing a ref filter; want it cleared", a.fileFilter)
	}
	if a.fileHistoryLoading {
		t.Error("applyHistoryRef left fileHistoryLoading set")
	}
	if a.historyPinned != "" {
		t.Errorf("historyPinned = %q when all refs are listed; want it cleared", a.historyPinned)
	}
}

// The selector's dot and the header chip have to wear the color the graph drew
// the line with, not the hash slot a colliding branch pushed aside.
func TestRefColorFollowsTheGraphWhenItShiftedALane(t *testing.T) {
	palette := []ui.Color{ui.RGB(1, 2, 3), ui.RGB(4, 5, 6)}
	graph := &commitGraph{Palette: palette, RefLanes: map[string]int{"main": 1}}
	a := &app{historyGraph: graph}
	if got := a.refGraphColor(palette, "main"); got != palette[1] {
		t.Errorf("refGraphColor(main) = %v; want the drawn lane color %v", got, palette[1])
	}
	if got, want := a.refGraphColor(palette, "dev"), refLaneColor(palette, "dev"); got != want {
		t.Errorf("refGraphColor(dev) = %v; want the hash color %v for a ref the graph does not draw", got, want)
	}
}
