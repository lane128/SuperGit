package desktop

import (
	"context"
	"supergit/internal/gitrepo"
	"testing"
)

func workspaceSelectionApp() *app {
	a := &app{page: "workspace", repo: &gitrepo.Repository{Root: "/fixture"}, path: "both.txt"}
	a.data.status.Entries = []gitrepo.Entry{
		{Path: "both.txt", Index: 'M', Worktree: 'M'},
		{Path: " odd\nfile.txt ", Index: '.', Worktree: 'M'},
		{Path: "index.txt", Index: 'A', Worktree: '.'},
	}
	return a
}
func TestWorkspacePresentationGroupsAndExactIdentities(t *testing.T) {
	a := workspaceSelectionApp()
	m := a.workspaceModel()
	if len(m.rows) != 6 {
		t.Fatalf("rows=%d, want two headers and four files", len(m.rows))
	}
	for _, i := range []int{0, 3} {
		if !m.rows[i].header {
			t.Fatalf("row %d is not a header", i)
		}
		if _, ok := a.workspaceSelection(m, i); ok {
			t.Fatal("header selectable")
		}
	}
	want := []workspaceFileKey{{"both.txt", false}, {" odd\nfile.txt ", false}, {"both.txt", true}, {"index.txt", true}}
	for i, row := range []int{1, 2, 4, 5} {
		item, ok := a.workspaceSelection(m, row)
		if !ok || item.key != want[i] {
			t.Fatalf("row %d = %+v %t", row, item, ok)
		}
	}
	if m.rows[1].key == m.rows[4].key {
		t.Fatal("Index and worktree share identity")
	}
	if a.workspaceModel() != m {
		t.Fatal("presentation rebuilt on unchanged frame")
	}
}
func TestWorkspaceCursorPreservesInputAndFollowsReconciliation(t *testing.T) {
	a := workspaceSelectionApp()
	m := a.workspaceModel()
	var cursor workspaceCursor
	cursor.sync(m, a.path, a.staged)
	if cursor.row != 1 {
		t.Fatal("wrong initial file cursor")
	}
	cursor.row = 2
	cursor.sync(m, a.path, a.staged)
	if cursor.row != 2 {
		t.Fatal("view overwrote new native input")
	}
	a.path = " odd\nfile.txt "
	cursor.sync(m, a.path, a.staged)
	if cursor.row != 2 {
		t.Fatal("accepted raw path lost")
	}
	a.data.status.Entries = append([]gitrepo.Entry{{Path: "first.txt", Index: '.', Worktree: 'M'}}, a.data.status.Entries...)
	a.workspaceRevision++
	cursor.sync(a.workspaceModel(), a.path, a.staged)
	if cursor.row != 3 {
		t.Fatal("inserted file changed selected path")
	}
	a.path = "both.txt"
	a.staged = false
	a.data.status.Entries = []gitrepo.Entry{{Path: "both.txt", Index: 'M', Worktree: '.'}}
	a.workspaceRevision++
	a.reconcileFileSelection()
	cursor.sync(a.workspaceModel(), a.path, a.staged)
	if !a.staged || cursor.row != 2 {
		t.Fatal("cursor did not follow remaining Index group")
	}
	a.data.status.Entries = []gitrepo.Entry{{Path: "fallback.txt", Index: '.', Worktree: 'M'}}
	a.workspaceRevision++
	a.reconcileFileSelection()
	cursor.sync(a.workspaceModel(), a.path, a.staged)
	if a.path != "fallback.txt" || cursor.row != 1 {
		t.Fatal("cursor did not follow fallback")
	}
	a.data.status.Entries = nil
	a.workspaceRevision++
	a.reconcileFileSelection()
	cursor.sync(a.workspaceModel(), a.path, a.staged)
	if cursor.row != -1 {
		t.Fatal("empty workspace has a selected row")
	}
}
func TestWorkspaceReconciliationDefersNativeResetUntilView(t *testing.T) {
	for _, changed := range []bool{false, true} {
		a := workspaceSelectionApp()
		model := a.workspaceModel()
		a.workspaceCursor.sync(model, a.path, a.staged)
		a.workspaceList.Key = func(i int) any { return model.rows[i].key }
		if changed {
			// The successor takes the vanished file's numeric row. Native
			// ListState must forget the old key instead of clearing row 1.
			a.data.status.Entries = a.data.status.Entries[1:]
			a.workspaceRevision++
		}
		a.reconcileFileSelection()
		// MyGo's committed accessibility tree borrows ListState.frame.
		// Between callback and redraw, the previous descriptor must stay live.
		if a.workspaceList.Key == nil {
			t.Fatal("callback destroyed live native list before the next view")
		}
		if a.workspaceSelectionReset != changed {
			t.Fatal("target change did not request exactly one deferred list reset")
		}
	}
}
func TestWorkspaceSelectionRejectsStaleAndBusyCallbacks(t *testing.T) {
	for _, state := range []string{"revision", "repository", "identity", "group", "writing", "opening", "page", "no repo"} {
		a := workspaceSelectionApp()
		m := a.workspaceModel()
		switch state {
		case "revision":
			a.workspaceRevision++
		case "repository":
			a.guard.Switch()
		case "identity":
			a.data.status.Entries[0].Path = "replaced.txt"
		case "group":
			a.data.status.Entries[0].Worktree = '.'
		case "writing":
			a.guard.writing = true
		case "opening":
			a.opening = true
		case "page":
			a.page = "action"
		case "no repo":
			a.repo = nil
		}
		if _, ok := a.workspaceSelection(m, 1); ok {
			t.Fatalf("accepted %s callback", state)
		}
	}
	a := workspaceSelectionApp()
	a.loading = true
	m := a.workspaceModel()
	if _, ok := a.workspaceSelection(m, 1); !ok {
		t.Fatal("snapshot read blocked read-only browsing")
	}
	for _, row := range []int{-1, 6, 1000} {
		if _, ok := a.workspaceSelection(m, row); ok {
			t.Fatal("accepted invalid row")
		}
	}
	if _, ok := a.workspaceSelection(nil, 1); ok {
		t.Fatal("accepted nil model")
	}
}
func TestWorkspaceShortcutFocusesFilesAndProtectsDraft(t *testing.T) {
	a := workspaceSelectionApp()
	a.message = "draft\nbody"
	if !a.navigateShortcut("workspace") || !a.focusWorkspaceList || a.message != "draft\nbody" {
		t.Fatal("workspace shortcut failed to focus file list or lost draft")
	}
	a.focusWorkspaceList = false
	a.page = "action"
	if a.navigateShortcut("workspace") || a.focusWorkspaceList {
		t.Fatal("action allowed workspace focus")
	}
}
func TestWorkspaceReadCancelsSupersededDiffAndFollowsRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &app{ctx: ctx}
	first, endFirst := a.newWorkspaceRead()
	defer endFirst()
	second, endSecond := a.newWorkspaceRead()
	defer endSecond()
	if first.Err() != context.Canceled || second.Err() != nil {
		t.Fatal("superseded Diff read was not canceled independently")
	}
	cancel()
	if second.Err() != context.Canceled {
		t.Fatal("Diff read outlived repository context")
	}
}
