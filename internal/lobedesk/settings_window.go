package lobedesk

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// settingsWindowWidth and settingsWindowHeight open the panel at a size that
// shows the cards at their capped measure with room for the scrollbar. The
// window keeps its own size and position under the "settings" state key, so a
// reader who widened it finds it that way again.
const (
	settingsWindowWidth  = 820
	settingsWindowHeight = 720
	settingsWindowMinW   = 560
	settingsWindowMinH   = 480
)

// settingsWindowTitle names the panel in the title bar and the window list.
func (a *app) settingsWindowTitle() string {
	return "SuperGit · " + a.t("settings")
}

// openSettingsWindow shows the settings panel, or brings the one already open
// back in front. A closed window is gone, so the next open makes a new one.
func (a *app) openSettingsWindow() {
	if w := a.settingsWin.Load(); w != nil && !w.IsDestroyed() {
		// Show alone does not bring back a minimized panel; Restore does
		// nothing to one already on screen.
		w.Restore()
		w.Show()
		w.Focus()
		return
	}
	// The closed panel's dropdown state does not belong to the new one.
	a.paletteOpen = false
	w := mygo.NewWindow(mygo.WindowOptions{
		Title:           a.settingsWindowTitle(),
		Width:           settingsWindowWidth,
		Height:          settingsWindowHeight,
		MinWidth:        settingsWindowMinW,
		MinHeight:       settingsWindowMinH,
		BackgroundColor: "light-dark(#f7f8fa, #09090b)",
		StateKey:        "settings",
		Content:         ui.View(a.settingsRoot),
	})
	a.settingsWin.Store(w)
}

// settingsRoot draws the panel in its own window: the same cards as the page
// it used to be, without the rail or the title bar of the main window.
func (a *app) settingsRoot(c *ui.Context) {
	a.applyTheme(c)
	tk := theme.TokensOf(c)
	a.syncDockIcon(tk.Dark)
	// Dialog state is shared with the main window; only the window with the
	// keyboard draws it, so a confirmation opened here stays here and one
	// opened in the main window does not appear twice.
	focused := a.settingsFocused()
	// The page keeps one identity across theme and palette changes. Its
	// controls read the new tokens every frame and animate to them over the
	// transition, so there is no need to rebuild the page (which dropped the
	// focused control); a headless check reads the selected block in the new
	// theme's color after the transition.
	ui.Column(c).Fill().Background(tk.BgLayout).Children(func() {
		a.settingsPage(c)
		if focused {
			a.confirmDialog(c)
			a.promptDialog(c)
			a.resetDialog(c)
			a.checkoutBranchDialog(c)
			a.blameDialog(c)
			a.worktreeFormDialog(c)
		}
		feedback.ToastHost(c, feedback.ToastHostProps{})
		if focused {
			a.flushNotice(c, toastSettings)
		}
	})
}

// settingsFocused reports whether the settings window is the one taking
// keyboard input. A closed window is not.
func (a *app) settingsFocused() bool {
	w := a.settingsWin.Load()
	return w != nil && !w.IsDestroyed() && w.IsFocused()
}

// retitleSettingsWindow keeps the title in the current language after a
// language change inside the panel.
func (a *app) retitleSettingsWindow() {
	if w := a.settingsWin.Load(); w != nil && !w.IsDestroyed() {
		w.SetTitle(a.settingsWindowTitle())
	}
}
