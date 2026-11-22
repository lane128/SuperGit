package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	// DefaultManifestURL points to the static latest-release manifest asset on
	// GitHub Releases to avoid unauthenticated GitHub REST API rate limits.
	DefaultManifestURL = "https://github.com/lane128/SuperGit/releases/latest/download/update.json"

	// DefaultReleasesURL is the fallback human-readable releases page.
	DefaultReleasesURL = "https://github.com/lane128/SuperGit/releases/latest"

	// ExpectedBundleID identifies the genuine SuperGit macOS application bundle.
	ExpectedBundleID = "cc.supermix.supergit"

	// EnvManifestURL overrides the update manifest URL for local testing.
	EnvManifestURL = "SUPERGIT_UPDATE_URL"

	// EnvPublicKey overrides the Ed25519 public key (base64) for local testing.
	EnvPublicKey = "SUPERGIT_UPDATE_PUBKEY"

	// EnvDisableAutoUpdate suppresses background startup update checks when set.
	EnvDisableAutoUpdate = "SUPERGIT_DISABLE_AUTO_UPDATE"

	maxManifestBytes = 1 << 20 // 1 MiB
	maxRedirects     = 5
)

// DefaultPublicKeyBase64 is the base64-encoded Ed25519 public key embedded at
// release build time via -ldflags "-X supergit/internal/updater.DefaultPublicKeyBase64=...".
var DefaultPublicKeyBase64 = ""

// Manifest is the release metadata published as update.json.
type Manifest struct {
	Version   string                   `json:"version"`
	Build     int                      `json:"build,omitempty"`
	PubDate   string                   `json:"pubDate,omitempty"`
	NotesURL  string                   `json:"notesURL,omitempty"`
	Notes     map[string]string        `json:"notes,omitempty"`
	MinOS     map[string]string        `json:"minOS,omitempty"`
	Platforms map[string]PlatformAsset `json:"platforms"`
}

// PlatformAsset describes a downloadable archive for a single OS/arch pair.
type PlatformAsset struct {
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Ed25519 string `json:"ed25519"`
	Size    int64  `json:"size,omitempty"`
}

// ReleaseInfo is the UI-ready projection of an available update.
type ReleaseInfo struct {
	Version  string
	Build    int
	PubDate  string
	NotesZH  string
	NotesEN  string
	NotesURL string
	Asset    PlatformAsset
	HasAsset bool
}

// NotesFor returns localized release notes with a sensible fallback.
func (r ReleaseInfo) NotesFor(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		if strings.TrimSpace(r.NotesZH) != "" {
			return r.NotesZH
		}
		return r.NotesEN
	}
	if strings.TrimSpace(r.NotesEN) != "" {
		return r.NotesEN
	}
	return r.NotesZH
}

// EffectiveManifestURL returns the configured manifest URL and whether loopback
// URLs are explicitly allowed (only when overridden to a local test server).
func EffectiveManifestURL(configured string) (string, bool) {
	raw := strings.TrimSpace(configured)
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(EnvManifestURL))
	}
	if raw == "" {
		return DefaultManifestURL, false
	}
	return raw, isLoopbackURL(raw)
}

// EffectivePublicKeyBase64 returns the configured Ed25519 public key in base64.
func EffectivePublicKeyBase64(configured string) string {
	if s := strings.TrimSpace(configured); s != "" {
		return s
	}
	if s := strings.TrimSpace(os.Getenv(EnvPublicKey)); s != "" {
		return s
	}
	return strings.TrimSpace(DefaultPublicKeyBase64)
}

func isLoopbackURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// ValidateUpdateURL ensures update manifests and downloads only talk to trusted
// HTTPS GitHub hosts unless loopback is explicitly permitted for local testing.
func ValidateUpdateURL(raw string, allowLoopback bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("update URL is empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid update URL: %w", err)
	}
	if u.User != nil {
		return errors.New("update URL must not contain credentials")
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "" {
		return errors.New("update URL has no host")
	}
	if allowLoopback && (host == "127.0.0.1" || host == "localhost" || host == "::1") {
		if scheme == "http" || scheme == "https" {
			return nil
		}
		return fmt.Errorf("unsupported update URL scheme %q", u.Scheme)
	}
	if scheme != "https" {
		return fmt.Errorf("update URL must use HTTPS: %s", u.Scheme)
	}
	if host == "github.com" || strings.HasSuffix(host, ".githubusercontent.com") {
		return nil
	}
	return fmt.Errorf("untrusted update host %q", host)
}

func newSecureHTTPClient(base *http.Client, timeout time.Duration, allowLoopback bool) *http.Client {
	var transport http.RoundTripper
	if base != nil {
		transport = base.Transport
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			return ValidateUpdateURL(req.URL.String(), allowLoopback)
		},
	}
}

// FetchManifest downloads and parses the update manifest JSON.
func FetchManifest(ctx context.Context, client *http.Client, manifestURL string, allowLoopback bool) (Manifest, error) {
	if err := ValidateUpdateURL(manifestURL, allowLoopback); err != nil {
		return Manifest{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return Manifest{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SuperGit-Updater")

	httpClient := newSecureHTTPClient(client, 20*time.Second, allowLoopback)
	resp, err := httpClient.Do(req)
	if err != nil {
		return Manifest{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Manifest{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return Manifest{}, err
	}
	if len(data) > maxManifestBytes {
		return Manifest{}, errors.New("update manifest exceeds maximum allowed size")
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("invalid update manifest JSON: %w", err)
	}
	if _, err := ParseVersion(m.Version); err != nil {
		return Manifest{}, fmt.Errorf("manifest version invalid: %w", err)
	}
	m.Version = NormalizeVersion(m.Version)
	return m, nil
}

// SelectAsset picks the matching platform asset for goos/goarch, falling back
// to darwin-universal on macOS when available.
func (m Manifest) SelectAsset(goos, goarch string) (PlatformAsset, bool) {
	if len(m.Platforms) == 0 {
		return PlatformAsset{}, false
	}
	key := goos + "-" + goarch
	if asset, ok := m.Platforms[key]; ok && strings.TrimSpace(asset.URL) != "" {
		return asset, true
	}
	if goos == "darwin" {
		if asset, ok := m.Platforms["darwin-universal"]; ok && strings.TrimSpace(asset.URL) != "" {
			return asset, true
		}
	}
	return PlatformAsset{}, false
}

// CurrentPlatform returns runtime.GOOS and runtime.GOARCH.
func CurrentPlatform() (string, string) {
	return runtime.GOOS, runtime.GOARCH
}

// ToReleaseInfo projects a manifest into ReleaseInfo for the given OS/arch.
func (m Manifest) ToReleaseInfo(goos, goarch string) ReleaseInfo {
	asset, hasAsset := m.SelectAsset(goos, goarch)
	notesURL := strings.TrimSpace(m.NotesURL)
	if notesURL == "" {
		notesURL = DefaultReleasesURL
	}
	info := ReleaseInfo{
		Version:  NormalizeVersion(m.Version),
		Build:    m.Build,
		PubDate:  strings.TrimSpace(m.PubDate),
		NotesURL: notesURL,
		Asset:    asset,
		HasAsset: hasAsset,
	}
	if m.Notes != nil {
		info.NotesZH = strings.TrimSpace(m.Notes["zh"])
		info.NotesEN = strings.TrimSpace(m.Notes["en"])
	}
	return info
}
