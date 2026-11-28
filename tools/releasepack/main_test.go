package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"supergit/internal/updater"
)

func TestGenerateKeypairAndPackageRelease(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("releasepack uses macOS ditto")
	}
	tmp := t.TempDir()
	privPath := filepath.Join(tmp, "update.key")
	pubPath := filepath.Join(tmp, "update.pub")
	if err := generateKeypair(privPath, pubPath); err != nil {
		t.Fatalf("generateKeypair: %v", err)
	}

	pubRaw, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	pubBase64 := strings.TrimSpace(string(pubRaw))
	appDir := writeAppBundle(t, tmp, "<key>SuperGitUpdatePublicKey</key><string>"+pubBase64+"</string>")

	outDir := filepath.Join(tmp, "dist")
	if err := packageRelease(packOptions{
		AppPath:  appDir,
		OutDir:   outDir,
		PrivPath: privPath,
		Platform: "darwin-arm64",
		BaseURL:  "http://127.0.0.1:9999",
		NotesZH:  "支持自动更新",
		NotesEN:  "Adds automatic updates",
	}); err != nil {
		t.Fatalf("packageRelease: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "update.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m updater.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.Version != "0.2.0" || m.Build != 45 {
		t.Fatalf("unexpected manifest version/build: %+v", m)
	}
	asset, ok := m.SelectAsset("darwin", "arm64")
	if !ok {
		t.Fatalf("missing darwin-arm64 asset in %+v", m.Platforms)
	}
	pubKey, err := updater.ParsePublicKey(pubBase64)
	if err != nil {
		t.Fatal(err)
	}
	sum, size, err := fileSHA256(filepath.Join(outDir, "SuperGit-0.2.0-darwin-arm64.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if size != asset.Size {
		t.Fatalf("asset size %d != file size %d", asset.Size, size)
	}
	if err := updater.VerifyArchiveDigest(sum, asset.SHA256, asset.Ed25519, pubKey); err != nil {
		t.Fatalf("VerifyArchiveDigest failed: %v", err)
	}
}

func TestPackageReleaseRequiresMatchingEmbeddedPublicKey(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("releasepack uses macOS ditto")
	}
	tmp := t.TempDir()
	privPath := filepath.Join(tmp, "update.key")
	pubPath := filepath.Join(tmp, "update.pub")
	if err := generateKeypair(privPath, pubPath); err != nil {
		t.Fatalf("generateKeypair: %v", err)
	}
	pubRaw, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatal(err)
	}

	const keyStringStart = "<key>SuperGitUpdatePublicKey</key><string>"
	matchingKeyEntry := keyStringStart + " " + string(pubRaw) + "</string>"
	pubBase64 := strings.TrimSpace(string(pubRaw))
	mid := len(pubBase64) / 2
	left, right := pubBase64[:mid], pubBase64[mid:]
	for _, tc := range []struct {
		name     string
		keyEntry string
		wantErr  bool
	}{
		{name: "missing", wantErr: true},
		{name: "empty", keyEntry: "<key>SuperGitUpdatePublicKey</key><string></string>", wantErr: true},
		{name: "whitespace_only", keyEntry: "<key>SuperGitUpdatePublicKey</key><string> </string>", wantErr: true},
		{name: "commented_out", keyEntry: "<!-- " + matchingKeyEntry + " -->", wantErr: true},
		{name: "nested", keyEntry: "<key>Other</key><dict>" + matchingKeyEntry + "</dict>", wantErr: true},
		{name: "comment_in_string", keyEntry: keyStringStart + left + "<!-- split -->" + right + "</string>", wantErr: true},
		{name: "cdata_in_string", keyEntry: keyStringStart + left + "<![CDATA[" + right + "]]></string>", wantErr: true},
		{name: "wrong", keyEntry: "<key>SuperGitUpdatePublicKey</key><string>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=</string>", wantErr: true},
		{name: "correct", keyEntry: matchingKeyEntry},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			err := packageRelease(packOptions{
				AppPath:  writeAppBundle(t, tmp, tc.keyEntry),
				OutDir:   filepath.Join(tmp, "dist"),
				PrivPath: privPath,
				Platform: "darwin-arm64",
			})
			if tc.wantErr {
				if err == nil || !strings.HasPrefix(err.Error(), "embedded update public key") {
					t.Fatalf("packageRelease error = %v, want an embedded update public key error", err)
				}
			} else if err != nil {
				t.Fatalf("packageRelease: %v", err)
			}
		})
	}
}

func writeAppBundle(t *testing.T, dir, keyEntry string) string {
	t.Helper()
	appDir := filepath.Join(dir, "SuperGit.app")
	if err := os.MkdirAll(filepath.Join(appDir, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>supergit</string>
<key>CFBundleIdentifier</key><string>` + updater.ExpectedBundleID + `</string>
<key>CFBundleShortVersionString</key><string>0.2.0</string>
<key>CFBundleVersion</key><string>45</string>
<key>LSMinimumSystemVersion</key><string>12.0</string>
` + keyEntry + `
</dict></plist>`
	if err := os.WriteFile(filepath.Join(appDir, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "Contents", "MacOS", "supergit"), []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return appDir
}
