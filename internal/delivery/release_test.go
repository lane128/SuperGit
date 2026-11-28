package delivery

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"supergit/internal/updater"
)

const fixtureArchiveName = "SuperGit-1.2.3-darwin-arm64.zip"

var releaseCheckOrder = [...]string{"manifest", "size", "sha256", "ed25519", "zip_safety", "extract", "codesign"}

func TestVerifyReleaseAcceptsMatchingSignedArchive(t *testing.T) {
	opts, asset := releaseFixture(t, "cc.supermix.supergit", false)
	report, err := VerifyRelease(t.Context(), opts)
	if err != nil {
		t.Fatalf("VerifyRelease: %v (%+v)", err, report.Checks)
	}
	assertReleaseChecks(t, report, len(releaseCheckOrder), false)
	if report.Version != "1.2.3" || report.Build != 42 || report.ArchivePath != filepath.Join(opts.OutDir, fixtureArchiveName) || report.SHA256 != asset.SHA256 || report.Size != asset.Size {
		t.Errorf("unexpected release metadata: %+v", report)
	}
	if filepath.Base(report.BundlePath) != "SuperGit.app" {
		t.Errorf("bundle path = %q", report.BundlePath)
	}
	if _, err := os.Stat(filepath.Join(report.BundlePath, "Contents", "MacOS", "supergit")); err != nil {
		t.Errorf("missing extracted bundle executable: %v", err)
	}
}

func TestVerifyReleaseRejectsTamperedArchive(t *testing.T) {
	opts, _ := releaseFixture(t, "cc.supermix.supergit", false)
	archive := filepath.Join(opts.OutDir, fixtureArchiveName)
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	data[0] ^= 1
	if err := os.WriteFile(archive, data, 0o600); err != nil {
		t.Fatal(err)
	}
	assertReleaseFailure(t, opts, 3, updater.ErrChecksumMismatch)
}

func TestVerifyReleaseRejectsBadSignature(t *testing.T) {
	opts, asset := releaseFixture(t, "cc.supermix.supergit", false)
	sig, err := base64.StdEncoding.DecodeString(asset.Ed25519)
	if err != nil {
		t.Fatal(err)
	}
	sig[0] ^= 1
	asset.Ed25519 = base64.StdEncoding.EncodeToString(sig)
	writeReleaseManifest(t, opts.OutDir, asset)
	assertReleaseFailure(t, opts, 4, updater.ErrSignatureMismatch)
}

func TestVerifyReleaseRejectsWrongBundleID(t *testing.T) {
	opts, _ := releaseFixture(t, "invalid.example.bundle", false)
	assertReleaseFailure(t, opts, 6, updater.ErrBundleMismatch)
}

func TestVerifyReleaseRejectsUnsafeZip(t *testing.T) {
	opts, _ := releaseFixture(t, "cc.supermix.supergit", true)
	assertReleaseFailure(t, opts, 5, updater.ErrInvalidArchive)
}

func TestVerifyReleaseRejectsSizeMismatch(t *testing.T) {
	opts, asset := releaseFixture(t, "cc.supermix.supergit", false)
	asset.Size++
	writeReleaseManifest(t, opts.OutDir, asset)
	assertReleaseFailure(t, opts, 2, nil)
}

func TestVerifyReleaseRejectsFailedCodeSignature(t *testing.T) {
	opts, _ := releaseFixture(t, "cc.supermix.supergit", false)
	signErr := errors.New("fixture code signature rejected")
	opts.VerifySignature = func(context.Context, string) error { return signErr }
	assertReleaseFailure(t, opts, 7, signErr)
}

func TestVerifyReleaseRejectsOversizedManifest(t *testing.T) {
	opts, _ := releaseFixture(t, "cc.supermix.supergit", false)
	manifestPath := filepath.Join(opts.OutDir, "update.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, strings.Repeat(" ", (1<<20)+1-len(data))...)
	if err := os.WriteFile(manifestPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	assertReleaseFailure(t, opts, 1, nil)
}

func assertReleaseFailure(t *testing.T, opts VerifyOptions, count int, cause error) {
	t.Helper()
	report, err := VerifyRelease(t.Context(), opts)
	if err == nil || cause != nil && !errors.Is(err, cause) {
		t.Fatalf("VerifyRelease error = %v, want failure (cause %v); checks = %+v", err, cause, report.Checks)
	}
	assertReleaseChecks(t, report, count, true)
}

func assertReleaseChecks(t *testing.T, report ReleaseVerification, count int, failed bool) {
	t.Helper()
	if len(report.Checks) != count {
		t.Fatalf("checks = %+v, want %d checks", report.Checks, count)
	}
	for i, check := range report.Checks {
		passed := !failed || i < count-1
		if check.Name != releaseCheckOrder[i] || check.Passed != passed || !passed && check.Detail == "" {
			t.Errorf("check %d = %+v, want %s with Passed=%t and detail on failure", i, check, releaseCheckOrder[i], passed)
		}
	}
}

func releaseFixture(t *testing.T, bundleID string, unsafeZip bool) (VerifyOptions, updater.PlatformAsset) {
	t.Helper()
	root := t.TempDir()
	// Keep any verifier workspace under test cleanup on each supported OS.
	for _, name := range []string{"TMPDIR", "TEMP", "TMP"} {
		t.Setenv(name, root)
	}
	executable := "SuperGit.app/Contents/MacOS/supergit"
	plist := "SuperGit.app/Contents/Info.plist"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, executable)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, executable), []byte("fixture executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	plistData := `<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>` + bundleID + `</string>
<key>CFBundleExecutable</key><string>supergit</string>
<key>CFBundleShortVersionString</key><string>1.2.3</string>
<key>CFBundleVersion</key><string>42</string>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(root, plist), []byte(plistData), 0o644); err != nil {
		t.Fatal(err)
	}

	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for _, name := range []string{executable, plist} {
		filename := filepath.Join(root, name)
		info, err := os.Stat(filename)
		if err != nil {
			t.Fatal(err)
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			t.Fatal(err)
		}
		header.Name = name
		entry, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if unsafeZip {
		if _, err := zw.Create("outside-app.txt"); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	data := archive.Bytes()
	if err := os.WriteFile(filepath.Join(root, fixtureArchiveName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	asset := updater.PlatformAsset{
		URL:     "https://example.invalid/releases/" + fixtureArchiveName + "?download=1",
		SHA256:  hex.EncodeToString(sum[:]),
		Ed25519: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sum[:])),
		Size:    int64(len(data)),
	}
	writeReleaseManifest(t, root, asset)
	return VerifyOptions{
		OutDir:          root,
		PublicKey:       base64.StdEncoding.EncodeToString(pub),
		VerifySignature: func(context.Context, string) error { return nil },
	}, asset
}

func writeReleaseManifest(t *testing.T, dir string, asset updater.PlatformAsset) {
	t.Helper()
	data, err := json.Marshal(updater.Manifest{
		Version:   "1.2.3",
		Build:     42,
		Platforms: map[string]updater.PlatformAsset{"darwin-arm64": asset},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "update.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}
