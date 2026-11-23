package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// revealIfSelected keeps keyboard navigation visible: the arrow keys can move
// the selection beyond the viewport, so the row that just became current
// scrolls itself into view. Mouse selection never sets the flag, so clicking a
// row does not yank the list.
func (a *app) revealIfSelected(row ui.Element, selected bool) {
	if !selected || !a.revealSelection {
		return
	}
	row.ScrollIntoView()
	a.revealSelection = false
}

// worktreeFilesShown reports whether the working tree's file list is the one
// on screen. The history page's first row carries the working tree, so
// selecting it is what makes the files column the live changes.
func (a *app) worktreeFilesShown() bool {
	return a.page == "history" && a.worktreeSelected
}

// toggleSelectedStage stages or unstages the working tree's selected file, the
// way Space works in Sublime Merge's file list.
func (a *app) toggleSelectedStage() {
	if !a.canToggleSelectedStage() {
		return
	}
	entry, _ := a.selectedEntry()
	a.toggleStage(entry, a.selectedStaged)
}

// canToggleSelectedStage keeps Space quiet while a field takes text, a diff
// line selection is active, or the selected file is conflicted.
func (a *app) canToggleSelectedStage() bool {
	if !a.worktreeFilesShown() || len(a.lineSel) > 0 || a.repo == nil || a.loading || a.busy {
		return false
	}
	entry, ok := a.selectedEntry()
	if !ok {
		return false
	}
	_, ready := a.fileActionTarget(entry, a.selectedStaged)
	return ready
}

// openSelectedFile opens the working tree's selected file in its default app,
// the keyboard equivalent of a double click.
func (a *app) openSelectedFile() {
	if !a.canOpenSelectedFile() {
		return
	}
	a.openFile(a.selectedPath, "open")
}

// canOpenSelectedFile keeps the window's Enter fallback out of the Diff.
// A replaced hunk can lose focus before its queued key arrives; that key must
// not open the file externally. Focused Diff controls handle their own Enter.
func (a *app) canOpenSelectedFile() bool {
	if !a.worktreeFilesShown() || a.worktreeDiffShown() || a.selectedPath == "" || len(a.lineSel) > 0 || a.repo == nil || a.loading || a.exclusiveBusy() {
		return false
	}
	return !a.confirm.open && !a.prompt.open && !a.blameOpen && !a.worktreeForm.open && !a.reset.open
}

// The working tree composer stays above the patch on both Summary and file tabs.
func (a *app) canCommitShortcut() bool {
	if a.repo == nil || a.busy || a.loading {
		return false
	}
	if a.historyDetailsHidden || a.historyRefOpen {
		return false
	}
	if a.confirm.open || a.prompt.open || a.blameOpen || a.worktreeForm.open || a.reset.open {
		return false
	}
	return a.worktreeFilesShown()
}

// worktreeDiffShown reports whether the working tree's patch is the one on
// screen: the history page with its first row selected and the file tab in
// front.
func (a *app) worktreeDiffShown() bool {
	if !a.worktreeFilesShown() || !a.detailPaneVisible() {
		return false
	}
	if a.detailTabShown() == detailTabFile {
		return true
	}
	return a.overviewLineTarget != nil && a.overviewTargetVisible(*a.overviewLineTarget)
}

// extendDiffSelection grows a diff line selection by one row. The working
// tree and a commit keep separate selections; only the patch on screen moves.
func (a *app) extendDiffSelection(delta int) {
	if a.worktreeDiffShown() && a.moveLineSelection(delta, true) {
		return
	}
	a.moveCommitLineSelection(delta, true)
}

// pageStep is how far PageUp and PageDown move a list's selection. The rows
// differ in height between pages (a commit row is 50pt, a file row 30pt) and
// the viewport is not measured here, so one step covers roughly the smaller
// screenful; the selection scrolls itself into view either way.
const pageStep = 10

// moveSelection walks the current page's selectable list with the arrow keys.
// Pages without a list selection, or with an empty list, do nothing.
func (a *app) moveSelection(delta int) {
	if a.repo == nil || a.loading || a.exclusiveBusy() {
		return
	}
	// The ref picker covers the history list, and its search box holds the
	// keyboard; with Tab on the picker's Retry, an arrow or page key moved
	// and loaded the commit list behind the popup.
	if a.historyRefOpen {
		return
	}
	// Once a diff line is selected, the arrows walk changed lines; leaving
	// the selection or switching pages restores list navigation.
	if a.moveLineSelection(delta, false) {
		return
	}
	switch a.page {
	case "history":
		if !a.moveCommitLineSelection(delta, false) {
			a.moveHistorySelection(delta)
		}
	case "branches":
		a.moveBranchSelection(delta)
		return
	case "worktrees":
		a.moveWorkspaceSelection(delta)
		return
	default:
		a.moveFileSelection(delta)
	}
}

// listStartIndex is the row a list move begins at when nothing is selected
// yet: a step down starts at the first row, a jump to the end (End, and
// PageDown, which is a large step) lands on the last one, and a step up or
// Home has nowhere before the list and stays on the first row.
func listStartIndex(delta, length int) int {
	return clampIndex(delta-1, length)
}

type fileRef struct {
	path   string
	staged bool
}

// The workspace's three sections, as the keys the folded state is kept under.
// The order here is the order they are drawn in.
const (
	filesSectionUnstaged  = "unstaged"
	filesSectionStaged    = "staged"
	filesSectionUntracked = "untracked"
)

func (a *app) fileSectionCollapsed(key string) bool {
	return a.filesCollapsed[key]
}

// fileRefs lists the workspace rows in the order the page draws them. Folded
// sections are skipped: the arrows must not move onto rows that are not on
// screen, which is the same rule the history list follows while a walk is out.
func (a *app) fileRefs() []fileRef {
	var refs []fileRef
	add := func(key string, entries []gitrepo.Entry, staged bool) {
		if a.fileSectionCollapsed(key) {
			return
		}
		for _, entry := range entries {
			refs = append(refs, fileRef{path: entry.Path, staged: staged})
		}
	}
	for _, group := range a.workingFileGroups() {
		add(group.Key, group.Entries, group.Staged)
	}
	return refs
}

func (a *app) moveFileSelection(delta int) {
	refs := a.fileRefs()
	if len(refs) == 0 {
		return
	}
	index := -1
	for i, ref := range refs {
		if ref.path == a.selectedPath && ref.staged == a.selectedStaged {
			index = i
			break
		}
	}
	if index < 0 {
		index = listStartIndex(delta, len(refs))
	} else {
		index = clampIndex(index+delta, len(refs))
	}
	ref := refs[index]
	if ref.path == a.selectedPath && ref.staged == a.selectedStaged {
		return
	}
	a.revealSelection = true
	a.selectWorkspaceFile(ref, 0, false)
}

func (a *app) moveHistorySelection(delta int) {
	// The list draws a placeholder while a new walk is out. Arrows would then
	// move a selection the reader cannot see, and the row they land on belongs
	// to the filter that was just replaced.
	if a.historyReloading {
		return
	}
	// A failed read leaves the old rows hidden behind the error placeholder;
	// moving the selection there would load details for a list nobody sees.
	if a.historyErr != "" {
		return
	}
	commits := a.projectHistoryMerges(a.historyWithStashes(a.filteredHistoryRows()))
	if len(commits) == 0 {
		return
	}
	total := len(commits) + 1
	position, found := 0, a.worktreeSelected
	if !found {
		for i, commit := range commits {
			if commit.SHA == a.selectedSHA || commit.SHA == a.selectedStash {
				position, found = i+1, true
				break
			}
		}
	}
	if !found {
		position = listStartIndex(delta, total)
	} else {
		position = clampIndex(position+delta, total)
	}
	a.revealSelection = true
	if position == 0 {
		a.selectWorktree()
		return
	}
	index := position - 1
	if stash := a.stashTimeline.bySHA[commits[index].SHA]; stash != nil {
		a.selectStash(stash.SHA)
		return
	}
	if commits[index].SHA == a.selectedSHA {
		// Down on the last loaded row still asks for the next page: the
		// selection cannot move, but the list can grow under it.
		if delta > 0 && index+3 >= len(commits) {
			a.loadMoreHistory()
		}
		return
	}
	a.revealSelection = true
	a.selectCommit(commits[index].SHA)
	// Reaching the end of the loaded window asks for the next page, so the
	// arrows can keep walking into older history.
	if index+3 >= len(commits) {
		a.loadMoreHistory()
	}
}

func clampIndex(index, length int) int {
	return min(max(index, 0), length-1)
}
