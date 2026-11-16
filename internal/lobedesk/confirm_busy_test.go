package lobedesk

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestRepositoryConfirmationWaitsForReadOrWrite(t *testing.T) {
	for _, state := range []string{"loading", "busy"} {
		t.Run(state, func(t *testing.T) {
			a := &app{repo: &gitrepo.Repository{Root: t.TempDir()}}
			acted := 0
			a.askConfirm("Original target", "Original details", "Confirm target", true, func() { acted++ })
			a.loading = state == "loading"
			a.busy = state == "busy"
			tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialog(c) }, 640, 420)
			button, ok := tt.Find("Confirm target")
			if !ok {
				t.Fatal("confirmation action missing")
			}
			tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
			if acted != 0 || !a.confirm.open || a.confirm.title != "Original target" {
				t.Fatal("blocked confirmation consumed the action or lost its original target")
			}
			a.loading, a.busy = false, false
			tt.Frame()
			button, ok = tt.Find("Confirm target")
			if !ok {
				t.Fatal("ready confirmation action missing")
			}
			tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
			if acted != 1 || a.confirm.open {
				t.Fatal("ready confirmation did not execute its retained action once")
			}
		})
	}
}

func TestRepositoryConfirmationCanCancelDuringRead(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: t.TempDir()}}
	a.askConfirm("Target", "Details", "Confirm target", true, func() { t.Error("cancel ran action") })
	a.loading = true
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialog(c) }, 640, 420)
	button, ok := tt.Find(a.t("cancel"))
	if !ok {
		t.Fatal("cancel missing during read")
	}
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	if a.confirm.open {
		t.Fatal("read prevented cancellation")
	}
}

func TestIndependentSettingsConfirmationCanRunDuringRepositoryRead(t *testing.T) {
	a := &app{loading: true, repo: &gitrepo.Repository{Root: t.TempDir()}}
	a.settingsWin.Store(&mygo.Window{})
	acted := false
	a.askConfirmInWindow(toastSettings, "", "Settings target", "Details", "Confirm settings", false, func() { acted = true })
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.confirmDialogFor(c, toastSettings) }, 640, 420)
	button, ok := tt.Find("Confirm settings")
	if !ok {
		t.Fatal("settings action missing")
	}
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	if !acted || a.confirm.open {
		t.Fatal("unrelated repository read blocked settings confirmation")
	}
}
