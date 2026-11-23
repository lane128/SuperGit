package lobedesk

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"supergit/internal/updater"
	"testing"
	"time"
)

func TestSettingsRedesignAllPagesStayInsideGutters(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			for _, scale := range []float32{1, 1.15} {
				for _, width := range []int{560, 687, 688, 848} {
					t.Run(fmt.Sprintf("%s/%s/%g/%d", language, mode, scale, width), func(t *testing.T) {
						setUIFontScale(scale)
						a := &app{locale: language, themeMode: mode, showGraph: true}
						v := ui.NewTester(a.settingsRoot, width, 480)
						for _, section := range settingsSections {
							a.selectSettingsSection(section.key)
							v.Frame()
							v.Frame()
							for _, n := range v.AccessAll() {
								// The viewport's scrollbar thumb intentionally occupies the outer gutter.
								if n.Role == ui.RoleButton && n.Name == "Scroll" {
									continue
								}
								switch n.Role {
								case ui.RoleButton, ui.RoleToggleButton, ui.RoleCheckBox:
									if n.Bounds.W > 0 && (n.Bounds.X < 184 || n.Bounds.X+n.Bounds.W > float32(width)-23) {
										t.Errorf("%s control %q leaves gutters: %+v", section.key, n.Name, n.Bounds)
									}
								}
							}
						}
					})
				}
			}
		}
	}
}

func TestSettingsUpdateStatusPreservesCompleteStateMatrix(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	setUIFontScale(1.15)
	for _, language := range []locale{localeEnglish, localeChinese} {
		a := &app{locale: language}
		for _, tc := range []struct {
			state   updater.State
			total   int64
			actions []string
		}{
			{updater.StateIdle, 0, []string{"checkForUpdates"}},
			{updater.StateChecking, 0, []string{"updateStatusChecking"}},
			{updater.StateUpToDate, 0, []string{"checkForUpdates"}},
			{updater.StateAvailable, 0, []string{"updateViewDetailsAction", "checkForUpdates"}},
			{updater.StateDownloading, 100, []string{"updateViewDetailsAction", "updateCancelDownload"}},
			{updater.StateDownloading, 0, []string{"updateViewDetailsAction", "updateCancelDownload"}},
			{updater.StateVerifying, 0, []string{"updateViewDetailsAction"}},
			{updater.StateReadyToRestart, 0, []string{"updateRestartAndInstall", "updateViewDetailsAction"}},
			{updater.StateApplying, 0, []string{"checkForUpdates"}},
			{updater.StateError, 0, []string{"checkForUpdates"}},
		} {
			t.Run(fmt.Sprintf("%s/%s/%d", language, tc.state, tc.total), func(t *testing.T) {
				snap := updater.Snapshot{State: tc.state, TotalBytes: tc.total, DownloadedBytes: 42, Release: updater.ReleaseInfo{Version: "2026.11.24"}, Error: strings.Repeat("Connection interrupted. ", 8)}
				for _, last := range []time.Time{{}, time.Date(2026, 11, 24, 10, 30, 0, 0, time.UTC)} {
					snap.LastCheckedAt = last
					v := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Width(352).Children(func() { a.settingsUpdateStatus(c, snap) })
					}, 352, 800)
					status := a.updateSettingsStatusText(snap)
					if !v.HasText(status) {
						t.Fatal("update status missing")
					}
					if tc.state == updater.StateDownloading && strings.Contains(status, "%") != (tc.total > 0) {
						t.Fatal("download percentage does not match known total")
					}
					if !last.IsZero() && !v.HasText(a.tf("updateLastChecked", last.Local().Format("2006-01-02 15:04"))) {
						t.Fatal("last checked time missing")
					}
					count := 0
					for _, n := range v.AccessAll() {
						if n.Role == ui.RoleButton {
							count++
							if n.Bounds.X < 0 || n.Bounds.X+n.Bounds.W > 352 {
								t.Fatalf("update action overflows: %+v", n)
							}
						}
					}
					if count != len(tc.actions) {
						t.Fatalf("got %d actions, want %d", count, len(tc.actions))
					}
					for _, key := range tc.actions {
						n, ok := v.AccessByRole(ui.RoleButton, a.t(key))
						if !ok || n.Disabled != (tc.state == updater.StateChecking) {
							t.Fatalf("action %s has incorrect state", key)
						}
					}
				}
			})
		}
	}
}
