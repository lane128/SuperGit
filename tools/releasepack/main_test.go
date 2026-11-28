package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
	appDir := writeAppBundle(t, tmp, "<key>SuperGitUpdatePublicKey</key><string>"+pubBase64+"</string>", pubBase64)

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
		name          string
		keyEntry      string
		executableKey *string
		wantErr       bool
	}{
		{name: "missing", wantErr: true},
		{name: "empty", keyEntry: "<key>SuperGitUpdatePublicKey</key><string></string>", wantErr: true},
		{name: "whitespace_only", keyEntry: "<key>SuperGitUpdatePublicKey</key><string> </string>", wantErr: true},
		{name: "commented_out", keyEntry: "<!-- " + matchingKeyEntry + " -->", wantErr: true},
		{name: "nested", keyEntry: "<key>Other</key><dict>" + matchingKeyEntry + "</dict>", wantErr: true},
		{name: "comment_in_string", keyEntry: keyStringStart + left + "<!-- split -->" + right + "</string>", wantErr: true},
		{name: "cdata_in_string", keyEntry: keyStringStart + left + "<![CDATA[" + right + "]]></string>", wantErr: true},
		{name: "wrong", keyEntry: "<key>SuperGitUpdatePublicKey</key><string>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=</string>", wantErr: true},
		{name: "binary_mismatch", keyEntry: matchingKeyEntry, executableKey: new("wrong key"), wantErr: true},
		{name: "binary_missing", keyEntry: matchingKeyEntry, executableKey: new(""), wantErr: true},
		{name: "correct", keyEntry: matchingKeyEntry},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			key := pubBase64
			if tc.executableKey != nil {
				key = *tc.executableKey
			}
			outDir := filepath.Join(tmp, "dist")
			app := writeAppBundle(t, tmp, tc.keyEntry, key)
			if tc.name == "binary_missing" {
				if err := os.Remove(filepath.Join(app, "Contents", "MacOS", "supergit")); err != nil {
					t.Fatal(err)
				}
			}
			err := packageRelease(packOptions{
				AppPath:  app,
				OutDir:   outDir,
				PrivPath: privPath,
				Platform: "darwin-arm64",
			})
			if tc.wantErr {
				if err == nil || !strings.HasPrefix(err.Error(), "embedded update public key") {
					t.Errorf("packageRelease error = %v, want an embedded update public key error", err)
				}
				if _, err := os.Stat(outDir); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("output directory exists or could not be checked: %v", err)
				}
			} else if err != nil {
				t.Fatalf("packageRelease: %v", err)
			}
		})
	}
}

func TestPackageReleaseRedactsInaccessiblePrivateKeyPath(t *testing.T) {
	out := filepath.Join(t.TempDir(), "dist")
	const supplied = "input-that-must-not-appear-in-errors"
	err := packageRelease(packOptions{OutDir: out, PrivPath: filepath.Join(t.TempDir(), supplied)})
	if err == nil || strings.Contains(err.Error(), supplied) {
		t.Error("missing private key error disclosed its supplied path or was absent")
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("key error created output directory: %v", err)
	}
}

func TestPackageReleaseRejectsTagVersionMismatch(t *testing.T) {
	tmp := t.TempDir()
	priv, pub := filepath.Join(tmp, "update.key"), filepath.Join(tmp, "update.pub")
	if err := generateKeypair(priv, pub); err != nil {
		t.Fatal(err)
	}
	publicKey, err := os.ReadFile(pub)
	if err != nil {
		t.Fatal(err)
	}
	key := strings.TrimSpace(string(publicKey))
	app := writeAppBundle(t, tmp, "<key>SuperGitUpdatePublicKey</key><string>"+key+"</string>", key)
	out := filepath.Join(tmp, "dist")
	cmd := exec.Command("go", "run", ".", "-app", app, "-priv", priv, "-out", out, "-tag", "v0.2.1")
	log, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(log), "CFBundleShortVersionString") || !strings.Contains(string(log), "v0.2.1") {
		t.Fatalf("package release error = %v, output = %s", err, log)
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("mismatching tag created the output directory: %v", err)
	}
}

func TestPackageReleaseRejectsNonTopLevelVersion(t *testing.T) {
	tmp := t.TempDir()
	priv, pub := filepath.Join(tmp, "update.key"), filepath.Join(tmp, "update.pub")
	if err := generateKeypair(priv, pub); err != nil {
		t.Fatal(err)
	}
	publicKey, err := os.ReadFile(pub)
	if err != nil {
		t.Fatal(err)
	}
	key := strings.TrimSpace(string(publicKey))
	const version = "<key>CFBundleShortVersionString</key><string>0.2.0</string>"
	for _, tc := range []struct{ name, value string }{
		{"commented", "<!-- " + version + " -->"},
		{"nested", "<key>Other</key><dict>" + version + "</dict>"},
		{"split", "<key>CFBundleShortVersionString</key><string>0.2.<!-- split -->0</string>"},
		{"padded", "<key>CFBundleShortVersionString</key><string> 0.2.0&#32;</string>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			app := writeAppBundle(t, root, "<key>SuperGitUpdatePublicKey</key><string>"+key+"</string>", key)
			file := filepath.Join(app, "Contents", "Info.plist")
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(strings.Replace(string(data), version, tc.value, 1)), 0600); err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(root, "dist")
			err = packageRelease(packOptions{AppPath: app, OutDir: out, PrivPath: priv, Platform: "darwin-arm64"})
			if err == nil || !strings.Contains(err.Error(), "CFBundleShortVersionString") {
				t.Errorf("packageRelease = %v, want version error", err)
			}
			if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("invalid version created output directory: %v", err)
			}
		})
	}
}

func TestPackageReleasePreservesExistingArtifacts(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("releasepack uses macOS ditto")
	}
	for _, name := range []string{"SuperGit-0.2.0-darwin-arm64.zip", "update.json"} {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			priv, pub := filepath.Join(tmp, "update.key"), filepath.Join(tmp, "update.pub")
			if err := generateKeypair(priv, pub); err != nil {
				t.Fatal(err)
			}
			pubBytes, err := os.ReadFile(pub)
			if err != nil {
				t.Fatal(err)
			}
			key := strings.TrimSpace(string(pubBytes))
			app := writeAppBundle(t, tmp, "<key>SuperGitUpdatePublicKey</key><string>"+key+"</string>", key)
			out := filepath.Join(tmp, "dist")
			if err := os.Mkdir(out, 0700); err != nil {
				t.Fatal(err)
			}
			existing := filepath.Join(out, name)
			const sentinel = "existing verified artifact"
			if err := os.WriteFile(existing, []byte(sentinel), 0600); err != nil {
				t.Fatal(err)
			}
			err = packageRelease(packOptions{AppPath: app, OutDir: out, PrivPath: priv, Platform: "darwin-arm64"})
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("packageRelease = %v, want conflict for %s", err, name)
			}
			data, err := os.ReadFile(existing)
			if err != nil || string(data) != sentinel {
				t.Errorf("existing artifact overwritten, read error = %v", err)
			}
			entries, err := os.ReadDir(out)
			if err != nil || len(entries) != 1 {
				t.Errorf("packaging conflict left additional artifacts: %v, %v", entries, err)
			}
		})
	}
}

func TestPackageReleaseRejectsLinkedExecutable(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("releasepack uses macOS ditto")
	}
	root := t.TempDir()
	priv, pub := filepath.Join(root, "update.key"), filepath.Join(root, "update.pub")
	if err := generateKeypair(priv, pub); err != nil {
		t.Fatal(err)
	}
	keyData, err := os.ReadFile(pub)
	if err != nil {
		t.Fatal(err)
	}
	key := strings.TrimSpace(string(keyData))
	app := writeAppBundle(t, root, "<key>SuperGitUpdatePublicKey</key><string>"+key+"</string>", key)
	bin := filepath.Join(app, "Contents", "MacOS", "supergit")
	external := filepath.Join(root, "external-executable")
	if err := os.Rename(bin, external); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, bin); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "release")
	if err := packageRelease(packOptions{AppPath: app, OutDir: out, PrivPath: priv, Platform: "darwin-arm64"}); err == nil {
		t.Fatal("packaged a symlink using the key in a file outside the bundle")
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid bundle created an output directory: %v", err)
	}
}

func TestPackageReleaseRecoversInterruptedPair(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("releasepack uses macOS ditto")
	}
	for _, scenario := range []string{"same bundle", "changed bundle", "changed notes", "omitted notes", "extra recovery file", "already complete"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			priv, pub := filepath.Join(root, "update.key"), filepath.Join(root, "update.pub")
			if err := generateKeypair(priv, pub); err != nil {
				t.Fatal(err)
			}
			keyData, err := os.ReadFile(pub)
			if err != nil {
				t.Fatal(err)
			}
			key := strings.TrimSpace(string(keyData))
			app := writeAppBundle(t, root, "<key>SuperGitUpdatePublicKey</key><string>"+key+"</string>", key)
			out := filepath.Join(root, "dist")
			opts := packOptions{AppPath: app, OutDir: out, PrivPath: priv, Platform: "darwin-arm64", Tag: "v0.2.0", NotesEN: "Original notes"}
			if err := packageRelease(opts); err != nil {
				t.Fatal(err)
			}
			const archiveName = "SuperGit-0.2.0-darwin-arm64.zip"
			archive := filepath.Join(out, archiveName)
			manifest := filepath.Join(out, "update.json")
			original, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatal(err)
			}
			oldSum, _, err := fileSHA256(archive)
			if err != nil {
				t.Fatal(err)
			}
			// Simulate a stop after the prepared archive is installed but
			// before its manifest is linked into the output directory.
			pending := filepath.Join(out, ".release-pack-pending")
			if scenario != "already complete" {
				if err := os.Mkdir(pending, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Link(archive, filepath.Join(pending, archiveName)); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(manifest, filepath.Join(pending, "update.json")); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "extra recovery file" {
				if err := os.WriteFile(filepath.Join(pending, "keep.txt"), []byte("operator notes"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			when := time.Date(2020, 1, 2, 3, 4, 6, 0, time.UTC)
			if err := os.Chtimes(filepath.Join(app, "Contents", "Info.plist"), when, when); err != nil {
				t.Fatal(err)
			}
			if scenario == "changed bundle" {
				if err := os.WriteFile(filepath.Join(app, "Contents", "MacOS", "supergit"), []byte("#!/bin/sh\n# "+key+"\necho different\n"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "changed notes" {
				opts.NotesEN = "Different notes"
			}
			if scenario == "omitted notes" {
				opts.NotesEN = ""
			}
			fresh := filepath.Join(root, "fresh.zip")
			if err := archiveAppBundle(app, fresh); err != nil {
				t.Fatal(err)
			}
			freshSum, _, err := fileSHA256(fresh)
			if err != nil || freshSum == oldSum {
				t.Fatalf("fixture must re-archive to different bytes: %v", err)
			}
			err = packageRelease(opts)
			if sum, _, readErr := fileSHA256(archive); readErr != nil || sum != oldSum {
				t.Fatalf("retry modified previously installed archive: %v", readErr)
			}
			if scenario != "same bundle" && scenario != "already complete" {
				if scenario == "extra recovery file" {
					if got, err := os.ReadFile(filepath.Join(pending, "keep.txt")); err != nil || string(got) != "operator notes" {
						t.Errorf("retry did not preserve the additional recovery file: %v", err)
					}
				}
				if err == nil {
					t.Error("retry accepted changed release inputs")
				}
				if _, err := os.Stat(manifest); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("conflicting retry installed a manifest: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(manifest); err != nil || !bytes.Equal(got, original) {
				t.Errorf("retry failed to restore the original manifest: %v", err)
			}
			if _, err := os.Stat(pending); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("successful retry left its pending directory: %v", err)
			}
		})
	}
}

func writeAppBundle(t *testing.T, dir, keyEntry, executableKey string) string {
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
	if err := os.WriteFile(filepath.Join(appDir, "Contents", "MacOS", "supergit"), []byte("#!/bin/sh\n# "+executableKey+"\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return appDir
}

func TestReleaseMakePassesArgumentsWithoutEchoingValues(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make unavailable")
	}
	tmp := t.TempDir()
	stub := filepath.Join(tmp, "go-stub")
	captured := filepath.Join(tmp, "args")
	if err := os.WriteFile(stub, []byte(`#!/bin/sh
printf '%s\000' "$@" > "$CAPTURED_ARGS"
`), 0700); err != nil {
		t.Fatal(err)
	}
	const privatePath = `/tmp/key path with "quotes" and $HOME`
	const notes = "Line one \"quoted\" and $HOME\nLine two"
	const publicKey = "fixture-public-key"
	t.Setenv("CAPTURED_ARGS", captured)
	t.Setenv("MAKEFLAGS", "")
	t.Setenv("MAKEOVERRIDES", "")
	t.Setenv("RELEASE_BASE_URL", "https://example.invalid/download")
	t.Setenv("RELEASE_NOTES_EN", notes)
	t.Setenv("RELEASE_NOTES_ZH", "更新说明")
	t.Setenv("RELEASE_OUT", tmp)
	for _, tc := range []struct {
		name       string
		args, want []string
	}{
		{"preflight", []string{"release-preflight"}, []string{"run", "./tools/delivery", "preflight", "-tag", "v0.2.0", "-pubkey", publicKey, "-privkey", privatePath}},
		{"pack", []string{"-o", "app", "-o", "release-preflight", "release-pack"}, []string{"run", "./tools/releasepack", "-app", "build/SuperGit.app", "-tag", "v0.2.0", "-out", tmp, "-priv", privatePath, "-base-url", "https://example.invalid/download", "-notes-zh", "更新说明", "-notes-en", notes}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"GO=" + stub, "RELEASE_TAG=v0.2.0", "UPDATE_PUBKEY=" + publicKey, "UPDATE_PRIVKEY=" + privatePath}, tc.args...)
			cmd := exec.Command("make", args...)
			cmd.Dir = "../.."
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("make failed: %v", err)
			}
			if len(output) != 0 {
				t.Error("release recipe emitted output with a silent tool stub")
			}
			data, err := os.ReadFile(captured)
			if err != nil {
				t.Fatal(err)
			}
			if want := []byte(strings.Join(tc.want, "\x00") + "\x00"); !bytes.Equal(data, want) {
				t.Error("arguments differed from the literal input values")
			}
		})
	}
}
