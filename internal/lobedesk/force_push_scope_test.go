package lobedesk

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestForcePushDoesNotTreatFailedReadAsAbsent(t *testing.T) {
	a, _, _ := worktreeApp(t)
	a.remotes = []gitrepo.Remote{{Name: "origin"}}
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.remoteBranchesErr = "fixture remote reference read failed"
	a.confirmForcePush()
	if a.confirm.open || a.busy {
		t.Fatal("failed reference read authorized an absent-target force push")
	}
}

func TestForcePushConfirmationKeepsSourceRemoteAndLease(t *testing.T) {
	for _, change := range []string{"source", "remote", "remote-tip-and-fetch"} {
		t.Run(change, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			runAt := func(dir string, args ...string) string {
				t.Helper()
				out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
				if err != nil {
					t.Fatalf("fixture Git: %v: %s", err, out)
				}
				return string(out)
			}
			base := strings.TrimSpace(runAt(root, "rev-parse", "HEAD"))
			runAt(root, "commit", "--allow-empty", "-m", "remote baseline")
			lease := strings.TrimSpace(runAt(root, "rev-parse", "HEAD"))
			first, second := filepath.Join(t.TempDir(), "first.git"), filepath.Join(t.TempDir(), "second.git")
			runAt(root, "clone", "--bare", root, first)
			runAt(root, "clone", "--bare", root, second)
			runAt(root, "remote", "add", "origin", first)
			runAt(root, "fetch", "origin")
			runAt(root, "reset", "--hard", base)
			runAt(root, "branch", "--set-upstream-to=origin/main", "main")
			var err error
			a.remotes, err = a.repo.Remotes(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.remoteBranches, err = a.repo.RemoteBranches(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.status, err = a.repo.Status(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.confirmForcePush()
			if !a.confirm.open || !strings.Contains(a.confirm.message, shortSHA(base)) || !strings.Contains(a.confirm.message, shortSHA(lease)) {
				t.Fatal("force push confirmation does not identify both commits")
			}
			switch change {
			case "source":
				runAt(root, "commit", "--allow-empty", "-m", "later local")
			case "remote":
				runAt(root, "remote", "set-url", "origin", second)
			case "remote-tip-and-fetch":
				peer := filepath.Join(t.TempDir(), "peer")
				runAt(root, "clone", first, peer)
				runAt(peer, "config", "user.name", "Fixture")
				runAt(peer, "config", "user.email", "fixture@example.invalid")
				runAt(peer, "commit", "--allow-empty", "-m", "later remote")
				runAt(peer, "push", "origin", "main")
				runAt(root, "fetch", "origin")
				a.remoteBranches, err = a.repo.RemoteBranches(a.ctx)
				if err != nil {
					t.Fatal(err)
				}
			}
			firstRefs := runAt(first, "for-each-ref", "--format=%(refname) %(objectname)")
			secondRefs := runAt(second, "for-each-ref", "--format=%(refname) %(objectname)")
			done := make(chan func(), 1)
			a.runOnMain = func(fn func()) {
				release := make(chan struct{})
				done <- func() { defer close(release); fn() }
				<-release
			}
			a.confirm.action()
			runContinuation(t, done)
			if runAt(first, "for-each-ref", "--format=%(refname) %(objectname)") != firstRefs || runAt(second, "for-each-ref", "--format=%(refname) %(objectname)") != secondRefs {
				t.Fatal("old force confirmation changed remote refs")
			}
			key := "forcePushTargetChanged"
			if change == "source" {
				key = "pushTargetChanged"
			}
			if change == "remote" {
				key = "remoteTargetChanged"
			}
			found := false
			for _, n := range a.notices[toastMain] {
				if strings.HasPrefix(n.description, a.t(key)) {
					if change == "remote-tip-and-fetch" && !strings.Contains(n.description, "git push:") {
						t.Fatal("lease failure lost Git output")
					}
					found = true
				}
			}
			if !found {
				t.Fatal("force push refusal lacks localized recovery")
			}
		})
	}
}
