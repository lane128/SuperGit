package lobedesk

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCodeFontPresetsDefaultAndPersistence(t *testing.T) {
	prev := codeFontKeyValue()
	defer setCodeFontKey(prev)

	if got := codeFontKeyOf(""); got != codeFontJetBrainsMono {
		t.Fatalf("empty codeFontKeyOf = %q; want %q", got, codeFontJetBrainsMono)
	}
	if got := codeFontKeyOf("unknown-font"); got != codeFontJetBrainsMono {
		t.Fatalf("unknown codeFontKeyOf = %q; want %q", got, codeFontJetBrainsMono)
	}
	if got := codeFontSetting(codeFontJetBrainsMono); got != "" {
		t.Fatalf("default code font should omit persisted key, got %q", got)
	}
	if len(codeFontOptions) < 5 {
		t.Fatalf("expected multiple popular monospace options, got %d", len(codeFontOptions))
	}
	for _, opt := range codeFontOptions {
		if got := codeFontFamilyFor(opt.Key); got != opt.Family || got == "" {
			t.Errorf("codeFontFamilyFor(%q) = %q; want %q", opt.Key, got, opt.Family)
		}
	}

	path := filepath.Join(t.TempDir(), "settings.json")
	if err := writeSettings(path, persistedSettings{CodeFont: codeFontFiraCode}); err != nil {
		t.Fatal(err)
	}
	loaded := readSettings(path)
	if loaded.CodeFont != codeFontFiraCode {
		t.Fatalf("loaded CodeFont = %q; want %q", loaded.CodeFont, codeFontFiraCode)
	}
	a := &app{}
	a.applySettings(loaded)
	if a.codeFontKey() != codeFontFiraCode || a.patchFontFamily() != "SuperGit Fira Code" {
		t.Fatalf("applied codeFont = %q (%q)", a.codeFontKey(), a.patchFontFamily())
	}
	if codeFontKeyValue() != codeFontFiraCode {
		t.Fatalf("global activeCodeFont = %q; want %q", codeFontKeyValue(), codeFontFiraCode)
	}

	// Copy preferences across repository tabs.
	tab := &app{}
	tab.copyWindowPreferences(a)
	if tab.codeFontKey() != codeFontFiraCode {
		t.Fatalf("copied tab codeFont = %q; want %q", tab.codeFontKey(), codeFontFiraCode)
	}

	// Reset restores JetBrains Mono.
	a.applyDefaults()
	if a.codeFontKey() != codeFontJetBrainsMono || a.patchFontFamily() != "SuperGit JetBrains Mono" {
		t.Fatalf("reset codeFont = %q (%q); want JetBrains Mono", a.codeFontKey(), a.patchFontFamily())
	}
}

func TestSettingsAppearanceCardSwitchesCodeFont(t *testing.T) {
	prev := codeFontKeyValue()
	defer setCodeFontKey(prev)
	setCodeFontKey(codeFontJetBrainsMono)

	dir := t.TempDir()
	origDir := settingsConfigDir
	settingsConfigDir = func() (string, error) { return dir, nil }
	defer func() { settingsConfigDir = origDir }()

	a := &app{locale: localeEnglish, themeMode: "light", palette: themePaletteSuperGit, codeFont: codeFontJetBrainsMono}
	tt := ui.NewTester(a.settingsRoot, settingsWindowWidth, settingsWindowHeight)

	trigger, ok := tt.AccessByRole(ui.RoleButton, "Code font")
	if !ok || trigger.Description != "JetBrains Mono" {
		t.Fatalf("Code font dropdown should default to JetBrains Mono: %+v", trigger)
	}
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "func diff(a, b int)") {
		t.Fatal("missing live monospace preview sample in appearance card")
	}

	tt.ClickAt(trigger.Bounds.X+trigger.Bounds.W/2, trigger.Bounds.Y+trigger.Bounds.H/2)
	for range 3 {
		tt.Frame()
	}
	tt.Click("Geist Mono")
	for range 3 {
		tt.Frame()
	}
	if a.codeFontKey() != codeFontGeistMono || a.patchFontFamily() != "SuperGit Geist Mono" {
		t.Fatalf("clicking Geist Mono did not switch active code font: %q (%q)", a.codeFontKey(), a.patchFontFamily())
	}
}
