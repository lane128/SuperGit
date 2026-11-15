package desktop

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"supergit/internal/gitrepo"
)

func pagedHistoryApp() *app {
	a := selectionHistoryApp()
	a.ctx = context.Background()
	a.historyLimit = 300
	a.data.historyTips = []string{a.data.history[0].SHA}
	a.data.historyMore = true
	a.selectedSHA = a.data.history[0].SHA
	a.commitPath = "selected.txt"
	a.commitDiff = selectionDiff()
	a.message = "draft to preserve"
	return a
}

func nextHistoryPage(a *app) gitrepo.HistoryPage {
	return gitrepo.HistoryPage{Tips: append([]string(nil), a.data.historyTips...), Commits: []gitrepo.Commit{{SHA: "cccccccccccccccccccccccccccccccccccccccc", Subject: "older commit"}}}
}

func TestHistoryPageAppendPreservesSelectionAndPreview(t *testing.T) {
	a := pagedHistoryApp()
	a.data.history = performanceCommits(2100)
	a.historyLimit = 2100
	beforeSHA, beforeDiff, beforePath, beforeDraft := a.selectedSHA, a.commitDiff, a.commitPath, a.message
	beforeLength, beforeRevision := len(a.data.history), a.historyRevision
	oldModel := a.historyModel()
	req, ctx, ok := a.beginHistoryPage()
	if !ok || !a.historyPageLoading || a.canLoadHistory() {
		t.Fatal("page did not enter its own loading state")
	}
	defer a.cancelHistoryPage()
	if ctx.Err() != nil || a.loading || a.guard.writing {
		t.Fatal("page blocked snapshot or started a write")
	}
	if !a.finishHistoryPage(req, nextHistoryPage(a), nil) {
		t.Fatal("current page rejected")
	}
	if len(a.data.history) != beforeLength+1 || a.historyLimit != beforeLength+1 || a.historyRevision != beforeRevision+1 || a.data.historyMore || a.historyPageLoading {
		t.Fatal("page was not appended or end state was not recorded")
	}
	if a.selectedSHA != beforeSHA || a.commitPath != beforePath || !reflect.DeepEqual(a.commitDiff, beforeDiff) || a.message != beforeDraft {
		t.Fatal("append changed selected details or workspace draft")
	}
	if _, ok := a.historySelectionIndex(oldModel, 0); ok {
		t.Fatal("old presentation callback survived append")
	}
	if a.historyModel() == oldModel || a.canLoadHistory() {
		t.Fatal("append kept stale cache or allowed loading after end")
	}
	if ctx.Err() != context.Canceled || a.finishHistoryPage(req, nextHistoryPage(a), nil) {
		t.Fatal("finished page leaked context or accepted a duplicate callback")
	}
}

func TestHistoryPageRejectsStaleAndMalformedResults(t *testing.T) {
	for _, kind := range []string{"repository", "revision", "offset", "ticket", "tips", "writing", "opening"} {
		t.Run(kind, func(t *testing.T) {
			a := pagedHistoryApp()
			req, _, _ := a.beginHistoryPage()
			defer a.cancelHistoryPage()
			page := nextHistoryPage(a)
			before := len(a.data.history)
			switch kind {
			case "repository":
				a.guard.Switch()
			case "revision":
				a.historyRevision++
			case "offset":
				req.offset++
			case "ticket":
				a.guard.Begin("history-page")
			case "tips":
				a.data.historyTips = []string{"different traversal"}
			case "writing":
				a.guard.writing = true
			case "opening":
				a.opening = true
			}
			if a.finishHistoryPage(req, page, nil) || len(a.data.history) != before {
				t.Fatal("stale page changed history")
			}
		})
	}
	for _, kind := range []string{"error", "duplicate", "internal duplicate", "empty with more", "wrong tips"} {
		t.Run(kind, func(t *testing.T) {
			a := pagedHistoryApp()
			req, _, _ := a.beginHistoryPage()
			defer a.cancelHistoryPage()
			page := nextHistoryPage(a)
			var err error
			switch kind {
			case "error":
				err = errors.New("read failed")
			case "duplicate":
				page.Commits = a.data.history[:1]
			case "internal duplicate":
				page.Commits = append(page.Commits, page.Commits[0])
			case "empty with more":
				page.Commits, page.HasMore = nil, true
			case "wrong tips":
				page.Tips = []string{"different"}
			}
			before := append([]gitrepo.Commit(nil), a.data.history...)
			if !a.finishHistoryPage(req, page, err) || a.historyPageLoading || !a.canLoadHistory() || !reflect.DeepEqual(a.data.history, before) || a.notice == "" {
				t.Fatal("failed page lost history or retry state")
			}
		})
	}
}

func TestHistoryPageGuardsAndCancellation(t *testing.T) {
	for _, kind := range []string{"no repo", "page", "snapshot", "write", "open", "pending", "end", "empty tips"} {
		a := pagedHistoryApp()
		switch kind {
		case "no repo":
			a.repo = nil
		case "page":
			a.page = "workspace"
		case "snapshot":
			a.loading = true
		case "write":
			a.guard.writing = true
		case "open":
			a.opening = true
		case "pending":
			a.historyPageLoading = true
		case "end":
			a.data.historyMore = false
		case "empty tips":
			a.data.historyTips = nil
		}
		if _, _, ok := a.beginHistoryPage(); ok {
			a.cancelHistoryPage()
			t.Fatalf("page started during %s", kind)
		}
	}
	a := pagedHistoryApp()
	req, ctx, _ := a.beginHistoryPage()
	a.cancelHistoryPage()
	if ctx.Err() != context.Canceled || a.historyPageLoading || a.finishHistoryPage(req, nextHistoryPage(a), nil) {
		t.Fatal("cancel failed to stop/reject a pending page")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	a.ctx = cancelled
	_, ctx, _ = a.beginHistoryPage()
	cancel()
	defer a.cancelHistoryPage()
	if ctx.Err() != context.Canceled {
		t.Fatal("page is detached from repository lifetime")
	}
}
