package lobedesk

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
	"testing"
	"time"

	"supergit/internal/gitrepo"
	"supergit/internal/updater"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func testUpdateServer(t *testing.T, version string) (srv *httptest.Server, pubB64 string, exePath string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubB64 = base64.StdEncoding.EncodeToString(pub)

	zipPath := filepath.Join(t.TempDir(), "SuperGit.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>supergit</string>
<key>CFBundleIdentifier</key><string>` + updater.ExpectedBundleID + `</string>
<key>CFBundleShortVersionString</key><string>` + version + `</string>
<key>CFBundleVersion</key><string>50</string>
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
	} else if _, err := w.Write([]byte("#!/bin/sh\necho " + version + "\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(zipBytes)
	sig := updater.SignDigest(priv, sum)

	goos, goarch := updater.CurrentPlatform()
	platKey := goos + "-" + goarch
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/update.json":
			m := updater.Manifest{
				Version:  version,
				Build:    50,
				PubDate:  "2026-12-01T10:00:00Z",
				NotesURL: updater.DefaultReleasesURL,
				Notes:    map[string]string{"zh": "修复若干问题并支持自动更新。", "en": "Bug fixes and automatic updates."},
				Platforms: map[string]updater.PlatformAsset{
					platKey: {
						URL:     srv.URL + "/SuperGit.zip",
						SHA256:  hex.EncodeToString(sum[:]),
						Ed25519: sig,
						Size:    int64(len(zipBytes)),
					},
				},
			}
			_ = json.NewEncoder(w).Encode(m)
		case "/SuperGit.zip":
			_, _ = w.Write(zipBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	installDir := t.TempDir()
	macosDir := filepath.Join(installDir, "SuperGit.app", "Contents", "MacOS")
	if err := os.MkdirAll(macosDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "SuperGit.app", "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	exePath = filepath.Join(macosDir, "supergit")
	if err := os.WriteFile(exePath, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	return srv, pubB64, exePath
}

func TestUpdateAutoDownloadOffShowsBadgeAndDialogCheckbox(t *testing.T) {
	isolateSettings(t)
	srv, pubB64, exePath := testUpdateServer(t, "0.2.0")

	callbacks := make(chan func(), 64)
	a := &app{
		ctx:                 context.Background(),
		locale:              localeChinese,
		themeMode:           "light",
		page:                "history",
		autoCheckUpdates:    true,
		autoDownloadUpdates: false,
		runOnMain:           func(fn func()) { callbacks <- fn },
	}
	tabs := newRepositoryTabs(context.Background(), a)
	ctrl := &updateController{}
	ctrl.attachManager(a, updater.Config{
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
	tabs.updater = ctrl

	ctrl.mgr.CheckIfDue(context.Background())

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case fn := <-callbacks:
			fn()
		default:
			time.Sleep(10 * time.Millisecond)
		}
		if a.updateSnapshot().State == updater.StateAvailable {
			break
		}
	}
	if got := a.updateSnapshot().State; got != updater.StateAvailable {
		t.Fatalf("state=%q want %q", got, updater.StateAvailable)
	}
	if ctrl.dialogOpen {
		t.Fatal("auto-download OFF should not pop up modal automatically")
	}

	view := ui.NewTester(tabs.view, 1100, 720)
	view.Frame()

	badgeLabel := a.tf("updateBadgeAvailable", "v0.2.0")
	badge, ok := view.AccessByRole(ui.RoleButton, badgeLabel)
	if !ok {
		t.Fatalf("expected title bar update badge %q to be visible", badgeLabel)
	}

	// Click badge to open the update dialog.
	view.ClickAt(badge.Bounds.X+badge.Bounds.W/2, badge.Bounds.Y+badge.Bounds.H/2)
	view.Frame()
	if !ctrl.dialogOpen {
		t.Fatal("clicking title bar badge did not open update dialog")
	}
	if !view.HasText("修复若干问题并支持自动更新。") {
		t.Fatal("update dialog did not render localized release notes")
	}

	// Toggle the auto-download checkbox directly inside the update dialog.
	cb, ok := view.AccessByRole(ui.RoleCheckBox, a.t("autoDownloadUpdates"))
	if !ok {
		t.Fatal("update dialog missing auto-download checkbox")
	}
	view.ClickAt(cb.Bounds.X+cb.Bounds.W/2, cb.Bounds.Y+cb.Bounds.H/2)
	view.Frame()
	if !a.autoDownloadUpdates {
		t.Fatal("toggling auto-download checkbox in update dialog did not enable autoDownloadUpdates")
	}
	saved := loadSettings()
	if saved.AutoDownloadUpdates == nil || !*saved.AutoDownloadUpdates {
		t.Fatalf("autoDownloadUpdates not persisted to settings: %+v", saved)
	}
}

func TestUpdateAutoDownloadOnPromptsReadyAndGuardsRestart(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS in-place update flow")
	}
	isolateSettings(t)
	srv, pubB64, exePath := testUpdateServer(t, "0.2.0")

	callbacks := make(chan func(), 128)
	var stagedCall, targetCall string
	quitCalled := false

	a := &app{
		ctx:                 context.Background(),
		locale:              localeEnglish,
		themeMode:           "light",
		page:                "history",
		autoCheckUpdates:    true,
		autoDownloadUpdates: true,
		runOnMain:           func(fn func()) { callbacks <- fn },
	}
	tabs := newRepositoryTabs(context.Background(), a)
	ctrl := &updateController{
		quitFn: func() { quitCalled = true },
	}
	ctrl.attachManager(a, updater.Config{
		CurrentVersion:  "0.1.0",
		CurrentBuild:    "30",
		ManifestURL:     srv.URL + "/update.json",
		PublicKeyBase64: pubB64,
		CacheDir:        t.TempDir(),
		ExecutablePath:  exePath,
		HTTPClient:      srv.Client(),
		VerifyCodeSign:  func(context.Context, string) error { return nil },
		StageAndLaunch: func(stagedApp, targetApp string, _ int) error {
			stagedCall, targetCall = stagedApp, targetApp
			return nil
		},
		AutoCheck:    true,
		AutoDownload: true,
	})
	tabs.updater = ctrl

	ctrl.mgr.CheckIfDue(context.Background())

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case fn := <-callbacks:
			fn()
		default:
			time.Sleep(10 * time.Millisecond)
		}
		if a.updateSnapshot().State == updater.StateReadyToRestart && ctrl.dialogOpen {
			break
		}
	}
	if got := a.updateSnapshot().State; got != updater.StateReadyToRestart {
		t.Fatalf("state=%q want %q", got, updater.StateReadyToRestart)
	}
	if !ctrl.dialogOpen {
		t.Fatal("auto-download ON should automatically prompt when ready to restart")
	}

	view := ui.NewTester(tabs.view, 1100, 720)
	view.Frame()
	if !view.HasText(a.t("updateTitleReady")) {
		t.Fatal("ready prompt title not rendered")
	}

	// Verify user can turn off auto-download right inside the ready prompt.
	cb, ok := view.AccessByRole(ui.RoleCheckBox, a.t("autoDownloadUpdates"))
	if !ok {
		t.Fatal("ready prompt missing auto-download checkbox")
	}
	view.ClickAt(cb.Bounds.X+cb.Bounds.W/2, cb.Bounds.Y+cb.Bounds.H/2)
	view.Frame()
	if a.autoDownloadUpdates {
		t.Fatal("unchecking auto-download inside ready prompt did not disable it")
	}

	// Busy Git operation must block restart.
	a.busy = true
	a.applyPreparedUpdate(toastMain)
	if quitCalled {
		t.Fatal("restart should be blocked while a Git operation is running")
	}
	if len(a.notices[toastMain]) == 0 || a.notices[toastMain][len(a.notices[toastMain])-1].title != a.t("updateBusyBeforeRestart") {
		t.Fatalf("expected busy blocker warning notice, got %+v", a.notices[toastMain])
	}
	a.busy = false

	// Unsaved conflict editor draft must also block restart.
	a.conflictEditors = map[conflictEditorKey]*conflictEditorState{
		{Root: "/tmp/r", Path: "a.txt"}: {
			Loaded:   true,
			Original: gitrepo.Conflict{Working: "original"},
			Text:     "edited",
		},
	}
	a.applyPreparedUpdate(toastMain)
	if quitCalled {
		t.Fatal("restart should be blocked while conflict editor has unsaved edits")
	}
	a.conflictEditors = nil

	// Once idle, clicking Restart & Update stages and quits cleanly.
	restartBtn, ok := view.AccessByRole(ui.RoleButton, a.t("updateRestartAndInstall"))
	if !ok {
		t.Fatal("ready prompt missing Restart & Update button")
	}
	view.ClickAt(restartBtn.Bounds.X+restartBtn.Bounds.W/2, restartBtn.Bounds.Y+restartBtn.Bounds.H/2)
	view.Frame()
	if !quitCalled || stagedCall == "" || targetCall == "" {
		t.Fatalf("expected apply to stage and quit: quit=%v staged=%q target=%q", quitCalled, stagedCall, targetCall)
	}
}

func TestSettingsAboutUpdatesCardAndAppMenu(t *testing.T) {
	isolateSettings(t)
	a := &app{
		locale:              localeEnglish,
		themeMode:           "light",
		settingsSection:     "about",
		autoCheckUpdates:    true,
		autoDownloadUpdates: false,
		window:              &mygo.Window{},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() { a.settingsPage(c) })
	}, settingsWindowWidth, settingsWindowHeight)
	tt.Frame()

	if !tt.HasText(a.t("softwareUpdates")) {
		t.Fatal("Settings About page missing Software Updates card")
	}
	checkCB, ok := tt.AccessByRole(ui.RoleCheckBox, a.t("autoCheckUpdates"))
	if !ok {
		t.Fatal("Settings missing autoCheckUpdates checkbox")
	}
	dlCB, ok := tt.AccessByRole(ui.RoleCheckBox, a.t("autoDownloadUpdates"))
	if !ok || dlCB.Disabled {
		t.Fatalf("expected enabled autoDownloadUpdates checkbox, got ok=%v disabled=%v", ok, dlCB.Disabled)
	}
	// Turning off auto-check disables auto-download checkbox.
	tt.ClickAt(checkCB.Bounds.X+checkCB.Bounds.W/2, checkCB.Bounds.Y+checkCB.Bounds.H/2)
	tt.Frame()
	dlCB, ok = tt.AccessByRole(ui.RoleCheckBox, a.t("autoDownloadUpdates"))
	if !ok || !dlCB.Disabled {
		t.Fatalf("expected autoDownloadUpdates disabled when autoCheckUpdates is off, got ok=%v disabled=%v", ok, dlCB.Disabled)
	}

	if runtime.GOOS == "darwin" {
		oldMenu := mygo.App.Menu()
		t.Cleanup(func() { mygo.App.SetMenu(oldMenu) })
		tabs := newRepositoryTabs(context.Background(), a)
		tabs.installRepositoryMenu()
		found := false
		for _, item := range mygo.App.Menu().Items() {
			if item.Role != mygo.RoleAppMenu {
				continue
			}
			for _, child := range item.Submenu {
				if child.Label == a.t("checkForUpdatesAction") && child.Click != nil {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("macOS AppMenu missing Check for Updates… item")
		}
	}
}
