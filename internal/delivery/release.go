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
	"strings"

	"supergit/internal/updater"
)

const maxReleaseManifestBytes = 1 << 20

// VerifyOptions identifies the packaged release and its Ed25519 public key.
type VerifyOptions struct {
	OutDir          string
	ArchivePath     string // empty: derive from manifest asset
	ManifestPath    string // empty: OutDir/update.json
	PublicKey       string
	Platform        string // default "darwin-arm64"
	VerifySignature func(context.Context, string) error
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
	pass := func(name string) {
		report.Checks = append(report.Checks, PreflightCheck{Name: name, Passed: true})
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
	pass("manifest")

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
	pass("size")

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
	pass("sha256")

	pubKey, err := updater.ParsePublicKey(opts.PublicKey)
	if err != nil {
		return fail("ed25519", err)
	}
	if err := updater.VerifyArchiveDigest(sum, expectedSHA256, asset.Ed25519, pubKey); err != nil {
		return fail("ed25519", err)
	}
	pass("ed25519")

	if _, err := updater.InspectZipSafety(archiveCopy); err != nil {
		return fail("zip_safety", err)
	}
	pass("zip_safety")

	var verifySignature func(context.Context, string) error
	if opts.VerifySignature != nil {
		verifySignature = func(ctx context.Context, appPath string) error {
			if err := opts.VerifySignature(ctx, appPath); err != nil {
				return fmt.Errorf("%w: %w", updater.ErrCodeSignVerify, err)
			}
			return nil
		}
	}
	bundlePath, err := updater.ExtractAndVerifyApp(ctx, archiveCopy, filepath.Join(workspace, "bundle"), updater.ExpectedBundleID, verifySignature)
	if errors.Is(err, updater.ErrCodeSignVerify) {
		// The updater only calls the signature verifier once extraction and
		// the bundle metadata checks have succeeded.
		pass("extract")
		return fail("codesign", err)
	}
	if err != nil {
		return fail("extract", err)
	}
	if err := updater.VerifyAppBundleMetadata(bundlePath, updater.ExpectedBundleID); err != nil {
		return fail("extract", err)
	}
	report.BundlePath = bundlePath
	pass("extract")
	pass("codesign")
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
	copy(sum[:], hash.Sum(nil))
	return sum, nil
}
