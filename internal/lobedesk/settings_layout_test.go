package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestSettingsDetailKeepsItsMeasureAndSidePadding(t *testing.T) {
	for _, mode := range []string{"light", "dark"} {
		t.Run(mode, func(t *testing.T) {
			a := &app{locale: localeEnglish, themeMode: mode}
			view := ui.NewTester(a.settingsRoot, 820, 720)
			for _, width := range []int{560, 687, 688, 848, 1200} {
				view.SetSize(width, 720)
				view.Frame()
				view.Frame()
				panel, ok := view.AccessByRole(ui.RoleGroup, a.t("settingsContent"))
				if !ok {
					t.Fatal("selected settings card has no named group")
				}
				left := panel.Bounds.X - 160
				right := float32(width) - panel.Bounds.X - panel.Bounds.W
				if left != 24 || right < 24 {
					t.Errorf("at width %d, content must follow navigation by 24: left=%g right=%g", width, left, right)
				}
				if want := float32(min(640, width-160-48)); panel.Bounds.W != want {
					t.Errorf("at width %d, content width=%g; want %g", width, panel.Bounds.W, want)
				}
			}
		})
	}
}

func TestSettingsTitleBarTracksAppearanceAndClearsWindowControls(t *testing.T) {
	a := &app{locale: localeEnglish, palette: themePaletteCatppuccinMacchiato}
	view := ui.NewTester(a.settingsRoot, settingsWindowMinW, settingsWindowMinH)
	bar := ui.TitleBar{Left: 76, Right: 100, Height: 38}
	view.SetTitleBar(bar)
	for _, tc := range []struct {
		mode                 string
		systemDark, wantDark bool
		palette              string
	}{
		{"light", true, false, themePaletteCatppuccinMacchiato},
		{"dark", false, true, themePaletteCatppuccinMacchiato},
		{"system", true, true, themePaletteCatppuccinMacchiato},
		{"system", false, false, themePaletteCatppuccinMacchiato},
		{"dark", false, true, themePaletteSuperGit},
	} {
		a.themeMode, a.palette = tc.mode, tc.palette
		view.SetDark(tc.systemDark)
		view.Frame()
		title, ok := view.Find(a.settingsWindowTitle())
		if !ok || title.X < bar.Left || title.X+title.W > settingsWindowMinW-bar.Right || title.Y+title.H > bar.Height {
			t.Fatalf("title overlaps window controls or is outside the title bar: %+v", title)
		}
		want := themeConfig(tc.palette).Tokens(tc.wantDark).BgLayout
		img := view.Image()
		for _, x := range []int{4, settingsWindowMinW / 2, settingsWindowMinW - 4} {
			p := img.RGBAAt(x, 4)
			if p.R != want.R || p.G != want.G || p.B != want.B {
				t.Fatalf("%s/%s systemDark=%v at x=%d: got %v, want %v", tc.palette, tc.mode, tc.systemDark, x, p, want)
			}
		}
	}
}

func TestCodeFontCanBeChangedAtMinimumSettingsSize(t *testing.T) {
	path := isolateSettings(t)
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, tc := range []struct {
		language         locale
		down, up, follow string
	}{
		{localeEnglish, "Decrease code font size", "Increase code font size", "Follow interface size"},
		{localeChinese, "减小代码字号", "增大代码字号", "跟随界面字号"},
	} {
		t.Run(string(tc.language), func(t *testing.T) {
			setUIFontScale(fontScaleLarge)
			a := &app{locale: tc.language, themeMode: "dark", codeFontSize: 16.5}
			view := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				a.settingsPage(c)
			}, settingsWindowMinW, settingsWindowMinH)
			click := func(label string, disabled bool) {
				t.Helper()
				button, ok := view.AccessByRole(ui.RoleButton, label)
				// The padded, narrower card can wrap the follow-size control.
				// Reach it through the real scroll viewport before clicking.
				if ok && button.Bounds.Y+button.Bounds.H > settingsWindowMinH {
					view.Scroll(settingsWindowMinW-40, settingsWindowMinH-40, 0, button.Bounds.Y+button.Bounds.H-settingsWindowMinH+24)
					view.Frame()
					button, ok = view.AccessByRole(ui.RoleButton, label)
				}
				if !ok || button.Offscreen || button.Disabled != disabled || button.Bounds.X < 132 || button.Bounds.X+button.Bounds.W > settingsWindowMinW || button.Bounds.Y+button.Bounds.H > settingsWindowMinH {
					t.Fatalf("control %q (want disabled=%v) has unexpected state: %+v", label, disabled, button)
				}
				view.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
				view.Frame()
			}
			checkSize := func(want float32) {
				t.Helper()
				var restarted app
				restarted.applySettings(readSettings(path))
				if a.codeFontSize != want || restarted.codeFontSize != want {
					t.Fatalf("want %g pt, got %g pt in the UI and %g pt after reload", want, a.codeFontSize, restarted.codeFontSize)
				}
			}
			click(tc.up, false)
			checkSize(17)
			click(tc.up, false)
			click(tc.up, false)
			click(tc.up, true)
			checkSize(19)
			for range 8 {
				click(tc.down, false)
			}
			click(tc.down, true)
			checkSize(11)
			click(tc.follow, false)
			checkSize(0)
			click(tc.follow, true)
			if a.patchFontSize() != 11.5*fontScaleLarge {
				t.Fatal("the restored automatic code size should follow the large interface scale")
			}
		})
	}
}

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
						{"git", "settingsGit", []string{a.t("autoFetch"), a.t("autoFetchPrune"), a.fetchIntervalLabel(0), a.t("autoFetchEvery")}},
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
