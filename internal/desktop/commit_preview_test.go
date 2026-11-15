package desktop

import (
	"context"
	"errors"
	"reflect"
	"supergit/internal/gitrepo"
	"testing"
)

func previewApp() *app {
	a := messageApp()
	a.commitText = commitMessageState{sha: a.selectedSHA, message: "Title\n\nBody", loaded: true}
	return a
}

func TestCommitPreviewFileFailureRetryAndEmptySuccess(t *testing.T) {
	a := previewApp()
	message := a.commitText
	req, ctx, ok := a.beginCommitPreview("")
	if !ok || !a.commitLoading || a.commitFilesLoaded {
		t.Fatal("file read did not start")
	}
	failure := errors.New("missing tree")
	if !a.finishCommitFiles(req, []gitrepo.ChangedFile{{Path: "partial"}}, failure) || a.commitFilesErr != failure || !a.commitFilesLoaded || a.commitLoading || len(a.commitFiles) != 0 || ctx.Err() != context.Canceled {
		t.Fatal("failed file list reported empty success or kept partial data")
	}
	if a.finishCommitFiles(req, nil, nil) {
		t.Fatal("accepted duplicate completion")
	}
	retry, _, ok := a.beginCommitPreview("")
	if !ok || a.commitFilesErr != nil {
		t.Fatal("retry retained old error")
	}
	if !a.finishCommitFiles(retry, nil, nil) || !a.commitFilesLoaded || a.commitFilesErr != nil || len(a.commitFiles) != 0 {
		t.Fatal("empty success not distinguished from error")
	}
	if a.commitText != message {
		t.Fatal("file retry changed message")
	}
}

func TestCommitPreviewDiffFailureRecoversExactPathAndKeepsFiles(t *testing.T) {
	a := previewApp()
	files := []gitrepo.ChangedFile{{Path: " odd\nfile.txt ", Status: "M"}, {Path: "second", Status: "A"}}
	a.replaceCommitFiles(files)
	a.commitFilesLoaded = true
	req, ctx, ok := a.beginCommitPreview(files[0].Path)
	if !ok {
		t.Fatal("diff read refused")
	}
	failure := errors.New("missing blob")
	if !a.finishCommitDiff(req, selectionDiff(), failure) || a.commitDiffErr != failure || !a.commitDiffLoaded || a.commitLoading || a.commitDiff.Text != "" || ctx.Err() != context.Canceled {
		t.Fatal("failed Diff displayed partial or empty success")
	}
	retry, _, ok := a.beginCommitPreview(files[0].Path)
	if !ok || retry.sha != a.selectedSHA || retry.path != files[0].Path || a.commitDiffErr != nil {
		t.Fatal("retry lost immutable target")
	}
	if !a.finishCommitDiff(retry, selectionDiff(), nil) || a.commitDiffErr != nil || !a.commitDiffLoaded || a.commitDiff.Text == "" {
		t.Fatal("retry failed to install Diff")
	}
	if !reflect.DeepEqual(a.commitFiles, files) || !a.commitText.loaded {
		t.Fatal("Diff retry replaced files/message")
	}
	next, _, ok := a.beginCommitPreview(files[1].Path)
	if !ok || a.commitDiffErr != nil || a.commitDiffLoaded || a.finishCommitDiff(retry, selectionDiff(), failure) {
		t.Fatal("file change kept old result/error")
	}
	a.finishCommitDiff(next, gitrepo.Diff{}, nil)
	if !a.commitDiffLoaded || a.commitDiffErr != nil {
		t.Fatal("valid no-text Diff became failure")
	}
}

func TestCommitPreviewRejectsStaleIdentityAndSupersedesContext(t *testing.T) {
	for _, kind := range []string{"repository", "sha", "ticket", "path", "closed", "interrupted"} {
		t.Run(kind, func(t *testing.T) {
			a := previewApp()
			a.replaceCommitFiles([]gitrepo.ChangedFile{{Path: "a", Status: "M"}})
			req, _, ok := a.beginCommitPreview("a")
			if !ok {
				t.Fatal("start")
			}
			defer a.commitCancel()
			switch kind {
			case "repository":
				a.guard.Switch()
			case "sha":
				a.selectedSHA = "another"
			case "ticket":
				a.guard.Begin("commit")
			case "path":
				a.commitPath = "b"
			case "closed":
				a.closed = true
			case "interrupted":
				a.commitLoading = false
			}
			if a.finishCommitDiff(req, selectionDiff(), errors.New("old")) || a.commitDiffErr != nil || a.commitDiff.Text != "" {
				t.Fatal("accepted stale Diff")
			}
		})
	}
	a := previewApp()
	first, ctx, _ := a.beginCommitPreview("")
	second, _, _ := a.beginCommitPreview("")
	if ctx.Err() != context.Canceled || a.finishCommitFiles(first, nil, errors.New("old")) || !a.finishCommitFiles(second, nil, nil) {
		t.Fatal("file reads did not supersede")
	}
}

func TestCommitPreviewStartGuardsAndInterruptedRetry(t *testing.T) {
	for _, kind := range []string{"writing", "opening", "closed", "no repo", "no sha", "unknown file"} {
		a := previewApp()
		path := ""
		switch kind {
		case "writing":
			a.guard.writing = true
		case "opening":
			a.opening = true
		case "closed":
			a.closed = true
		case "no repo":
			a.repo = nil
		case "no sha":
			a.selectedSHA = ""
		case "unknown file":
			path = "absent"
		}
		if _, _, ok := a.beginCommitPreview(path); ok {
			t.Fatalf("started during %s", kind)
		}
	}
	a := previewApp()
	req, _, _ := a.beginCommitPreview("")
	a.guard.Begin("commit") // Writes invalidate the commit lane and stop its spinner.
	a.commitLoading = false
	if a.finishCommitFiles(req, nil, nil) || a.commitFilesLoaded {
		t.Fatal("interrupted read reported success")
	}
	retry, _, ok := a.beginCommitPreview("")
	if !ok || !a.finishCommitFiles(retry, []gitrepo.ChangedFile{{Path: "a", Status: "M"}}, nil) {
		t.Fatal("interrupted read cannot recover")
	}
}
