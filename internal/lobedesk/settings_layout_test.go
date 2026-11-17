package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestSettingsChoicesFitMinimumWindowAndResetCanBeReached(t *testing.T) {
	isolateSettings(t)
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			for _, scale := range []float32{1, 1.15} {
				t.Run(fmt.Sprintf("%s/%s/%.2f", language, mode, scale), func(t *testing.T) {
					setUIFontScale(scale)
					a := &app{locale: language, themeMode: mode, showGraph: true, autoFetch: true}
					a.settingsWin.Store(&mygo.Window{})
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Fill().Children(func() {
							a.settingsPage(c)
							a.confirmDialogFor(c, toastSettings)
						})
					}, settingsWindowMinW, settingsWindowMinH)
					cases := []struct {
						key, label string
						controls   []string
					}{
						{"appearance", "appearance", []string{a.t("themeSystem"), a.t("themeLight"), a.t("themeDark"), "中文", "English", a.t("fontSizeSmall"), a.t("fontSizeStandard"), a.t("fontSizeLarge")}},
						{"history", "history", []string{a.t("showAuthor"), a.t("showTime"), a.t("showHistoryGraph"), a.t("graphLineThin"), a.t("graphLineStandard"), a.t("graphLineThick"), a.t("graphNodeSmall"), a.t("graphNodeStandard"), a.t("graphNodeLarge"), a.t("detailsInfo"), a.t("detailsChanges"), a.t("detailsDiff")}},
						{"git", "settingsGit", []string{a.t("autoFetch"), a.t("autoFetchPrune"), a.fetchIntervalLabel(0), a.fetchIntervalLabel(5), a.fetchIntervalLabel(15), a.fetchIntervalLabel(30), a.fetchIntervalLabel(60)}},
						{"shortcuts", "shortcuts", nil},
						{"about", "about", []string{a.t("resetSettingsAction")}},
					}
					for _, category := range cases {
						tab, ok := tt.AccessByRole(ui.RoleTab, a.t(category.label))
						if !ok {
							t.Fatalf("category %q missing", category.key)
						}
						tt.ClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
						for range 4 {
							tt.Frame()
						}
						if a.activeSettingsSection() != category.key {
							t.Fatalf("category %s did not activate", category.key)
						}
						for _, label := range category.controls {
							bounds, ok := tt.Find(label)
							if !ok {
								t.Errorf("setting %q missing from %s", label, category.key)
								continue
							}
							if bounds.X < 132 || bounds.X+bounds.W > settingsWindowMinW+1 {
								t.Errorf("setting %q outside content pane: %+v", label, bounds)
							}
						}
						if category.key == "history" && a.settingsScroll.MaxY <= 0 {
							t.Error("History has no independent scroll range")
						}
					}
					a.settingsScroll.Y = a.settingsScroll.MaxY
					for range 4 {
						tt.Frame()
					}
					reset, ok := tt.Find(a.t("resetSettingsAction"))
					if !ok || reset.Y < 0 || reset.Y+reset.H > settingsWindowMinH+1 {
						t.Fatalf("reset cannot be reached: %+v", reset)
					}
					tt.ClickAt(reset.X+reset.W/2, reset.Y+reset.H/2)
					tt.Frame()
					if !a.confirm.open || a.confirm.owner != toastSettings {
						t.Fatal("visible reset did not open settings-owned confirmation")
					}
				})
			}
		}
	}
}
