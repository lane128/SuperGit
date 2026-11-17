package lobedesk

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestExpandedNoticesFitMinimumSettingsWindow(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	setUIFontScale(1.15)
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			t.Run(string(language)+"/"+mode, func(t *testing.T) {
				a := &app{locale: language, themeMode: mode}
				var titles []string
				for i := range 3 {
					title := fmt.Sprintf("%s %d", a.t("settingsSaveFailed"), i+1)
					titles = append(titles, title)
					a.pushNotice(toastSettings, &toastNotice{kind: "warning", title: title, description: "open /Users/example/Library/Application Support/SuperGit/settings.json: permission denied", timeout: -1})
				}
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					a.toastHost(c, toastSettings)
					a.flushNotice(c, toastSettings)
				}, settingsWindowMinW, settingsWindowMinH)
				for range 8 {
					tt.Advance(100 * time.Millisecond)
				}
				front, ok := tt.Find(titles[2])
				if !ok {
					t.Fatal("newest notice missing")
				}
				tt.Move(front.X+front.W/2, front.Y+front.H/2)
				tt.Frame()
				for range 8 {
					tt.Advance(100 * time.Millisecond)
				}
				var previous ui.Rect
				for i, title := range titles {
					r, ok := tt.Find(title)
					if !ok || r.X < 16 || r.Y < 16 || r.X+r.W > settingsWindowMinW-16 || r.Y+r.H > settingsWindowMinH-16 {
						t.Fatalf("expanded notice outside minimum window: %q %+v", title, r)
					}
					if i > 0 && previous.Y+previous.H >= r.Y {
						t.Fatalf("expanded notices overlap: %+v %+v", previous, r)
					}
					previous = r
				}
			})
		}
	}
}
