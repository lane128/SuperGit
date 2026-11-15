package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// A file-history load that is thrown away -- the filter is cleared, or another
// repository opens -- never reaches its callback's success path. Its loading
// flag has to be cleared where the load is abandoned, or the history list sits
// on its loading placeholder with nothing left to finish it.
func TestAbandonedFileHistoryLoadStopsLoading(t *testing.T) {
	a := &app{fileFilter: "pkg/constants/links.go", fileHistoryLoading: true, fileHistorySeq: 4}
	a.clearFileFilter()
	if a.fileHistoryLoading {
		t.Error("clearFileFilter left fileHistoryLoading set; the history list would keep its loading placeholder")
	}
	if a.fileHistorySeq <= 4 {
		t.Errorf("clearFileFilter left fileHistorySeq at %d; the in-flight callback would still be accepted", a.fileHistorySeq)
	}
}

// The diff's line selection indexes belong to the diff that was on screen. A
// reload that swaps the commit under the selection has to drop them there,
// because no click ever happens to reset them.
func TestSelectionChangeDropsTheDiffLineSelection(t *testing.T) {
	a := &app{
		selectedSHA:  "aaaa",
		commitSel:    map[int]bool{3: true, 4: true},
		commitAnchor: 3,
		commitCursor: 4,
	}
	a.setSelectedCommit("bbbb")
	if a.commitSel != nil || a.commitAnchor != -1 || a.commitCursor != -1 {
		t.Errorf("setSelectedCommit kept %v (anchor %d, cursor %d) after the commit changed",
			a.commitSel, a.commitAnchor, a.commitCursor)
	}
	if a.selectedSHA != "bbbb" {
		t.Errorf("selectedSHA = %q; want bbbb", a.selectedSHA)
	}

	// Re-selecting the same commit must not throw the selection away: a reload
	// of the same commit keeps its rows.
	b := &app{selectedSHA: "aaaa", commitSel: map[int]bool{3: true}}
	b.setSelectedCommit("aaaa")
	if b.commitSel == nil {
		t.Error("setSelectedCommit dropped the selection although the commit did not change")
	}
}

// A reload can move the selection to another commit (the old row is gone, or
// a file history picked one). The details blocks kept their scroll offsets,
// so the new commit's message opened halfway down where the previous one was
// being read.
func TestSelectionChangeResetsTheDetailScroll(t *testing.T) {
	a := &app{
		selectedSHA:       "aaaa",
		commitInfoScroll:  ui.ScrollState{Y: 120},
		commitFilesScroll: ui.ScrollState{Y: 40},
		commitDiffScroll:  ui.ScrollState{Y: 300},
	}
	a.setSelectedCommit("bbbb")
	if a.commitInfoScroll.Y != 0 || a.commitFilesScroll.Y != 0 || a.commitDiffScroll.Y != 0 {
		t.Errorf("detail scrolls = (%v, %v, %v); want all 0",
			a.commitInfoScroll.Y, a.commitFilesScroll.Y, a.commitDiffScroll.Y)
	}

	// The same commit coming back must keep the reader's place.
	b := &app{selectedSHA: "aaaa", commitInfoScroll: ui.ScrollState{Y: 90}}
	b.setSelectedCommit("aaaa")
	if b.commitInfoScroll.Y != 90 {
		t.Errorf("same-commit reload moved the info scroll to %v; want 90", b.commitInfoScroll.Y)
	}
}

// A stash with no message is listed by its reference, so copying has to use
// that same text instead of putting an empty string on the clipboard.
func TestStashTitleFallsBackToTheReference(t *testing.T) {
	if got := stashTitle(gitrepo.Stash{Ref: "stash@{0}"}); got != "stash@{0}" {
		t.Errorf("stashTitle = %q; want the reference the row shows, %q", got, "stash@{0}")
	}
	if got := stashTitle(gitrepo.Stash{Ref: "stash@{1}", Message: "wip: links"}); got != "wip: links" {
		t.Errorf("stashTitle = %q; want the message %q", got, "wip: links")
	}
}

// Deleting a branch leaves no row for the selection to sit on. Without a
// search to explain it away, the selection moves to the first row so the list
// still shows one and Enter has something to act on.
func TestDeletedBranchMovesTheSelectionToTheFirstRow(t *testing.T) {
	a := &app{
		branches:  []gitrepo.Branch{{Name: "main", Current: true}, {Name: "topic"}},
		branchSel: "local:topic",
	}
	a.syncBranchSelection()
	if a.branchSel != "local:topic" {
		t.Fatalf("syncBranchSelection moved a selection whose row is still listed: %q", a.branchSel)
	}
	a.branches = a.branches[:1]
	a.syncBranchSelection()
	if a.branchSel != "local:main" {
		t.Errorf("branchSel = %q after the selected branch was deleted; want the first row, local:main", a.branchSel)
	}

	// Nothing selected and no search: keep the page's opening state.
	b := &app{branches: []gitrepo.Branch{{Name: "main"}}}
	b.syncBranchSelection()
	if b.branchSel != "" {
		t.Errorf("branchSel = %q on a page with no selection; want it left empty", b.branchSel)
	}

	// A reload empties the list for a frame; with no search running that must
	// not be read as "the branch is gone".
	c := &app{branchSel: "local:x"}
	c.syncBranchSelection()
	if c.branchSel != "local:x" {
		t.Errorf("branchSel = %q while the list reloaded; want it kept", c.branchSel)
	}
}
