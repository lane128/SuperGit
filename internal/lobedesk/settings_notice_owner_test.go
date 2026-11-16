package lobedesk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSettingsPageSaveWarningBelongsToItsInitiatingWindow(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "blocked-home")
			if err := os.WriteFile(home, []byte("not a directory"), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", home)
			a := &app{locale: language, themeMode: "light"}
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				a.appearanceCard(c)
			}, 820, 720)
			tt.Click(a.t("themeDark"))
			tt.Frame()
			if a.themeMode != "dark" {
				t.Fatal("theme action did not run")
			}
			if len(a.notices[toastSettings]) != 1 || len(a.notices[toastMain]) != 0 {
				t.Fatal("settings action routed its save warning by keyboard focus instead of its initiating window")
			}
			if n := a.notices[toastSettings][0]; n.title != a.t("settingsSaveFailed") || n.kind != "warning" {
				t.Fatalf("wrong settings warning: %+v", n)
			}
		})
	}
}

func TestSettingsResetResultBelongsToItsConfirmationWindow(t *testing.T) {
	for _, writable := range []bool{false, true} {
		t.Run(map[bool]string{false: "failure", true: "success"}[writable], func(t *testing.T) {
			home := t.TempDir()
			if !writable {
				home = filepath.Join(home, "blocked-home")
				if err := os.WriteFile(home, []byte("not a directory"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("HOME", home)
			before := uiFontScaleValue()
			t.Cleanup(func() { setUIFontScale(before) })
			a := &app{locale: localeEnglish, themeMode: "dark"}
			a.confirmResetSettings()
			if !a.confirm.open || a.confirm.owner != toastSettings {
				t.Fatal("reset confirmation missing or incorrectly owned")
			}
			a.confirm.action()
			if len(a.notices[toastSettings]) != 1 || len(a.notices[toastMain]) != 0 {
				t.Fatal("reset result escaped its confirmation window")
			}
			want := a.t("settingsSaveFailed")
			if writable {
				want = a.t("resetSettingsDone")
			}
			if got := a.notices[toastSettings][0].title; got != want {
				t.Fatalf("reset result %q, want %q", got, want)
			}
		})
	}
}
