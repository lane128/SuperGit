package lobedesk

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestSettingsSaveFailureRemainsVisibleUntilDismissed(t *testing.T) {
	for _, owner := range []toastOwner{toastMain, toastSettings} {
		t.Run(map[toastOwner]string{toastMain: "main", toastSettings: "settings"}[owner], func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "blocked-home")
			if err := os.WriteFile(home, []byte("not a directory"), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("HOME", home)
			a := &app{locale: localeEnglish}
			if a.saveSettingsFor(owner) {
				t.Fatal("save unexpectedly succeeded with a blocked settings path")
			}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.toastHost(c, owner)
				a.flushNotice(c, owner)
			}, 820, 720)
			for range 5 {
				tt.Advance(50 * time.Millisecond)
				tt.Frame()
			}
			tt.Advance(30 * time.Second)
			tt.Frame()
			tt.Advance(time.Second)
			tt.Frame()
			if !tt.HasText(a.t("settingsSaveFailed")) {
				t.Fatal("unsaved settings warning expired without being dismissed")
			}
			if err := tt.Click(a.t("close")); err != nil {
				t.Fatalf("persistent warning cannot be dismissed: %v", err)
			}
			for range 6 {
				tt.Advance(100 * time.Millisecond)
			}
			if tt.HasText(a.t("settingsSaveFailed")) {
				t.Fatal("dismissed settings warning remains visible")
			}
		})
	}
}
