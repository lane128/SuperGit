package desktop

import (
	"supergit/internal/gitrepo"
	"testing"
)

func TestOldRepositoryAndSelectionResultsAreRejected(t *testing.T) {
	var g jobGuard
	g.Switch()
	old := g.Begin("snapshot")
	g.Switch()
	if g.Accept(old) {
		t.Fatal("old repository result accepted")
	}
	a := g.Begin("diff")
	b := g.Begin("diff")
	if g.Accept(a) || !g.Accept(b) {
		t.Fatal("old file selection can overwrite new diff")
	}
	snapshot := g.Begin("snapshot")
	if !g.Accept(b) || !g.Accept(snapshot) {
		t.Fatal("independent read lanes interfere")
	}
}
func TestWorkspaceAndCommitDiffResultsStayIndependent(t *testing.T) {
	a := &app{}
	a.guard.Switch()
	work := a.guard.Begin("diff")
	history := a.guard.Begin("commit")
	a.applyDiffResult(work, gitrepo.Diff{Text: "workspace changes"}, false)
	a.applyDiffResult(history, gitrepo.Diff{Text: "commit changes"}, true)
	if a.diff.Text != "workspace changes" || a.commitDiff.Text != "commit changes" {
		t.Fatal("history result overwrote working diff")
	}
	stale := a.guard.Begin("diff")
	a.guard.Begin("diff")
	a.applyDiffResult(stale, gitrepo.Diff{Text: "stale"}, false)
	if a.diff.Text != "workspace changes" {
		t.Fatal("stale result accepted")
	}
}
func TestFileSelectionFollowsStageAndUnstage(t *testing.T) {
	a := &app{path: "file.txt", diff: gitrepo.Diff{Text: "old preview"}}
	a.data.status.Entries = []gitrepo.Entry{{Path: "file.txt", Index: 'M', Worktree: '.'}}
	a.reconcileFileSelection()
	if !a.staged || a.path != "file.txt" {
		t.Fatal("staged file selection stayed in empty unstaged group")
	}
	a.data.status.Entries = []gitrepo.Entry{{Path: "file.txt", Index: '.', Worktree: 'M'}}
	a.reconcileFileSelection()
	if a.staged {
		t.Fatal("unstaged file selection stayed in empty staged group")
	}
	a.data.status.Entries = nil
	a.reconcileFileSelection()
	if a.path != "" || a.diff.Text != "" {
		t.Fatal("clean workspace retained stale selection")
	}
}
func TestWriteGuardRejectsOverlappingMutations(t *testing.T) {
	var g jobGuard
	if !g.StartWrite() {
		t.Fatal("first write rejected")
	}
	if g.StartWrite() {
		t.Fatal("overlapping write accepted")
	}
	g.EndWrite()
	if !g.StartWrite() {
		t.Fatal("write guard never released")
	}
}
func TestGraphKeepsFirstParentInLaneUntilMerge(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "merge", Parents: []string{"main", "feature"}}, {SHA: "feature", Parents: []string{"root"}}, {SHA: "main", Parents: []string{"root"}}, {SHA: "root"}}
	rows := layoutGraph(commits)
	if rows[2].Edges[0].ViaLane != rows[2].Lane {
		t.Fatal("first-parent edge joins another lane before reaching parent")
	}
	if rows[1].Lane == rows[2].Lane {
		t.Fatal("independent commits occupy same lane")
	}
}
func TestGraphUsesParentsAndMarksPageBoundaries(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "merge", Parents: []string{"main", "feature"}}, {SHA: "main", Parents: []string{"root"}}, {SHA: "feature", Parents: []string{"root"}}, {SHA: "root", Parents: []string{"outside"}}}
	rows := layoutGraph(commits)
	edges := map[[2]int]bool{}
	for i, r := range rows {
		for _, e := range r.Edges {
			edges[[2]int{i, e.Row}] = true
		}
	}
	want := [][2]int{{0, 1}, {0, 2}, {1, 3}, {2, 3}}
	if len(edges) != 4 {
		t.Fatalf("unexpected graph edges: %v", edges)
	}
	for _, edge := range want {
		if !edges[edge] {
			t.Fatalf("missing edge %v", edge)
		}
	}
	if rows[1].Lane == rows[2].Lane {
		t.Fatal("branch lanes collapsed")
	}
	if !rows[3].Boundary {
		t.Fatal("missing parent must mark boundary")
	}
}
