package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// A walk from a new tip leaves the previous filter's rows in a.commits until
// it answers. The list has to show the loading placeholder instead of those
// rows: after the header's ref changed, a list still showing the old commits
// reads as if the filter did nothing.
func TestHistoryListShowsAReloadInsteadOfStaleRows(t *testing.T) {
	a := &app{
		historyReloading: true,
		commits:          []gitrepo.Commit{{SHA: "a"}, {SHA: "b"}},
	}
	if got := a.historyListState(2); got != historyListReloading {
		t.Errorf("state with a walk out = %v; want historyListReloading", got)
	}
	a.historyReloading = false
	if got := a.historyListState(2); got != historyListRows {
		t.Errorf("state after the walk = %v; want historyListRows", got)
	}
}

// An error line is drawn above the list by the page; a placeholder under it
// would contradict the line, and a walk out is no reason to hold the error
// back behind a loading state.
func TestHistoryErrorsOutrankAReload(t *testing.T) {
	a := &app{historyErr: "boom", historyReloading: true}
	if got := a.historyListState(0); got != historyListError {
		t.Errorf("state = %v; want historyListError", got)
	}
}

// Arrows reach the list only when it is on screen. During a walk the rows
// belong to the filter that was replaced, so moving the hidden selection
// would load details for a commit the reader cannot see.
func TestCommitArrowsWaitForTheWalk(t *testing.T) {
	a := &app{
		historyReloading: true,
		commits:          []gitrepo.Commit{{SHA: "a"}, {SHA: "b"}},
		selectedSHA:      "a",
	}
	a.moveHistorySelection(1)
	if a.selectedSHA != "a" {
		t.Errorf("selection moved to %q while the list was reloading", a.selectedSHA)
	}
}

// Paging the old walk while a new one is out would append the old tip's
// commits to the new list; the footer is not drawn during a walk, but the
// jump paths call this too.
func TestPagingWaitsForTheWalk(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{}, historyMore: true, historyReloading: true}
	a.loadMoreHistory()
	if a.historyLoadingMore {
		t.Error("a page of the replaced walk was started during the reload")
	}
}

// Starting a walk without a repository has nothing to answer it, so it must
// not mark the list as loading: the placeholder would never go away.
func TestAReloadWithoutARepositoryLeavesNoPlaceholder(t *testing.T) {
	a := &app{}
	a.reloadHistoryFromTip("")
	if a.historyReloading {
		t.Error("the list was left loading with no walk to finish it")
	}
}
