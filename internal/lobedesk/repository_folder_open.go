package lobedesk

import (
	"path/filepath"

	"github.com/egoist/mygo"
)

// The system may send multiple folder events before startup restore finishes,
// or while a tab holds an operation or dialog open.
func (tabs *repositoryTabs) queueRepositoryFolder(path string) {
	if !filepath.IsAbs(path) || tabs.active.localWatch.closed {
		return
	}
	tabs.repositoryFolders = append(tabs.repositoryFolders, path)
	// AppKit can deliver a folder while the window is not drawing frames.
	if !tabs.openNextRepositoryFolder() {
		tabs.active.invalidate()
	}
}

// Dispatch one request when the active tab can accept it. Rendering retries
// after UI state changes, and the existing watch tick retries without a frame.
func (tabs *repositoryTabs) openNextRepositoryFolder() bool {
	if !tabs.sessionStarted || len(tabs.repositoryFolders) == 0 || !tabs.canChange() {
		return false
	}
	path := tabs.repositoryFolders[0]
	tabs.repositoryFolders[0] = ""
	tabs.repositoryFolders = tabs.repositoryFolders[1:]
	tabs.open(tabs.active, path)
	if w := tabs.active.window; w != nil && !w.IsDestroyed() && !w.IsFocused() {
		if w.IsMinimized() {
			w.Restore()
		}
		mygo.App.Focus()
		w.Focus()
	}
	return true
}
