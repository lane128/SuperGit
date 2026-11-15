package desktop

import (
	"supergit/internal/gitrepo"
)

func (a *app) installSnapshot(s snapshot) snapshot {
	// Auxiliary history failure must not erase a usable loaded traversal. Keep
	// the failure distinct from an exhausted history until a refresh succeeds.
	if s.historyErr != nil {
		s.history, s.historyTips, s.historyMore = a.data.history, a.data.historyTips, a.data.historyMore
	}
	a.data = s
	return s
}

// reconcileFileSelection follows a file when it moves between Index and
// worktree groups. A vanished file falls back to a current change.
func (a *app) reconcileFileSelection() {
	path, staged := a.path, a.staged
	defer func() {
		if a.path != path || a.staged != staged {
			a.clearLineSelection()
			a.diff = gitrepo.Diff{}
			a.diffLoaded, a.diffErr = false, nil
			a.workspaceDiffReset = true
			// Reset in the view: the committed accessibility tree still borrows
			// the current ListState until the next frame replaces it.
			a.workspaceSelectionReset = true
		}
	}()
	for _, e := range a.data.status.Entries {
		if e.Path == a.path {
			if a.staged && e.Staged() || !a.staged && e.Unstaged() {
				return
			}
			if e.Unstaged() {
				a.staged = false
				return
			}
			if e.Staged() {
				a.staged = true
				return
			}
		}
	}
	for _, e := range a.data.status.Entries {
		if e.Unstaged() {
			a.path = e.Path
			a.staged = false
			return
		}
	}
	for _, e := range a.data.status.Entries {
		if e.Staged() {
			a.path = e.Path
			a.staged = true
			return
		}
	}
	a.path = ""
	a.diff = gitrepo.Diff{}
	a.diffLoaded, a.diffErr = false, nil
	a.diffLoading = false
	a.guard.Begin("diff")
}

// applyDiffResult keeps the workspace preview separate from immutable history.
func (a *app) applyDiffResult(t ticket, d gitrepo.Diff, historical bool) {
	if !a.guard.Accept(t) {
		return
	}
	if historical {
		a.commitDiff = d
	} else {
		if a.diff.Text != d.Text {
			a.clearLineSelection()
		}
		a.diff = d
		a.diffLoaded, a.diffErr = true, nil
	}
}
