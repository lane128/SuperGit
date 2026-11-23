package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestSettingsFormStacksAtCompactBreakpoint(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, scale := range []float32{1, 1.15} {
			setUIFontScale(scale)
			a := &app{locale: language, themeMode: "light"}
			v := ui.NewTester(a.settingsRoot, 848, 680)
			for _, width := range []int{560, 687, 688, 848} {
				v.SetSize(width, 680)
				v.Frame()
				v.Frame()
				label, ok := v.Find(a.t("themeMode"))
				if !ok {
					t.Fatal("theme label missing")
				}
				first, ok := v.AccessByRole(ui.RoleToggleButton, a.t("themeSystem"))
				if !ok {
					t.Fatal("theme option missing")
				}
				if width < 688 && first.Bounds.Y < label.Y+label.H {
					t.Errorf("width %d scale %g: compact control is not below label", width, scale)
				}
				if width >= 688 && first.Bounds.Y > label.Y+label.H {
					t.Errorf("width %d: wide row unexpectedly stacked", width)
				}
				for _, key := range []string{"themeLight", "themeDark"} {
					option, ok := v.AccessByRole(ui.RoleToggleButton, a.t(key))
					if !ok {
						t.Fatal(key)
					}
					if option.Bounds.W != first.Bounds.W || option.Bounds.Y != first.Bounds.Y {
						t.Errorf("%s width %d: segmented options are not equal-width on one line", language, width)
					}
					if option.Bounds.X+option.Bounds.W > float32(width)-24 {
						t.Errorf("%s width %d: option overflows gutter: %+v", language, width, option.Bounds)
					}
				}
			}
		}
	}
}

func TestSettingsFetchIntervalMenuPreservesTimingAndPreferences(t *testing.T) {
	path := isolateSettings(t)
	a := &app{locale: localeEnglish, settingsSection: "git", autoFetch: true}
	v := ui.NewTester(a.settingsRoot, 560, 480)
	trigger, ok := v.AccessByRole(ui.RoleButton, a.t("autoFetchEvery"))
	if !ok {
		t.Fatal("fetch interval must be a menu trigger")
	}
	v.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
	v.Frame()
	item, ok := v.Find(a.fetchIntervalLabel(15))
	if !ok {
		t.Fatal("15-minute menu option missing")
	}
	v.ClickAt(item.X+item.W/2, item.Y+item.H/2)
	v.Frame()
	if a.autoFetchEvery != 15 || a.lastAutoFetch.IsZero() {
		t.Fatal("interval choice did not start a fresh timer")
	}
	var reload app
	reload.applySettings(readSettings(path))
	if reload.autoFetchEvery != 15 || !reload.autoFetch {
		t.Fatal("interval did not persist or changed automatic fetch")
	}
}
