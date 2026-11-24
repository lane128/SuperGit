package lobedesk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestDiffPaletteDefaultsPersistenceAndTabs(t *testing.T) {
	for _, value := range []string{"", "unknown", "blue-orange"} {
		a := &app{diffPalette: diffPaletteVivid}
		a.applySettings(persistedSettings{DiffPalette: value})
		if a.diffPaletteKey() != diffPaletteDelta {
			t.Fatalf("%q did not fall back to Delta", value)
		}
	}
	if defaultSettings().DiffPalette != diffPaletteDelta {
		t.Fatal("fresh installs must use Delta")
	}
	for _, option := range diffPaletteOptions {
		t.Run(option.key, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := writeSettings(path, persistedSettings{DiffPalette: option.key}); err != nil {
				t.Fatal(err)
			}
			a := &app{}
			a.applySettings(readSettings(path))
			b := &app{}
			b.copyWindowPreferences(a)
			if a.diffPaletteKey() != option.key || b.diffPaletteKey() != option.key {
				t.Fatal("palette lost on load or repository switch")
			}
			a.applyDefaults()
			if a.diffPaletteKey() != diffPaletteDelta {
				t.Fatal("reset did not restore Delta")
			}
		})
	}
}

func TestDiffPaletteSettingsChoiceAndSave(t *testing.T) {
	dir := t.TempDir()
	prev := settingsConfigDir
	settingsConfigDir = func() (string, error) { return dir, nil }
	defer func() { settingsConfigDir = prev }()
	for _, locale := range []locale{localeEnglish, localeChinese} {
		for _, width := range []float32{848, 560} {
			a := &app{locale: locale, themeMode: "light", palette: themePaletteSuperGit}
			view := ui.NewTester(a.settingsRoot, int(width), 680)
			trigger, ok := view.AccessByRole(ui.RoleButton, a.t("diffPalette"))
			if !ok || trigger.Description != a.t("diffPaletteDelta") {
				t.Fatalf("missing Delta default: %+v", trigger)
			}
			for _, key := range []string{diffPaletteClassic, diffPaletteVivid, diffPaletteDelta} {
				trigger, _ = view.AccessByRole(ui.RoleButton, a.t("diffPalette"))
				if trigger.Bounds.X < 0 || trigger.Bounds.X+trigger.Bounds.W > width {
					t.Fatal("dropdown outside settings window")
				}
				view.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
				for range 3 {
					view.Frame()
				}
				view.Click(a.t(diffPaletteOptionFor(key).label))
				for range 3 {
					view.Frame()
				}
				if a.diffPaletteKey() != key {
					t.Fatalf("choice did not apply %s", key)
				}
				path, err := settingsPath()
				if err != nil {
					t.Fatal(err)
				}
				if readSettings(path).DiffPalette != key {
					t.Fatal("choice not saved")
				}
			}
		}
	}
}

func TestDiffPaletteSaveFailureKeepsSessionChoice(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	previous := settingsConfigDir
	settingsConfigDir = func() (string, error) { return blocked, nil }
	t.Cleanup(func() { settingsConfigDir = previous })
	a := &app{locale: localeEnglish, themeMode: "light"}
	view := ui.NewTester(a.settingsRoot, 848, 680)
	trigger, ok := view.AccessByRole(ui.RoleButton, a.t("diffPalette"))
	if !ok {
		t.Fatal("palette control missing")
	}
	view.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
	view.Frame()
	view.Click(a.t("diffPaletteVivid"))
	view.Frame()
	if a.diffPaletteKey() != diffPaletteVivid {
		t.Fatal("save failure discarded the in-session choice")
	}
	if len(a.notices[toastSettings]) != 1 || a.notices[toastSettings][0].title != a.t("settingsSaveFailed") || len(a.notices[toastMain]) != 0 {
		t.Fatal("save failure must be reported in Settings")
	}
	settingsConfigDir = func() (string, error) { return dir, nil }
	if !a.saveSettingsFor(toastSettings) {
		t.Fatal("save did not recover")
	}
	path, err := settingsPath()
	if err != nil || readSettings(path).DiffPalette != diffPaletteVivid {
		t.Fatal("recovered save lost the palette")
	}
}

func TestDiffPaletteContrastAcrossUIThemes(t *testing.T) {
	for _, option := range diffPaletteOptions {
		for _, theme := range appThemes {
			for _, dark := range []bool{false, true} {
				tk := theme.config.Tokens(dark)
				a := &app{diffPalette: option.key}
				for _, kind := range []string{"add", "del", "hunk", "context", "meta"} {
					c := a.diffColors(tk, kind)
					minimum := 4.5
					if kind == "add" || kind == "del" {
						minimum = 7
					}
					if r := contrastRatio(c.text, c.background); r < minimum {
						t.Errorf("%s/%s/%t/%s text %.2f", option.key, theme.key, dark, kind, r)
					}
					if r := contrastRatio(c.gutter, c.background); r < 4.5 {
						t.Errorf("%s/%s/%t/%s gutter %.2f", option.key, theme.key, dark, kind, r)
					}
					if r := contrastRatio(c.selection, c.background); r < 3 {
						t.Errorf("%s/%s/%t/%s selection %.2f", option.key, theme.key, dark, kind, r)
					}
				}
				for _, active := range []bool{false, true} {
					bg, fg := diffSearchColors(dark, active)
					if contrastRatio(fg, bg) < 4.5 {
						t.Fatal("search contrast")
					}
				}
			}
		}
	}
}

func TestDiffPaletteRowsKeepSemanticBackgroundWhenSelected(t *testing.T) {
	for _, key := range []string{diffPaletteClassic, diffPaletteDelta, diffPaletteVivid} {
		for _, dark := range []bool{false, true} {
			for _, kind := range []string{"add", "del"} {
				a := &app{locale: localeEnglish, diffPalette: key, themeMode: "light", palette: themePaletteSuperGit, commitSel: map[int]bool{0: true}, commitCursor: 0}
				if dark {
					a.themeMode = "dark"
				}
				line := gitrepo.DiffLine{Kind: kind, Text: "+kept content", New: 1, Old: 1}
				view := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					a.commitDiffLine(c, line, "file.go", 400, 0)
				}, 400, 100)
				view.Frame()
				colors := a.diffColors(themeConfig(themePaletteSuperGit).Tokens(dark), kind)
				got := view.Image().RGBAAt(350, 10)
				if got.R != colors.background.R || got.G != colors.background.G || got.B != colors.background.B {
					t.Fatalf("%s/%t/%s selected row lost semantic background: %v want %v", key, dark, kind, got, colors.background)
				}
				found := false
				for _, info := range view.AccessAll() {
					if info.Role == ui.RoleButton && info.Checked {
						found = true
					}
				}
				if !found {
					t.Fatal("selected history gutter missing accessibility state")
				}
			}
		}
	}
}
