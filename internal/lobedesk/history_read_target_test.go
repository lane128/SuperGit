package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"os"
	"os/exec"
	"strings"
	"supergit/internal/gitrepo"
	"testing"
	"time"
)

func TestHistoricalMessageFailureKeepsPatchAndRetryTarget(t *testing.T) {
	a, callbacks := overviewFixture(t)
	git := func(input string, args ...string) string {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = a.repo.Root
		command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
		command.Stdin = strings.NewReader(input)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v: %s", err, output)
		}
		return strings.TrimSpace(string(output))
	}
	tree := git("", "rev-parse", "HEAD^{tree}")
	sha := git("oversized message\n\n"+strings.Repeat("x", (1<<20)+1), "commit-tree", tree)
	a.worktreeSelected = false
	a.selectedSHA = sha
	a.commits = []gitrepo.Commit{{SHA: sha, Subject: "oversized message"}}
	finish := func() {
		t.Helper()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		for a.commitDiffLoading {
			select {
			case callback := <-callbacks:
				callback()
			case <-timer.C:
				t.Fatal("history read did not finish")
			}
		}
	}
	a.loadCommitDetail(true)
	finish()
	if !a.commitFilesLoaded || a.commitDiffErr != "" || a.commitDiff.Text == "" {
		t.Fatal("message failure erased valid file/patch data")
	}
	tt := pageTester(a, a.historyDiffPane)
	if _, ok := tt.Find("提交说明超过 1 MiB 显示上限"); !ok {
		t.Fatal("message read error was silently hidden")
	}
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("retryCommitMessage"))
	if !ok {
		t.Fatal("failed message has no retry")
	}
	seq := a.commitSeq
	tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	if a.commitSeq != seq+1 {
		t.Fatal("retry did not start a read")
	}
	finish()
	if a.selectedSHA != sha || a.commitDiffErr != "" || a.commitMessageErr == "" {
		t.Fatal("retry changed immutable target or broke patch")
	}
}

func TestHistoricalDetailReadKeepsItsTarget(t *testing.T) {
	for _, change := range []string{"unchanged", "root", "closed repository", "SHA", "generation", "worktree"} {
		t.Run(change, func(t *testing.T) {
			a, callbacks := overviewFixture(t)
			commits, err := a.repo.History(a.ctx, 1)
			if err != nil || len(commits) != 1 {
				t.Fatalf("fixture history: %v", err)
			}
			a.worktreeSelected = false
			a.selectedSHA = commits[0].SHA
			a.commitMessage = "sentinel"
			a.commitSig = gitrepo.Signature{Signer: "sentinel"}
			a.loadCommitDetail(true)
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "closed repository":
				a.repo = nil
			case "SHA":
				a.selectedSHA = "other"
			case "generation":
				a.commitSeq++
			case "worktree":
				a.worktreeSelected = true
			}
			timer := time.NewTimer(3 * time.Second)
			defer timer.Stop()
			for range 2 {
				select {
				case callback := <-callbacks:
					callback()
				case <-timer.C:
					t.Fatal("historical reads did not dispatch through the UI completion seam")
				}
			}
			if change == "unchanged" {
				if !a.commitFilesLoaded || a.commitMessage == "sentinel" {
					t.Fatal("current history result was not applied")
				}
			} else if a.commitFilesLoaded || a.commitMessage != "sentinel" || a.commitSig.Signer != "sentinel" {
				t.Fatal("obsolete history result replaced the current context")
			}
		})
	}
}
