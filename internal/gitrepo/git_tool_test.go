package gitrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The macOS /usr/bin/git launcher can create xcrun_db in TMPDIR and emit
// Xcode diagnostics before Git starts, especially under a restricted sandbox.
// Temp-ownership assertions must run Git itself, not measure launcher caches.
func useDirectGitForTempAssertions(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if gitPath != "/usr/bin/git" {
		return
	}
	out, err := exec.Command("/usr/bin/xcrun", "--find", "git").Output()
	if err != nil {
		t.Fatalf("resolve developer Git: %v", err)
	}
	direct := strings.TrimSpace(string(out))
	if direct == gitPath || !filepath.IsAbs(direct) {
		t.Fatalf("expected developer Git, got %q", direct)
	}
	if info, err := os.Stat(direct); err != nil || info.IsDir() || info.Mode()&0111 == 0 {
		t.Fatalf("developer Git is not executable: %q (%v)", direct, err)
	}
	t.Setenv("PATH", filepath.Dir(direct)+string(os.PathListSeparator)+os.Getenv("PATH"))
}
