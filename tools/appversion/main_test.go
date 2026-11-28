package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const fixturePlist = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>SuperGit</string>
    <key>CFBundleShortVersionString</key><string>0.1.0</string>
    <key>CFBundleVersion</key><string>1</string>
</dict>
</plist>
`

// A tag names the version; the commit count names the build. Anything git
// answers that is not a legal bundle version is ignored rather than written
// into the plist, because macOS refuses to launch a bundle whose version is
// not one to three integers.
func TestVersionFieldsFollowTheNearestTag(t *testing.T) {
	tests := []struct {
		name, tag, commits   string
		wantShort, wantBuild string
	}{
		{"tag and commits", "v1.2.3", "42", "1.2.3", "42"},
		{"tag without v", "0.4", "7", "0.4", "7"},
		{"no tag keeps the plist", "", "7", "0.1.0", "7"},
		{"no commits keeps the plist", "v2.0.0", "", "2.0.0", "1"},
		{"a release train name is not a version", "v1.2.3-rc1", "9", "0.1.0", "9"},
		{"a word is not a version", "nightly", "9", "0.1.0", "9"},
		{"four parts are not a version", "v1.2.3.4", "9", "0.1.0", "9"},
		{"a count that is not a number is ignored", "v1.0.0", "many", "1.0.0", "1"},
		{"nothing known", "", "", "0.1.0", "1"},
	}
	for _, test := range tests {
		short, build := versionFields(test.tag, test.commits, "0.1.0", "1")
		if short != test.wantShort || build != test.wantBuild {
			t.Errorf("%s: versionFields(%q, %q) = %q/%q; want %q/%q",
				test.name, test.tag, test.commits, short, build, test.wantShort, test.wantBuild)
		}
	}
}

// The rewrite touches only the two version values: the checked-in plist is
// hand-read, so a reordered or reformatted file would be noise in every diff.
func TestTheRewriteKeepsTheRestOfThePlist(t *testing.T) {
	versioned := setPlistString([]byte(fixturePlist), "CFBundleShortVersionString", "1.2.3")
	versioned = setPlistString(versioned, "CFBundleVersion", "42")
	for _, want := range []string{"<key>CFBundleShortVersionString</key><string>1.2.3</string>", "<key>CFBundleVersion</key><string>42</string>"} {
		if !strings.Contains(string(versioned), want) {
			t.Errorf("the versioned plist lacks %q:\n%s", want, versioned)
		}
	}
	if strings.Contains(string(versioned), "0.1.0") || strings.Contains(string(versioned), "<string>1</string>") {
		t.Errorf("the versioned plist still holds the old values:\n%s", versioned)
	}
	if strings.Count(string(versioned), "\n") != strings.Count(fixturePlist, "\n") {
		t.Errorf("the rewrite changed the number of lines:\n%s", versioned)
	}
	if !strings.Contains(string(versioned), "<key>CFBundleName</key><string>SuperGit</string>") {
		t.Error("the rewrite dropped an unrelated key")
	}
}

// gitVersion reads the nearest tag and the count from a real checkout: the
// fallbacks above are only reachable if these two commands are right.
func TestGitVersionReadsTheCheckout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull,
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %q: %v\n%s", args, err, out)
		}
	}
	git("init", "-b", "main")
	git("commit", "--allow-empty", "-m", "one")
	git("tag", "v1.2.3")
	git("commit", "--allow-empty", "-m", "two")
	tag, commits := gitVersion(dir)
	if tag != "v1.2.3" || commits != "2" {
		t.Errorf("gitVersion = %q/%q; want v1.2.3/2", tag, commits)
	}
	short, build := versionFields(tag, commits, "0.1.0", "1")
	if short != "1.2.3" || build != "2" {
		t.Errorf("versionFields from the checkout = %q/%q; want 1.2.3/2", short, build)
	}
}

func TestAppVersionOverride(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.plist")
	if err := os.WriteFile(in, []byte(fixturePlist), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		version string
		valid   bool
	}{{"2.3.4", true}, {"v2.3.4", false}, {"2.3.4-rc1", false}} {
		t.Run(tc.version, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out.plist")
			cmd := exec.Command("go", "run", ".", "-in", in, "-out", out, "-repo", dir, "-version", tc.version)
			log, err := cmd.CombinedOutput()
			if !tc.valid {
				if err == nil || !strings.Contains(string(log), "invalid -version") {
					t.Fatalf("appversion error = %v, output = %s", err, log)
				}
				if _, err := os.Stat(out); !os.IsNotExist(err) {
					t.Errorf("invalid version wrote output: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("appversion: %v, %s", err, log)
			}
			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			short, _ := plistString(data, "CFBundleShortVersionString")
			build, _ := plistString(data, "CFBundleVersion")
			if short != tc.version || build != "1" {
				t.Fatalf("version/build = %q/%q", short, build)
			}
		})
	}
}
