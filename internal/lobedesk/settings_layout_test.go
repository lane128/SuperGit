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
					for range 4 {
						tt.Frame()
					}
					labels := []string{
						a.t("themeSystem"), a.t("themeLight"), a.t("themeDark"), "中文", "English",
						a.t("fontSizeSmall"), a.t("fontSizeStandard"), a.t("fontSizeLarge"),
						a.t("showAuthor"), a.t("showTime"), a.t("showHistoryGraph"),
						a.t("graphLineThin"), a.t("graphLineStandard"), a.t("graphLineThick"),
						a.t("graphNodeSmall"), a.t("graphNodeStandard"), a.t("graphNodeLarge"),
						a.t("detailsInfo"), a.t("detailsChanges"), a.t("detailsDiff"),
						a.t("autoFetch"), a.t("autoFetchPrune"), a.fetchIntervalLabel(0),
						a.fetchIntervalLabel(5), a.fetchIntervalLabel(15), a.fetchIntervalLabel(30), a.fetchIntervalLabel(60),
						a.t("resetSettingsAction"),
					}
					for _, label := range labels {
						bounds, ok := tt.Find(label)
						if !ok {
							t.Errorf("setting %q is missing", label)
							continue
						}
						if bounds.X < 0 || bounds.X+bounds.W > settingsWindowMinW+1 {
							t.Errorf("setting %q extends outside minimum window: %+v", label, bounds)
						}
					}
					if a.settingsScroll.MaxY <= 0 {
						t.Fatal("settings do not retain a scroll range")
					}
					a.settingsScroll.Y = a.settingsScroll.MaxY
					for range 4 {
						tt.Frame()
					}
					reset, ok := tt.Find(a.t("resetSettingsAction"))
					if !ok || reset.Y < 0 || reset.Y+reset.H > settingsWindowMinH+1 {
						t.Fatalf("reset cannot be reached at the bottom: %+v", reset)
					}
					tt.ClickAt(reset.X+reset.W/2, reset.Y+reset.H/2)
					tt.Frame()
					if !a.confirm.open || a.confirm.owner != toastSettings {
						t.Fatal("visible reset action did not open its settings-owned confirmation")
					}
				})
			}
		}
	}
}
