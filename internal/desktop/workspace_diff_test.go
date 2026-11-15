package desktop

import (
	"context"
	"errors"
	"github.com/egoist/mygo/ui"
	"reflect"
	"supergit/internal/gitrepo"
	"testing"
	"time"
)

func workspaceDiffApp() *app {
	a := watchApp()
	a.ctx = context.Background()
	a.diffLoaded = true
	a.commitDiff = selectionDiff()
	a.message = "draft"
	return a
}

func TestWorkspaceDiffFailureRetryKeepsExactGroupAndClearsUnsafeData(t *testing.T) {
	a := workspaceDiffApp()
	historical := a.commitDiff
	req, ctx, ok := a.beginWorkspaceDiff(" odd\npath ", true, false)
	if !ok || !a.diffLoading || a.diffLoaded || a.diffErr != nil || len(a.selectedLines) != 0 {
		t.Fatal("new target not reset")
	}
	failure := errors.New("output limit")
	if !a.finishWorkspaceDiff(req, selectionDiff(), failure) || a.diffErr != failure || !a.diffLoaded || a.diffLoading || a.diff.Text != "" || ctx.Err() != context.Canceled || a.diffCancel != nil {
		t.Fatal("failed read left partial/unsafe data or pending context")
	}
	if a.linesAllowed(false) {
		t.Fatal("failed Diff allowed line action")
	}
	if a.finishWorkspaceDiff(req, selectionDiff(), nil) {
		t.Fatal("duplicate completion accepted")
	}
	retry, _, ok := a.beginWorkspaceDiff(a.path, a.staged, false)
	if !ok || retry.path != " odd\npath " || !retry.staged || a.diffErr != nil {
		t.Fatal("retry lost path/group or kept error")
	}
	if !a.finishWorkspaceDiff(retry, gitrepo.Diff{}, nil) || !a.diffLoaded || a.diffErr != nil {
		t.Fatal("valid empty Diff became failure")
	}
	if !reflect.DeepEqual(a.commitDiff, historical) || a.message != "draft" {
		t.Fatal("workspace retry altered historical preview or draft")
	}
}

func TestWorkspaceDiffAutomaticReadRetainsOnlyUnchangedSelectionAndScroll(t *testing.T) {
	a := workspaceDiffApp()
	a.diffList.ScrollTo(120, ui.Start)
	list := a.diffList
	original := a.diff
	req, _, ok := a.beginWorkspaceDiff(a.path, a.staged, true)
	if !ok || !a.selectedLines[2] || a.diff.Text != original.Text || a.workspaceDiffReset || !reflect.DeepEqual(a.diffList, list) {
		t.Fatal("automatic read cleared unchanged preview before completion")
	}
	if !a.finishWorkspaceDiff(req, original, nil) || !a.selectedLines[2] {
		t.Fatal("unchanged automatic read lost selection")
	}
	req, _, _ = a.beginWorkspaceDiff(a.path, a.staged, true)
	if !a.finishWorkspaceDiff(req, gitrepo.Diff{Text: "changed"}, nil) || len(a.selectedLines) != 0 {
		t.Fatal("changed automatic read kept stale lines")
	}
	a.selectedLines = map[int]bool{2: true}
	req, _, _ = a.beginWorkspaceDiff(a.path, a.staged, true)
	if !a.finishWorkspaceDiff(req, original, errors.New("failed")) || len(a.selectedLines) != 0 || a.diff.Text != "" || !reflect.DeepEqual(a.diffList, list) {
		t.Fatal("failure kept unsafe lines or reset borrowed ListState")
	}
}

func TestWorkspaceDiffInterruptedAutomaticRecoveryRemainsUnread(t *testing.T) {
	for _, interruption := range []string{"failed open", "write then failed snapshot"} {
		t.Run(interruption, func(t *testing.T) {
			a := workspaceDiffApp()
			a.failWorkspaceDiff(errors.New("previous failure"))
			req, ctx, ok := a.beginWorkspaceDiff(a.path, a.staged, true)
			if !ok {
				t.Fatal("automatic recovery did not start")
			}
			defer a.diffCancel()
			if interruption == "failed open" {
				a.guard.Switch()
			} else {
				a.guard.Begin("diff")
			}
			a.diffCancel()
			a.diffLoading = false // Open/write interruption; failed open/snapshot retains the old preview.
			if a.diffLoaded || a.diffErr != nil || a.diff.Text != "" || ctx.Err() != context.Canceled {
				t.Fatal("interrupted recovery was promoted to successful empty Diff")
			}
			if a.finishWorkspaceDiff(req, gitrepo.Diff{}, nil) {
				t.Fatal("interrupted recovery accepted late completion")
			}
			retry, _, ok := a.beginWorkspaceDiff(a.path, a.staged, false)
			if !ok || !a.finishWorkspaceDiff(retry, selectionDiff(), nil) || !a.diffLoaded {
				t.Fatal("interrupted recovery could not retry")
			}
		})
	}
}

func TestWorkspaceDiffRejectsStaleResultsAndCancelsSupersededReads(t *testing.T) {
	for _, kind := range []string{"repository", "ticket", "path", "group", "closed", "interrupted"} {
		t.Run(kind, func(t *testing.T) {
			a := workspaceDiffApp()
			req, _, _ := a.beginWorkspaceDiff(a.path, false, false)
			defer a.diffCancel()
			switch kind {
			case "repository":
				a.guard.Switch()
			case "ticket":
				a.guard.Begin("diff")
			case "path":
				a.path = "other"
			case "group":
				a.staged = true
			case "closed":
				a.closed = true
			case "interrupted":
				a.diffLoading = false
			}
			if a.finishWorkspaceDiff(req, selectionDiff(), errors.New("old")) || a.diffErr != nil || a.diff.Text != "" {
				t.Fatal("stale result published")
			}
		})
	}
	a := workspaceDiffApp()
	first, ctx, _ := a.beginWorkspaceDiff("same", false, false)
	second, current, _ := a.beginWorkspaceDiff("same", true, false)
	if ctx.Err() != context.Canceled || a.finishWorkspaceDiff(first, selectionDiff(), errors.New("old group")) || current.Err() != nil || !a.diffLoading {
		t.Fatal("old group ended current read")
	}
	if !a.finishWorkspaceDiff(second, selectionDiff(), nil) || current.Err() != context.Canceled {
		t.Fatal("current group failed")
	}
}

func TestWorkspaceDiffStartGuardsAndFallbackClearErrors(t *testing.T) {
	for _, kind := range []string{"writing", "opening", "closed", "no repo", "empty path"} {
		a := workspaceDiffApp()
		path := a.path
		switch kind {
		case "writing":
			a.guard.writing = true
		case "opening":
			a.opening = true
		case "closed":
			a.closed = true
		case "no repo":
			a.repo = nil
		case "empty path":
			path = ""
		}
		if _, _, ok := a.beginWorkspaceDiff(path, false, false); ok {
			t.Fatalf("started %s", kind)
		}
	}
	a := workspaceDiffApp()
	a.diffErr = errors.New("old")
	a.reconcileFileSelection() // Empty status removes the selected target.
	if a.path != "" || a.diffErr != nil || a.diffLoaded {
		t.Fatal("empty workspace retained previous error")
	}
}

func TestWorkspaceDiffWatcherFailureClearsOnlyCurrentTarget(t *testing.T) {
	for _, kind := range []string{"current", "path", "group", "sequence"} {
		t.Run(kind, func(t *testing.T) {
			a := workspaceDiffApp()
			req, _ := a.beginProbe(time.Now())
			switch kind {
			case "path":
				a.path = "other"
			case "group":
				a.staged = !a.staged
			case "sequence":
				a.guard.Begin("diff")
			}
			failure := errors.New("selected Diff too large")
			if a.acceptProbe(req, "base", gitrepo.Diff{}, failure, time.Now()) {
				t.Fatal("failed probe requested success refresh")
			}
			if kind == "current" {
				if a.diffErr != failure || a.diff.Text != "" || len(a.selectedLines) != 0 || !a.diffLoaded {
					t.Fatal("current failed probe retained stale actionable preview")
				}
			} else if a.diffErr != nil || a.diff.Text == "" || !a.selectedLines[2] {
				t.Fatal("stale probe changed current preview")
			}
		})
	}
}

func TestWorkspaceDiffWatcherRecoversFailedEmptyPreview(t *testing.T) {
	a := workspaceDiffApp()
	a.diff = gitrepo.Diff{}
	a.diffErr = errors.New("previous failure")
	req, ok := a.beginProbe(time.Now())
	if !ok {
		t.Fatal("probe refused")
	}
	if !a.acceptProbe(req, "base", gitrepo.Diff{}, nil, time.Now()) {
		t.Fatal("successful empty read could not clear a failed preview")
	}
}
