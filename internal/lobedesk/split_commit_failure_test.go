package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRejectedCommitKeepsDraftIndexAndFailure(t *testing.T) {
	for _, amend := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "amend"}[amend], func(t *testing.T) {
			testRejectedCommitRecovery(t, amend)
		})
	}
}

func testRejectedCommitRecovery(t *testing.T, amend bool) {
	t.Helper()
	a, _, root := worktreeApp(t)
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	t.Cleanup(func() {
		a.resetHistoryTraversal()
		a.resetFileHistory()
		if a.overviewCancel != nil {
			a.overviewCancel()
		}
	})
	a.page, a.locale, a.worktreeSelected = "changes", localeEnglish, true
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("first part\n"), 0644); err != nil {
		t.Fatal(err)
	}
	splitCommitGit(t, root, "add", "--", "base.txt")
	head := splitCommitGit(t, root, "rev-parse", "HEAD")
	index := splitCommitGit(t, root, "ls-files", "--stage")
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nprintf 'owned hook rejected commit\\n' >&2\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.commitIdentity = readCommitIdentity(a.ctx, a.repo, true)
	const draft = "First logical change\n\nRetain this explanation."
	a.commitText, a.amend = draft, amend
	a.commit()
	var completion func()
	select {
	case completion = <-callbacks:
	case <-time.After(5 * time.Second):
		t.Fatal("commit completion missing")
	}
	// Git already returned a failure. A late cancel or worker cleanup before
	// the UI drains this continuation must not relabel that result as canceled.
	a.cancelOperation()
	completion()
	if a.commitText != draft || a.amend != amend || splitCommitGit(t, root, "rev-parse", "HEAD") != head || splitCommitGit(t, root, "ls-files", "--stage") != index {
		t.Fatal("failed commit changed draft, HEAD or Index")
	}
	found := false
	for _, notice := range a.notices[toastMain] {
		if notice.kind == "error" && notice.title == a.t("commitFailed") && strings.Contains(notice.description, "owned hook rejected commit") {
			found = true
		}
	}
	if !found {
		t.Fatalf("Git failure was lost or reclassified: %+v", a.notices[toastMain])
	}
	for a.loading || a.diffLoading {
		runContinuation(t, callbacks)
	}
	// Correct the owned fixture's rejecting hook, then retry the retained draft.
	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	a.commit()
	if !a.busy {
		t.Fatalf("retry blocked: %s", a.commitDisabledReason())
	}
	for a.busy || a.loading || a.diffLoading {
		runContinuation(t, callbacks)
	}
	if a.commitText != "" || a.amend {
		t.Fatal("successful retry did not clear draft and Amend")
	}
	if got := splitCommitGit(t, root, "log", "-1", "--format=%B"); got != draft+"\n\n" {
		t.Fatalf("retry message = %q", got)
	}
	if got := splitCommitGit(t, root, "show", "HEAD:base.txt"); got != "first part\n" {
		t.Fatalf("retry tree = %q", got)
	}
	wantCount := "2\n"
	if amend {
		wantCount = "1\n"
	}
	if got := splitCommitGit(t, root, "rev-list", "--count", "HEAD"); got != wantCount {
		t.Fatalf("retry commit count = %q", got)
	}
}
