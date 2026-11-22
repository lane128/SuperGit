package lobedesk

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
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
	if w := a.settingsWindowSlot().Load(); w != nil && !w.IsDestroyed() {
		// Show alone does not bring back a minimized panel; Restore does
		// nothing to one already on screen.
		w.Restore()
		w.Show()
		w.Focus()
		return
	}
	// The closed panel's dropdown state does not belong to the new one.
	a.paletteOpen = false
	a.uiFontOpen = false
	a.codeFontOpen = false
	w := mygo.NewWindow(mygo.WindowOptions{
		Title:                a.settingsWindowTitle(),
		Width:                settingsWindowWidth,
		Height:               settingsWindowHeight,
		MinWidth:             settingsWindowMinW,
		MinHeight:            settingsWindowMinH,
		TitleBarStyle:        mygo.TitleBarHidden,
		TrafficLightPosition: &mygo.Point{X: 12, Y: 12},
		BackgroundColor:      "light-dark(#f7f8fa, #09090b)",
		StateKey:             "settings",
		Content:              ui.View(func(c *ui.Context) { a.activeRepositorySession().settingsRoot(c) }),
	})
	a.settingsWindowSlot().Store(w)
}

// settingsRoot applies the same theme to the panel and its title bar.
func (a *app) settingsRoot(c *ui.Context) {
	a.applyTheme(c)
	tk := theme.TokensOf(c)
	a.syncDockIcon(tk.Dark)
	// Dialogs retain their initiating window even when keyboard focus changes.
	// Focus determines whether this window drains its notification queue.
	focused := a.settingsFocused()
	// The page keeps one identity across theme and palette changes. Its
	// controls read the new tokens every frame and animate to them over the
	// transition, so there is no need to rebuild the page (which dropped the
	// focused control); a headless check reads the selected block in the new
	// theme's color after the transition.
	ui.Column(c).Fill().Background(tk.BgLayout).Children(func() {
		a.settingsPage(c)
		a.confirmDialogFor(c, toastSettings)
		a.noticeDetailsDialog(c, toastSettings)
		a.promptDialogFor(c, toastSettings)
		a.resetDialogFor(c, toastSettings)
		a.checkoutBranchDialogFor(c, toastSettings)
		a.worktreeFormDialogFor(c, toastSettings)
		a.blameDialogFor(c, toastSettings)
		a.updateDialog(c, toastSettings)
		a.toastHost(c, toastSettings)
		if focused {
			a.flushNotice(c, toastSettings)
		}
	})
}

// settingsFocused reports whether the settings window is the one taking
// keyboard input. A closed window is not.
func (a *app) settingsFocused() bool {
	w := a.settingsWindowSlot().Load()
	return w != nil && !w.IsDestroyed() && w.IsFocused()
}

// retitleSettingsWindow keeps the title in the current language after a
// language change inside the panel.
func (a *app) retitleSettingsWindow() {
	if w := a.settingsWindowSlot().Load(); w != nil && !w.IsDestroyed() {
		w.SetTitle(a.settingsWindowTitle())
	}
}
