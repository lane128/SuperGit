package lobedesk

import (
	"os/exec"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestTagDeleteConfirmationKeepsOriginalObject(t *testing.T) {
	a, _, root := worktreeApp(t)
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v: %s", err, out)
		}
		return string(out)
	}
	run("tag", "-a", "--no-sign", "-m", "original annotation", "--", "release")
	tags, err := a.repo.Tags(a.ctx)
	if err != nil || len(tags) != 1 {
		t.Fatalf("fixture tag missing: %v", err)
	}
	selected := tags[0]
	if selected.ObjectSHA == selected.TargetSHA {
		t.Fatal("fixture lacks distinct annotated object")
	}
	a.confirmDeleteTag(selected)
	if !a.confirm.open {
		t.Fatal("tag delete confirmation missing")
	}
	run("tag", "-f", "-a", "--no-sign", "-m", "replacement annotation", "--", "release")
	before := run("for-each-ref", "--format=%(refname) %(objectname)", "refs/tags")
	done := make(chan func(), 1)
	a.runOnMain = func(fn func()) {
		release := make(chan struct{})
		done <- func() { defer close(release); fn() }
		<-release
	}
	a.confirm.action()
	runContinuation(t, done)
	if after := run("for-each-ref", "--format=%(refname) %(objectname)", "refs/tags"); after != before {
		t.Fatal("old confirmation deleted replacement annotated tag at same commit")
	}
	if !strings.Contains(a.confirm.message, shortSHA(selected.ObjectSHA)) {
		t.Fatal("confirmation does not identify original tag object")
	}
	found := false
	for _, n := range a.notices[toastMain] {
		if strings.HasPrefix(n.description, a.t("tagTargetChanged")) && strings.Contains(n.description, "git update-ref:") {
			found = true
		}
	}
	if !found {
		t.Fatal("changed tag refusal lost localized guidance or original Git output")
	}
}

func TestTagDeleteRefusalUsesCurrentLanguage(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		a := &app{locale: language}
		if got := a.operationErrorDescription(gitrepo.ErrTagTargetChanged); got != a.t("tagTargetChanged") {
			t.Fatalf("%s: no localized tag recovery: %q", language, got)
		}
	}
}
