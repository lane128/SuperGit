package updater

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// State represents the current phase of the update manager state machine.
type State string

const (
	StateIdle           State = "idle"
	StateChecking       State = "checking"
	StateUpToDate       State = "upToDate"
	StateAvailable      State = "available"
	StateDownloading    State = "downloading"
	StateVerifying      State = "verifying"
	StateReadyToRestart State = "readyToRestart"
	StateApplying       State = "applying"
	StateError          State = "error"
)

// DefaultCheckInterval is the minimum time between automatic background checks.
const DefaultCheckInterval = 24 * time.Hour

// Config initializes a Manager.
type Config struct {
	CurrentVersion   string
	CurrentBuild     string
	ManifestURL      string
	PublicKeyBase64  string
	CacheDir         string
	ExecutablePath   string
	ExpectedBundleID string
	PlatformOS       string
	PlatformArch     string
	HTTPClient       *http.Client
	VerifyCodeSign   func(context.Context, string) error
	StageAndLaunch   func(stagedApp, targetApp string, waitPID int) error
	Now              func() time.Time
	CheckInterval    time.Duration
	AutoCheck        bool
	AutoDownload     bool
	SkippedVersion   string
	LastCheckedAt    time.Time
	OnChange         func(Snapshot)
}

// Snapshot is an immutable view of the update manager's state for UI rendering.
type Snapshot struct {
	State           State
	CurrentVersion  string
	CurrentBuild    string
	ManualCheck     bool
	Release         ReleaseInfo
	DownloadedBytes int64
	TotalBytes      int64
	StagedAppPath   string
	Eligibility     Eligibility
	HasPublicKey    bool
	Error           string
	LastCheckedAt   time.Time
	SkippedVersion  string
	AutoCheck       bool
	AutoDownload    bool
	// AutoDownloadOrigin marks whether the current ReadyToRestart state came
	// from an automatic background download rather than a manual click.
	AutoDownloadOrigin bool
}

// CanAutoInstall reports whether in-place download + verify + apply is possible
// for the currently known release.
func (s Snapshot) CanAutoInstall() bool {
	return s.Eligibility.Eligible && s.HasPublicKey && s.Release.HasAsset
}

// Manager coordinates update checks, downloads, verification, and staging.
type Manager struct {
	mu               sync.Mutex
	manifestURL      string
	allowLoopback    bool
	pubKey           ed25519.PublicKey
	cacheDir         string
	expectedBundleID string
	platformOS       string
	platformArch     string
	httpClient       *http.Client
	verifyCodeSign   func(context.Context, string) error
	stageAndLaunch   func(stagedApp, targetApp string, waitPID int) error
	now              func() time.Time
	checkInterval    time.Duration
	onChange         func(Snapshot)

	snap           Snapshot
	opCancel       context.CancelFunc
	opSeq          uint64
	autoDownloaded bool
}

// NewManager creates a configured update manager without starting any network I/O.
func NewManager(cfg Config) *Manager {
	manifestURL, allowLoopback := EffectiveManifestURL(cfg.ManifestURL)
	pubEncoded := EffectivePublicKeyBase64(cfg.PublicKeyBase64)
	pubKey, _ := ParsePublicKey(pubEncoded)

	exePath := cfg.ExecutablePath
	if exePath == "" {
		exePath, _ = os.Executable()
	}
	eligibility := CheckEligibility(exePath)

	cacheDir := cfg.CacheDir
	if cacheDir == "" {
		if base, err := os.UserCacheDir(); err == nil {
			cacheDir = filepath.Join(base, "SuperGit", "updates")
		} else {
			cacheDir = filepath.Join(os.TempDir(), "SuperGit-updates")
		}
	}
	bundleID := cfg.ExpectedBundleID
	if bundleID == "" {
		bundleID = ExpectedBundleID
	}
	goos, goarch := cfg.PlatformOS, cfg.PlatformArch
	if goos == "" || goarch == "" {
		goos, goarch = CurrentPlatform()
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	interval := cfg.CheckInterval
	if interval <= 0 {
		interval = DefaultCheckInterval
	}
	stageFn := cfg.StageAndLaunch
	if stageFn == nil {
		stageFn = StageAndLaunchApply
	}
	currentVer := NormalizeVersion(cfg.CurrentVersion)
	if currentVer == "" {
		currentVer = "0.1.0"
	}

	return &Manager{
		manifestURL:      manifestURL,
		allowLoopback:    allowLoopback,
		pubKey:           pubKey,
		cacheDir:         cacheDir,
		expectedBundleID: bundleID,
		platformOS:       goos,
		platformArch:     goarch,
		httpClient:       cfg.HTTPClient,
		verifyCodeSign:   cfg.VerifyCodeSign,
		stageAndLaunch:   stageFn,
		now:              nowFn,
		checkInterval:    interval,
		onChange:         cfg.OnChange,
		snap: Snapshot{
			State:          StateIdle,
			CurrentVersion: currentVer,
			CurrentBuild:   strings.TrimSpace(cfg.CurrentBuild),
			Eligibility:    eligibility,
			HasPublicKey:   len(pubKey) == ed25519.PublicKeySize,
			LastCheckedAt:  cfg.LastCheckedAt,
			SkippedVersion: NormalizeVersion(cfg.SkippedVersion),
			AutoCheck:      cfg.AutoCheck,
			AutoDownload:   cfg.AutoDownload,
		},
	}
}

// Snapshot returns a thread-safe immutable copy of the current state.
func (m *Manager) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snap
}

// SetPreferences updates user preferences and emits an updated snapshot.
// If autoDownload is turned on while an eligible update is already available,
// it can optionally begin downloading immediately.
func (m *Manager) SetPreferences(autoCheck, autoDownload bool, skippedVersion string) Snapshot {
	m.mu.Lock()
	m.snap.AutoCheck = autoCheck
	m.snap.AutoDownload = autoDownload
	if skippedVersion != "" {
		m.snap.SkippedVersion = NormalizeVersion(skippedVersion)
	} else {
		m.snap.SkippedVersion = ""
	}
	snap := m.snap
	cb := m.onChange
	m.mu.Unlock()
	if cb != nil {
		cb(snap)
	}
	return snap
}

// CheckIfDue runs a background check when autoCheck is enabled and at least
// checkInterval has passed since the last check.
func (m *Manager) CheckIfDue(ctx context.Context) bool {
	m.mu.Lock()
	due := m.snap.AutoCheck &&
		(m.snap.State == StateIdle || m.snap.State == StateUpToDate || m.snap.State == StateError) &&
		(m.snap.LastCheckedAt.IsZero() || m.now().Sub(m.snap.LastCheckedAt) >= m.checkInterval)
	m.mu.Unlock()
	if !due {
		return false
	}
	return m.CheckNow(ctx, false)
}

// CheckNow checks the update manifest. Manual checks ignore the skipped version
// and report errors/up-to-date status visibly.
func (m *Manager) CheckNow(ctx context.Context, manual bool) bool {
	m.mu.Lock()
	switch m.snap.State {
	case StateChecking, StateDownloading, StateVerifying, StateApplying:
		m.mu.Unlock()
		return false
	case StateReadyToRestart:
		if !manual {
			m.mu.Unlock()
			return false
		}
		// Keep an already-staged release if the user clicks Check for Updates manually.
		m.snap.ManualCheck = true
		snap := m.snap
		cb := m.onChange
		m.mu.Unlock()
		if cb != nil {
			cb(snap)
		}
		return true
	}

	checkCtx, cancel := context.WithCancel(ctx)
	m.opCancel = cancel
	m.opSeq++
	seq := m.opSeq
	m.snap.State = StateChecking
	m.snap.ManualCheck = manual
	m.snap.Error = ""
	snap := m.snap
	cb := m.onChange
	manifestURL := m.manifestURL
	allowLoopback := m.allowLoopback
	httpClient := m.httpClient
	goos, goarch := m.platformOS, m.platformArch
	m.mu.Unlock()

	if cb != nil {
		cb(snap)
	}

	go func() {
		manifest, err := FetchManifest(checkCtx, httpClient, manifestURL, allowLoopback)
		m.finishCheck(ctx, seq, manual, manifest, err, goos, goarch)
	}()
	return true
}

func (m *Manager) finishCheck(parentCtx context.Context, seq uint64, manual bool, manifest Manifest, err error, goos, goarch string) {
	m.mu.Lock()
	if seq != m.opSeq {
		m.mu.Unlock()
		return
	}
	m.opCancel = nil
	checkedAt := m.now()

	if err != nil {
		if errors.Is(err, context.Canceled) {
			m.snap.State = StateIdle
			m.snap.Error = ""
		} else if manual {
			m.snap.State = StateError
			m.snap.Error = err.Error()
		} else {
			// Background check failures stay quiet and retry later.
			m.snap.State = StateIdle
			m.snap.Error = ""
		}
		snap := m.snap
		cb := m.onChange
		m.mu.Unlock()
		if cb != nil {
			cb(snap)
		}
		return
	}

	m.snap.LastCheckedAt = checkedAt
	newer, cmpErr := IsNewer(manifest.Version, m.snap.CurrentVersion)
	if cmpErr != nil {
		if manual {
			m.snap.State = StateError
			m.snap.Error = cmpErr.Error()
		} else {
			m.snap.State = StateIdle
		}
		snap := m.snap
		cb := m.onChange
		m.mu.Unlock()
		if cb != nil {
			cb(snap)
		}
		return
	}

	if !newer {
		m.snap.State = StateUpToDate
		m.snap.Release = ReleaseInfo{}
		m.snap.StagedAppPath = ""
		m.snap.Error = ""
		snap := m.snap
		cb := m.onChange
		m.mu.Unlock()
		if cb != nil {
			cb(snap)
		}
		return
	}

	release := manifest.ToReleaseInfo(goos, goarch)
	if !manual && m.snap.SkippedVersion != "" && NormalizeVersion(release.Version) == NormalizeVersion(m.snap.SkippedVersion) {
		m.snap.State = StateIdle
		m.snap.Release = ReleaseInfo{}
		snap := m.snap
		cb := m.onChange
		m.mu.Unlock()
		if cb != nil {
			cb(snap)
		}
		return
	}

	m.snap.Release = release
	m.snap.State = StateAvailable
	m.snap.DownloadedBytes = 0
	m.snap.TotalBytes = release.Asset.Size
	m.snap.Error = ""
	m.snap.AutoDownloadOrigin = false

	shouldAutoDownload := !manual && m.snap.AutoDownload && m.snap.CanAutoInstall()
	snap := m.snap
	cb := m.onChange
	m.mu.Unlock()

	if cb != nil {
		cb(snap)
	}
	if shouldAutoDownload {
		m.startDownloadInternal(parentCtx, true)
	}
}

// StartDownload begins downloading and verifying the available release.
func (m *Manager) StartDownload(ctx context.Context) bool {
	return m.startDownloadInternal(ctx, false)
}

func (m *Manager) startDownloadInternal(ctx context.Context, autoOrigin bool) bool {
	m.mu.Lock()
	if m.snap.State != StateAvailable && m.snap.State != StateError {
		m.mu.Unlock()
		return false
	}
	if !m.snap.CanAutoInstall() {
		m.mu.Unlock()
		return false
	}
	dlCtx, cancel := context.WithCancel(ctx)
	m.opCancel = cancel
	m.opSeq++
	seq := m.opSeq
	release := m.snap.Release
	m.snap.State = StateDownloading
	m.snap.DownloadedBytes = 0
	m.snap.TotalBytes = release.Asset.Size
	m.snap.Error = ""
	m.snap.AutoDownloadOrigin = autoOrigin
	snap := m.snap
	cb := m.onChange
	httpClient := m.httpClient
	pubKey := m.pubKey
	allowLoopback := m.allowLoopback
	cacheDir := m.cacheDir
	expectedBundleID := m.expectedBundleID
	verifyCodeSign := m.verifyCodeSign
	m.mu.Unlock()

	if cb != nil {
		cb(snap)
	}

	go func() {
		versionDir := filepath.Join(cacheDir, sanitizePathComponent(release.Version))
		zipPath := filepath.Join(versionDir, fmt.Sprintf("SuperGit-%s.zip", sanitizePathComponent(release.Version)))
		extractDir := filepath.Join(versionDir, "extracted")

		err := DownloadAndVerify(dlCtx, httpClient, release.Asset, zipPath, pubKey, allowLoopback, func(downloaded, total int64) {
			m.updateDownloadProgress(seq, downloaded, total)
		})
		if err != nil {
			m.finishDownload(seq, "", err)
			return
		}

		m.enterVerifying(seq)
		stagedApp, err := ExtractAndVerifyApp(dlCtx, zipPath, extractDir, expectedBundleID, verifyCodeSign)
		_ = os.Remove(zipPath)
		if err != nil {
			_ = os.RemoveAll(extractDir)
			m.finishDownload(seq, "", err)
			return
		}
		m.finishDownload(seq, stagedApp, nil)
	}()
	return true
}

func (m *Manager) updateDownloadProgress(seq uint64, downloaded, total int64) {
	m.mu.Lock()
	if seq != m.opSeq || m.snap.State != StateDownloading {
		m.mu.Unlock()
		return
	}
	m.snap.DownloadedBytes = downloaded
	if total > 0 {
		m.snap.TotalBytes = total
	}
	snap := m.snap
	cb := m.onChange
	m.mu.Unlock()
	if cb != nil {
		cb(snap)
	}
}

func (m *Manager) enterVerifying(seq uint64) {
	m.mu.Lock()
	if seq != m.opSeq || m.snap.State != StateDownloading {
		m.mu.Unlock()
		return
	}
	m.snap.State = StateVerifying
	snap := m.snap
	cb := m.onChange
	m.mu.Unlock()
	if cb != nil {
		cb(snap)
	}
}

func (m *Manager) finishDownload(seq uint64, stagedApp string, err error) {
	m.mu.Lock()
	if seq != m.opSeq {
		m.mu.Unlock()
		return
	}
	m.opCancel = nil
	if err != nil {
		if errors.Is(err, context.Canceled) {
			if m.snap.Release.Version != "" {
				m.snap.State = StateAvailable
			} else {
				m.snap.State = StateIdle
			}
			m.snap.Error = ""
		} else {
			m.snap.State = StateError
			m.snap.Error = err.Error()
		}
		m.snap.DownloadedBytes = 0
		snap := m.snap
		cb := m.onChange
		m.mu.Unlock()
		if cb != nil {
			cb(snap)
		}
		return
	}
	m.snap.State = StateReadyToRestart
	m.snap.StagedAppPath = stagedApp
	m.snap.Error = ""
	m.autoDownloaded = m.snap.AutoDownloadOrigin
	snap := m.snap
	cb := m.onChange
	m.mu.Unlock()
	if cb != nil {
		cb(snap)
	}
}

// Cancel cancels any active check or download and returns to Available/Idle.
func (m *Manager) Cancel() {
	m.mu.Lock()
	cancel := m.opCancel
	m.opCancel = nil
	if m.snap.State == StateDownloading || m.snap.State == StateVerifying || m.snap.State == StateChecking {
		m.opSeq++
		if m.snap.Release.Version != "" {
			m.snap.State = StateAvailable
		} else {
			m.snap.State = StateIdle
		}
		m.snap.DownloadedBytes = 0
		m.snap.Error = ""
	}
	snap := m.snap
	cb := m.onChange
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if cb != nil {
		cb(snap)
	}
}

// SkipAvailable marks the currently offered version as skipped and clears the
// pending update state.
func (m *Manager) SkipAvailable() string {
	m.mu.Lock()
	cancel := m.opCancel
	m.opCancel = nil
	m.opSeq++
	skipped := NormalizeVersion(m.snap.Release.Version)
	if skipped != "" {
		m.snap.SkippedVersion = skipped
	}
	if m.snap.StagedAppPath != "" {
		_ = os.RemoveAll(filepath.Dir(m.snap.StagedAppPath))
	}
	m.snap.State = StateIdle
	m.snap.Release = ReleaseInfo{}
	m.snap.StagedAppPath = ""
	m.snap.DownloadedBytes = 0
	m.snap.TotalBytes = 0
	m.snap.Error = ""
	snap := m.snap
	cb := m.onChange
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if cb != nil {
		cb(snap)
	}
	return skipped
}

// PrepareApply stages the verified app bundle beside the running bundle and
// spawns the detached helper process. The caller should immediately quit the
// application after PrepareApply returns nil.
func (m *Manager) PrepareApply(waitPID int) error {
	m.mu.Lock()
	if m.snap.State != StateReadyToRestart || m.snap.StagedAppPath == "" {
		m.mu.Unlock()
		return errors.New("no verified update is ready to install")
	}
	if !m.snap.Eligibility.Eligible || m.snap.Eligibility.BundlePath == "" {
		m.mu.Unlock()
		return errors.New("current installation cannot be updated in place")
	}
	staged := m.snap.StagedAppPath
	target := m.snap.Eligibility.BundlePath
	stageFn := m.stageAndLaunch
	m.snap.State = StateApplying
	snap := m.snap
	cb := m.onChange
	m.mu.Unlock()

	if cb != nil {
		cb(snap)
	}
	if err := stageFn(staged, target, waitPID); err != nil {
		m.mu.Lock()
		m.snap.State = StateError
		m.snap.Error = err.Error()
		errSnap := m.snap
		m.mu.Unlock()
		if cb != nil {
			cb(errSnap)
		}
		return err
	}
	return nil
}

func sanitizePathComponent(v string) string {
	var b strings.Builder
	for _, r := range v {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}
