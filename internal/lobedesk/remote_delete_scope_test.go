package lobedesk

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteDeleteConfirmationKeepsTipAndDestination(t *testing.T) {
	for _, change := range []string{"tip", "remote"} {
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
			run("branch", "delete-topic")
			first, second := filepath.Join(t.TempDir(), "first.git"), filepath.Join(t.TempDir(), "second.git")
			run("clone", "--bare", root, first)
			run("clone", "--bare", root, second)
			run("remote", "add", "origin", first)
			run("fetch", "origin")
			var err error
			a.remotes, err = a.repo.Remotes(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.remoteBranches, err = a.repo.RemoteBranches(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			expected := strings.TrimSpace(run("rev-parse", "refs/remotes/origin/delete-topic"))
			a.confirmDeleteRemoteBranch("origin", "delete-topic", expected)
			if !a.confirm.open || !strings.Contains(a.confirm.message, shortSHA(expected)) {
				t.Fatal("remote delete confirmation does not identify original tip")
			}
			if change == "tip" {
				run("commit", "--allow-empty", "-m", "later")
				run("push", "origin", "HEAD:refs/heads/delete-topic")
				// A later UI refresh must not update the already-open confirmation lease.
				a.remoteBranches, err = a.repo.RemoteBranches(a.ctx)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				run("remote", "set-url", "origin", second)
			}
			refs := func(bare string) string {
				t.Helper()
				out, err := exec.Command("git", "-C", bare, "for-each-ref", "--format=%(refname) %(objectname)").CombinedOutput()
				if err != nil {
					t.Fatal(err)
				}
				return string(out)
			}
			beforeFirst, beforeSecond := refs(first), refs(second)
			done := make(chan func(), 1)
			a.runOnMain = func(fn func()) {
				release := make(chan struct{})
				done <- func() { defer close(release); fn() }
				<-release
			}
			a.confirm.action()
			runContinuation(t, done)
			if refs(first) != beforeFirst || refs(second) != beforeSecond {
				t.Fatal("old confirmation deleted changed remote target")
			}
			key := "remoteDeleteTargetChanged"
			if change == "remote" {
				key = "remoteTargetChanged"
			}
			found := false
			for _, n := range a.notices[toastMain] {
				if strings.HasPrefix(n.description, a.t(key)) {
					if change == "tip" && !strings.Contains(n.description, "git push:") {
						t.Fatal("lease failure lost original Git output")
					}
					found = true
				}
			}
			if !found {
				t.Fatal("remote deletion refusal lacks localized recovery")
			}
		})
	}
}
