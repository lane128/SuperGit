package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain points the settings and preferences files at a throwaway home for
// the whole package. Several tests click a control that saves a choice, and a
// test run must not write into the reader's real ~/Library/Application Support.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "supergit-test-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	// Refuse to run against the reader's real config: this is the check that
	// the two settings above actually moved it.
	if dir, err := os.UserConfigDir(); err != nil || !strings.HasPrefix(dir, home) {
		panic("the test home is not in effect; the settings file would be written to " + dir)
	}
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
