package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"runtime"
)

func fileTabShortcutBindings() []appShortcut {
	binding := func(id string, mod ui.Modifiers, key ui.Key, keys string, previous bool) appShortcut {
		label, arrow := "shortcutNextFileTab", ui.KeyRight
		if previous {
			label, arrow = "shortcutPreviousFileTab", ui.KeyLeft
		}
		return appShortcut{ID: id, Mod: mod, Key: key, Keys: keys, Group: shortcutGroupGlobal, Label: label,
			Available: func(a *app) bool { return a.canNavigateFileTabs() },
			Run: func(a *app) {
				if a.canNavigateFileTabs() {
					a.navigateDetailTab(a.detailTabs().Active, arrow)
				}
			}}
	}
	bindings := []appShortcut{
		binding("detail.previousFileTab", ui.Ctrl, ui.KeyPageUp, "Ctrl+PageUp", true),
		binding("detail.nextFileTab", ui.Ctrl, ui.KeyPageDown, "Ctrl+PageDown", false),
	}
	if runtime.GOOS == "darwin" {
		bindings = append(bindings,
			binding("detail.previousFileTabBracket", ui.Cmd|ui.Shift, ui.KeyBracketLeft, "⌘⇧[", true),
			binding("detail.nextFileTabBracket", ui.Cmd|ui.Shift, ui.KeyBracketRight, "⌘⇧]", false),
			binding("detail.previousFileTabCommandPage", ui.Cmd, ui.KeyPageUp, "⌘PageUp", true),
			binding("detail.nextFileTabCommandPage", ui.Cmd, ui.KeyPageDown, "⌘PageDown", false),
		)
	}
	return bindings
}

// Editors own modified PageUp/PageDown before window shortcuts. Consume a
// matching, available binding here so a key runs once through the same guards.
func (a *app) fileTabTextKey(k ui.TextKey) bool {
	for _, s := range fileTabShortcutBindings() {
		if s.Mod == k.Mods && s.Key == k.Key && s.Available(a) {
			s.Run(a)
			return true
		}
	}
	return false
}

func (a *app) canFocusDetailTab() bool {
	return a.detailNavigationAllowed() && !a.exclusiveBusy() && !a.historyReloading &&
		!a.repositoryChooserOpen && !a.hasPendingDialog(toastMain)
}

func (a *app) canNavigateFileTabs() bool {
	return a.repo != nil && a.canFocusDetailTab() && a.detailFileShown()
}

func (a *app) discardStaleDetailTabFocus() {
	if a.detailFocusPending && (!a.canFocusDetailTab() ||
		a.detailFocusContext != a.currentDetailContext() || a.detailFocusSeq != a.loadSeq ||
		a.detailFocusTarget != a.detailTabs().Active ||
		!a.detailFocusTarget.Summary && !a.detailFileShown()) {
		a.detailFocusPending = false
	}
}
