package updater

import (
	"archive/zip"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

const maxArchiveBytes = 512 << 20 // 512 MiB

var (
	ErrNoPublicKey       = errors.New("update verification public key is not configured")
	ErrSignatureMismatch = errors.New("update archive signature verification failed")
	ErrChecksumMismatch  = errors.New("update archive SHA-256 checksum mismatch")
	ErrInvalidArchive    = errors.New("update archive failed safety inspection")
	ErrBundleMismatch    = errors.New("extracted bundle identifier does not match SuperGit")
	ErrCodeSignVerify    = errors.New("extracted app bundle failed code signature verification")
)

// ParsePublicKey decodes a base64-encoded Ed25519 public key.
func ParsePublicKey(encoded string) (ed25519.PublicKey, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, ErrNoPublicKey
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid public key encoding: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length %d", len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

// VerifyArchiveDigest checks both SHA-256 hex digest and Ed25519 signature
// (computed over the 32-byte SHA-256 digest of the archive).
func VerifyArchiveDigest(sum [sha256.Size]byte, expectedSHA256, signatureBase64 string, pubKey ed25519.PublicKey) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return ErrNoPublicKey
	}
	wantHex := strings.ToLower(strings.TrimSpace(expectedSHA256))
	if len(wantHex) != sha256.Size*2 {
		return fmt.Errorf("%w: invalid sha256 field", ErrChecksumMismatch)
	}
	gotHex := hex.EncodeToString(sum[:])
	if gotHex != wantHex {
		return fmt.Errorf("%w: got %s want %s", ErrChecksumMismatch, gotHex, wantHex)
	}
	sigBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signatureBase64))
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return ErrSignatureMismatch
	}
	if !ed25519.Verify(pubKey, sum[:], sigBytes) {
		return ErrSignatureMismatch
	}
	return nil
}

// SignDigest signs a 32-byte SHA-256 digest with an Ed25519 private key and
// returns the base64-encoded signature.
func SignDigest(priv ed25519.PrivateKey, sum [sha256.Size]byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sum[:]))
}

// DownloadAndVerify streams the platform asset to destPath while computing
// SHA-256 and verifying the Ed25519 signature before finalizing the file.
func DownloadAndVerify(
	ctx context.Context,
	client *http.Client,
	asset PlatformAsset,
	destPath string,
	pubKey ed25519.PublicKey,
	allowLoopback bool,
	onProgress func(downloaded, total int64),
) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return ErrNoPublicKey
	}
	if err := ValidateUpdateURL(asset.URL, allowLoopback); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "SuperGit-Updater")

	httpClient := newSecureHTTPClient(client, 15*time.Minute, allowLoopback)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength
	if total <= 0 && asset.Size > 0 {
		total = asset.Size
	}
	if total > maxArchiveBytes {
		return fmt.Errorf("archive size %d exceeds maximum allowed size", total)
	}

	tmpPath := destPath + ".part"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	hasher := sha256.New()
	limited := io.LimitReader(resp.Body, maxArchiveBytes+1)
	buf := make([]byte, 32*1024)
	var written int64
	lastReport := time.Time{}

	for {
		n, readErr := limited.Read(buf)
		if n > 0 {
			written += int64(n)
			if written > maxArchiveBytes {
				_ = f.Close()
				return errors.New("downloaded archive exceeds maximum allowed size")
			}
			if _, err := f.Write(buf[:n]); err != nil {
				_ = f.Close()
				return err
			}
			_, _ = hasher.Write(buf[:n])
			if onProgress != nil && (lastReport.IsZero() || time.Since(lastReport) >= 100*time.Millisecond) {
				lastReport = time.Now()
				onProgress(written, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = f.Close()
			return readErr
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if onProgress != nil {
		onProgress(written, max(total, written))
	}
	if asset.Size > 0 && written != asset.Size {
		return fmt.Errorf("downloaded size mismatch: got %d want %d", written, asset.Size)
	}

	var sum [sha256.Size]byte
	copy(sum[:], hasher.Sum(nil))
	if err := VerifyArchiveDigest(sum, asset.SHA256, asset.Ed25519, pubKey); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// InspectZipSafety verifies that every entry in a zip archive stays inside a
// single top-level ".app" bundle directory with no path traversal.
func InspectZipSafety(zipPath string) (string, error) {
	appName, _, err := inspectZipSafety(zipPath)
	return appName, err
}

func inspectZipSafety(zipPath string) (appRoot string, hasBackslashes bool, err error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return "", false, fmt.Errorf("%w: %v", ErrInvalidArchive, err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		clean, err := cleanZipEntry(f.Name)
		if err != nil {
			return "", false, err
		}
		hasBackslashes = hasBackslashes || strings.Contains(f.Name, `\`)
		top, _, _ := strings.Cut(clean, "/")
		if top == "__MACOSX" {
			continue
		}
		if !strings.HasSuffix(top, ".app") {
			return "", false, fmt.Errorf("%w: unexpected top-level entry %q", ErrInvalidArchive, top)
		}
		if appRoot == "" {
			appRoot = top
		} else if appRoot != top {
			return "", false, fmt.Errorf("%w: multiple app bundles (%q and %q)", ErrInvalidArchive, appRoot, top)
		}
	}
	if appRoot == "" {
		return "", false, fmt.Errorf("%w: archive contains no .app bundle", ErrInvalidArchive)
	}
	return appRoot, hasBackslashes, nil
}

// Check both ZIP separators so Windows-style paths and traversal are
// interpreted consistently, regardless of the host filesystem.
func cleanZipEntry(name string) (string, error) {
	normalized := strings.ReplaceAll(name, `\`, "/")
	clean := path.Clean(normalized)
	if path.IsAbs(normalized) || clean == "." || strings.Contains(normalized, ":") ||
		slices.Contains(strings.Split(normalized, "/"), "..") {
		return "", fmt.Errorf("%w: unsafe entry %q", ErrInvalidArchive, name)
	}
	return clean, nil
}

// ExtractAndVerifyApp validates the zip entries, extracts the .app bundle into
// destDir, checks Info.plist bundle ID & executable, and runs code signature
// verification on macOS.
func ExtractAndVerifyApp(ctx context.Context, zipPath, destDir, expectedBundleID string, verifySign func(context.Context, string) error) (string, error) {
	appName, hasBackslashes, err := inspectZipSafety(zipPath)
	if err != nil {
		return "", err
	}
	if err := os.RemoveAll(destDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", err
	}

	// Ditto treats backslashes literally, so use the normalizing extractor for
	// archives that need separator conversion.
	if runtime.GOOS == "darwin" && !hasBackslashes {
		cmd := exec.CommandContext(ctx, "ditto", "-x", "-k", zipPath, destDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("extract archive failed: %v (%s)", err, strings.TrimSpace(string(out)))
		}
	} else {
		if err := extractZipPureGo(zipPath, destDir); err != nil {
			return "", err
		}
	}

	appPath := filepath.Join(destDir, appName)
	if err := VerifyAppBundleMetadata(appPath, expectedBundleID); err != nil {
		return "", err
	}
	if verifySign == nil {
		verifySign = defaultVerifyCodeSign
	}
	if err := verifySign(ctx, appPath); err != nil {
		return "", err
	}
	return appPath, nil
}

// VerifyAppBundleMetadata verifies Info.plist bundle identifier and executable.
func VerifyAppBundleMetadata(appPath, expectedBundleID string) error {
	plist, err := ReadAppPlist(appPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBundleMismatch, err)
	}
	if err := plist.VerifyBundleID(expectedBundleID); err != nil {
		return err
	}
	if _, err := plist.ExecutablePath(appPath); err != nil {
		return fmt.Errorf("%w: %v", ErrBundleMismatch, err)
	}
	return nil
}

func defaultVerifyCodeSign(ctx context.Context, appPath string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	cmd := exec.CommandContext(ctx, "codesign", "--verify", "--deep", "--strict", appPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", ErrCodeSignVerify, strings.TrimSpace(string(out)))
	}
	return nil
}

func extractZipPureGo(zipPath, destDir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		clean, err := cleanZipEntry(f.Name)
		if err != nil {
			return err
		}
		top, _, _ := strings.Cut(clean, "/")
		if top == "__MACOSX" {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(clean))
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("%w: unsafe extraction path %q", ErrInvalidArchive, f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode().Perm()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		mode := f.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, maxArchiveBytes))
		closeErr := out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
