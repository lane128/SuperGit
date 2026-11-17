package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSettingsCategoriesExposeOnlySelectedContent(t *testing.T) {
	isolateSettings(t)
	a := &app{locale: localeEnglish, themeMode: "light"}
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() { a.settingsPage(c) })
	}, settingsWindowMinW, settingsWindowMinH)
	for _, key := range []string{"appearance", "history", "settingsGit", "shortcuts", "about"} {
		if _, ok := tt.AccessByRole(ui.RoleTab, a.t(key)); !ok {
			t.Fatalf("settings category %q has no native tab", key)
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleCheckBox, a.t("autoFetch")); ok {
		t.Error("Git controls appear in Appearance")
	}
	tab, _ := tt.AccessByRole(ui.RoleTab, a.t("history"))
	tt.ClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
	tt.Frame()
	if !tt.HasText(a.t("detailsDiff")) {
		t.Error("History does not expose detail settings")
	}
	if tt.HasText(a.t("themeDark")) {
		t.Error("Appearance controls remain in History")
	}
	tab, _ = tt.AccessByRole(ui.RoleTab, a.t("about"))
	tt.ClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
	tt.Frame()
	if !tt.HasText(a.t("resetSettingsAction")) {
		t.Error("About has no reset action")
	}
	if tt.HasText(a.t("autoFetch")) {
		t.Error("Git settings remain in About")
	}
	// Manual tab activation: arrow keys move within navigation, Enter selects.
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.activeSettingsSection() != "shortcuts" {
		t.Errorf("keyboard category activation selected %q; want shortcuts", a.activeSettingsSection())
	}
}

func TestSettingsCategoryScrollSurvivesSwitchAndThemeChange(t *testing.T) {
	isolateSettings(t)
	a := &app{locale: localeEnglish, themeMode: "light", showGraph: true}
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() { a.settingsPage(c) })
	}, settingsWindowMinW, settingsWindowMinH)
	selectTab := func(label string) {
		t.Helper()
		tab, ok := tt.AccessByRole(ui.RoleTab, a.t(label))
		if !ok {
			t.Fatalf("missing category %s", label)
		}
		tt.ClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
		tt.Frame()
		tt.Frame()
	}
	selectTab("history")
	a.settingsScroll.Y = a.settingsScroll.MaxY / 2
	tt.Frame()
	position := a.settingsScroll.Y
	if position <= 0 {
		t.Fatal("History needs scrollable content for restoration test")
	}
	selectTab("about")
	selectTab("history")
	if a.settingsScroll.Y != position {
		t.Errorf("returning to History scrolled to %.1f; want %.1f", a.settingsScroll.Y, position)
	}
	a.themeMode = "dark"
	tt.Frame()
	if a.activeSettingsSection() != "history" || a.settingsScroll.Y != position {
		t.Error("theme change lost current category or position")
	}
}
