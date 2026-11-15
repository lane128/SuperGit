package desktop

import (
	"reflect"
	"supergit/internal/gitrepo"
	"testing"
)

func selectionDiff() gitrepo.Diff {
	return gitrepo.Diff{Text: "preview", CanStageHunks: true, Lines: []gitrepo.DiffLine{
		{Kind: "hunk", Text: "@@ -1,3 +1,4 @@"},
		{Kind: "context", Text: " same", Old: 1, New: 1},
		{Kind: "del", Text: "-old", Old: 2},
		{Kind: "add", Text: "+new", New: 2},
		{Kind: "add", Text: "+extra", New: 3},
		{Kind: "meta", Text: "\\ No newline at end of file"},
		{Kind: "context", Text: " end", Old: 3, New: 4},
		{Kind: "hunk", Text: "@@ -20 +21 @@"},
		{Kind: "del", Text: "-gone", Old: 20},
	}}
}
func TestSplitRowsRetainOriginalIdentityAndEmptySides(t *testing.T) {
	d := selectionDiff()
	rows := splitDiffRows(d)
	want := [][3]int{{0, -1, -1}, {-1, 1, 1}, {-1, 2, 3}, {-1, -1, 4}, {5, -1, -1}, {-1, 6, 6}, {7, -1, -1}, {-1, 8, -1}}
	var got [][3]int
	for _, row := range rows {
		got = append(got, [3]int{row.Shared, row.Left, row.Right})
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("split identities %v, want %v", got, want)
	}
	if d.Lines[rows[2].Left].Old != 2 || d.Lines[rows[2].Right].New != 2 {
		t.Fatal("line numbers lost")
	}
}
func TestLineActionCapturesOriginalIndicesAndDirection(t *testing.T) {
	a := &app{path: "file.txt", staged: true, diff: selectionDiff(), selectedLines: map[int]bool{2: true, 4: true}}
	a.data.status.Entries = []gitrepo.Entry{{Path: "file.txt", Index: 'M', Worktree: 'M'}}
	target, ok := a.captureLines(false)
	if !ok || target.Path != "file.txt" || !target.Reverse || !reflect.DeepEqual(target.Selection, []int{2, 4}) {
		t.Fatalf("target %#v, %v", target, ok)
	}
	// Switching presentation shares selection; the target must survive subsequent UI changes.
	a.setDiffPresentation(true)
	rows := splitDiffRows(a.diff)
	if !a.selectedLines[rows[2].Left] || !a.selectedLines[rows[3].Right] {
		t.Fatal("split selection drifted")
	}
	a.selectedLines[2] = false
	a.path = "another.txt"
	if !reflect.DeepEqual(target.Selection, []int{2, 4}) || target.Path != "file.txt" {
		t.Fatal("captured target follows live selection")
	}
}
func TestLineActionRejectsHistoryBusyAndUnsupportedFiles(t *testing.T) {
	a := &app{path: "file.txt", diff: selectionDiff(), selectedLines: map[int]bool{2: true}}
	a.data.status.Entries = []gitrepo.Entry{{Path: "file.txt", Index: 'M', Worktree: 'M'}}
	if _, ok := a.captureLines(true); ok {
		t.Fatal("history can mutate workspace selection")
	}
	a.diffLoading = true
	if _, ok := a.captureLines(false); ok {
		t.Fatal("loading diff can mutate")
	}
	a.diffLoading = false
	a.guard.StartWrite()
	if _, ok := a.captureLines(false); ok {
		t.Fatal("overlapping write can mutate")
	}
	a.guard.EndWrite()
	a.data.status.Entries[0].Conflict = true
	if _, ok := a.captureLines(false); ok {
		t.Fatal("conflict can mutate by line")
	}
}
func TestAcceptedWorkspaceDiffClearsSelectionButHistoryAndStaleDoNot(t *testing.T) {
	a := &app{selectedLines: map[int]bool{2: true}}
	a.guard.Switch()
	historical := a.guard.Begin("commit")
	a.applyDiffResult(historical, selectionDiff(), true)
	if !a.selectedLines[2] {
		t.Fatal("history cleared workspace selection")
	}
	stale := a.guard.Begin("diff")
	fresh := a.guard.Begin("diff")
	a.applyDiffResult(stale, selectionDiff(), false)
	if !a.selectedLines[2] {
		t.Fatal("stale result changed selection")
	}
	a.applyDiffResult(fresh, selectionDiff(), false)
	if len(a.selectedLines) != 0 {
		t.Fatal("replacement diff retained old line selection")
	}
}
func TestReconcileGroupAndFileChangesClearLineSelection(t *testing.T) {
	a := &app{path: "file.txt", selectedLines: map[int]bool{2: true}}
	a.data.status.Entries = []gitrepo.Entry{{Path: "file.txt", Index: 'M', Worktree: '.'}}
	a.reconcileFileSelection()
	if !a.staged || len(a.selectedLines) != 0 {
		t.Fatal("group changed but line selection survived")
	}
	a.selectedLines = map[int]bool{2: true}
	a.data.status.Entries = []gitrepo.Entry{{Path: "other.txt", Index: '.', Worktree: 'M'}}
	a.reconcileFileSelection()
	if a.path != "other.txt" || len(a.selectedLines) != 0 {
		t.Fatal("file changed but line selection survived")
	}
}
