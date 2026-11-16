package lobedesk

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestMainConfirmationDoesNotDrawInSettingsWindow(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.askConfirm("Main object", "details", "Confirm main", true, func() {})
	settings := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialogFor(c, toastSettings) }, 640, 420)
	if settings.HasText("Main object") {
		t.Fatal("main confirmation moved into settings window")
	}
	main := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialog(c) }, 640, 420)
	if !main.HasText("Main object") {
		t.Fatal("main window lost its confirmation")
	}
}

func TestSettingsConfirmationStaysInItsWindow(t *testing.T) {
	a := &app{}
	a.settingsWin.Store(&mygo.Window{})
	a.askConfirmInWindow(toastSettings, "", "Settings object", "details", "Confirm settings", false, func() {})
	main := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialog(c) }, 640, 420)
	if main.HasText("Settings object") {
		t.Fatal("settings confirmation moved to main window")
	}
	settings := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialogFor(c, toastSettings) }, 640, 420)
	if !settings.HasText("Settings object") {
		t.Fatal("settings window lost its confirmation")
	}
}

func TestClosedSettingsWindowDoesNotLeaveBlockingConfirmation(t *testing.T) {
	a := &app{}
	a.askConfirmInWindow(toastSettings, "", "Closed settings object", "details", "Confirm settings", false, func() {})
	ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialog(c) }, 640, 420)
	if a.confirm.open {
		t.Fatal("closed settings window left an unreachable confirmation")
	}
}

func TestNewConfirmationCanReplaceStaleRepositoryConfirmation(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.askConfirm("Old object", "details", "Confirm old", true, func() {})
	a.repo = &gitrepo.Repository{Root: "second"}
	a.askConfirm("Current object", "details", "Confirm current", true, func() {})
	if a.confirm.title != "Current object" {
		t.Fatal("stale confirmation prevented a current request")
	}
}

func TestNewConfirmationDoesNotReplacePendingObject(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	first, second := false, false
	a.askConfirm("First object", "first details", "Confirm first", true, func() { first = true })
	a.askConfirm("Second object", "second details", "Confirm second", true, func() { second = true })
	if a.confirm.title != "First object" {
		t.Fatal("new confirmation replaced the object awaiting consent")
	}
	if len(a.notices[toastMain]) != 1 || a.notices[toastMain][0].kind != "warning" {
		t.Fatal("rejected request must explain that a confirmation is already open")
	}
	a.confirm.action()
	if !first || second {
		t.Fatal("confirmation did not retain the original action")
	}
}

func TestRepositoryConfirmationExitsOnRootChange(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.askConfirm("Old object", "old details", "Confirm old", true, func() {})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialog(c) }, 640, 420)
	if !tt.HasText("Old object") {
		t.Fatal("fixture did not show confirmation")
	}
	a.repo = &gitrepo.Repository{Root: "first"}
	tt.Frame()
	if !tt.HasText("Old object") {
		t.Fatal("same-root refresh removed confirmation")
	}
	a.repo = &gitrepo.Repository{Root: "second"}
	tt.Frame()
	if a.confirm.open || tt.HasText("Old object") {
		t.Fatal("old confirmation survived repository change")
	}
}

func TestSettingsResetConfirmationUsesPanelWithoutKeyboardFocus(t *testing.T) {
	a := &app{}
	// With no native event loop, no window can own keyboard focus. Invoke the
	// panel-only entry first, then supply its lifetime for the render checks.
	a.confirmResetSettings()
	a.settingsWin.Store(&mygo.Window{})
	main := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialog(c) }, 640, 420)
	if main.HasText(a.t("resetSettingsTitle")) {
		t.Fatal("settings reset confirmation followed keyboard focus into main window")
	}
	panel := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialogFor(c, toastSettings) }, 640, 420)
	if !panel.HasText(a.t("resetSettingsTitle")) {
		t.Fatal("settings panel lost its reset confirmation")
	}
}

func TestSettingsConfirmationSurvivesRepositoryChange(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	a.confirmResetSettings()
	a.settingsWin.Store(&mygo.Window{})
	a.repo = &gitrepo.Repository{Root: "second"}
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialogFor(c, toastSettings) }, 640, 420)
	if !tt.HasText(a.t("resetSettingsTitle")) {
		t.Fatal("repository change removed a settings confirmation")
	}
}

func TestRepositoryConfirmationActionRejectsChangedRoot(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "first"}}
	acted := false
	a.askConfirm("Old object", "old details", "Confirm old", true, func() { acted = true })
	staleAction := a.confirm.action
	a.repo = &gitrepo.Repository{Root: "second"}
	staleAction()
	if acted {
		t.Fatal("old confirmation action ran against another repository")
	}
}
