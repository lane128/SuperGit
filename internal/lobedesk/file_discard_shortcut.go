package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func (a *app) workspaceDiscardSurfaceAllowed() bool {
	return a.worktreeFilesShown() && a.detailNavigationAllowed() &&
		(!a.detailsChangesHidden || a.worktreeSelected) && !a.repositoryChooserOpen &&
		!a.hasPendingDialog(toastMain) && len(a.lineSel) == 0
}

// Bind only the tracked worktree version. Capturing the rendered source also
// prevents a retained row callback from following a refresh or repository switch.
func (a *app) wireFileDiscardKey(row ui.Element, entry gitrepo.Entry, staged bool) {
	if staged || isUntracked(entry) || !entry.WorkspaceUnstaged() {
		return
	}
	repo, generation := a.repo, a.loadSeq
	row.OnShortcut(0, ui.KeyBackspace, func() {
		if repo == nil || a.repo != repo || a.loadSeq != generation || !a.workspaceDiscardSurfaceAllowed() {
			return
		}
		targets := a.workspaceStageTargets(entry, false)
		if !a.discardTargetsCurrent(targets, false) {
			return
		}
		for _, target := range targets {
			if target.Staged || isUntracked(target.Entry) {
				return
			}
		}
		// Match existing row actions: focus outside the set selects this row, while
		// focus inside retains membership. Preparation freezes the resulting targets.
		a.selectWorkspaceFile(fileRef{entry.Path, false}, 0, true)
		a.prepareScopedDiscardTargets(targets, false, true)
	})
}
