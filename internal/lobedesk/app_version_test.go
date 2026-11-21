package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestParseBundleVersionAndFormatting(t *testing.T) {
	plist := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
	<key>CFBundleShortVersionString</key><string>0.1.0</string>
	<key>CFBundleVersion</key><string>42</string>
</dict></plist>`)
	short, build := parseBundleVersion(plist)
	if short != "0.1.0" || build != "42" {
		t.Fatalf("parseBundleVersion = (%q, %q), want (0.1.0, 42)", short, build)
	}
	if got := formatVersionLabel(short, build); got != "v0.1.0 (42)" {
		t.Fatalf("formatVersionLabel = %q, want v0.1.0 (42)", got)
	}
	if got := formatVersionLabel("v1.2.3", ""); got != "v1.2.3" {
		t.Fatalf("formatVersionLabel without build = %q, want v1.2.3", got)
	}
	if got := formatVersionLabel("", ""); got != "v"+defaultAppVersion {
		t.Fatalf("formatVersionLabel fallback = %q, want v%s", got, defaultAppVersion)
	}
}

func TestAboutCardShowsVersionLabel(t *testing.T) {
	isolateSettings(t)
	prevVersion, prevBuild := appVersion, appBuild
	appVersion, appBuild = "0.1.0", "12"
	t.Cleanup(func() {
		appVersion, appBuild = prevVersion, prevBuild
	})

	a := &app{locale: localeEnglish, themeMode: "light", settingsSection: "about"}
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() { a.settingsPage(c) })
	}, settingsWindowMinW, settingsWindowMinH)
	if !tt.HasText(a.t("settingsVersion")) {
		t.Fatalf("About card missing %q label", a.t("settingsVersion"))
	}
	if !tt.HasText("v0.1.0 (12)") {
		t.Fatalf("About card missing formatted version label")
	}
}
