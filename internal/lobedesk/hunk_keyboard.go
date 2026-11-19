package lobedesk

import "github.com/egoist/mygo/ui"

// Replacing a patch must also replace its focusable row. A retained focus
// cannot silently become permission to stage a different patch at that index.
func (a *app) hunkFocusKey(index int) any {
	root, patch := "", ""
	if a.repo != nil {
		root = a.repo.Root
	}
	if index >= 0 && index < len(a.diff.Hunks) {
		patch = a.diff.Hunks[index].Patch
	}
	return struct {
		Root, Path, Patch string
		Staged            bool
		Diff, Load        uint64
		Index             int
	}{root, a.selectedPath, patch, a.selectedStaged, a.diffSeq, a.loadSeq, index}
}

func (a *app) hunkEnterAllowed() bool {
	return a.worktreeDiffShown() && a.detailTabShown() == detailTabFile &&
		a.detailNavigationAllowed() && !a.repositoryChooserOpen && !a.hasPendingDialog(toastMain) &&
		!a.diffLoading && a.diffErr == "" && len(a.lineSel) == 0 && a.hunkActionsAllowed()
}

// Bind the focused file-tab header, not the file's global default action.
// The snapshot also rejects a key queued for a replaced file/version/patch.
func (a *app) wireHunkEnter(row ui.Element, index int) {
	if index < 0 || index >= len(a.diff.Hunks) {
		return
	}
	if a.hunkEnterAllowed() {
		action := "stageHunk"
		if a.selectedStaged {
			action = "unstageHunk"
		}
		row.KeyShortcuts(0, ui.KeyEnter).Description(a.tf("hunkEnterHint", a.t(action)))
	}
	// An ancestor shortcut would swallow Enter on the nested hover button.
	if !row.Focused() {
		return
	}
	root := ""
	if a.repo != nil {
		root = a.repo.Root
	}
	path, staged, seq, load := a.selectedPath, a.selectedStaged, a.diffSeq, a.loadSeq
	patch := a.diff.Hunks[index].Patch
	row.OnShortcut(0, ui.KeyEnter, func() {
		if !row.Focused() || !a.hunkEnterAllowed() || a.repo.Root != root ||
			a.selectedPath != path || a.selectedStaged != staged || a.diffSeq != seq || a.loadSeq != load ||
			index >= len(a.diff.Hunks) || a.diff.Hunks[index].Patch != patch {
			return
		}
		a.applyHunk(index)
	})
}
