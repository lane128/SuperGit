package updater

import (
	"archive/zip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestVersionComparison(t *testing.T) {
	cases := []struct {
		cand    string
		curr    string
		want    bool
		wantErr bool
	}{
		{"0.2.0", "0.1.0", true, false},
		{"v0.2.0", "0.1.0", true, false},
		{"0.1.0", "0.1.0", false, false},
		{"0.1.0", "0.2.0", false, false},
		{"1.0.0", "1.0.0-rc.1", true, false},
		{"1.0.0-rc.2", "1.0.0-rc.1", true, false},
		{"1.0.0-rc.10", "1.0.0-rc.2", true, false},
		{"1.0", "0.9.9", true, false},
		{"invalid", "0.1.0", false, true},
		{"0.2.0", "", false, true},
	}
	for _, tc := range cases {
		got, err := IsNewer(tc.cand, tc.curr)
		if (err != nil) != tc.wantErr {
			t.Fatalf("IsNewer(%q, %q) err=%v wantErr=%v", tc.cand, tc.curr, err, tc.wantErr)
		}
		if !tc.wantErr && got != tc.want {
			t.Fatalf("IsNewer(%q, %q) = %v, want %v", tc.cand, tc.curr, got, tc.want)
		}
	}
}

func TestValidateUpdateURL(t *testing.T) {
	valid := []string{
		"https://github.com/lane128/SuperGit/releases/latest/download/update.json",
		"https://objects.githubusercontent.com/github-production-release-asset/foo.zip",
		"https://release-assets.githubusercontent.com/bar.zip",
	}
	for _, raw := range valid {
		if err := ValidateUpdateURL(raw, false); err != nil {
			t.Errorf("ValidateUpdateURL(%q, false) unexpected err: %v", raw, err)
		}
	}

	invalid := []string{
		"",
		"http://github.com/lane128/SuperGit/releases/latest/download/update.json",
		"https://evil.example.com/update.json",
		"https://user:pass@github.com/lane128/SuperGit/update.json",
		"http://127.0.0.1:8080/update.json",
		"file:///etc/passwd",
	}
	for _, raw := range invalid {
		if err := ValidateUpdateURL(raw, false); err == nil {
			t.Errorf("ValidateUpdateURL(%q, false) expected error, got nil", raw)
		}
	}

	if err := ValidateUpdateURL("http://127.0.0.1:8080/update.json", true); err != nil {
		t.Errorf("expected loopback URL allowed when allowLoopback=true: %v", err)
	}
	if err := ValidateUpdateURL("https://evil.example.com/update.json", true); err == nil {
		t.Error("expected non-loopback non-GitHub URL rejected even when allowLoopback=true")
	}
}

func TestInspectZipSafetyRejectsTraversal(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "bad.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("SuperGit.app/../../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("nope"))
	_ = zw.Close()
	_ = f.Close()

	if _, err := InspectZipSafety(zipPath); err == nil {
		t.Fatal("expected Zip Slip traversal archive to be rejected")
	}
}

func makeFakeAppZip(t *testing.T, bundleID, shortVer, build string) (zipPath string, sumHex string, size int64, digest [sha256.Size]byte) {
	t.Helper()
	zipPath = filepath.Join(t.TempDir(), "SuperGit.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)

	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>supergit</string>
<key>CFBundleIdentifier</key><string>` + bundleID + `</string>
<key>CFBundleShortVersionString</key><string>` + shortVer + `</string>
<key>CFBundleVersion</key><string>` + build + `</string>
</dict></plist>`
	if w, err := zw.Create("SuperGit.app/Contents/Info.plist"); err != nil {
		t.Fatal(err)
	} else if _, err := w.Write([]byte(plist)); err != nil {
		t.Fatal(err)
	}

	hdr := &zip.FileHeader{Name: "SuperGit.app/Contents/MacOS/supergit", Method: zip.Deflate}
	hdr.SetMode(0o755)
	if w, err := zw.CreateHeader(hdr); err != nil {
		t.Fatal(err)
	} else if _, err := w.Write([]byte("#!/bin/sh\necho " + shortVer + "\n")); err != nil {
		t.Fatal(err)
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	digest = sha256.Sum256(data)
	return zipPath, hex.EncodeToString(digest[:]), int64(len(data)), digest
}

func makeFakeInstalledApp(t *testing.T, dir, bundleID, marker string) (bundlePath, exePath string) {
	t.Helper()
	bundlePath = filepath.Join(dir, "SuperGit.app")
	macosDir := filepath.Join(bundlePath, "Contents", "MacOS")
	if err := os.MkdirAll(macosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<plist><dict><key>CFBundleExecutable</key><string>supergit</string><key>CFBundleIdentifier</key><string>` + bundleID + `</string></dict></plist>`
	if err := os.WriteFile(filepath.Join(bundlePath, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	exePath = filepath.Join(macosDir, "supergit")
	if err := os.WriteFile(exePath, []byte(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	return bundlePath, exePath
}

func TestDownloadAndVerifySignatureChecks(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	zipPath, sumHex, size, digest := makeFakeAppZip(t, ExpectedBundleID, "0.2.0", "40")
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	sig := SignDigest(priv, digest)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	defer srv.Close()

	asset := PlatformAsset{
		URL:     srv.URL + "/SuperGit.zip",
		SHA256:  sumHex,
		Ed25519: sig,
		Size:    size,
	}

	dest := filepath.Join(t.TempDir(), "dl.zip")
	if err := DownloadAndVerify(context.Background(), srv.Client(), asset, dest, pub, true, nil); err != nil {
		t.Fatalf("valid DownloadAndVerify failed: %v", err)
	}

	// Bad signature must fail.
	badSigAsset := asset
	badSigAsset.Ed25519 = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	if err := DownloadAndVerify(context.Background(), srv.Client(), badSigAsset, filepath.Join(t.TempDir(), "bad-sig.zip"), pub, true, nil); err == nil {
		t.Fatal("expected signature verification failure")
	}

	// Bad checksum must fail.
	badHashAsset := asset
	badHashAsset.SHA256 = strings.Repeat("0", 64)
	if err := DownloadAndVerify(context.Background(), srv.Client(), badHashAsset, filepath.Join(t.TempDir(), "bad-hash.zip"), pub, true, nil); err == nil {
		t.Fatal("expected checksum verification failure")
	}

	// Missing public key must fail.
	if err := DownloadAndVerify(context.Background(), srv.Client(), asset, filepath.Join(t.TempDir(), "no-key.zip"), nil, true, nil); err == nil {
		t.Fatal("expected missing public key failure")
	}
}

func TestManagerAutoDownloadOffVsOn(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	zipPath, sumHex, size, digest := makeFakeAppZip(t, ExpectedBundleID, "0.2.0", "42")
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	sig := SignDigest(priv, digest)
	goos, goarch := CurrentPlatform()
	platKey := goos + "-" + goarch

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/update.json":
			m := Manifest{
				Version:  "0.2.0",
				Build:    42,
				PubDate:  "2026-12-01T10:00:00Z",
				NotesURL: DefaultReleasesURL,
				Notes:    map[string]string{"zh": "中文更新说明", "en": "English release notes"},
				Platforms: map[string]PlatformAsset{
					platKey: {
						URL:     srv.URL + "/SuperGit-0.2.0.zip",
						SHA256:  sumHex,
						Ed25519: sig,
						Size:    size,
					},
				},
			}
			_ = json.NewEncoder(w).Encode(m)
		case "/SuperGit-0.2.0.zip":
			_, _ = w.Write(zipBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	_, exePath := makeFakeInstalledApp(t, t.TempDir(), ExpectedBundleID, "old-v0.1.0")
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	waitState := func(m *Manager, want State) Snapshot {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			snap := m.Snapshot()
			if snap.State == want {
				return snap
			}
			if snap.State == StateError {
				t.Fatalf("unexpected StateError: %s", snap.Error)
			}
			time.Sleep(15 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for state %q; current=%+v", want, m.Snapshot())
		return Snapshot{}
	}

	// Case 1: AutoDownload OFF -> stops at StateAvailable.
	mOff := NewManager(Config{
		CurrentVersion:  "0.1.0",
		CurrentBuild:    "30",
		ManifestURL:     srv.URL + "/update.json",
		PublicKeyBase64: pubB64,
		CacheDir:        t.TempDir(),
		ExecutablePath:  exePath,
		HTTPClient:      srv.Client(),
		VerifyCodeSign:  func(context.Context, string) error { return nil },
		AutoCheck:       true,
		AutoDownload:    false,
	})
	if !mOff.CheckIfDue(context.Background()) {
		t.Fatal("expected CheckIfDue to start check")
	}
	snap := waitState(mOff, StateAvailable)
	if snap.Release.Version != "0.2.0" || snap.Release.NotesFor("zh") != "中文更新说明" {
		t.Fatalf("unexpected release info: %+v", snap.Release)
	}
	if snap.AutoDownloadOrigin {
		t.Fatal("expected AutoDownloadOrigin=false when auto-download is off")
	}
	// Skipping version clears it and suppresses subsequent automatic check.
	if skipped := mOff.SkipAvailable(); skipped != "0.2.0" {
		t.Fatalf("skipped=%q want 0.2.0", skipped)
	}
	if !mOff.CheckNow(context.Background(), false) {
		t.Fatal("expected automatic re-check to start")
	}
	waitState(mOff, StateIdle)

	// Case 2: AutoDownload ON (on macOS where .app is eligible) -> auto downloads to StateReadyToRestart.
	if runtime.GOOS == "darwin" {
		var mu sync.Mutex
		var states []State
		mOn := NewManager(Config{
			CurrentVersion:  "0.1.0",
			CurrentBuild:    "30",
			ManifestURL:     srv.URL + "/update.json",
			PublicKeyBase64: pubB64,
			CacheDir:        t.TempDir(),
			ExecutablePath:  exePath,
			HTTPClient:      srv.Client(),
			VerifyCodeSign:  func(context.Context, string) error { return nil },
			AutoCheck:       true,
			AutoDownload:    true,
			OnChange: func(s Snapshot) {
				mu.Lock()
				states = append(states, s.State)
				mu.Unlock()
			},
		})
		if !mOn.CheckIfDue(context.Background()) {
			t.Fatal("expected CheckIfDue to start check")
		}
		ready := waitState(mOn, StateReadyToRestart)
		if !ready.AutoDownloadOrigin {
			t.Fatal("expected AutoDownloadOrigin=true after automatic download")
		}
		if ready.StagedAppPath == "" {
			t.Fatal("expected non-empty StagedAppPath")
		}
		if _, err := os.Stat(filepath.Join(ready.StagedAppPath, "Contents", "Info.plist")); err != nil {
			t.Fatalf("staged app missing Info.plist: %v", err)
		}
	}
}

func TestDarwinEligibilityAndAtomicSwap(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS bundle tests only")
	}
	dir := t.TempDir()
	targetBundle, targetExe := makeFakeInstalledApp(t, dir, ExpectedBundleID, "old-binary")
	wantBundle, err := filepath.EvalSymlinks(targetBundle)
	if err != nil {
		wantBundle = targetBundle
	}
	elig := CheckEligibility(targetExe)
	if !elig.Eligible || elig.BundlePath != wantBundle {
		t.Fatalf("expected eligible bundle %q, got %+v", wantBundle, elig)
	}

	if got := CheckEligibility("/Volumes/SuperGit/AppTranslocation/ABC/d/SuperGit.app/Contents/MacOS/supergit"); got.Eligible || got.Reason != ReasonTranslocated {
		t.Fatalf("expected translocated rejection, got %+v", got)
	}
	if got := CheckEligibility("/usr/local/bin/supergit"); got.Eligible || got.Reason != ReasonNotAppBundle {
		t.Fatalf("expected non-bundle rejection, got %+v", got)
	}

	// Stage a new bundle right beside targetBundle and test atomic swap.
	stagedDir := filepath.Join(dir, ".SuperGit.app.update-staged-test")
	if err := os.MkdirAll(filepath.Join(stagedDir, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<plist><dict><key>CFBundleExecutable</key><string>supergit</string><key>CFBundleIdentifier</key><string>` + ExpectedBundleID + `</string></dict></plist>`
	if err := os.WriteFile(filepath.Join(stagedDir, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagedDir, "Contents", "MacOS", "supergit"), []byte("new-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	launched := ""
	if err := applyBundleUpdate(0, stagedDir, targetBundle, true, time.Second, func(path string) error {
		launched = path
		return nil
	}); err != nil {
		t.Fatalf("applyBundleUpdate failed: %v", err)
	}
	if launched != targetBundle {
		t.Fatalf("launched=%q want %q", launched, targetBundle)
	}
	gotBytes, err := os.ReadFile(filepath.Join(targetBundle, "Contents", "MacOS", "supergit"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBytes) != "new-binary" {
		t.Fatalf("expected swapped binary content, got %q", string(gotBytes))
	}
}
