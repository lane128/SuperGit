package lobedesk

import (
	"runtime"

	"github.com/egoist/mygo/ui"
)

// shortcutGroup is the card section a binding is listed under. The groups
// follow the page where the key means something; the global group holds the
// keys that work anywhere or pick the page themselves.
const (
	shortcutGroupGlobal  = "global"
	shortcutGroupHistory = "history"
	// The working-tree group holds the keys that act on the changes: staging a
	// file, and the diff selection, while the history page's first row is in
	// front.
	shortcutGroupWorkspace = "workspace"
	shortcutGroupBranches  = "branches"
	shortcutGroupStash     = "stash"
	shortcutGroupWorktrees = "worktrees"
)

// shortcutGroupOrder is the order the settings page shows the groups in, and
// the order the empty ones are dropped from. Stash has no bindings of its
// own today; the list stays here so a binding added later has a home.
var shortcutGroupOrder = []string{
	shortcutGroupGlobal, shortcutGroupHistory, shortcutGroupWorkspace,
	shortcutGroupBranches, shortcutGroupStash, shortcutGroupWorktrees,
}

// appShortcut is one window-level key the app registers. The same table
// drives the dispatcher and the settings page's card, so a binding cannot
// work without being listed, and a line in the card cannot name a key the
// window does not listen for.
type appShortcut struct {
	// ID is the stable name the tests and the card key off.
	ID string
	// Mod and Key are what the window registers; Keys is how the card spells
	// them ("⌘⇧P", "⇧↵").
	Mod  ui.Modifiers
	Key  ui.Key
	Keys string
	// Group is the card section; Label is the i18n key of the action.
	Group string
	Label string
	// SkipWhenTyping leaves the key to a focused text field, so typing a
	// space or copying a selection keeps its usual meaning.
	SkipWhenTyping bool
	// Field marks a key a text field binds itself (⌘F in the search bars).
	// The card lists it; the window dispatcher does not register it.
	Field bool
	// Local actions are bound by their owning control, never by the window.
	// Their keys and actions still share this table with the Settings card.
	Local bool
	// Available hides the binding while it would mean nothing. A key that
	// registered but did nothing would swallow the system's own action.
	Available func(a *app) bool
	Run       func(a *app)
}

// shortcutTable is the table. Every window key and every search-bar key the
// app registers must come from here; `TestEveryWindowShortcutComesFromTheTable`
// reads the package's sources to keep it that way. It is a function rather
// than a package-level table so the Run closures below may call methods that
// in turn read the table, without an initialization cycle.
func shortcutTable() []appShortcut {
	openKeys, headKeys, unstageKeys, stageTrackedKeys, refreshKeys := "Ctrl+O", "Ctrl+0", "Ctrl+Shift+R", "Ctrl+Shift+A", "Ctrl+R"
	if runtime.GOOS == "darwin" {
		openKeys, headKeys, unstageKeys, stageTrackedKeys, refreshKeys = "⌘O", "⌘0", "⌘⇧R", "⌘⇧A", "⌘R"
	}
	commitMessageKeys, newTabKeys, closeTabKeys := "Ctrl+9", "Ctrl+T", "Ctrl+W"
	if runtime.GOOS == "darwin" {
		commitMessageKeys, newTabKeys, closeTabKeys = "⌘9", "⌘T", "⌘W"
	}
	reopenTabKeys := "Ctrl+Shift+T"
	if runtime.GOOS == "darwin" {
		reopenTabKeys = "⌘⇧T"
	}
	bindings := []appShortcut{
		{ID: "workspace.focusCommitMessage", Mod: ui.Cmd, Key: ui.Key9, Keys: commitMessageKeys, Group: shortcutGroupGlobal, Label: "shortcutFocusCommitMessage",
			Available: func(a *app) bool { return a.canFocusCommitMessage() },
			Run:       func(a *app) { a.focusCommitMessage() }},
		{ID: "repo.newTab", Mod: ui.Cmd, Key: ui.KeyT, Keys: newTabKeys, Group: shortcutGroupGlobal, Label: "newRepositoryTab",
			Available: func(a *app) bool {
				return a.repositoryTabs != nil && a.repositoryTabs.active == a && a.repositoryTabs.canChange()
			},
			Run: func(a *app) {
				if tabs := a.repositoryTabs; tabs != nil && tabs.active == a {
					tabs.newTab()
				}
			}},
		{ID: "repo.closeTab", Mod: ui.Cmd, Key: ui.KeyW, Keys: closeTabKeys, Group: shortcutGroupGlobal, Label: "closeTab",
			Available: func(a *app) bool { return a.repositoryTabs != nil && a.repositoryTabs.active == a },
			Run: func(a *app) {
				if tabs := a.repositoryTabs; tabs != nil && tabs.active == a {
					tabs.askClose(a)
				}
			}},
		{ID: "repo.reopenClosedTab", Mod: ui.Cmd | ui.Shift, Key: ui.KeyT, Keys: reopenTabKeys, Group: shortcutGroupGlobal, Label: "reopenClosedTab",
			Available: func(a *app) bool {
				tabs := a.repositoryTabs
				return tabs != nil && tabs.active == a && len(tabs.closed) > 0 && tabs.canChange()
			},
			Run: func(a *app) { a.repositoryTabs.reopenClosedRepository() }},
		{ID: "repo.nextTab", Mod: ui.Ctrl, Key: ui.KeyTab, Keys: "Ctrl+Tab", Group: shortcutGroupGlobal, Label: "nextRepositoryTab",
			Available: (*app).canCycleRepositoryTabs,
			Run:       func(a *app) { a.cycleRepositoryTabs(1) }},
		{ID: "repo.previousTab", Mod: ui.Ctrl | ui.Shift, Key: ui.KeyTab, Keys: "Ctrl+Shift+Tab", Group: shortcutGroupGlobal, Label: "previousRepositoryTab",
			Available: (*app).canCycleRepositoryTabs,
			Run:       func(a *app) { a.cycleRepositoryTabs(-1) }},
		{ID: "repo.open", Mod: ui.Cmd, Key: ui.KeyO, Keys: openKeys, Group: shortcutGroupGlobal, Label: "openRepo",
			Run: func(a *app) { a.chooseRepository() }},
		{ID: "history.head", Mod: ui.Cmd, Key: ui.Key0, Keys: headKeys, Group: shortcutGroupGlobal, Label: "jumpToHead",
			Run: func(a *app) { a.navigateToHead() }},
		// Pages. The sidebar prints the same Keys text, so the numbers cannot
		// drift from the bindings. History is the main window and carries the
		// working-tree row, so it took the first key when the separate
		// workspace page went away.
		{ID: "page.history", Mod: ui.Cmd, Key: ui.Key1, Keys: "⌘1", Group: shortcutGroupGlobal, Label: "shortcutOpenHistory",
			Run: func(a *app) { a.openPage("history") }},
		{ID: "page.branches", Mod: ui.Cmd, Key: ui.Key2, Keys: "⌘2", Group: shortcutGroupGlobal, Label: "shortcutOpenBranches",
			Run: func(a *app) { a.openPage("branches") }},
		{ID: "page.stash", Mod: ui.Cmd, Key: ui.Key3, Keys: "⌘3", Group: shortcutGroupGlobal, Label: "shortcutOpenStash",
			Run: func(a *app) { a.openPage("stash") }},
		{ID: "page.worktrees", Mod: ui.Cmd, Key: ui.Key4, Keys: "⌘4", Group: shortcutGroupGlobal, Label: "shortcutOpenWorktrees",
			Run: func(a *app) { a.openPage("worktrees") }},
		{ID: "window.settings", Mod: ui.Cmd, Key: ui.KeyComma, Keys: "⌘,", Group: shortcutGroupGlobal, Label: "shortcutOpenSettings",
			Run: func(a *app) { a.openSettingsWindow() }},
		{ID: "repo.reload", Mod: ui.Cmd, Key: ui.KeyR, Keys: refreshKeys, Group: shortcutGroupGlobal, Label: "shortcutReload",
			Available: func(a *app) bool { return a.canRefreshShortcut() },
			Run:       func(a *app) { a.refreshFromShortcut() }},
		{ID: "repo.reloadF5", Key: ui.KeyF5, Keys: "F5", Group: shortcutGroupGlobal, Label: "shortcutReload",
			Available: func(a *app) bool { return a.canRefreshShortcut() },
			Run:       func(a *app) { a.refreshFromShortcut() }},

		{ID: "repo.pullF7", Key: ui.KeyF7, Keys: "F7", Group: shortcutGroupGlobal, Label: "pull",
			Available: func(a *app) bool { return a.canPullShortcut() },
			Run:       func(a *app) { a.pullFromShortcut() }},
		{ID: "repo.pushF8", Key: ui.KeyF8, Keys: "F8", Group: shortcutGroupGlobal, Label: "pushButton",
			Available: func(a *app) bool { return a.canPushShortcut() },
			Run:       func(a *app) { a.pushFromShortcut() }},
		// The commit button shows ⌘↩, so the key has to work from the message
		// field too, not only while the button has the focus.
		{ID: "repo.commit", Mod: ui.Cmd, Key: ui.KeyEnter, Keys: "⌘↵", Group: shortcutGroupGlobal, Label: "shortcutCommit",
			Run: func(a *app) {
				if a.canCommitShortcut() {
					a.commit()
				}
			}},
		// Lists. A focused field or scrolling control consumes the same keys
		// first, so these only walk a list the reader is actually looking at.
		{ID: "list.down", Key: ui.KeyDown, Keys: "↓", Group: shortcutGroupGlobal, Label: "shortcutSelectNext",
			Run: func(a *app) { a.moveSelection(1) }},
		{ID: "list.up", Key: ui.KeyUp, Keys: "↑", Group: shortcutGroupGlobal, Label: "shortcutSelectPrevious",
			Run: func(a *app) { a.moveSelection(-1) }},
		{ID: "list.home", Key: ui.KeyHome, Keys: "Home", Group: shortcutGroupGlobal, Label: "shortcutListStart",
			Run: func(a *app) { a.moveSelection(-1 << 30) }},
		{ID: "list.end", Key: ui.KeyEnd, Keys: "End", Group: shortcutGroupGlobal, Label: "shortcutListEnd",
			Run: func(a *app) { a.moveSelection(1 << 30) }},
		{ID: "list.pageUp", Key: ui.KeyPageUp, Keys: "PageUp", Group: shortcutGroupGlobal, Label: "shortcutPageUp",
			Run: func(a *app) { a.moveSelection(-pageStep) }},
		{ID: "list.pageDown", Key: ui.KeyPageDown, Keys: "PageDown", Group: shortcutGroupGlobal, Label: "shortcutPageDown",
			Run: func(a *app) { a.moveSelection(pageStep) }},
		// Diffs. Shift extends the line selection, Escape drops it, ⌘C copies it.
		// ⌘C exists only while a selection does: registered without one it would
		// take the key from the system's own copy.
		{ID: "diff.extendUp", Mod: ui.Shift, Key: ui.KeyUp, Keys: "⇧↑", Group: shortcutGroupGlobal, Label: "shortcutExtendUp",
			Run: func(a *app) { a.extendDiffSelection(-1) }},
		{ID: "diff.extendDown", Mod: ui.Shift, Key: ui.KeyDown, Keys: "⇧↓", Group: shortcutGroupGlobal, Label: "shortcutExtendDown",
			Run: func(a *app) { a.extendDiffSelection(1) }},
		{ID: "diff.clearSelection", Key: ui.KeyEscape, Keys: "Esc", Group: shortcutGroupGlobal, Label: "shortcutClearSelection",
			Run: func(a *app) {
				if a.page == "history" && len(a.commitSel) > 0 {
					a.clearCommitSelection()
					a.invalidate()
					return
				}
				if len(a.lineSel) > 0 {
					a.clearLineSelection()
					a.invalidate()
				}
			}},
		{ID: "diff.copyLines", Mod: ui.Cmd, Key: ui.KeyC, Keys: "⌘C", Group: shortcutGroupGlobal, Label: "shortcutCopyLines",
			SkipWhenTyping: true,
			Available: func(a *app) bool {
				switch {
				case a.worktreeDiffShown():
					return len(a.lineSel) > 0
				case a.page == "history":
					return a.commitDiffShown() && len(a.commitSel) > 0
				}
				return false
			},
			Run: func(a *app) {
				switch {
				case a.worktreeDiffShown() && len(a.lineSel) > 0:
					a.copyText(diffLinesText(a.activeWorktreeDiff(), sortedSelection(a.lineSel)))
				case a.page == "history" && a.commitDiffShown() && len(a.commitSel) > 0:
					a.copyText(diffLinesText(a.commitDiff, sortedSelection(a.commitSel)))
				}
			}},
		// Enter runs the current page's default action. One window shortcut has
		// to carry all three: the runtime hands a key to a single window-level
		// handler, so separate registrations would leave the later ones
		// unreachable.
		{ID: "page.activate", Key: ui.KeyEnter, Keys: "↵", Group: shortcutGroupGlobal, Label: "shortcutActivate",
			Run: func(a *app) {
				a.openSelectedFile()
				a.activateBranchSelection()
				a.activateWorktreeSelection()
			}},
		// Workspace. Space is the file list's quick toggle, Shift+Enter is
		// Sublime Merge's binding; both leave a text field alone.
		{ID: "file.stage", Key: ui.KeySpace, Keys: "Space", Group: shortcutGroupWorkspace, Label: "shortcutStageFile",
			SkipWhenTyping: true, Run: func(a *app) { a.toggleSelectedStage() }},
		{ID: "file.stageShift", Mod: ui.Shift, Key: ui.KeyEnter, Keys: "⇧↵", Group: shortcutGroupWorkspace, Label: "shortcutStageFile",
			SkipWhenTyping: true, Run: func(a *app) { a.toggleSelectedStage() }},
		{ID: "workspace.stageModified", Mod: ui.Cmd | ui.Shift, Key: ui.KeyA, Keys: stageTrackedKeys, Group: shortcutGroupWorkspace, Label: "stageTracked",
			Available: func(a *app) bool { return a.canStageTrackedShortcut() },
			Run: func(a *app) {
				if a.canStageTrackedShortcut() {
					a.runBatchStageKind("stageTracked")
				}
			}},
		{ID: "workspace.unstageAll", Mod: ui.Cmd | ui.Shift, Key: ui.KeyR, Keys: unstageKeys, Group: shortcutGroupWorkspace, Label: "unstageAll",
			Available: func(a *app) bool { return a.canUnstageAllShortcut() },
			Run: func(a *app) {
				if a.canUnstageAllShortcut() {
					a.unstageAll()
				}
			}},
		{ID: "history.parent", Mod: ui.Alt, Key: ui.KeyDown, Keys: "Alt+↓", Group: shortcutGroupHistory, Label: "jumpToParent",
			SkipWhenTyping: true,
			Available:      func(a *app) bool { return a.parentNavigationTarget() != "" },
			Run:            func(a *app) { a.navigateToParent() }},
		{ID: "history.secondParent", Mod: ui.Alt | ui.Shift, Key: ui.KeyDown, Keys: "Alt+Shift+↓", Group: shortcutGroupHistory, Label: "jumpToSecondParent", Local: true,
			Available: func(a *app) bool { return a.parentNavigationTargetAt(1) != "" },
			Run:       func(a *app) { a.navigateToSecondParent() }},
		// Searches are the fields' own bindings; the card still lists them so the
		// reader finds ⌘F where the page's other keys are.
		{ID: "search.history", Mod: ui.Cmd, Key: ui.KeyF, Keys: "⌘F", Group: shortcutGroupHistory, Label: "shortcutSearchHistory", Field: true},
		{ID: "search.branches", Mod: ui.Cmd, Key: ui.KeyF, Keys: "⌘F", Group: shortcutGroupBranches, Label: "shortcutSearchBranches", Field: true},
	}
	for _, binding := range bindings {
		if binding.ID == "history.parent" {
			bindings = append(bindings, childNavigationShortcut(binding))
			bindings = append(bindings, historyNavigationShortcuts(binding)...)
			break
		}
	}
	bindings = append(bindings, quickRepositorySwitchShortcuts()...)
	bindings = append(bindings, quickCheckoutShortcut())
	bindings = append(bindings, commandPaletteShortcutBindings()...)
	return append(bindings, fileTabShortcutBindings()...)
}

// Keep manual key requests separate from operation-completion reloads.
func (a *app) canRefreshShortcut() bool {
	return a.repo != nil && !a.busy && !a.loading && !a.repositoryChooserOpen && !a.hasPendingDialog(toastMain)
}

func (a *app) refreshFromShortcut() {
	if a.canRefreshShortcut() {
		a.reload()
	}
}

// registerShortcuts binds every applicable entry to the window. Field
// bindings are left to their text fields, and an unavailable action is not
// registered at all, so the system's own handling of the key survives.
func (a *app) registerShortcuts(c *ui.Context) {
	a.discardStaleCommitMessageFocus()
	a.discardStaleDetailTabFocus()
	a.discardStaleCommandPalette()
	a.discardStaleQuickCheckout()
	a.discardStaleChildNavigation()
	a.discardStaleQuickRepositorySwitch()
	a.discardStaleSnapshotFile()
	if a.snapshotView.open || a.commandPalette.open || a.quickCheckout.open || a.childNavigation.open || a.quickRepositorySwitch.open {
		return
	}
	typing := c.FocusTakesText()
	for _, s := range shortcutTable() {
		if s.Local {
			continue
		}
		if s.Field || (s.SkipWhenTyping && typing) {
			continue
		}
		if s.Available != nil && !s.Available(a) {
			continue
		}
		run := s.Run
		c.OnWindowShortcut(s.Mod, s.Key, func() { a.dispatchWindowShortcut(run) })
	}
}

// shortcutKeys is the display text of one binding, from the same table the
// dispatcher reads. The sidebar prints the page keys with it.
func shortcutKeys(id string) string {
	for _, s := range shortcutTable() {
		if s.ID == id {
			return s.Keys
		}
	}
	return ""
}

// shortcutGroupLabel names a group in the settings card: the global group has
// its own key, the others use the page's name.
func shortcutGroupLabel(group string) string {
	if group == shortcutGroupGlobal {
		return "shortcutGroupGlobal"
	}
	return group
}
