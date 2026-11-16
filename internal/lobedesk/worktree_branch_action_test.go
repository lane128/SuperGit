package lobedesk

import (
	"os/exec"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestRetainedWorktreeBranchActionRejectsLaterTip(t *testing.T) {
	a, _, root := worktreeApp(t)
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v: %s", err, out)
		}
		return string(out)
	}
	run("branch", "topic")
	selected := gitrepo.Branch{Name: "topic", SHA: strings.TrimSpace(run("rev-parse", "refs/heads/topic"))}
	run("commit", "--allow-empty", "-m", "advance")
	run("branch", "-f", "topic", "HEAD")
	before := run("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads")
	done := make(chan func(), 1)
	a.runOnMain = func(fn func()) {
		release := make(chan struct{})
		done <- func() { defer close(release); fn() }
		<-release
	}
	a.deleteBranchAfterWorktree(a.repo, selected)
	runContinuation(t, done)
	if run("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads") != before {
		t.Fatal("retained branch action deleted later branch tip")
	}
	found := false
	for _, n := range a.notices[toastMain] {
		if strings.HasPrefix(n.description, a.t("branchTargetChanged")) {
			found = true
		}
	}
	if !found {
		t.Fatal("retained branch refusal lacks localized recovery")
	}
}

func TestRetainedWorktreeBranchActionSurvivesSameRootRefresh(t *testing.T) {
	a, _, root := worktreeApp(t)
	cmd := exec.Command("git", "-C", root, "branch", "topic")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture branch: %v: %s", err, out)
	}
	original := a.repo
	tip, err := original.ResolveRef(a.ctx, "refs/heads/topic")
	if err != nil {
		t.Fatal(err)
	}
	selected := gitrepo.Branch{Name: "topic", SHA: tip}
	a.repo, err = gitrepo.Open(a.ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan func(), 1)
	a.runOnMain = func(fn func()) {
		release := make(chan struct{})
		done <- func() { defer close(release); fn() }
		<-release
	}
	a.deleteBranchAfterWorktree(original, selected)
	if !a.busy {
		t.Fatal("same-root refresh disabled retained branch action")
	}
	runContinuation(t, done)
	out, err := exec.Command("git", "-C", root, "branch", "--list", "topic").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "" {
		t.Fatalf("original retained branch was not deleted: %q %v", out, err)
	}
}

func TestWorktreeRemovalNoticeKeepsOriginalBranchIdentity(t *testing.T) {
	for _, change := range []string{"original", "moved"} {
		t.Run(change, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			run := func(args ...string) string {
				t.Helper()
				out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
				if err != nil {
					t.Fatalf("fixture Git: %v: %s", err, out)
				}
				return string(out)
			}
			path := t.TempDir() + "/linked"
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
			done := make(chan func(), 1)
			a.runOnMain = func(fn func()) {
				release := make(chan struct{})
				done <- func() { defer close(release); fn() }
				<-release
			}
			a.confirmRemoveWorktree(selected)
			a.confirm.action()
			runContinuation(t, done)
			var retainedAction func()
			for _, n := range a.notices[toastMain] {
				if n.title == a.t("worktreeRemoved") && n.action == a.t("delete") {
					retainedAction = n.onAction
				}
			}
			if retainedAction == nil {
				t.Fatal("successful removal lost retained branch action")
			}
			// Complete the real refresh, which replaces the Repository pointer.
			runContinuation(t, done)
			if change == "moved" {
				run("commit", "--allow-empty", "-m", "advance")
				run("branch", "-f", "topic", "HEAD")
			}
			before := run("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads/topic")
			retainedAction()
			if !a.busy {
				t.Fatal("actual retained action stopped working after refresh")
			}
			for a.busy {
				runContinuation(t, done)
			}
			after := run("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads/topic")
			if change == "moved" && after != before {
				t.Fatal("actual removal notice deleted later branch tip")
			}
			if change == "original" && strings.TrimSpace(after) != "" {
				t.Fatal("actual removal notice failed to delete original retained branch")
			}
		})
	}
}

func TestRetainedWorktreeBranchActionRejectsAnotherRoot(t *testing.T) {
	a, _, _ := worktreeApp(t)
	original := a.repo
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.deleteBranchAfterWorktree(original, gitrepo.Branch{Name: "topic"})
	if a.busy {
		t.Fatal("retained branch action entered another repository")
	}
}
