// Command releasepack generates Ed25519 update signing keys and packages a
// built SuperGit.app into a signed zip archive plus update.json manifest.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"supergit/internal/updater"
)

var (
	plistShortVerRE = regexp.MustCompile(`<key>CFBundleShortVersionString</key>\s*<string>([^<]*)</string>`)
	plistBuildVerRE = regexp.MustCompile(`<key>CFBundleVersion</key>\s*<string>([^<]*)</string>`)
	plistMinOSRE    = regexp.MustCompile(`<key>LSMinimumSystemVersion</key>\s*<string>([^<]*)</string>`)
)

func main() {
	genKey := flag.Bool("genkey", false, "generate a new Ed25519 signing keypair")
	privPath := flag.String("priv", "", "path to base64-encoded Ed25519 private key")
	pubPath := flag.String("pub", "", "path to write base64-encoded Ed25519 public key (used with -genkey)")
	appPath := flag.String("app", "build/SuperGit.app", "path to the built .app bundle")
	outDir := flag.String("out", "build/release", "output directory for zip and update.json")
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
	if err := updater.VerifyAppBundleMetadata(opts.AppPath, updater.ExpectedBundleID); err != nil {
		return err
	}
	plistBytes, err := os.ReadFile(filepath.Join(opts.AppPath, "Contents", "Info.plist"))
	if err != nil {
		return err
	}
	pubBase64 := base64.StdEncoding.EncodeToString(pub)
	embeddedPubKey, err := extractEmbeddedUpdatePublicKey(plistBytes)
	if err != nil {
		return fmt.Errorf("embedded update public key: %w", err)
	}
	if embeddedPubKey == "" {
		return fmt.Errorf("embedded update public key is missing or empty in Info.plist")
	}
	if embeddedPubKey != pubBase64 {
		return fmt.Errorf("embedded update public key in Info.plist does not match the signing key")
	}
	shortVer := extractPlistValue(plistBytes, plistShortVerRE)
	if _, err := updater.ParseVersion(shortVer); err != nil {
		return fmt.Errorf("invalid CFBundleShortVersionString %q: %w", shortVer, err)
	}
	shortVer = updater.NormalizeVersion(shortVer)
	buildNum, _ := strconv.Atoi(extractPlistValue(plistBytes, plistBuildVerRE))
	minOS := extractPlistValue(plistBytes, plistMinOSRE)

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}

	archiveName := fmt.Sprintf("SuperGit-%s-%s.zip", shortVer, opts.Platform)
	archivePath := filepath.Join(opts.OutDir, archiveName)
	_ = os.Remove(archivePath)

	if err := archiveAppBundle(opts.AppPath, archivePath); err != nil {
		return err
	}
	if _, err := updater.InspectZipSafety(archivePath); err != nil {
		return fmt.Errorf("generated archive failed safety check: %w", err)
	}

	sum, size, err := fileSHA256(archivePath)
	if err != nil {
		return err
	}
	sumHex := hex.EncodeToString(sum[:])
	sigB64 := updater.SignDigest(priv, sum)
	if err := updater.VerifyArchiveDigest(sum, sumHex, sigB64, pub); err != nil {
		return fmt.Errorf("self-verification failed: %w", err)
	}

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
		if json.Unmarshal(existing, &prev) == nil && updater.NormalizeVersion(prev.Version) == shortVer {
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
	if err := os.WriteFile(manifestPath, append(encoded, '\n'), 0o644); err != nil {
		return err
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
		return nil, nil, err
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

func extractEmbeddedUpdatePublicKey(data []byte) (string, error) {
	// Collect only direct children of the outer plist dictionary. Nested
	// dictionaries and comments must not supply the installed app's key.
	var plist struct {
		XMLName xml.Name `xml:"plist"`
		Values  []struct {
			XMLName xml.Name
			Entries []struct {
				XMLName  xml.Name
				Text     string     `xml:",chardata"`
				Children []struct{} `xml:",any"`
			} `xml:",any"`
		} `xml:",any"`
	}
	if err := xml.Unmarshal(data, &plist); err != nil || plist.XMLName != (xml.Name{Local: "plist"}) ||
		len(plist.Values) != 1 || plist.Values[0].XMLName != (xml.Name{Local: "dict"}) {
		return "", errors.New("invalid top-level dictionary in Info.plist")
	}
	entries := plist.Values[0].Entries
	if len(entries)%2 != 0 {
		return "", errors.New("invalid key/value pairs in Info.plist")
	}
	var embedded string
	found := false
	for i := 0; i < len(entries); i += 2 {
		key, value := entries[i], entries[i+1]
		if key.XMLName != (xml.Name{Local: "key"}) || len(key.Children) != 0 || value.XMLName.Local == "key" {
			return "", errors.New("invalid key/value pairs in Info.plist")
		}
		if key.Text != "SuperGitUpdatePublicKey" {
			continue
		}
		if found || value.XMLName != (xml.Name{Local: "string"}) || len(value.Children) != 0 {
			return "", errors.New("expected one string entry in Info.plist")
		}
		embedded = strings.TrimSpace(value.Text)
		found = true
	}
	return embedded, nil
}

func extractPlistValue(data []byte, re *regexp.Regexp) string {
	m := re.FindSubmatch(data)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(string(m[1]))
}
