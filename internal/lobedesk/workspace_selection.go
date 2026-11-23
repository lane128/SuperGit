package lobedesk

import (
	"context"
	"slices"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// The selection belongs to one visible group, independently of the preview tab.
// A refresh intersects identities rather than following a path into another group.
type workspaceFileSelection struct {
	Root, Group string
	Files       map[fileRef]bool
	Anchor      fileRef
}

func (a *app) workspaceGroup(ref fileRef) (workingFileGroup, bool) {
	for _, group := range a.workingFileGroups() {
		if group.Staged != ref.staged || a.fileSectionCollapsed(group.Key) {
			continue
		}
		for _, entry := range group.Entries {
			if entry.Path == ref.path {
				return group, true
			}
		}
	}
	return workingFileGroup{}, false
}

func (a *app) workspaceFileSelected(ref fileRef) bool {
	s := a.workspaceSelection
	if a.repo != nil && s.Root == a.repo.Root && s.Files != nil {
		return s.Files[ref]
	}
	return a.selectedPath == ref.path && a.selectedStaged == ref.staged
}

func (a *app) selectWorkspaceFile(ref fileRef, mods ui.Modifiers, contextMenu bool) {
	// A cached/read-only row can still be previewed without an open repo;
	// mutation targets separately require a live repository and status.
	if a.repo == nil {
		a.selectFile(ref.path, ref.staged)
		return
	}
	if a.loading || a.exclusiveBusy() || !a.detailNavigationAllowed() {
		return
	}
	group, ok := a.workspaceGroup(ref)
	if !ok {
		return
	}
	s := a.workspaceSelection
	if s.Root != a.repo.Root || s.Group != group.Key || s.Files == nil {
		s = workspaceFileSelection{Root: a.repo.Root, Group: group.Key, Files: map[fileRef]bool{}}
	}
	switch {
	case contextMenu && s.Files[ref]:
		// Keep membership and the range anchor; only the preview changes.
	case mods&ui.Shift != 0 && s.Files[s.Anchor]:
		from, to := -1, -1
		for i, entry := range group.Entries {
			if entry.Path == s.Anchor.path {
				from = i
			}
			if entry.Path == ref.path {
				to = i
			}
		}
		if from < 0 || to < 0 {
			return
		}
		if from > to {
			from, to = to, from
		}
		s.Files = map[fileRef]bool{}
		for _, entry := range group.Entries[from : to+1] {
			s.Files[fileRef{entry.Path, group.Staged}] = true
		}
	case !contextMenu && mods&ui.Cmd != 0:
		if s.Files[ref] {
			delete(s.Files, ref)
		} else {
			s.Files[ref] = true
			s.Anchor = ref
		}
	default:
		s.Files = map[fileRef]bool{ref: true}
		s.Anchor = ref
	}
	a.workspaceSelection = s
	if !s.Files[ref] {
		ref = fileRef{a.selectedPath, a.selectedStaged}
		if !s.Files[ref] {
			ref = fileRef{}
			for _, entry := range group.Entries {
				candidate := fileRef{entry.Path, group.Staged}
				if s.Files[candidate] {
					ref = candidate
					break
				}
			}
		}
		if !s.Files[s.Anchor] {
			a.workspaceSelection.Anchor = ref
		}
	}
	a.previewWorkspaceSelection(ref)
}

func (a *app) previewWorkspaceSelection(ref fileRef) {
	if ref.path != "" {
		a.previewWorkspaceFile(ref.path, ref.staged)
		return
	}
	a.selectedPath, a.selectedStaged = "", false
	a.diffSeq++
	a.diff = gitrepo.Diff{}
	a.diffLoading = false
	a.diffErr = ""
	a.clearLineSelection()
	if a.worktreeSelected {
		a.activateDetailTab(detailTabKey{Summary: true})
		return
	}
	a.invalidate()
}

func (a *app) selectWorkspaceGroup(ref fileRef) {
	if a.repo == nil || a.exclusiveBusy() || a.loading || !a.detailNavigationAllowed() {
		return
	}
	group, ok := a.workspaceGroup(ref)
	if !ok {
		return
	}
	s := a.workspaceSelection
	if s.Root != a.repo.Root || s.Group != group.Key || s.Files == nil {
		s = workspaceFileSelection{Root: a.repo.Root, Group: group.Key, Anchor: ref}
	}
	s.Files = map[fileRef]bool{}
	for _, entry := range group.Entries {
		s.Files[fileRef{entry.Path, group.Staged}] = true
	}
	a.workspaceSelection = s
	if !s.Files[fileRef{a.selectedPath, a.selectedStaged}] {
		a.previewWorkspaceSelection(ref)
	}
	a.invalidate()
}

func (a *app) reconcileWorkspaceSelection() {
	s := a.workspaceSelection
	if s.Files == nil {
		return
	}
	if a.repo == nil || s.Root != a.repo.Root {
		a.workspaceSelection = workspaceFileSelection{}
		return
	}
	retained := map[fileRef]bool{}
	var first fileRef
	for _, group := range a.workingFileGroups() {
		if group.Key != s.Group || a.fileSectionCollapsed(group.Key) {
			continue
		}
		for _, entry := range group.Entries {
			ref := fileRef{entry.Path, group.Staged}
			if s.Files[ref] {
				retained[ref] = true
				if first.path == "" {
					first = ref
				}
			}
		}
	}
	previous := s.Files
	s.Files = retained
	if !retained[s.Anchor] {
		s.Anchor = first
	}
	a.workspaceSelection = s
	var active fileRef
	if a.worktreeSelected {
		file := a.detailTabs().Active.File
		active = fileRef{file.Path, file.Staged}
		if active.path != "" && !previous[active] {
			// The reader opened this tab independently of the Files selection.
			// reconcileWorktreeTabs checks its version against the full status.
			return
		}
	}
	if !retained[fileRef{a.selectedPath, a.selectedStaged}] {
		// Refresh is background reconciliation, not a request to activate a
		// file tab. The normal tab reconciliation preserves Summary/focus.
		a.selectedPath, a.selectedStaged = first.path, first.staged
		a.diffSeq++
		a.diff = gitrepo.Diff{}
		a.diffLoading = false
		a.diffErr = ""
		a.clearLineSelection()
	}
	if active.path != "" {
		if len(retained) == 0 {
			a.activateDetailTab(detailTabKey{Summary: true})
		} else if !retained[active] {
			// Keep an already visible file preview inside the surviving set;
			// Summary is never activated as a file by background reconciliation.
			a.previewWorkspaceFile(a.selectedPath, a.selectedStaged)
		}
	}
}

// Snapshots are captured by the menu, never reconstructed when it is activated.
func (a *app) workspaceStageTargets(entry gitrepo.Entry, staged bool) []fileActionTarget {
	ref := fileRef{entry.Path, staged}
	s := a.workspaceSelection
	if a.repo == nil {
		return nil
	}
	if s.Root != a.repo.Root || !s.Files[ref] {
		return []fileActionTarget{{Root: a.repo.Root, Generation: a.loadSeq, Entry: entry, Staged: staged}}
	}
	var targets []fileActionTarget
	for _, group := range a.workingFileGroups() {
		if group.Key != s.Group || a.fileSectionCollapsed(group.Key) {
			continue
		}
		for _, candidate := range group.Entries {
			if s.Files[fileRef{candidate.Path, staged}] {
				targets = append(targets, fileActionTarget{Root: a.repo.Root, Generation: a.loadSeq, Entry: candidate, Staged: staged})
			}
		}
	}
	return targets
}

func (a *app) workspaceStageReady(targets []fileActionTarget) bool {
	if len(targets) == 0 || !a.worktreeFilesShown() || !a.detailNavigationAllowed() || a.repositoryChooserOpen || a.hasPendingDialog(toastMain) || len(a.lineSel) > 0 {
		return false
	}
	for _, target := range targets {
		if target.Staged != targets[0].Staged || !a.fileActionTargetCurrent(target) {
			return false
		}
	}
	return true
}

func (a *app) runWorkspaceStage(targets []fileActionTarget) {
	if !a.workspaceStageReady(targets) {
		return
	}
	if len(targets) == 1 {
		a.runFileStage(targets[0])
		return
	}
	targets = slices.Clone(targets)
	staged := targets[0].Staged
	label := a.tf("stageSelectedFiles", len(targets))
	if staged {
		label = a.tf("unstageSelectedFiles", len(targets))
	}
	paths := []string{}
	for _, target := range targets {
		if staged && target.Entry.OriginalPath != "" {
			paths = append(paths, target.Entry.OriginalPath)
		}
		paths = append(paths, target.Entry.Path)
	}
	a.perform(label, "", func(ctx context.Context, repo *gitrepo.Repository) error {
		if err := validateFileActionTargets(ctx, repo, targets, fileTargetsOnly); err != nil {
			return err
		}
		if staged {
			return repo.UnstagePaths(ctx, paths)
		}
		return repo.StagePaths(ctx, paths)
	})
}
