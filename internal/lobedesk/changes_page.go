package lobedesk

import "github.com/egoist/mygo/ui"

// Changes is not a page any more: the working tree's row is the first row of
// the History list, and choosing it makes the live files and their commit box
// the subject of the details. The helpers here are the selection transitions
// that page used to own.

// openChanges selects the working tree's row, snapshotting the historical
// context before its selected SHA is cleared by selectWorktree. All targets
// still pass through the normal selection/loading guards.
func (a *app) openChanges() {
	if a.worktreeSelected {
		a.saveDetailScroll()
		return
	}
	a.selectWorktree()
	a.commitFilesScroll = ui.ScrollState{X: a.detailTabs().FilesScroll.X, Y: a.detailTabs().FilesScroll.Y}
}

// restoreHistorySelection returns from the working tree's row to the commit
// the reader was last reading.
func (a *app) restoreHistorySelection() {
	if !a.worktreeSelected {
		return
	}
	sha := ""
	if a.repo != nil && a.detailContexts.Root == a.repo.Root {
		sha = a.detailContexts.HistorySHA
	}
	if sha == "" && len(a.historySource()) > 0 {
		sha = a.historySource()[0].SHA
	}
	if sha != "" {
		a.selectCommit(sha)
		pos := a.detailTabs().FilesScroll
		a.commitFilesScroll = ui.ScrollState{X: pos.X, Y: pos.Y}
	}
}
