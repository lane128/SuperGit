package lobedesk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestHistoryVisualSettingsRoundTrip(t *testing.T) {
	oldDir, oldScale := settingsConfigDir, uiFontScaleValue()
	dir := t.TempDir()
	settingsConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { settingsConfigDir = oldDir; setUIFontScale(oldScale) })
	for _, graph := range []string{"rounded", "chamfer", "compact"} {
		for _, ref := range []string{"segmented", "stripe"} {
			var saved persistedSettings
			if err := json.Unmarshal([]byte(`{"historyGraphStyle":"`+graph+`","historyRefStyle":"`+ref+`","historyLine":"thick","historyNode":"large"}`), &saved); err != nil {
				t.Fatal(err)
			}
			a := &app{}
			a.applySettings(saved)
			if !a.saveSettings() {
				t.Fatal("saving visual settings failed")
			}
			raw, err := os.ReadFile(filepath.Join(dir, "supergit", "settings.json"))
			if err != nil {
				t.Fatal(err)
			}
			var values map[string]any
			if err = json.Unmarshal(raw, &values); err != nil {
				t.Fatal(err)
			}
			if values["historyGraphStyle"] != graph || values["historyRefStyle"] != ref || values["historyLine"] != "thick" || values["historyNode"] != "large" {
				t.Fatalf("lost independent preferences: %s", raw)
			}
		}
	}
}

func TestHistoryVisualSettingsFallbackAndReset(t *testing.T) {
	oldDir, oldScale := settingsConfigDir, uiFontScaleValue()
	dir := t.TempDir()
	settingsConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { settingsConfigDir = oldDir; setUIFontScale(oldScale) })
	for _, raw := range []string{`{}`, `{"historyGraphStyle":"unknown","historyRefStyle":"unknown"}`} {
		var s persistedSettings
		_ = json.Unmarshal([]byte(raw), &s)
		a := &app{}
		a.applySettings(s)
		for _, reset := range []bool{false, true} {
			if reset {
				a.applyDefaults()
			}
			if !a.saveSettings() {
				t.Fatal("save failed")
			}
			b, _ := os.ReadFile(filepath.Join(dir, "supergit", "settings.json"))
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			if m["historyGraphStyle"] != "rounded" || m["historyRefStyle"] != "segmented" {
				t.Fatalf("invalid defaults/reset: %s", b)
			}
		}
	}
}

func TestHistoryVisualChoicesSaveIndependently(t *testing.T) {
	oldDir, oldScale := settingsConfigDir, uiFontScaleValue()
	dir := t.TempDir()
	settingsConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { settingsConfigDir = oldDir; setUIFontScale(oldScale) })
	a := &app{locale: localeEnglish}
	view := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).FillWidth().Children(func() { a.historyColumnsCard(c); a.graphCard(c) })
	}, 640, 800)
	for _, label := range []string{"Compact stripe", "Short curve", "Chamfer", "Split capsule"} {
		button, ok := view.AccessByRole(ui.RoleToggleButton, label)
		if !ok {
			t.Fatalf("missing option %s", label)
		}
		view.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
		for range 3 {
			view.Frame()
		}
	}
	raw, err := os.ReadFile(filepath.Join(dir, "supergit", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if m["historyGraphStyle"] != "chamfer" || m["historyRefStyle"] != "segmented" {
		t.Fatalf("choices overwrite each other: %s", raw)
	}
}
