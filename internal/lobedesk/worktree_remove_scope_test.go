package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestWorktreeRemoveConfirmationKeepsOriginalTarget(t *testing.T) {
	a, _, root := worktreeApp(t)
	path := filepath.Join(t.TempDir(), "linked")
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v: %s", err, out)
		}
		return string(out)
	}
	run("worktree", "add", "-b", "topic", path, "HEAD")
	list, err := a.repo.Worktrees(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	var selected gitrepo.Worktree
	for _, w := range list {
		if w.Branch == "topic" {
			selected = w
		}
	}
	if selected.HEAD == "" {
		t.Fatal("fixture worktree missing")
	}
	a.confirmRemoveWorktree(selected)
	if !a.confirm.open {
		t.Fatal("Worktree remove confirmation missing")
	}
	run("worktree", "remove", path)
	run("worktree", "add", "-b", "replacement", path, "HEAD")
	before := run("worktree", "list", "--porcelain", "-z")
	done := make(chan func(), 1)
	a.runOnMain = func(fn func()) {
		release := make(chan struct{})
		done <- func() { defer close(release); fn() }
		<-release
	}
	a.confirm.action()
	runContinuation(t, done)
	if run("worktree", "list", "--porcelain", "-z") != before {
		t.Fatal("old confirmation removed replacement worktree")
	}
	if b, err := os.ReadFile(filepath.Join(path, "base.txt")); err != nil || string(b) != "base\n" {
		t.Fatal("old confirmation removed replacement files")
	}
	found := false
	for _, n := range a.notices[toastMain] {
		if strings.HasPrefix(n.description, a.t("worktreeTargetChanged")) {
			found = true
		}
	}
	if !found {
		t.Fatal("changed Worktree refusal lacks localized recovery")
	}
}
