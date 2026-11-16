package lobedesk

import (
	"os/exec"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestBranchDeleteConfirmationKeepsOriginalTip(t *testing.T) {
	for _, force := range []bool{false, true} {
		label := "normal"
		if force {
			label = "force"
		}
		t.Run(label, func(t *testing.T) {
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
			run("commit", "--allow-empty", "-m", "advance main")
			branch := gitrepo.Branch{Name: "topic", SHA: original}
			if force {
				a.confirmForceDeleteBranch(branch)
			} else {
				a.confirmDeleteBranch(branch)
			}
			if !a.confirm.open {
				t.Fatal("delete confirmation missing")
			}
			run("branch", "-f", "topic", "HEAD")
			before := run("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads")
			done := make(chan func(), 1)
			a.runOnMain = func(fn func()) {
				release := make(chan struct{})
				done <- func() { defer close(release); fn() }
				<-release
			}
			a.confirm.action()
			runContinuation(t, done)
			after := run("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads")
			if after != before {
				t.Fatal("old confirmation deleted a branch whose tip changed")
			}
			if !strings.Contains(a.confirm.message, shortSHA(original)) {
				t.Fatal("confirmation does not identify selected branch SHA")
			}
		})
	}
}

func TestBranchDeleteChangedTargetHasLocalizedRecovery(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		a := &app{locale: language}
		if got := a.operationErrorDescription(gitrepo.ErrBranchTargetChanged); got != a.t("branchTargetChanged") {
			t.Fatalf("%s: changed branch lacks localized recovery: %q", language, got)
		}
	}
}
