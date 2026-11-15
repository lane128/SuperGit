package desktop

import (
	"github.com/egoist/mygo/ui"
	"reflect"
	"supergit/internal/gitrepo"
	"testing"
)

func commitReadyApp() *app {
	a := &app{repo: &gitrepo.Repository{Root: "/fixture"}, page: "workspace", message: "ready draft"}
	a.data.status.Entries = []gitrepo.Entry{{Path: "file.txt", Index: 'M', Worktree: '.'}}
	return a
}
func TestCommitRequiresVisibleWorkspaceAndExistingGuards(t *testing.T) {
	a := commitReadyApp()
	if !a.canCommit() {
		t.Fatal("workspace commit unexpectedly blocked")
	}
	for _, page := range []string{"history", "action", "conflicts", "stash", "repositories", "branches", "modules", "remotes", "worktrees", "logs", ""} {
		a.page = page
		if a.canCommit() {
			t.Fatalf("hidden workspace commit allowed from %q", page)
		}
	}
	a.page = "workspace"
	for _, state := range []string{"writing", "loading", "opening", "conflict", "empty draft", "empty index", "no repo"} {
		a = commitReadyApp()
		switch state {
		case "writing":
			a.guard.writing = true
		case "loading":
			a.loading = true
		case "opening":
			a.opening = true
		case "conflict":
			a.data.status.Entries[0].Conflict = true
		case "empty draft":
			a.message = " \n"
		case "empty index":
			a.data.status.Entries[0].Index = '.'
		case "no repo":
			a.repo = nil
		}
		if a.canCommit() {
			t.Fatalf("commit allowed with %s", state)
		}
	}
}
func TestPageShortcutPreservesDraftAndProtectsActionsAndWrites(t *testing.T) {
	a := commitReadyApp()
	if !a.navigateShortcut("history") || a.page != "history" || a.message != "ready draft" {
		t.Fatal("navigation lost draft")
	}
	a.page = "action"
	a.form = actionDraft{Kind: actionCreateBranch, First: "draft-branch"}
	a.returnPage = "branches"
	target := &actionTarget{}
	a.confirmation = target
	if a.navigateShortcut("workspace") || a.page != "action" || a.confirmation != target || a.form.First != "draft-branch" {
		t.Fatal("navigation bypassed action")
	}
	a.page = "history"
	a.guard.writing = true
	if a.navigateShortcut("workspace") || a.page != "history" {
		t.Fatal("navigation during write")
	}
	a.guard.writing = false
	a.opening = true
	if a.navigateShortcut("workspace") {
		t.Fatal("navigation while opening")
	}
	a.opening = false
	a.repo = nil
	if a.navigateShortcut("history") || !a.navigateShortcut("repositories") {
		t.Fatal("invalid no-repository navigation")
	}
	if a.navigateShortcut("unknown") {
		t.Fatal("unknown destination accepted")
	}
}
func TestEscapeCancelOnlyIdleActionAndKeepsWorkspaceDraft(t *testing.T) {
	a := commitReadyApp()
	a.page = "action"
	a.returnPage = "branches"
	a.confirmation = &actionTarget{}
	if !a.cancelAction() || a.page != "branches" || a.confirmation != nil || a.message != "ready draft" {
		t.Fatal("action cancellation lost state")
	}
	if a.cancelAction() {
		t.Fatal("cancel handled outside action page")
	}
	for _, state := range []string{"writing", "loading", "opening"} {
		a.page = "action"
		a.confirmation = &actionTarget{}
		a.guard.writing = state == "writing"
		a.loading = state == "loading"
		a.opening = state == "opening"
		if a.cancelAction() || a.page != "action" || a.confirmation == nil {
			t.Fatalf("cancel bypassed busy %s", state)
		}
	}
}
func TestAutomaticFileTargetChangeResetsOnlyWorkspacePreviewAndScroll(t *testing.T) {
	for _, entries := range [][]gitrepo.Entry{
		{{Path: "other.txt", Index: '.', Worktree: 'M'}},
		{{Path: "file.txt", Index: 'M', Worktree: '.'}},
		nil,
	} {
		a := &app{path: "file.txt", diff: selectionDiff(), selectedLines: map[int]bool{2: true}}
		a.diffList.ScrollTo(500, ui.Start)
		workspace := a.diffList
		a.historyDiffList.ScrollTo(77, ui.Center)
		history := a.historyDiffList
		a.data.status.Entries = entries
		a.reconcileFileSelection()
		if !a.workspaceDiffReset || a.diff.Text != "" || len(a.selectedLines) != 0 {
			t.Fatal("new target did not request fresh preview/scroll/selection")
		}
		if !reflect.DeepEqual(a.diffList, workspace) {
			t.Fatal("callback destroyed committed workspace list before redraw")
		}
		if !reflect.DeepEqual(a.historyDiffList, history) {
			t.Fatal("workspace changed history scroll")
		}
	}
	a := &app{path: "file.txt", diff: selectionDiff(), selectedLines: map[int]bool{2: true}}
	a.diffList.ScrollTo(500, ui.Start)
	before := a.diffList
	a.data.status.Entries = []gitrepo.Entry{{Path: "file.txt", Index: '.', Worktree: 'M'}}
	a.reconcileFileSelection()
	if !reflect.DeepEqual(before, a.diffList) || a.diff.Text == "" || !a.selectedLines[2] {
		t.Fatal("unchanged target lost scroll/preview/selection")
	}
}
func TestHistoricalPreviewResetLeavesWorkspacePositionIntact(t *testing.T) {
	a := &app{diff: selectionDiff(), commitDiff: selectionDiff()}
	a.diffList.ScrollTo(500, ui.Start)
	workspace := a.diffList
	a.historyDiffList.ScrollTo(80, ui.Start)
	historical := a.historyDiffList
	a.resetCommitDiff("next.txt")
	if a.commitPath != "next.txt" || a.commitDiff.Text != "" || !a.historyDiffReset {
		t.Fatal("historical target did not request a fresh preview/scroll")
	}
	if !reflect.DeepEqual(a.historyDiffList, historical) {
		t.Fatal("callback destroyed committed history list before redraw")
	}
	if !reflect.DeepEqual(workspace, a.diffList) || a.diff.Text == "" {
		t.Fatal("historical target reset workspace")
	}
}
