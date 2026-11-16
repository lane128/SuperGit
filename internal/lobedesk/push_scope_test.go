package lobedesk

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestPushConfirmationKeepsSourceAndRemote(t *testing.T) {
	for _, mode := range []string{"current", "branch"} {
		for _, change := range []string{"source", "remote"} {
			t.Run(mode+"/"+change, func(t *testing.T) {
				a, _, root := worktreeApp(t)
				run := func(args ...string) string {
					t.Helper()
					out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
					if err != nil {
						t.Fatalf("fixture Git: %v: %s", err, out)
					}
					return string(out)
				}
				first, second := filepath.Join(t.TempDir(), "first.git"), filepath.Join(t.TempDir(), "second.git")
				run("init", "--bare", first)
				run("init", "--bare", second)
				run("remote", "add", "origin", first)
				run("branch", "topic")
				var err error
				a.remotes, err = a.repo.Remotes(a.ctx)
				if err != nil {
					t.Fatal(err)
				}
				a.status, err = a.repo.Status(a.ctx)
				if err != nil {
					t.Fatal(err)
				}
				selected := gitrepo.Branch{Name: "topic", SHA: a.status.Head}
				if mode == "current" {
					a.confirmPush()
				} else {
					a.confirmPushBranch(selected)
				}
				if !a.confirm.open || !strings.Contains(a.confirm.message, shortSHA(selected.SHA)) {
					t.Fatal("push confirmation does not identify original commit")
				}
				if change == "source" {
					run("commit", "--allow-empty", "-m", "later")
					if mode == "branch" {
						run("branch", "-f", "topic", "HEAD")
					}
					a.status, err = a.repo.Status(a.ctx)
					if err != nil {
						t.Fatal(err)
					}
				} else {
					run("remote", "set-url", "origin", second)
				}
				done := make(chan func(), 1)
				a.runOnMain = func(fn func()) {
					release := make(chan struct{})
					done <- func() { defer close(release); fn() }
					<-release
				}
				a.confirm.action()
				runContinuation(t, done)
				for _, bare := range []string{first, second} {
					out, err := exec.Command("git", "-C", bare, "for-each-ref", "--format=%(refname)").CombinedOutput()
					if err != nil || strings.TrimSpace(string(out)) != "" {
						t.Fatalf("old push confirmation wrote remote: %s %v", out, err)
					}
				}
				key := "pushTargetChanged"
				if change == "remote" {
					key = "remoteTargetChanged"
				}
				found := false
				for _, n := range a.notices[toastMain] {
					if strings.HasPrefix(n.description, a.t(key)) {
						found = true
					}
				}
				if !found {
					t.Fatal("stale push refusal lacks localized recovery")
				}
			})
		}
	}
}
