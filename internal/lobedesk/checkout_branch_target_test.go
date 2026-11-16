package lobedesk

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCommitBranchPickerRejectsMovedTipWithoutUIRefresh(t *testing.T) {
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
	run("commit", "--allow-empty", "-m", "advance")
	newer := strings.TrimSpace(run("rev-parse", "HEAD"))
	a.status.Head = newer
	a.branches = []gitrepo.Branch{{Name: "topic", SHA: original}}
	a.openCheckoutBranchDialog(gitrepo.Commit{SHA: original, Subject: "original"})
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.checkoutBranchDialog(c) }, 640, 480)
	run("branch", "-f", "topic", newer)
	beforeHead, beforeIndex := run("symbolic-ref", "HEAD"), run("ls-files", "--stage")
	beforeBytes, err := os.ReadFile(filepath.Join(root, "base.txt"))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan func(), 1)
	a.runOnMain = func(fn func()) {
		release := make(chan struct{})
		done <- func() { defer close(release); fn() }
		<-release
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	runContinuation(t, done)
	if run("symbolic-ref", "HEAD") != beforeHead || run("rev-parse", "HEAD") != newer+"\n" {
		t.Fatal("commit picker switched to moved branch after old selection")
	}
	if run("ls-files", "--stage") != beforeIndex {
		t.Fatal("refused switch changed Index")
	}
	afterBytes, err := os.ReadFile(filepath.Join(root, "base.txt"))
	if err != nil || string(afterBytes) != string(beforeBytes) {
		t.Fatal("refused switch changed working bytes")
	}
	found := false
	for _, n := range a.notices[toastMain] {
		if strings.HasPrefix(n.description, a.t("switchTargetChanged")) {
			found = true
		}
	}
	if !found {
		t.Fatal("changed branch refusal lacks localized recovery")
	}
}

func TestCommitBranchSwitchRefusalKeepsLanguageAndGitOutput(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		a := &app{locale: language}
		if got := a.operationErrorDescription(gitrepo.ErrSwitchTargetChanged); got != a.t("switchTargetChanged") {
			t.Fatalf("%s: no localized switch recovery: %q", language, got)
		}
		cause := errors.New("git rev-parse: missing reference")
		got := a.operationErrorDescription(errors.Join(gitrepo.ErrSwitchTargetChanged, cause))
		if !strings.HasPrefix(got, a.t("switchTargetChanged")) || !strings.Contains(got, cause.Error()) {
			t.Fatalf("%s: lost recovery or original output: %q", language, got)
		}
	}
}
