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
	appPath := opts.AppPath
	if runtime.GOOS == "darwin" {
		// Validate and archive a private copy, not a build path that can be
		// replaced by a concurrent build while the packager waits for its lock.
		root, err := os.MkdirTemp("", "supergit-release-bundle-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(root)
		appPath, err = snapshotAppBundle(opts.AppPath, root)
		if err != nil {
			return err
		}
	}
	if err := requireBundleFile(appPath, "Contents", "Info.plist"); err != nil {
		return fmt.Errorf("embedded update public key: %w", err)
	}
	plist, err := updater.ReadAppPlist(appPath)
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
	executable, err := plist.ExecutablePath(appPath)
	if err != nil {
		return fmt.Errorf("embedded update public key: %w", err)
	}
	if err := requireBundleFile(appPath, "Contents", "MacOS", filepath.Base(executable)); err != nil {
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

	if opts.Platform == "" || strings.ContainsAny(opts.Platform, `/\`) {
		return errors.New("invalid platform for release archive")
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}
	output, err := lockReleaseOutput(opts.OutDir)
	if err != nil {
		return err
	}
	defer output.Close()

	pending := filepath.Join(opts.OutDir, pendingPackageDir)
	pendingInfo, err := os.Lstat(pending)
	recovering := err == nil
	if recovering && !pendingInfo.IsDir() {
		return errors.New("pending release package must be a directory")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	stage, err := os.MkdirTemp(opts.OutDir, ".release-pack-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	prepared := filepath.Join(stage, "prepared")
	archiveName := fmt.Sprintf("SuperGit-%s-%s.zip", shortVer, opts.Platform)
	archivePath := filepath.Join(opts.OutDir, archiveName)
	manifestPath := filepath.Join(opts.OutDir, "update.json")
	archiveSource := prepared
	if recovering {
		if err := validatePendingPackage(pending, archiveName); err != nil {
			return err
		}
		archiveSource = pending
	} else {
		complete, err := existingPackage(archivePath, manifestPath)
		if err != nil {
			return err
		}
		if complete {
			archiveSource = opts.OutDir
		}
	}
	reusing := archiveSource != prepared
	freshArchive := filepath.Join(stage, archiveName)
	if !reusing {
		if err := os.Mkdir(prepared, 0o700); err != nil {
			return err
		}
		freshArchive = filepath.Join(prepared, archiveName)
	}
	if err := archiveAppBundle(appPath, freshArchive); err != nil {
		return err
	}
	if _, err := updater.InspectZipSafety(freshArchive); err != nil {
		return fmt.Errorf("generated archive failed safety check: %w", err)
	}

	stagedArchive := filepath.Join(archiveSource, archiveName)
	if reusing {
		if err := requireRegularArtifact(stagedArchive); err != nil {
			return err
		}
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

	prevManifestPath := manifestPath
	if reusing {
		prevManifestPath = filepath.Join(archiveSource, "update.json")
		if err := requireRegularArtifact(prevManifestPath); err != nil {
			return err
		}
	}
	prevData, readErr := os.ReadFile(prevManifestPath)
	if reusing && readErr != nil {
		return fmt.Errorf("read existing release manifest: %w", readErr)
	}
	var manifest updater.Manifest
	var prev updater.Manifest
	if readErr == nil && json.Unmarshal(prevData, &prev) == nil && prev.Version == shortVer {
		manifest = prev
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
	for lang, notes := range map[string]string{"zh": opts.NotesZH, "en": opts.NotesEN} {
		if notes = strings.TrimSpace(notes); notes != "" {
			manifest.Notes[lang] = notes
		} else {
			delete(manifest.Notes, lang)
		}
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
	stagedManifest := filepath.Join(archiveSource, "update.json")
	encoded = append(encoded, '\n')
	if reusing {
		if !bytes.Equal(prevData, encoded) {
			return errors.New("existing release manifest differs from the requested package")
		}
	} else if err := os.WriteFile(stagedManifest, encoded, 0o644); err != nil {
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
	if reusing {
		if err := compareBundleArchives(freshArchive, stagedArchive); err != nil {
			return fmt.Errorf("reuse release package: %w", err)
		}
	}
	if recovering || !reusing {
		artifacts := []releaseArtifact{{archiveName, sum}, {"update.json", sha256.Sum256(stagedData)}}
		if err := installPackage(output, prepared, pending, recovering, artifacts); err != nil {
			return err
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

// Reject links in the metadata and executable paths so validation cannot
// consult anything outside the private snapshot being archived.
func requireBundleFile(appPath string, components ...string) error {
	for _, component := range append([]string{"."}, components[:len(components)-1]...) {
		appPath = filepath.Join(appPath, component)
		info, err := os.Lstat(appPath)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("bundle directory %q must not be a link", component)
		}
	}
	return requireRegularArtifact(filepath.Join(appPath, components[len(components)-1]))
}

func snapshotAppBundle(appPath, root string) (string, error) {
	source, err := filepath.Abs(appPath)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(root, filepath.Base(source))
	cmd := exec.Command("ditto", source, dest)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("copy app bundle: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return dest, nil
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

func requireRegularArtifact(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("existing artifact %s is not a regular file", path)
	}
	return nil
}

func compareArtifact(path string, expected [sha256.Size]byte) error {
	if err := requireRegularArtifact(path); err != nil {
		return err
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
