package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestHistoryGraphReusesUnchangedFrames(t *testing.T) {
	a := &app{showGraph: true, commits: distantParentHistory(64)}
	first := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	a.frameSeq++
	second := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	if first != second {
		t.Fatal("unchanged repaint rebuilt graph")
	}
	a.historyQuery = "long"
	filtered := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	a.frameSeq++
	if next := a.cachedCommitGraph(a.historyRows(), lanePalette, 100); next != filtered {
		t.Fatal("unchanged search rebuilt graph")
	}
	a.commits = append(a.commits, gitrepo.Commit{SHA: "older"})
	appended := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	if appended == filtered {
		t.Fatal("appended history reused stale graph")
	}
	a.commits = append([]gitrepo.Commit(nil), a.commits...)
	replaced := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	if replaced == appended {
		t.Fatal("same-length replacement reused stale graph")
	}
	palette := append([]ui.Color(nil), lanePalette...)
	palette[0] = ui.Hex("#abcdef")
	recolored := a.cachedCommitGraph(a.historyRows(), palette, 100)
	if recolored == replaced || recolored.LaneColor(0) != palette[0] {
		t.Fatal("theme change retained old graph colors")
	}
}

func TestHistoryGraphReusesRefFilteredFrames(t *testing.T) {
	a := &app{showGraph: true, commits: []gitrepo.Commit{{SHA: "tip", Parents: []string{"base"}}, {SHA: "other"}, {SHA: "base"}}, branches: []gitrepo.Branch{{Name: "main", SHA: "tip"}}, historyRef: historyRef{Kind: historyRefLocal, Name: "main"}}
	first := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	a.frameSeq++
	if next := a.cachedCommitGraph(a.historyRows(), lanePalette, 100); next != first {
		t.Fatal("ref-filtered repaint rebuilt graph")
	}
	a.branches = []gitrepo.Branch{{Name: "main", SHA: "other"}}
	next := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	if next == first || len(next.Rows) != 1 || a.historyRows()[0].SHA != "other" {
		t.Fatal("moved ref retained stale reachable history")
	}
	a.historyRef = historyRef{}
	all := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	if len(all.Rows) != 3 {
		t.Fatal("cleared ref retained filtered graph")
	}
	a.fileFilter = "path"
	a.fileCommits = []gitrepo.Commit{{SHA: "file"}}
	file := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	if len(file.Rows) != 1 || a.historyRows()[0].SHA != "file" {
		t.Fatal("file history retained repository graph")
	}
}

func TestCachedGraphRefreshesWorktreeAndWidth(t *testing.T) {
	a := &app{showGraph: true, commits: []gitrepo.Commit{{SHA: "head", Parents: []string{"base"}}, {SHA: "base"}}}
	graph := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	cleanWidth := graph.Width
	a.status.Head = "head"
	a.status.Entries = []gitrepo.Entry{{Path: "dirty"}}
	dirty := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	if dirty != graph || dirty.Worktree == nil || dirty.Worktree.HeadRow != 0 {
		t.Fatal("dirty state did not update cached graph link")
	}
	if dirty.Width != cleanWidth {
		t.Errorf("adjacent HEAD should use the existing lane: clean width %.1f, dirty width %.1f", cleanWidth, dirty.Width)
	}
	a.status.Head = "base"
	if next := a.cachedCommitGraph(a.historyRows(), lanePalette, 100); next.Worktree.HeadRow != 1 {
		t.Fatal("HEAD change retained old link")
	}
	for i := 0; i < 3; i++ {
		if next := a.cachedCommitGraph(a.historyRows(), lanePalette, 100); next.Width != cleanWidth+worktreeGraphGutter {
			t.Fatalf("older HEAD should reserve exactly one gutter: clean width %.1f, dirty width %.1f", cleanWidth, next.Width)
		}
	}
	a.status.Entries = nil
	clean := a.cachedCommitGraph(a.historyRows(), lanePalette, 100)
	if clean.Worktree != nil || clean.Width != cleanWidth {
		t.Fatal("clean state retained worktree gutter")
	}
	narrow := a.cachedCommitGraph(a.historyRows(), lanePalette, 24)
	if narrow != graph || narrow.Width >= cleanWidth {
		t.Fatal("width change failed to refit cached graph")
	}
}
