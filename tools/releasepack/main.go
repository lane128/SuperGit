// Command releasepack generates Ed25519 update signing keys and packages a
// built SuperGit.app into a signed zip archive plus update.json manifest.
package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"supergit/internal/updater"
)

func main() {
	genKey := flag.Bool("genkey", false, "generate a new Ed25519 signing keypair")
	privPath := flag.String("priv", "", "path to base64-encoded Ed25519 private key")
	pubPath := flag.String("pub", "", "path to write base64-encoded Ed25519 public key (used with -genkey)")
	appPath := flag.String("app", "build/SuperGit.app", "path to the built .app bundle")
	outDir := flag.String("out", "build/release", "output directory for zip and update.json")
	tag := flag.String("tag", "", "selected release tag, which must match the bundle version")
	platform := flag.String("platform", runtime.GOOS+"-"+runtime.GOARCH, "platform key in update.json")
	baseURL := flag.String("base-url", "", "download base URL for the release assets")
	notesURL := flag.String("notes-url", "", "release notes web URL")
	notesZH := flag.String("notes-zh", "", "Chinese release notes text")
	notesEN := flag.String("notes-en", "", "English release notes text")
	flag.Parse()

	if *genKey {
		if err := generateKeypair(*privPath, *pubPath); err != nil {
			fmt.Fprintf(os.Stderr, "releasepack: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := packageRelease(packOptions{
		AppPath:  *appPath,
		OutDir:   *outDir,
		PrivPath: *privPath,
		Platform: *platform,
		Tag:      *tag,
		BaseURL:  *baseURL,
		NotesURL: *notesURL,
		NotesZH:  *notesZH,
		NotesEN:  *notesEN,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "releasepack: %v\n", err)
		os.Exit(1)
	}
}

func generateKeypair(privPath, pubPath string) error {
	if strings.TrimSpace(privPath) == "" {
		return errors.New("-priv is required with -genkey")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	privB64 := base64.StdEncoding.EncodeToString(priv)
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	if err := os.MkdirAll(filepath.Dir(privPath), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(privPath, []byte(privB64+"\n"), 0o600); err != nil {
		return err
	}
	if strings.TrimSpace(pubPath) != "" {
		if err := os.MkdirAll(filepath.Dir(pubPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(pubPath, []byte(pubB64+"\n"), 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("Private key saved to %s (keep secret, never commit)\n", privPath)
	if strings.TrimSpace(pubPath) != "" {
		fmt.Printf("Public key saved to %s\n", pubPath)
	}
	fmt.Printf("UPDATE_PUBKEY=%s\n", pubB64)
	return nil
}

type packOptions struct {
	AppPath  string
	OutDir   string
	PrivPath string
	Platform string
	Tag      string
	BaseURL  string
	NotesURL string
	NotesZH  string
	NotesEN  string
}

func packageRelease(opts packOptions) error {
	if strings.TrimSpace(opts.PrivPath) == "" {
		return errors.New("-priv is required")
	}
	priv, pub, err := loadPrivateKey(opts.PrivPath)
	if err != nil {
		return err
	}
	plist, err := updater.ReadAppPlist(opts.AppPath)
	if err != nil {
		return fmt.Errorf("embedded update public key: %w", err)
	}
	pubBase64 := base64.StdEncoding.EncodeToString(pub)
	embeddedPubKey, err := plist.String("SuperGitUpdatePublicKey")
	if err != nil {
		return fmt.Errorf("embedded update public key: %w", err)
	}
	if embeddedPubKey == "" {
		return fmt.Errorf("embedded update public key: missing or empty in Info.plist")
	}
	if embeddedPubKey != pubBase64 {
		return fmt.Errorf("embedded update public key: Info.plist key does not match the signing key")
	}
	executable, err := plist.ExecutablePath(opts.AppPath)
	if err != nil {
		return fmt.Errorf("embedded update public key: %w", err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		return fmt.Errorf("embedded update public key: read executable: %w", err)
	}
	if !bytes.Contains(binary, []byte(pubBase64)) {
		return errors.New("embedded update public key: executable does not contain the signing public key")
	}
	if err := plist.VerifyBundleID(updater.ExpectedBundleID); err != nil {
		return err
	}
	shortVer, buildNum, err := plist.ReleaseVersion()
	if err != nil {
		return err
	}
	if opts.Tag != "" && shortVer != strings.TrimPrefix(opts.Tag, "v") {
		return fmt.Errorf("CFBundleShortVersionString %q does not match release tag %q", shortVer, opts.Tag)
	}
	minOS, err := plist.String("LSMinimumSystemVersion")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}

	if opts.Platform == "" || strings.ContainsAny(opts.Platform, `/\`) {
		return errors.New("invalid platform for release archive")
	}
	stage, err := os.MkdirTemp(opts.OutDir, ".release-pack-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	archiveName := fmt.Sprintf("SuperGit-%s-%s.zip", shortVer, opts.Platform)
	archivePath := filepath.Join(opts.OutDir, archiveName)
	stagedArchive := filepath.Join(stage, archiveName)

	if err := archiveAppBundle(opts.AppPath, stagedArchive); err != nil {
		return err
	}
	if _, err := updater.InspectZipSafety(stagedArchive); err != nil {
		return fmt.Errorf("generated archive failed safety check: %w", err)
	}

	sum, size, err := fileSHA256(stagedArchive)
	if err != nil {
		return err
	}
	sumHex := hex.EncodeToString(sum[:])
	sigB64 := updater.SignDigest(priv, sum)

	baseURL := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://github.com/lane128/SuperGit/releases/download/v%s", shortVer)
	}
	notesURL := strings.TrimSpace(opts.NotesURL)
	if notesURL == "" {
		notesURL = fmt.Sprintf("https://github.com/lane128/SuperGit/releases/tag/v%s", shortVer)
	}

	manifestPath := filepath.Join(opts.OutDir, "update.json")
	var manifest updater.Manifest
	if existing, err := os.ReadFile(manifestPath); err == nil {
		var prev updater.Manifest
		if json.Unmarshal(existing, &prev) == nil && prev.Version == shortVer {
			manifest = prev
		}
	}
	manifest.Version = shortVer
	manifest.Build = buildNum
	if manifest.PubDate == "" {
		manifest.PubDate = time.Now().UTC().Format(time.RFC3339)
	}
	manifest.NotesURL = notesURL
	if manifest.Notes == nil {
		manifest.Notes = map[string]string{}
	}
	if strings.TrimSpace(opts.NotesZH) != "" {
		manifest.Notes["zh"] = strings.TrimSpace(opts.NotesZH)
	}
	if strings.TrimSpace(opts.NotesEN) != "" {
		manifest.Notes["en"] = strings.TrimSpace(opts.NotesEN)
	}
	if minOS != "" {
		if manifest.MinOS == nil {
			manifest.MinOS = map[string]string{}
		}
		manifest.MinOS["darwin"] = minOS
	}
	if manifest.Platforms == nil {
		manifest.Platforms = map[string]updater.PlatformAsset{}
	}
	manifest.Platforms[opts.Platform] = updater.PlatformAsset{
		URL:     baseURL + "/" + archiveName,
		SHA256:  sumHex,
		Ed25519: sigB64,
		Size:    size,
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	stagedManifest := filepath.Join(stage, "update.json")
	encoded = append(encoded, '\n')
	if err := os.WriteFile(stagedManifest, encoded, 0o644); err != nil {
		return err
	}
	stagedData, err := os.ReadFile(stagedManifest)
	if err != nil {
		return err
	}
	var verified updater.Manifest
	if err := json.Unmarshal(stagedData, &verified); err != nil {
		return fmt.Errorf("verify staged manifest: %w", err)
	}
	asset := verified.Platforms[opts.Platform]
	if verified.Version != shortVer || verified.Build != buildNum || asset.Size != size {
		return errors.New("staged manifest metadata does not match the archive")
	}
	if err := updater.VerifyArchiveDigest(sum, asset.SHA256, asset.Ed25519, pub); err != nil {
		return fmt.Errorf("self-verification failed: %w", err)
	}
	artifacts := []struct {
		staged, final string
		digest        [sha256.Size]byte
	}{{stagedArchive, archivePath, sum}, {stagedManifest, manifestPath, sha256.Sum256(stagedData)}}
	// Check both destinations before publishing either staged file.
	for _, artifact := range artifacts {
		if err := compareArtifact(artifact.final, artifact.digest); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for _, artifact := range artifacts {
		// A hard link atomically installs a complete file and never clobbers a
		// destination another attempt published after the initial checks.
		if err := os.Link(artifact.staged, artifact.final); err != nil {
			if !errors.Is(err, os.ErrExist) {
				return err
			}
			if err := compareArtifact(artifact.final, artifact.digest); err != nil {
				return err
			}
		}
	}
	fmt.Printf("Packaged %s (%d bytes)\n", archivePath, size)
	fmt.Printf("Manifest %s (v%s build %d)\n", manifestPath, shortVer, buildNum)
	fmt.Printf("SHA-256  %s\n", sumHex)
	fmt.Printf("PubKey   %s\n", pubBase64)
	return nil
}

func loadPrivateKey(path string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, errors.New("cannot read private key file")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid private key base64: %w", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, nil, fmt.Errorf("invalid private key length %d", len(raw))
	}
	priv := ed25519.PrivateKey(raw)
	pub, _ := priv.Public().(ed25519.PublicKey)
	return priv, pub, nil
}

func archiveAppBundle(appPath, zipPath string) error {
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("ditto", "-c", "-k", "--sequesterRsrc", "--keepParent", appPath, zipPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ditto archive failed: %v (%s)", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	return errors.New("packaging macOS .app bundles requires macOS ditto")
}

func fileSHA256(path string) ([sha256.Size]byte, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return [sha256.Size]byte{}, 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return [sha256.Size]byte{}, 0, err
	}
	var sum [sha256.Size]byte
	copy(sum[:], h.Sum(nil))
	return sum, n, nil
}

func compareArtifact(path string, expected [sha256.Size]byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("existing artifact %s is not a regular file", path)
	}
	sum, _, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if sum != expected {
		return fmt.Errorf("existing artifact %s differs from the staged artifact", path)
	}
	return nil
}
