package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// The history page builds its rows once and the list, the selection and the
// details pane read them. The cache only lives for the frame that built it: a
// later frame, or a caller between frames, must see the current filter.
func TestHistoryRowsReusesTheFramesRows(t *testing.T) {
	a := &app{
		commits: []gitrepo.Commit{
			{SHA: "aaaa", Subject: "first"},
			{SHA: "bbbb", Subject: "second"},
		},
		historyQuery: "nothing matches",
	}
	// No page built this frame, so the rows come from the history itself.
	if got := a.historyRows(); len(got) != 0 {
		t.Fatalf("before any page build, historyRows() = %d rows; want the filtered history's none", len(got))
	}
	// A page build stores the rows it drew, and the frame's callers share them.
	a.historyRowsCache = []gitrepo.Commit{{SHA: "cached"}}
	a.historyRowsFrame = a.frameSeq
	a.historyRowsSet = true
	if got := a.historyRows(); len(got) != 1 || got[0].SHA != "cached" {
		t.Fatalf("in the page's frame, historyRows() = %v; want the rows the page built", got)
	}
	// The next frame starts over from the history, so a changed filter shows.
	a.frameSeq++
	if got := a.historyRows(); len(got) != 0 {
		t.Errorf("in a new frame, historyRows() = %d rows; want the current filter applied again", len(got))
	}
}
