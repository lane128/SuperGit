package lobedesk

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSubmoduleUpdateMenuKeepsDisplayedIndexTarget(t *testing.T) {
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	a, _, parent := worktreeApp(t)
	_, _, child := worktreeApp(t)
	run := func(dir string, args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v: %s", err, out)
		}
		return string(out)
	}
	base := strings.TrimSpace(run(child, "rev-parse", "HEAD"))
	run(child, "commit", "--allow-empty", "-m", "later child")
	later := strings.TrimSpace(run(child, "rev-parse", "HEAD"))
	run(parent, "-c", "protocol.file.allow=always", "submodule", "add", child, "module")
	run(parent, "commit", "-am", "module")
	module := filepath.Join(parent, "module")
	run(module, "checkout", "--detach", base)
	run(parent, "update-index", "--cacheinfo", "160000,"+base+",module")
	var err error
	a.submodules, err = a.repo.Submodules(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	tt := worktreeRowsTester(a)
	row, ok := tt.AccessByRole(ui.RoleOption, "module")
	if !ok {
		t.Fatal("module row missing")
	}
	tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	item, ok := tt.Find(a.t("updateSubmodule"))
	if !ok {
		t.Fatal("module update action missing")
	}
	run(parent, "update-index", "--cacheinfo", "160000,"+later+",module")
	index := run(parent, "ls-files", "--stage")
	done := make(chan func(), 1)
	a.runOnMain = func(fn func()) {
		release := make(chan struct{})
		done <- func() { defer close(release); fn() }
		<-release
	}
	tt.ClickAt(item.X+item.W/2, item.Y+item.H/2)
	runContinuation(t, done)
	if strings.TrimSpace(run(module, "rev-parse", "HEAD")) != base || run(parent, "ls-files", "--stage") != index {
		t.Fatal("stale module menu updated later Index target")
	}
	found := false
	for _, n := range a.notices[toastMain] {
		if strings.HasPrefix(n.description, a.t("submoduleTargetChanged")) {
			found = true
		}
	}
	if !found {
		t.Fatal("submodule refusal lacks localized recovery")
	}
}
