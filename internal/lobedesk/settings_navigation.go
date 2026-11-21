package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

var settingsSections = [...]struct{ key, label string }{
	{"appearance", "appearance"},
	{"history", "history"},
	{"git", "settingsGit"},
	{"shortcuts", "shortcuts"},
	{"about", "about"},
}

func (a *app) activeSettingsSection() string {
	for _, section := range settingsSections {
		if section.key == a.settingsSection {
			return section.key
		}
	}
	return "appearance"
}

func (a *app) selectSettingsSection(key string) {
	valid := false
	for _, section := range settingsSections {
		valid = valid || section.key == key
	}
	if !valid || key == a.activeSettingsSection() {
		return
	}
	if a.settingsPositions == nil {
		a.settingsPositions = make(map[string]detailScrollPosition)
	}
	a.settingsPositions[a.activeSettingsSection()] = detailScrollPosition{X: a.settingsScroll.X, Y: a.settingsScroll.Y}
	a.settingsSection = key
	position := a.settingsPositions[key]
	a.settingsScroll = ui.ScrollState{X: position.X, Y: position.Y}
	a.paletteOpen = false
	a.uiFontOpen = false
	a.codeFontOpen = false
	a.invalidate()
}

func (a *app) settingsNavigation(c *ui.Context) {
	tk := theme.TokensOf(c)
	ui.Column(c).Width(132).Shrink(0).Padding(12, 8).Gap(6).
		Background(tk.BgContainerSecondary).Role(ui.RoleTabList).Label(a.t("settingsCategories")).FocusGroup(ui.Vertical).
		Children(func() {
			for _, section := range settingsSections {
				current := section.key == a.activeSettingsSection()
				button := styleWorkbenchButton(c.Key(section.key), ui.ButtonBase(c.Key(section.key)), buttonNormal, false, current).
					FillWidth().Role(ui.RoleTab).Label(a.t(section.label)).AccessSelected(current).Justify(ui.Start)
				button.Children(func() { ui.Text(c, a.t(section.label)).SingleLine() })
				if button.Clicked() {
					a.selectSettingsSection(section.key)
				}
			}
		})
}
