package lobedesk

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// appVersion and appBuild can be set at link time via -ldflags. When left
// empty, bundleVersionInfo reads the enclosing macOS .app Info.plist and falls
// back to defaultAppVersion for local CLI runs.
var (
	appVersion = ""
	appBuild   = ""
)

const defaultAppVersion = "0.1.0"

var (
	plistShortVersionRE = regexp.MustCompile(`<key>CFBundleShortVersionString</key>\s*<string>([^<]*)</string>`)
	plistBuildVersionRE = regexp.MustCompile(`<key>CFBundleVersion</key>\s*<string>([^<]*)</string>`)
)

func parseBundleVersion(plist []byte) (short, build string) {
	if m := plistShortVersionRE.FindSubmatch(plist); m != nil {
		short = strings.TrimSpace(string(m[1]))
	}
	if m := plistBuildVersionRE.FindSubmatch(plist); m != nil {
		build = strings.TrimSpace(string(m[1]))
	}
	return short, build
}

func formatVersionLabel(short, build string) string {
	short = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(short), "v"))
	build = strings.TrimSpace(build)
	if short == "" {
		short = defaultAppVersion
	}
	if build != "" {
		return fmt.Sprintf("v%s (%s)", short, build)
	}
	return "v" + short
}

func currentAppVersionLabel() string {
	short := strings.TrimSpace(appVersion)
	build := strings.TrimSpace(appBuild)
	if short == "" || build == "" {
		if exe, err := os.Executable(); err == nil {
			plistPath := filepath.Join(filepath.Dir(exe), "..", "Info.plist")
			if data, err := os.ReadFile(plistPath); err == nil {
				plistShort, plistBuild := parseBundleVersion(data)
				if short == "" {
					short = plistShort
				}
				if build == "" {
					build = plistBuild
				}
			}
		}
	}
	return formatVersionLabel(short, build)
}
