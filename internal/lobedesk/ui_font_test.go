package lobedesk

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestUIFontPresetsDefaultPersistenceAndThemePropagation(t *testing.T) {
	prev := uiFontKeyValue()
	defer setUIFontKey(prev)

	if got := uiFontKeyOf(""); got != uiFontSystem {
		t.Fatalf("empty uiFontKeyOf = %q; want %q", got, uiFontSystem)
	}
	if got := uiFontKeyOf("unknown-ui-font"); got != uiFontSystem {
		t.Fatalf("unknown uiFontKeyOf = %q; want %q", got, uiFontSystem)
	}
	if got := uiFontKeyOf("plex"); got != uiFontIBMPlexSans {
		t.Fatalf("alias uiFontKeyOf(plex) = %q; want %q", got, uiFontIBMPlexSans)
	}
	if got := uiFontSetting(uiFontSystem); got != "" {
		t.Fatalf("default UI font should omit persisted key, got %q", got)
	}
	if got := uiFontFamilyFor(uiFontSystem); got != "" {
		t.Fatalf("system UI font family = %q; want empty", got)
	}
	for _, opt := range uiFontOptions {
		if opt.Key == uiFontSystem {
			continue
		}
		if got := uiFontFamilyFor(opt.Key); got != opt.Family || got == "" {
			t.Errorf("uiFontFamilyFor(%q) = %q; want %q", opt.Key, got, opt.Family)
		}
	}

	path := filepath.Join(t.TempDir(), "settings.json")
	if err := writeSettings(path, persistedSettings{UIFont: uiFontInter}); err != nil {
		t.Fatal(err)
	}
	loaded := readSettings(path)
	if loaded.UIFont != uiFontInter {
		t.Fatalf("loaded UIFont = %q; want %q", loaded.UIFont, uiFontInter)
	}
	a := &app{palette: themePaletteSuperGit, themeMode: "light"}
	a.applySettings(loaded)
	if a.uiFontKey() != uiFontInter || !strings.Contains(a.uiFontFamily(), "SuperGit Inter") {
		t.Fatalf("applied uiFont = %q (%q)", a.uiFontKey(), a.uiFontFamily())
	}
	if uiFontKeyValue() != uiFontInter {
		t.Fatalf("global activeUIFont = %q; want %q", uiFontKeyValue(), uiFontInter)
	}

	// Verify applyTheme sets Theme.Font while preserving palette token lookup.
	var resolvedFont string
	var resolvedBg ui.Color
	ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		resolvedFont = c.Theme().Font
		resolvedBg = theme.TokensOf(c).BgLayout
	}, 200, 120)
	if resolvedFont != a.uiFontFamily() {
		t.Fatalf("applyTheme Theme.Font = %q; want %q", resolvedFont, a.uiFontFamily())
	}
	wantBg := supergitTokens(false).BgLayout
	if resolvedBg != wantBg {
		t.Fatalf("applyTheme lost palette tokens: BgLayout = %v; want %v", resolvedBg, wantBg)
	}

	// Cross-tab sync preserves uiFont.
	tab := &app{}
	tab.copyWindowPreferences(a)
	if tab.uiFontKey() != uiFontInter {
		t.Fatalf("copied tab uiFont = %q; want %q", tab.uiFontKey(), uiFontInter)
	}

	// Reset restores system default.
	a.applyDefaults()
	if a.uiFontKey() != uiFontSystem || a.uiFontFamily() != "" {
		t.Fatalf("reset uiFont = %q (%q); want system default", a.uiFontKey(), a.uiFontFamily())
	}
}

func TestSettingsAppearanceCardSwitchesUIFontIndependently(t *testing.T) {
	prevUI := uiFontKeyValue()
	prevCode := codeFontKeyValue()
	defer func() {
		setUIFontKey(prevUI)
		setCodeFontKey(prevCode)
	}()
	setUIFontKey(uiFontSystem)
	setCodeFontKey(codeFontJetBrainsMono)

	dir := t.TempDir()
	origDir := settingsConfigDir
	settingsConfigDir = func() (string, error) { return dir, nil }
	defer func() { settingsConfigDir = origDir }()

	a := &app{
		locale:    localeEnglish,
		themeMode: "light",
		palette:   themePaletteSuperGit,
		uiFont:    uiFontSystem,
		codeFont:  codeFontJetBrainsMono,
	}
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Width(680).Children(func() {
			a.appearanceCard(c)
		})
	}, 680, 540)

	trigger, ok := tt.AccessByRole(ui.RoleButton, "Interface font")
	if !ok || trigger.Description != "System Default" {
		t.Fatalf("Interface font dropdown should default to System Default: %+v", trigger)
	}
	allText := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(allText, "Interface font preview") || !strings.Contains(allText, "func diff(a, b int)") {
		t.Fatal("missing dual UI + code preview in appearance card")
	}

	tt.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
	for range 3 {
		tt.Frame()
	}
	tt.Click("Inter")
	for range 3 {
		tt.Frame()
	}
	if a.uiFontKey() != uiFontInter {
		t.Fatalf("clicking Inter did not switch active UI font: %q", a.uiFontKey())
	}
	if a.codeFontKey() != codeFontJetBrainsMono {
		t.Fatalf("switching UI font unexpectedly changed code font: %q", a.codeFontKey())
	}
}
