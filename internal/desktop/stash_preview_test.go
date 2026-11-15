package desktop

import (
	"context"
	"errors"
	"github.com/egoist/mygo/ui"
	"reflect"
	"supergit/internal/gitrepo"
	"testing"
)

func stashPreviewApp() *app {
	a := messageApp()
	a.stashSHA = "first"
	a.stashDiff = selectionDiff()
	a.stashLoaded = true
	a.message = "draft"
	return a
}

func TestStashPreviewFailureRetryAndSuccessfulEmpty(t *testing.T) {
	a := stashPreviewApp()
	workspace, historical := a.diff, a.commitDiff
	req, ctx, ok := a.beginStashPreview("first")
	if !ok || !a.stashLoading || a.stashLoaded || a.stashErr != nil || a.stashDiff.Text != "" {
		t.Fatal("read did not start cleanly")
	}
	failure := errors.New("too large")
	if !a.finishStashPreview(req, selectionDiff(), failure) || a.stashErr != failure || !a.stashLoaded || a.stashLoading || a.stashDiff.Text != "" || ctx.Err() != context.Canceled || a.stashCancel != nil {
		t.Fatal("failure retained partial data or leaked read")
	}
	if a.finishStashPreview(req, selectionDiff(), nil) {
		t.Fatal("duplicate completion accepted")
	}
	retry, _, ok := a.beginStashPreview(a.stashSHA)
	if !ok || retry.sha != "first" || a.stashErr != nil {
		t.Fatal("retry changed SHA or kept error")
	}
	if !a.finishStashPreview(retry, gitrepo.Diff{}, nil) || !a.stashLoaded || a.stashErr != nil {
		t.Fatal("empty success not recorded")
	}
	if !reflect.DeepEqual(a.diff, workspace) || !reflect.DeepEqual(a.commitDiff, historical) || a.message != "draft" {
		t.Fatal("read modified another preview/draft")
	}
}

func TestStashPreviewReplacementCancelsReadAndDefersListReset(t *testing.T) {
	a := stashPreviewApp()
	a.stashList.ScrollTo(500, ui.Start)
	list := a.stashList
	first, ctx, _ := a.beginStashPreview("first")
	second, current, _ := a.beginStashPreview("second")
	if ctx.Err() != context.Canceled || !a.stashListReset || !reflect.DeepEqual(a.stashList, list) {
		t.Fatal("replacement failed to cancel or destroyed borrowed ListState")
	}
	if a.finishStashPreview(first, selectionDiff(), errors.New("old")) || current.Err() != nil || !a.stashLoading {
		t.Fatal("old result changed current read")
	}
	if !a.finishStashPreview(second, selectionDiff(), nil) || a.stashSHA != "second" || a.stashDiff.Text == "" {
		t.Fatal("current preview failed")
	}
	loaded := a.stashDiff
	a.installSnapshot(snapshot{stashes: []gitrepo.Stash{{SHA: "third", Ref: "stash@{0}"}, {SHA: "second", Ref: "stash@{1}"}}})
	if a.stashSHA != "second" || !reflect.DeepEqual(a.stashDiff, loaded) {
		t.Fatal("reflog shift retargeted preview")
	}
}

func TestStashPreviewRejectsStaleAndBusyResults(t *testing.T) {
	for _, kind := range []string{"repository", "ticket", "sha", "closed", "interrupted"} {
		a := stashPreviewApp()
		req, _, _ := a.beginStashPreview("first")
		defer a.cancelStashPreview()
		switch kind {
		case "repository":
			a.guard.Switch()
		case "ticket":
			a.guard.Begin("stash")
		case "sha":
			a.stashSHA = "other"
		case "closed":
			a.closed = true
		case "interrupted":
			a.stashLoading = false
		}
		if a.finishStashPreview(req, selectionDiff(), errors.New("old")) || a.stashErr != nil || a.stashDiff.Text != "" {
			t.Fatalf("accepted %s", kind)
		}
	}
	for _, kind := range []string{"writing", "opening", "closed", "no repo", "empty sha"} {
		a := stashPreviewApp()
		sha := "first"
		switch kind {
		case "writing":
			a.guard.writing = true
		case "opening":
			a.opening = true
		case "closed":
			a.closed = true
		case "no repo":
			a.repo = nil
		case "empty sha":
			sha = ""
		}
		if _, _, ok := a.beginStashPreview(sha); ok {
			t.Fatalf("started %s", kind)
		}
	}
}

func TestStashPreviewCancellationAndInterruptedRetry(t *testing.T) {
	a := stashPreviewApp()
	req, ctx, _ := a.beginStashPreview("first")
	a.cancelStashPreview()
	if ctx.Err() != context.Canceled || a.stashLoading || a.stashLoaded || a.finishStashPreview(req, selectionDiff(), nil) {
		t.Fatal("canceled read remained loading or became empty success")
	}
	retry, _, ok := a.beginStashPreview(a.stashSHA)
	if !ok || !a.finishStashPreview(retry, selectionDiff(), nil) {
		t.Fatal("interrupted retry failed")
	}
	loaded := a.stashDiff
	a.cancelStashPreview() // Failed repository open must retain a completed preview.
	if !a.stashLoaded || !reflect.DeepEqual(a.stashDiff, loaded) {
		t.Fatal("cancel erased accepted preview")
	}
	parent, cancel := context.WithCancel(context.Background())
	a.ctx = parent
	_, ctx, _ = a.beginStashPreview("first")
	cancel()
	if ctx.Err() != context.Canceled {
		t.Fatal("read survived repository lifetime")
	}
	a.cancelStashPreview()
}
