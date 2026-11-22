package lobedesk

import (
	"slices"

	"github.com/egoist/mygo"
)

func (tabs *repositoryTabs) refreshRepositoryRecents() {
	a := tabs.active
	if tabs.reopenMenuItem != nil && (tabs.menuRecentRoot != a.repositoryMenuRoot() || !slices.Equal(tabs.menuRecentRoots, a.recentRepositories)) {
		tabs.installRepositoryMenu()
	}
}

func (a *app) repositoryMenuRoot() string {
	if a.repo != nil {
		return a.repo.Root
	}
	return ""
}

func (tabs *repositoryTabs) installRepositoryMenu() {
	a := tabs.active
	if a.window == nil {
		return
	}
	item := func(label string, action func(*app)) *mygo.MenuItem {
		return &mygo.MenuItem{Label: a.t(label), Click: func(_ *mygo.MenuItem, win *mygo.Window) {
			current := tabs.active
			if current.window == nil || current.window.IsDestroyed() ||
				(win != nil && win != current.window) || !current.window.IsFocused() || !tabs.canChange() {
				return
			}
			action(current)
		}}
	}
	reopen := item("reopenClosedTab", func(*app) { tabs.reopenClosedRepository() })
	reopen.Accelerator = "Cmd+Shift+T"
	reopen.Disabled = len(tabs.closed) == 0
	tabs.reopenMenuItem = reopen
	tabs.menuRecentRoot, tabs.menuRecentRoots = a.repositoryMenuRoot(), a.recentRepositories
	recent := &mygo.MenuItem{Label: a.t("openRecentRepositories"), Submenu: []*mygo.MenuItem{reopen}}
	if len(a.recentRepositories) > 0 {
		recent.Submenu = append(recent.Submenu, mygo.Separator())
	}
	for _, root := range a.recentRepositories {
		entry := item("openRecentRepositories", func(current *app) { current.openRecentRepository(root) })
		entry.Label = root
		entry.Disabled = root == tabs.menuRecentRoot
		recent.Submenu = append(recent.Submenu, entry)
	}
	checkUpdates := &mygo.MenuItem{Label: a.t("checkForUpdatesAction"), Click: func(_ *mygo.MenuItem, win *mygo.Window) {
		current := tabs.active
		if current.window == nil || current.window.IsDestroyed() {
			return
		}
		owner := toastMain
		if settings := tabs.settings.Load(); settings != nil && !settings.IsDestroyed() && (win == settings || settings.IsFocused()) {
			owner = toastSettings
		}
		current.checkForUpdates(owner)
	}}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu, Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout},
			checkUpdates,
			mygo.Separator(),
			{Role: mygo.RoleServices},
			mygo.Separator(),
			{Role: mygo.RoleHide},
			{Role: mygo.RoleHideOthers},
			{Role: mygo.RoleUnhide},
			mygo.Separator(),
			{Role: mygo.RoleQuit},
		}},
		{Role: mygo.RoleFileMenu, Submenu: []*mygo.MenuItem{
			item("newRepositoryTab", func(*app) { tabs.newTab() }),
			item("newRepo", (*app).showNewRepository),
			item("cloneRepo", (*app).showCloneRepository),
			item("openRepo", (*app).chooseRepository),
			recent,
			item("quickRepositorySwitchCommand", (*app).openQuickRepositorySwitch),
			mygo.Separator(),
			{Label: a.t("closeTab"), Accelerator: "Cmd+W", Click: func(_ *mygo.MenuItem, win *mygo.Window) {
				current := tabs.active
				if settings := tabs.settings.Load(); settings != nil && !settings.IsDestroyed() && settings.IsFocused() {
					if win == nil || win == settings {
						settings.Close()
					}
					return
				}
				if current.window != nil && !current.window.IsDestroyed() && current.window.IsFocused() && (win == nil || win == current.window) {
					tabs.askClose(current)
				}
			}},
			{Role: mygo.RoleClose, Accelerator: "Cmd+Shift+" + "W"},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	}))
}
