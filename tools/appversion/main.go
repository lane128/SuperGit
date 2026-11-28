// Command appversion writes the bundle version fields into a copy of the
// packaged Info.plist.
//
// The short version comes from the nearest tag (v1.2.3 → 1.2.3); without a
// usable tag it stays whatever packaging/Info.plist already says, so a
// preview build made outside a tagged checkout keeps its hand-written number.
// The build number is the commit count, which keeps two builds of the same tag
// apart. Both fall back to the plist when git is unavailable or its answer is
// not a legal version, so "make app" works from an exported tree.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// A bundle version is one to three dot-separated integers; Apple rejects
// anything else. A build number is an integer.
var (
	shortVersionPattern = regexp.MustCompile(`^\d+(\.\d+){0,2}$`)
	buildPattern        = regexp.MustCompile(`^\d+$`)
)

func main() {
	in := flag.String("in", "packaging/Info.plist", "the plist to read")
	out := flag.String("out", "", "where to write the versioned plist (required)")
	override := flag.String("version", "", "short bundle version to use in place of the nearest tag")
	repo := flag.String("repo", ".", "the checkout git is asked about")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "appversion: -out is required")
		os.Exit(2)
	}
	if *override != "" && !shortVersionPattern.MatchString(*override) {
		fmt.Fprintln(os.Stderr, "appversion: invalid -version, expected one to three dot-separated integers")
		os.Exit(2)
	}
	src, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "appversion: %v\n", err)
		os.Exit(1)
	}
	shortFallback, ok := plistString(src, "CFBundleShortVersionString")
	if !ok {
		fmt.Fprintf(os.Stderr, "appversion: %s has no CFBundleShortVersionString\n", *in)
		os.Exit(1)
	}
	buildFallback, ok := plistString(src, "CFBundleVersion")
	if !ok {
		fmt.Fprintf(os.Stderr, "appversion: %s has no CFBundleVersion\n", *in)
		os.Exit(1)
	}
	tag, commits := gitVersion(*repo)
	if *override != "" {
		tag = *override
	}
	short, build := versionFields(tag, commits, shortFallback, buildFallback)
	versioned := setPlistString(src, "CFBundleShortVersionString", short)
	versioned = setPlistString(versioned, "CFBundleVersion", build)
	if err := os.WriteFile(*out, versioned, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "appversion: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("SuperGit %s (%s)\n", short, build)
}

// versionFields picks the two bundle fields. The tag and the commit count are
// only used when they are legal: a tag named "nightly" or a count git could
// not answer leaves the plist's own value in place.
func versionFields(tag, commits, fallbackShort, fallbackBuild string) (string, string) {
	short := fallbackShort
	if candidate := strings.TrimPrefix(strings.TrimSpace(tag), "v"); shortVersionPattern.MatchString(candidate) {
		short = candidate
	}
	build := fallbackBuild
	if candidate := strings.TrimSpace(commits); buildPattern.MatchString(candidate) {
		build = candidate
	}
	return short, build
}

// gitVersion reads the nearest tag and the commit count. An empty string means
// git had no answer (no tags, no history, or no repository).
func gitVersion(repo string) (string, string) {
	return gitOutput(repo, "describe", "--tags", "--abbrev=0"),
		gitOutput(repo, "rev-list", "--count", "HEAD")
}

func gitOutput(repo string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// plistString reads a top-level string value from the plist text.
func plistString(src []byte, key string) (string, bool) {
	match := plistEntry(key).FindSubmatch(src)
	if match == nil {
		return "", false
	}
	return string(match[2]), true
}

// setPlistString replaces one top-level string value, leaving the rest of the
// file byte-for-byte intact so the checked-in plist stays readable.
func setPlistString(src []byte, key, value string) []byte {
	return plistEntry(key).ReplaceAll(src, []byte("${1}"+value+"${3}"))
}

func plistEntry(key string) *regexp.Regexp {
	return regexp.MustCompile(`(<key>` + regexp.QuoteMeta(key) + `</key>\s*<string>)([^<]*)(</string>)`)
}
