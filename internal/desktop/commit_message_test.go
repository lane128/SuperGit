package desktop

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func messageApp() *app {
	a := selectionHistoryApp()
	a.ctx = context.Background()
	a.selectedSHA = a.data.history[0].SHA
	a.commitDiff = selectionDiff()
	return a
}

func TestCommitMessageReadCancellationAndIdentity(t *testing.T) {
	a := messageApp()
	first, ctx, ok := a.beginCommitMessage(a.selectedSHA)
	if !ok || !a.commitText.loading {
		t.Fatal("message read did not start")
	}
	defer a.cancelCommitMessageRead()
	second, current, ok := a.beginCommitMessage(a.selectedSHA)
	if !ok || ctx.Err() != context.Canceled || current.Err() != nil || a.finishCommitMessage(first, a.selectedSHA, "old", nil) {
		t.Fatal("superseded message read remained current")
	}
	diff := a.commitDiff
	a.guard.writing = true // Reading this immutable SHA may finish during a write.
	if !a.finishCommitMessage(second, a.selectedSHA, "Title\n\nBody\n", nil) || a.commitText.loading || !a.commitText.loaded || current.Err() != context.Canceled {
		t.Fatal("current immutable read did not finish and release context")
	}
	if !reflect.DeepEqual(a.commitDiff, diff) || a.commitText.message != "Title\n\nBody\n" || a.finishCommitMessage(second, a.selectedSHA, "duplicate", nil) {
		t.Fatal("message mutated Diff or accepted a duplicate callback")
	}
}

func TestCommitMessageRejectsStaleRepoSHAClosedAndBusyStart(t *testing.T) {
	for _, kind := range []string{"repository", "sha", "ticket", "closed"} {
		a := messageApp()
		sha := a.selectedSHA
		ticket, _, _ := a.beginCommitMessage(sha)
		defer a.cancelCommitMessageRead()
		switch kind {
		case "repository":
			a.guard.Switch()
		case "sha":
			a.selectedSHA = a.data.history[1].SHA
		case "ticket":
			a.guard.Begin("commit-message")
		case "closed":
			a.closed = true
		}
		if a.finishCommitMessage(ticket, sha, "old body", nil) || a.commitText.loaded {
			t.Fatalf("accepted stale %s body", kind)
		}
	}
	for _, kind := range []string{"no repo", "write", "open", "different sha", "empty sha"} {
		a := messageApp()
		sha := a.selectedSHA
		switch kind {
		case "no repo":
			a.repo = nil
		case "write":
			a.guard.writing = true
		case "open":
			a.opening = true
		case "different sha":
			sha = a.data.history[1].SHA
		case "empty sha":
			sha = ""
		}
		if _, _, ok := a.beginCommitMessage(sha); ok {
			a.cancelCommitMessageRead()
			t.Fatalf("started message read for %s", kind)
		}
	}
}

func TestCommitMessageErrorRetryAndRepositoryLifetime(t *testing.T) {
	a := messageApp()
	req, _, _ := a.beginCommitMessage(a.selectedSHA)
	defer a.cancelCommitMessageRead()
	failure := errors.New("message could not be read")
	if !a.finishCommitMessage(req, a.selectedSHA, "partial old text", failure) || a.commitText.err != failure || a.commitText.message != "" {
		t.Fatal("failed read retained stale text or lost retry error")
	}
	req, _, ok := a.beginCommitMessage(a.selectedSHA)
	if !ok || a.commitText.err != nil || !a.commitText.loading {
		t.Fatal("current SHA retry unavailable")
	}
	if !a.finishCommitMessage(req, a.selectedSHA, "fresh message", nil) {
		t.Fatal("retry did not finish")
	}
	before := a.commitText
	// Snapshot replacement does not invalidate an immutable selected message.
	a.installSnapshot(a.data)
	if !reflect.DeepEqual(before, a.commitText) {
		t.Fatal("snapshot replaced selected message")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	_, child, _ := a.beginCommitMessage(a.selectedSHA)
	cancel()
	if child.Err() != context.Canceled {
		t.Fatal("message read outlived repository context")
	}
	a.cancelCommitMessageRead()
	if a.commitText.loading {
		t.Fatal("cancel left a permanent loading indicator")
	}
}

func TestCommitMessageParentNavigationRespectsWriteAndOpenGuards(t *testing.T) {
	a := messageApp()
	if !a.canSelectCommit(1) {
		t.Fatal("idle parent navigation blocked")
	}
	a.loading = true
	if !a.canSelectCommit(1) {
		t.Fatal("background snapshot blocked immutable navigation")
	}
	for _, kind := range []string{"writing", "opening", "closed", "no repo"} {
		a := messageApp()
		switch kind {
		case "writing":
			a.guard.writing = true
		case "opening":
			a.opening = true
		case "closed":
			a.closed = true
		case "no repo":
			a.repo = nil
		}
		if a.canSelectCommit(1) {
			t.Fatalf("parent selection allowed during %s", kind)
		}
	}
	for _, index := range []int{-1, len(a.data.history)} {
		if a.canSelectCommit(index) {
			t.Fatal("invalid parent index accepted")
		}
	}
}

func TestCommitMessageDisplayPartsPreserveOriginalLines(t *testing.T) {
	for _, tc := range []struct{ text, title, body string }{
		{"Title\n\nBody\n  indent\n\nTrailer: value\n", "Title", "Body\n  indent\n\nTrailer: value\n"},
		{"First\ncontinued paragraph\n\nBody\n", "First", "continued paragraph\n\nBody\n"},
		{"Title\n", "Title", ""},
		{"", "", ""},
	} {
		s := commitMessageState{sha: "current", message: tc.text, loaded: true}
		title, body := s.parts("current", "folded fallback")
		if title != tc.title || body != tc.body {
			t.Fatalf("display changed message: title=%q body=%q", title, body)
		}
		title, body = s.parts("other", "fallback")
		if title != "fallback" || body != "" {
			t.Fatal("display exposed another commit's message")
		}
	}
}
