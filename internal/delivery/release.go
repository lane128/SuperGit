package delivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"supergit/internal/updater"
)

const maxReleaseManifestBytes = 1 << 20

var nativeCodeSignAvailable = func() bool { return runtime.GOOS == "darwin" }

// VerifyOptions identifies the packaged release and its Ed25519 public key.
type VerifyOptions struct {
	OutDir          string
	ArchivePath     string // empty: derive from manifest asset
	ManifestPath    string // empty: OutDir/update.json
	PublicKey       string
	Platform        string                              // default "darwin-arm64"
	VerifySignature func(context.Context, string) error // nil: use native macOS verification
}

// ReleaseVerification contains release metadata and checks in execution order.
type ReleaseVerification struct {
	Version     string
	Build       int
	ArchivePath string
	SHA256      string
	Size        int64
	BundlePath  string
	Checks      []PreflightCheck
}

// VerifyRelease verifies a local release archive against its manifest. On
// success, BundlePath points to the extracted app under the bundle subdirectory
// of a temporary workspace. The caller can remove the workspace after inspection.
func VerifyRelease(ctx context.Context, opts VerifyOptions) (ReleaseVerification, error) {
	var report ReleaseVerification
	pass := func(name, detail string) {
		report.Checks = append(report.Checks, PreflightCheck{Name: name, Passed: true, Detail: detail})
	}
	fail := func(name string, err error) (ReleaseVerification, error) {
		report.Checks = append(report.Checks, PreflightCheck{Name: name, Detail: err.Error()})
		return report, fmt.Errorf("%s: %w", name, err)
	}

	manifestPath := opts.ManifestPath
	if manifestPath == "" {
		manifestPath = filepath.Join(opts.OutDir, "update.json")
	}
	manifest, err := readReleaseManifest(manifestPath)
	if err != nil {
		return fail("manifest", err)
	}
	report.Version, report.Build = manifest.Version, manifest.Build
	platform := opts.Platform
	if platform == "" {
		platform = "darwin-arm64"
	}
	asset, ok := manifest.Platforms[platform]
	if !ok {
		return fail("manifest", fmt.Errorf("no asset for platform %q", platform))
	}
	report.ArchivePath = opts.ArchivePath
	if report.ArchivePath == "" {
		assetURL, err := url.Parse(asset.URL)
		if err != nil {
			return fail("manifest", fmt.Errorf("invalid asset URL: %w", err))
		}
		name := path.Base(assetURL.Path)
		if name == "." || name == ".." || name == "/" || strings.ContainsAny(name, `\:`) {
			return fail("manifest", errors.New("asset URL has no usable archive filename"))
		}
		report.ArchivePath = filepath.Join(opts.OutDir, name)
	}
	pass("manifest", fmt.Sprintf("version %s, build %d", report.Version, report.Build))

	archive, err := os.Open(report.ArchivePath)
	if err != nil {
		return fail("size", err)
	}
	defer archive.Close()
	info, err := archive.Stat()
	if err != nil {
		return fail("size", err)
	}
	if !info.Mode().IsRegular() {
		return fail("size", errors.New("archive must be a regular file"))
	}
	report.Size = info.Size()
	if report.Size != asset.Size {
		return fail("size", fmt.Errorf("archive size mismatch: got %d, want %d bytes", report.Size, asset.Size))
	}
	pass("size", fmt.Sprintf("%d bytes", report.Size))

	workspace, err := os.MkdirTemp("", "supergit-verify-release-*")
	if err != nil {
		return fail("sha256", err)
	}
	archiveCopy := filepath.Join(workspace, "archive.zip")
	defer func() {
		if report.BundlePath == "" {
			_ = os.RemoveAll(workspace)
			return
		}
		_ = os.Remove(archiveCopy)
	}()
	// Inspect and extract the same bytes that were hashed, even if the
	// original archive is replaced while verification is in progress.
	sum, err := copyReleaseArchive(archiveCopy, archive, report.Size)
	if err != nil {
		return fail("sha256", err)
	}
	report.SHA256 = hex.EncodeToString(sum[:])
	expectedSHA256 := strings.ToLower(strings.TrimSpace(asset.SHA256))
	if len(expectedSHA256) != sha256.Size*2 {
		return fail("sha256", fmt.Errorf("%w: invalid sha256 field", updater.ErrChecksumMismatch))
	}
	if report.SHA256 != expectedSHA256 {
		return fail("sha256", fmt.Errorf("%w: got %s, want %q", updater.ErrChecksumMismatch, report.SHA256, expectedSHA256))
	}
	pass("sha256", report.SHA256)

	pubKey, err := updater.ParsePublicKey(opts.PublicKey)
	if err != nil {
		return fail("ed25519", err)
	}
	if err := updater.VerifyArchiveDigest(sum, expectedSHA256, asset.Ed25519, pubKey); err != nil {
		return fail("ed25519", err)
	}
	pass("ed25519", "signature verified")

	appName, err := updater.InspectZipSafety(archiveCopy)
	if err != nil {
		return fail("zip_safety", err)
	}
	pass("zip_safety", "zip safe")

	var verifySignature func(context.Context, string) error
	switch {
	case opts.VerifySignature != nil:
		verifySignature = func(ctx context.Context, appPath string) error {
			if err := opts.VerifySignature(ctx, appPath); err != nil {
				return fmt.Errorf("%w: %w", updater.ErrCodeSignVerify, err)
			}
			return nil
		}
	case !nativeCodeSignAvailable():
		verifySignature = func(context.Context, string) error {
			return fmt.Errorf("%w: code signature verification requires macOS", updater.ErrCodeSignVerify)
		}
	}
	bundleDir := filepath.Join(workspace, "bundle")
	bundlePath, err := updater.ExtractAndVerifyApp(ctx, archiveCopy, bundleDir, updater.ExpectedBundleID, verifySignature)
	if err != nil && !errors.Is(err, updater.ErrCodeSignVerify) {
		return fail("extract", err)
	}
	// A signature failure still leaves the extracted bundle available to check.
	if err != nil {
		bundlePath = filepath.Join(bundleDir, appName)
	}
	plist, plistErr := updater.ReadAppPlist(bundlePath)
	if plistErr != nil {
		return fail("extract", plistErr)
	}
	version, build, plistErr := plist.ReleaseVersion()
	if plistErr != nil {
		return fail("extract", plistErr)
	}
	if version != report.Version {
		return fail("extract", fmt.Errorf("bundle version %q does not match manifest version %q", version, report.Version))
	}
	if build != report.Build {
		return fail("extract", fmt.Errorf("bundle build %d does not match manifest build %d", build, report.Build))
	}
	pass("extract", bundlePath)
	if err != nil {
		return fail("codesign", err)
	}
	report.BundlePath = bundlePath
	pass("codesign", "code signature verified")
	return report, nil
}

func readReleaseManifest(manifestPath string) (updater.Manifest, error) {
	var manifest updater.Manifest
	file, err := os.Open(manifestPath)
	if err != nil {
		return manifest, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxReleaseManifestBytes+1))
	if err != nil {
		return manifest, err
	}
	if len(data) > maxReleaseManifestBytes {
		return manifest, errors.New("update manifest exceeds 1 MiB")
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("invalid update manifest JSON: %w", err)
	}
	if _, err := updater.ParseVersion(manifest.Version); err != nil {
		return manifest, fmt.Errorf("invalid manifest version: %w", err)
	}
	return manifest, nil
}

func copyReleaseArchive(dest string, archive io.Reader, size int64) ([sha256.Size]byte, error) {
	var sum [sha256.Size]byte
	file, err := os.Create(dest)
	if err != nil {
		return sum, err
	}
	hash := sha256.New()
	_, copyErr := io.CopyN(io.MultiWriter(file, hash), archive, size)
	closeErr := file.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return sum, fmt.Errorf("copy release archive: %w", err)
	}
	var trailing [1]byte
	if n, err := archive.Read(trailing[:]); n != 0 || err == nil {
		return sum, fmt.Errorf("archive grew during verification: expected %d bytes", size)
	} else if err != io.EOF {
		return sum, fmt.Errorf("check end of release archive: %w", err)
	}
	copy(sum[:], hash.Sum(nil))
	return sum, nil
}
