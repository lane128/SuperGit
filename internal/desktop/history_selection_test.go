package desktop

import (
	"context"
	"supergit/internal/gitrepo"
	"testing"
)

func selectionHistoryApp() *app {
	a := &app{page: "history", repo: &gitrepo.Repository{Root: "/fixture"}}
	a.data.history = performanceCommits(60)
	return a
}
func TestNewCommitReadCancelsSupersededReadAndFollowsRepositoryContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &app{ctx: ctx}
	first, finishFirst := a.newCommitRead()
	defer finishFirst()
	second, finishSecond := a.newCommitRead()
	defer finishSecond()
	if first.Err() != context.Canceled || second.Err() != nil {
		t.Fatal("superseded history read was not canceled independently")
	}
	cancel()
	if second.Err() != context.Canceled {
		t.Fatal("history read outlived repository context")
	}
}
func TestHistoryCursorTracksSHAWithoutOverwritingListInput(t *testing.T) {
	a := selectionHistoryApp()
	model := a.historyModel()
	var cursor historyCursor
	cursor.sync(model, model.Commits[4].SHA)
	if cursor.row != 4 {
		t.Fatal("cursor did not follow SHA")
	}
	cursor.row = 5 // New keyboard input arrives before the next view build.
	cursor.sync(model, model.Commits[4].SHA)
	if cursor.row != 5 {
		t.Fatal("view overwrote list input using previous SHA")
	}
	cursor.sync(model, model.Commits[5].SHA)
	if cursor.row != 5 {
		t.Fatal("accepted choice did not stay selected")
	}
	sha := model.Commits[5].SHA
	a.data.history = append([]gitrepo.Commit{{SHA: "new", Subject: "inserted"}}, a.data.history...)
	a.historyRevision++
	cursor.sync(a.historyModel(), sha)
	if cursor.row != 6 {
		t.Fatalf("inserted history moved SHA selection to row %d", cursor.row)
	}
	a.historyQuery = "no matching subject"
	cursor.sync(a.historyModel(), sha)
	if cursor.row != -1 {
		t.Fatal("hidden selected commit chose an unrelated result")
	}
}
func TestHistoryKeyboardSelectionUsesFilteredOriginalIndex(t *testing.T) {
	a := selectionHistoryApp()
	a.data.history[4].Subject = "needle first"
	a.data.history[33].Subject = "needle second"
	a.historyQuery = "needle"
	model := a.historyModel()
	index, ok := a.historySelectionIndex(model, 1)
	if !ok || index != 33 || a.data.history[index].SHA != model.Commits[1].SHA {
		t.Fatal("filtered keyboard selection targeted wrong original index")
	}
	for _, row := range []int{-1, 2, 2000} {
		if _, ok := a.historySelectionIndex(model, row); ok {
			t.Fatalf("invalid row %d accepted", row)
		}
	}
	if _, ok := a.historySelectionIndex(nil, 0); ok {
		t.Fatal("missing presentation accepted")
	}
	a.historyQuery = "not present"
	if _, ok := a.historySelectionIndex(a.historyModel(), 0); ok {
		t.Fatal("empty search accepted selection")
	}
}
func TestHistoryKeyboardBrowseDuringSnapshotReadKeepsCurrentIdentityGuard(t *testing.T) {
	a := selectionHistoryApp()
	model := a.historyModel()
	a.loading = true
	if index, ok := a.historySelectionIndex(model, 4); !ok || index != 4 {
		t.Fatal("background snapshot blocked read-only browsing of current model")
	}
	a.historyRevision++
	if _, ok := a.historySelectionIndex(model, 4); ok {
		t.Fatal("new snapshot accepted old presentation")
	}
}
func TestHistoryKeyboardRejectsOldQuerySnapshotRepositoryAndBusyState(t *testing.T) {
	for _, change := range []string{"query", "snapshot", "repository", "writing", "opening", "no repo", "other page", "identity"} {
		a := selectionHistoryApp()
		model := a.historyModel()
		switch change {
		case "query":
			a.historyQuery = "other query"
		case "snapshot":
			a.historyRevision++
		case "repository":
			a.guard.Switch()
		case "writing":
			a.guard.writing = true
		case "loading":
			a.loading = true
		case "opening":
			a.opening = true
		case "no repo":
			a.repo = nil
		case "other page":
			a.page = "workspace"
		case "identity":
			a.data.history[0].SHA = "another-sha"
		}
		if _, ok := a.historySelectionIndex(model, 0); ok {
			t.Fatalf("accepted stale/invalid selection after %s", change)
		}
	}
}
