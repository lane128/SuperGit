package updater

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type plistEntry struct {
	XMLName xml.Name
	Text    string `xml:",chardata"`
	Raw     string `xml:",innerxml"`
}

// AppPlist holds direct entries of the bundle's Info.plist dictionary.
type AppPlist struct {
	entries map[string]plistEntry
}

func ReadAppPlist(appPath string) (AppPlist, error) {
	data, err := os.ReadFile(filepath.Join(appPath, "Contents", "Info.plist"))
	if err != nil {
		return AppPlist{}, err
	}
	var plist struct {
		XMLName xml.Name `xml:"plist"`
		Values  []struct {
			XMLName xml.Name
			Entries []plistEntry `xml:",any"`
		} `xml:",any"`
	}
	if err := xml.Unmarshal(data, &plist); err != nil {
		return AppPlist{}, fmt.Errorf("decode Info.plist: %w", err)
	}
	if plist.XMLName != (xml.Name{Local: "plist"}) || len(plist.Values) != 1 || plist.Values[0].XMLName != (xml.Name{Local: "dict"}) {
		return AppPlist{}, errors.New("invalid top-level dictionary in Info.plist")
	}
	entries := plist.Values[0].Entries
	if len(entries)%2 != 0 {
		return AppPlist{}, errors.New("invalid key/value pairs in Info.plist")
	}
	result := AppPlist{entries: make(map[string]plistEntry, len(entries)/2)}
	for i := 0; i < len(entries); i += 2 {
		key, value := entries[i], entries[i+1]
		if key.XMLName != (xml.Name{Local: "key"}) || strings.Contains(key.Raw, "<") || value.XMLName.Local == "key" {
			return AppPlist{}, errors.New("invalid key/value pairs in Info.plist")
		}
		if _, exists := result.entries[key.Text]; exists {
			return AppPlist{}, fmt.Errorf("duplicate Info.plist key %q", key.Text)
		}
		result.entries[key.Text] = value
	}
	return result, nil
}

// String reads one direct string value. Missing keys return an empty string.
func (p AppPlist) String(key string) (string, error) {
	value, err := p.exactString(key)
	return strings.TrimSpace(value), err
}

func (p AppPlist) exactString(key string) (string, error) {
	value, ok := p.entries[key]
	if !ok {
		return "", nil
	}
	if value.XMLName != (xml.Name{Local: "string"}) || strings.Contains(value.Raw, "<") {
		return "", fmt.Errorf("Info.plist %s must be a plain string", key)
	}
	return value.Text, nil
}

// ExecutablePath resolves a regular bundle executable named by the plist.
func (p AppPlist) ExecutablePath(appPath string) (string, error) {
	name, err := p.exactString("CFBundleExecutable")
	if err != nil {
		return "", err
	}
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return "", fmt.Errorf("invalid CFBundleExecutable %q", name)
	}
	exe := filepath.Join(appPath, "Contents", "MacOS", name)
	info, err := os.Stat(exe)
	if err != nil {
		return "", fmt.Errorf("cannot stat executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("bundle executable %q is not a regular file", name)
	}
	return exe, nil
}

var bundleVersionPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,2}$`)

// ReleaseVersion reads and validates the bundle's short version and build.
func (p AppPlist) ReleaseVersion() (string, int, error) {
	version, err := p.exactString("CFBundleShortVersionString")
	if err != nil {
		return "", 0, err
	}
	if !bundleVersionPattern.MatchString(version) {
		return "", 0, fmt.Errorf("invalid CFBundleShortVersionString %q", version)
	}
	if _, err := ParseVersion(version); err != nil {
		return "", 0, fmt.Errorf("invalid CFBundleShortVersionString: %w", err)
	}
	buildText, err := p.exactString("CFBundleVersion")
	if err != nil {
		return "", 0, err
	}
	build, err := strconv.Atoi(buildText)
	if err != nil || build < 0 {
		return "", 0, fmt.Errorf("invalid CFBundleVersion %q", buildText)
	}
	return version, build, nil
}

// VerifyBundleID checks the identifier declared in the bundle metadata.
func (p AppPlist) VerifyBundleID(expected string) error {
	id, err := p.String("CFBundleIdentifier")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBundleMismatch, err)
	}
	if id != expected {
		return ErrBundleMismatch
	}
	return nil
}
