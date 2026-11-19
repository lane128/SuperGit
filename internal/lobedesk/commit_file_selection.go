package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

type historicalFileSelection struct {
	// nil retains the legacy active-preview selection until an explicit gesture.
	Files  map[string]bool
	Anchor string
}

type historicalFileOrigin struct {
	repo          *gitrepo.Repository
	root, sha     string
	epoch, detail uint64
	selection     *historicalFileSelection
}

func (a *app) historicalFileSelectionState() *historicalFileSelection {
	if a.repo == nil || a.selectedSHA == "" || a.worktreeSelected {
		return nil
	}
	tabs := a.detailTabs()
	if tabs.HistoricalSelection == nil {
		tabs.HistoricalSelection = &historicalFileSelection{}
	}
	return tabs.HistoricalSelection
}

func (a *app) historicalFileOrigin() historicalFileOrigin {
	origin := historicalFileOrigin{repo: a.repo, sha: a.selectedSHA, epoch: a.loadSeq, detail: a.commitSeq, selection: a.historicalFileSelectionState()}
	if a.repo != nil {
		origin.root = a.repo.Root
	}
	return origin
}

func (a *app) historicalFileSelected(path string) bool {
	if selection := a.historicalFileSelectionState(); selection != nil && selection.Files != nil {
		return selection.Files[path]
	}
	return a.detailTabShown() == detailTabFile && a.detailTabs().Active.File.Path == path
}

func (a *app) historicalFileOriginCurrent(origin historicalFileOrigin) bool {
	return origin.repo != nil && a.repo == origin.repo && a.repo.Root == origin.root && a.selectedSHA == origin.sha &&
		a.loadSeq == origin.epoch && a.commitSeq == origin.detail && !a.worktreeSelected && a.page == "history" &&
		!a.busy && !a.loading && !a.historyReloading && !a.repositoryChooserOpen && !a.hasPendingDialog(toastMain) &&
		!a.detailsChangesHidden && !a.commitFilesLoading && a.commitFilesErr == "" && a.detailNavigationAllowed() &&
		origin.selection != nil && a.historicalFileSelectionState() == origin.selection
}

func (a *app) selectHistoricalFile(origin historicalFileOrigin, file gitrepo.ChangedFile, mods ui.Modifiers, contextMenu bool) {
	if !a.historicalFileOriginCurrent(origin) {
		return
	}
	at := -1
	for index, current := range a.commitFiles {
		if current == file {
			at = index
			break
		}
	}
	if at < 0 {
		return
	}
	s := origin.selection
	if s.Files == nil {
		s.Files = map[string]bool{}
		if a.detailTabShown() == detailTabFile {
			active := a.detailTabs().Active.File.Path
			for _, current := range a.commitFiles {
				if current.Path == active {
					s.Files[active] = true
					s.Anchor = active
					break
				}
			}
		}
	}
	switch {
	case contextMenu && s.Files[file.Path]:
		// A single-file menu keeps membership and its anchor, while its clicked
		// path remains the only action target and becomes the primary preview.
	case mods&ui.Shift != 0 && s.Files[s.Anchor]:
		from := -1
		for index, current := range a.commitFiles {
			if current.Path == s.Anchor {
				from = index
				break
			}
		}
		if from < 0 {
			return
		}
		if from > at {
			from, at = at, from
		}
		s.Files = map[string]bool{}
		for _, current := range a.commitFiles[from : at+1] {
			s.Files[current.Path] = true
		}
	case !contextMenu && mods&ui.Cmd != 0:
		if s.Files[file.Path] {
			delete(s.Files, file.Path)
		} else {
			s.Files[file.Path] = true
			s.Anchor = file.Path
		}
	default:
		s.Files = map[string]bool{file.Path: true}
		s.Anchor = file.Path
	}
	path := file.Path
	if !s.Files[path] {
		path = a.detailTabs().Active.File.Path
		if !s.Files[path] {
			path = a.firstSelectedHistoricalFile(s)
		}
	}
	if !s.Files[s.Anchor] {
		s.Anchor = path
	}
	a.previewHistoricalSelection(path)
}

func (a *app) firstSelectedHistoricalFile(s *historicalFileSelection) string {
	for _, file := range a.commitFiles {
		if s.Files[file.Path] {
			return file.Path
		}
	}
	return ""
}

func (a *app) previewHistoricalSelection(path string) {
	if path != "" {
		a.openDetailFile(detailFileKey{Path: path})
		return
	}
	a.activateDetailTab(detailTabKey{Summary: true})
	a.commitPath = ""
	a.clearCommitSelection()
	a.invalidate()
}

func (a *app) reconcileHistoricalFileSelection() {
	if a.worktreeSelected || a.repo == nil || a.selectedSHA == "" || !a.commitFilesLoaded || a.commitFilesLoading || a.commitFilesErr != "" {
		return
	}
	s := a.historicalFileSelectionState()
	if s.Files == nil {
		return
	}
	present := make(map[string]bool, len(a.commitFiles))
	for _, file := range a.commitFiles {
		present[file.Path] = true
	}
	changed := false
	for path := range s.Files {
		if !present[path] {
			delete(s.Files, path)
			changed = true
		}
	}
	if !changed {
		return
	}
	remaining := a.firstSelectedHistoricalFile(s)
	if !s.Files[s.Anchor] {
		s.Anchor = remaining
	}
	active := a.detailTabs().Active
	if active.Summary {
		if !present[a.commitPath] {
			a.commitPath = ""
		}
		a.invalidate()
		return
	}
	if !present[active.File.Path] {
		a.previewHistoricalSelection(remaining)
	} else {
		a.invalidate()
	}
}
