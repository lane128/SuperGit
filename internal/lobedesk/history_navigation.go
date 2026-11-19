package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"runtime"
)

func historyNavigationShortcuts(parent appShortcut) []appShortcut {
	back, forward := parent, parent
	back.ID, forward.ID = "history.back", "history.forward"
	back.Label, forward.Label = "historyBack", "historyForward"
	back.Key, forward.Key = ui.KeyLeft, ui.KeyRight
	back.Mod, forward.Mod = ui.Alt, ui.Alt
	back.Keys, forward.Keys = "Alt+←", "Alt+→"
	if runtime.GOOS == "darwin" {
		back.Mod, forward.Mod = ui.Cmd, ui.Cmd
		back.Keys, forward.Keys = "⌘←", "⌘→"
	}
	back.Available = func(a *app) bool { return a.canNavigateHistory(-1) }
	forward.Available = func(a *app) bool { return a.canNavigateHistory(1) }
	back.Run = func(a *app) { a.navigateHistory(-1) }
	forward.Run = func(a *app) { a.navigateHistory(1) }
	return []appShortcut{back, forward}
}

// Browsing history records explicit successful jumps, not ordinary selection.
// A scope change invalidates the route rather than silently widening history.
type historyPosition struct{ sha, query string }
type historyJump struct {
	source        historyPosition
	target        string
	messageOrigin *messageCommitOrigin
}
type historyNavigationState struct {
	root      string
	epoch     uint64
	reload    int
	positions []historyPosition
	cursor    int
	pending   *historyJump
	selecting bool
}

func (a *app) historyNavigationBlocked() bool {
	return a.busy || a.loading || a.historyReloading || a.repositoryChooserOpen || a.historyRefOpen || a.hasPendingDialog(toastMain)
}

func (a *app) cancelPendingHistoryNavigation() {
	if p := a.historyNavigation.pending; p != nil {
		if a.pendingJumpSHA == p.target {
			a.pendingJumpSHA = ""
		}
		a.historyNavigation.pending = nil
	}
}

func (a *app) clearHistoryNavigation() {
	if p := a.historyNavigation.pending; p != nil && a.pendingJumpSHA == p.target {
		a.pendingJumpSHA = ""
	}
	a.historyNavigation = historyNavigationState{}
}

func (a *app) historyNavigationMember(sha string) bool {
	for _, c := range a.commits {
		if c.SHA == sha {
			return true
		}
	}
	return false
}

func (a *app) historyNavigationVisible() bool {
	if a.repo == nil || a.fileFilter != "" || a.worktreeSelected || !a.historyNavigationMember(a.selectedSHA) {
		return false
	}
	// Never trust a frame cache built before a query was restored.
	for _, c := range a.filteredHistoryRows() {
		if c.SHA == a.selectedSHA {
			return true
		}
	}
	return false
}

func (a *app) discardStaleHistoryNavigation() {
	s := &a.historyNavigation
	if s.root == "" {
		return
	}
	if a.repo == nil || s.root != a.repo.Root || s.epoch != a.loadSeq || s.reload != a.historyReload || a.fileFilter != "" || a.worktreeSelected || !a.historyNavigationMember(a.selectedSHA) {
		a.clearHistoryNavigation()
		return
	}
	if p := s.pending; p != nil && (a.selectedSHA != p.source.sha || a.historyQuery != "" || a.pendingJumpSHA != p.target || !a.messageCommitPagingCurrent(p.messageOrigin)) {
		if a.pendingJumpSHA == p.target {
			a.pendingJumpSHA = ""
		}
		s.pending = nil
	}
	if len(s.positions) > 0 && (s.positions[s.cursor].sha != a.selectedSHA || !a.historyNavigationVisible()) {
		a.clearHistoryNavigation()
	}
}

// jumpWithHistory opts an existing explicit relationship command into recording.
// Cross-file and cross-scope callers keep jumpToCommit's existing behavior.
func (a *app) jumpWithHistory(sha string) {
	a.jumpWithHistoryTarget(sha, nil)
}

func (a *app) jumpWithHistoryTarget(sha string, messageOrigin *messageCommitOrigin) {
	a.discardStaleHistoryNavigation()
	if !a.historyNavigationVisible() || a.page != "history" {
		a.clearHistoryNavigation()
		a.jumpToCommit(sha)
		return
	}
	if a.historyNavigation.pending != nil || a.pendingJumpSHA != "" {
		return
	}
	s := &a.historyNavigation
	if s.root == "" {
		s.root, s.epoch, s.reload = a.repo.Root, a.loadSeq, a.historyReload
	}
	s.pending = &historyJump{source: historyPosition{a.selectedSHA, a.historyQuery}, target: sha, messageOrigin: messageOrigin}
	a.jumpToCommit(sha)
	if a.pendingJumpSHA == "" {
		a.historyNavigation.pending = nil
	}
}

// Called only by jumpToCommit when it has the target in the real history source.
// The generic selection function never writes positions.
func (a *app) selectJumpCommit(sha string) {
	s := &a.historyNavigation
	p := s.pending
	if p != nil && a.historyNavigationBlocked() {
		a.cancelPendingHistoryNavigation()
		return
	}
	tracked := p != nil && p.target == sha && a.repo != nil && s.root == a.repo.Root && s.epoch == a.loadSeq && s.reload == a.historyReload && a.fileFilter == "" && a.selectedSHA == p.source.sha && a.historyNavigationMember(sha)
	if tracked {
		if len(s.positions) == 0 {
			s.positions = []historyPosition{p.source}
			s.cursor = 0
		} else {
			s.positions[s.cursor] = p.source
		}
		if p.source.sha != sha {
			s.positions = append(s.positions[:s.cursor+1], historyPosition{sha, a.historyQuery})
			s.cursor++
		} else {
			s.positions[s.cursor].query = a.historyQuery
		}
		s.pending = nil
		s.selecting = true
	} else {
		a.clearHistoryNavigation()
	}
	a.selectCommit(sha)
	a.historyNavigation.selecting = false
}

// A user query edit cancels the transaction before a late page can clear it.
func (a *app) resumePendingCommitJump() {
	a.discardStaleHistoryNavigation()
	if a.historyNavigation.pending != nil && a.historyNavigationBlocked() {
		a.cancelPendingHistoryNavigation()
		return
	}
	if sha := a.pendingJumpSHA; sha != "" {
		a.pendingJumpSHA = ""
		a.jumpToCommit(sha)
	}
}

func (a *app) canNavigateHistory(direction int) bool {
	a.discardStaleHistoryNavigation()
	s := &a.historyNavigation
	if direction != -1 && direction != 1 || a.page != "history" || !a.historyNavigationVisible() || a.busy || a.loading || a.historyReloading || a.repositoryChooserOpen || a.historyRefOpen || a.hasPendingDialog(toastMain) || s.pending != nil || a.pendingJumpSHA != "" {
		return false
	}
	target := s.cursor + direction
	return len(s.positions) > 0 && target >= 0 && target < len(s.positions)
}

func (a *app) navigateHistory(direction int) {
	if !a.canNavigateHistory(direction) {
		return
	}
	s := &a.historyNavigation
	next := s.cursor + direction
	position := s.positions[next]
	if !a.historyNavigationMember(position.sha) {
		a.clearHistoryNavigation()
		return
	}
	// Preserve the current edited query when leaving a still-visible anchor.
	s.positions[s.cursor].query = a.historyQuery
	a.historyQuery = position.query
	a.historyRowsSet = false
	s.cursor = next
	s.selecting = true
	a.selectCommit(position.sha)
	a.historyNavigation.selecting = false
	a.revealSelection = true
	a.invalidate()
}
