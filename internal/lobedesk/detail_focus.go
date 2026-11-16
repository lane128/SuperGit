package lobedesk

import "github.com/egoist/mygo/ui"

func (a *app) detailNavigationAllowed() bool {
	return a.page == "history" && !a.historyDetailsHidden && !a.historyRefOpen && !a.loading &&
		!a.confirm.open && !a.prompt.open && !a.blameOpen && !a.worktreeForm.open && !a.reset.open
}

func (a *app) requestDetailTabFocus() {
	a.detailFocusPending = true
	a.detailFocusContext = a.currentDetailContext()
}

func (a *app) navigateDetailTab(from detailTabKey, key ui.Key) {
	if !a.detailNavigationAllowed() {
		return
	}
	keys := []detailTabKey{{Summary: true}}
	if a.detailFileShown() {
		for _, file := range a.detailTabs().OpenFiles {
			keys = append(keys, detailTabKey{File: file})
		}
	}
	at := 0
	for i, tab := range keys {
		if tab == from {
			at = i
			break
		}
	}
	switch key {
	case ui.KeyLeft:
		at = (at - 1 + len(keys)) % len(keys)
	case ui.KeyRight:
		at = (at + 1) % len(keys)
	case ui.KeyHome:
		at = 0
	case ui.KeyEnd:
		at = len(keys) - 1
	default:
		return
	}
	a.activateDetailTab(keys[at])
	a.requestDetailTabFocus()
}

func (a *app) wireDetailTabKeys(tab ui.Element, key detailTabKey) {
	for _, arrow := range []ui.Key{ui.KeyLeft, ui.KeyRight, ui.KeyHome, ui.KeyEnd} {
		tab.OnShortcut(0, arrow, func() { a.navigateDetailTab(key, arrow) })
	}
}

func (a *app) focusRequestedDetailTab(tab ui.Element, key detailTabKey) {
	if !a.detailFocusPending {
		return
	}
	if a.detailFocusContext != a.currentDetailContext() {
		a.detailFocusPending = false
		return
	}
	if a.detailTabs().Active == key {
		tab.Focus()
		a.detailFocusPending = false
	}
}

// The focused row, rather than a hidden patch or the commit list, owns file
// navigation. Opening its neighbor also gives that row focus on the next frame.
func (a *app) navigateHistoryFile(from detailFileKey, delta int) {
	if !a.detailNavigationAllowed() || a.detailsChangesHidden {
		return
	}
	var files []detailFileKey
	if a.worktreeSelected {
		for _, ref := range a.fileRefs() {
			files = append(files, detailFileKey{Path: ref.path, Staged: ref.staged})
		}
	} else {
		if a.commitFilesLoading || a.commitFilesErr != "" {
			return
		}
		for _, file := range a.commitFiles {
			files = append(files, detailFileKey{Path: file.Path})
		}
	}
	if len(files) == 0 {
		return
	}
	at := 0
	for i, file := range files {
		if file == from {
			at = i
			break
		}
	}
	target := files[clampIndex(at+delta, len(files))]
	if a.worktreeSelected {
		a.selectFile(target.Path, target.Staged)
	} else {
		a.selectCommitFile(target.Path)
	}
	a.detailFileFocus = &target
	a.detailFileFocusContext = a.currentDetailContext()
	a.invalidate()
}

func (a *app) wireHistoryFileKeys(row ui.Element, file detailFileKey) {
	if a.page != "history" {
		return
	}
	for _, step := range []struct {
		key   ui.Key
		delta int
	}{
		{ui.KeyUp, -1}, {ui.KeyDown, 1}, {ui.KeyHome, -1 << 30}, {ui.KeyEnd, 1 << 30},
		{ui.KeyPageUp, -pageStep}, {ui.KeyPageDown, pageStep},
	} {
		row.OnShortcut(0, step.key, func() { a.navigateHistoryFile(file, step.delta) })
	}
	if a.detailFileFocus == nil {
		return
	}
	if a.detailFileFocusContext != a.currentDetailContext() {
		a.detailFileFocus = nil
		return
	}
	if *a.detailFileFocus == file {
		row.Focus()
		row.ScrollIntoView()
		a.detailFileFocus = nil
	}
}
