package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestHardResetRechecksGitAfterConfirmationWithoutUIRefresh(t *testing.T) {
	a, _, root := worktreeApp(t)
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v: %s", err, out)
		}
		return string(out)
	}
	target := strings.TrimSpace(run("rev-parse", "HEAD"))
	run("commit", "--allow-empty", "-m", "second")
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.openResetDialog(gitrepo.Commit{SHA: target})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.resetDialog(c) }, 640, 480)
	button, ok := tt.Find(a.t("resetHard"))
	if !ok {
		t.Fatal("hard reset choice missing")
	}
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	run("switch", "-c", "other")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("staged bytes\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "--", "base.txt")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("working bytes\n"), 0600); err != nil {
		t.Fatal(err)
	}
	beforeHead, beforeIndex := run("rev-parse", "HEAD"), run("ls-files", "--stage", "-z")
	done := make(chan func(), 1)
	a.runOnMain = func(fn func()) {
		release := make(chan struct{})
		done <- func() { defer close(release); fn() }
		<-release
	}
	a.confirm.action()
	if !a.busy {
		t.Fatal("fixture did not reach operation worker")
	}
	runContinuation(t, done)
	if run("rev-parse", "HEAD") != beforeHead || run("ls-files", "--stage", "-z") != beforeIndex {
		t.Fatal("old Hard Reset confirmation changed new branch HEAD or Index")
	}
	bytes, err := os.ReadFile(filepath.Join(root, "base.txt"))
	if err != nil || string(bytes) != "working bytes\n" {
		t.Fatal("old Hard Reset confirmation discarded new branch bytes")
	}
	found := false
	for _, n := range a.notices[toastMain] {
		if n.description == a.t("resetTargetChanged") {
			found = true
		}
	}
	if !found {
		t.Fatal("worker refusal lacks localized recovery guidance")
	}
}

func TestResetBaselineFailureUsesCurrentLanguage(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		a := &app{locale: language}
		if got := a.operationErrorDescription(gitrepo.ErrResetTargetChanged); got != a.t("resetTargetChanged") {
			t.Fatalf("%s: baseline error lacks localized recovery: %q", language, got)
		}
	}
}

func TestResetFormRejectsChangedHeadOrBranch(t *testing.T) {
	for _, change := range []string{"head", "branch"} {
		t.Run(change, func(t *testing.T) {
			a := &app{repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Head: strings.Repeat("b", 40), Branch: "main"}}
			a.openResetDialog(gitrepo.Commit{SHA: strings.Repeat("a", 40)})
			tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.resetDialog(c) }, 640, 480)
			if change == "head" {
				a.status.Head = strings.Repeat("c", 40)
			} else {
				a.status.Branch = "other"
			}
			tt.Frame()
			if a.reset.open {
				t.Fatal("reset still offers modes against changed current HEAD or branch")
			}
			if len(a.notices[toastMain]) == 0 {
				t.Fatal("changed reset baseline was silently dismissed")
			}
		})
	}
}

func TestHardResetConfirmationKeepsHeadAndBranchBaseline(t *testing.T) {
	a := &app{ctx: t.Context(), runOnMain: func(func()) {}, repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Head: strings.Repeat("b", 40), Branch: "main"}}
	a.openResetDialog(gitrepo.Commit{SHA: strings.Repeat("a", 40)})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.resetDialog(c) }, 640, 480)
	button, ok := tt.Find(a.t("resetHard"))
	if !ok {
		t.Fatal("hard reset mode missing")
	}
	tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
	if !a.confirm.open {
		t.Fatal("hard reset second confirmation missing")
	}
	a.status.Branch = "other"
	a.confirm.action()
	if a.busy {
		t.Fatal("hard reset confirmation forgot original branch baseline")
	}
}
