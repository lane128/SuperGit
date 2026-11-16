package lobedesk

import (
	"os/exec"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestBranchRenamePromptKeepsOriginalTip(t *testing.T) {
	a, _, root := worktreeApp(t)
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v: %s", err, out)
		}
		return string(out)
	}
	original := strings.TrimSpace(run("rev-parse", "HEAD"))
	run("branch", "topic")
	a.branches = []gitrepo.Branch{{Name: "topic", SHA: original}}
	a.promptRenameBranch(a.branches[0])
	if !a.prompt.open {
		t.Fatal("rename input missing")
	}
	run("commit", "--allow-empty", "-m", "advance")
	run("branch", "-f", "topic", "HEAD")
	before := run("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads")
	done := make(chan func(), 1)
	a.runOnMain = func(fn func()) {
		release := make(chan struct{})
		done <- func() { defer close(release); fn() }
		<-release
	}
	a.prompt.action("renamed")
	runContinuation(t, done)
	if after := run("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads"); after != before {
		t.Fatal("old rename input renamed a replacement branch tip")
	}
	found := false
	for _, n := range a.notices[toastMain] {
		if strings.HasPrefix(n.description, a.t("renameTargetChanged")) {
			found = true
		}
	}
	if !found {
		t.Fatal("rename refusal lacks localized recovery")
	}
}
