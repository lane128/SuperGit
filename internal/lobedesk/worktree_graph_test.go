package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestWorkingChangesUseDashedHeadConnection(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "other", Parents: []string{"actual"}}, {SHA: "actual"}}
	a := &app{showGraph: true, status: gitrepo.Status{Head: "actual", Entries: []gitrepo.Entry{{Path: "a", Worktree: 'M'}}}}
	g := newCommitGraph(commits, nil)
	link, ok := a.worktreeGraphConnection(commits, g)
	if !ok || link.HeadRow != 1 {
		t.Fatalf("worktree does not target actual HEAD: %+v visible=%v", link, ok)
	}
	// Reserve a separate lane: the temporary edge must not pass through 'other'.
	g.Worktree = &link
	rect := ui.Rect{X: 0, Y: 100, W: 100, H: 50}
	points := link.rowPoints(g, 1, rect, 25)
	if len(points) == 0 {
		t.Fatal("no connection reaches actual HEAD")
	}
	last := points[len(points)-1]
	wantX := laneXGap(int(worktreeGraphGutter), g.Rows[1].Lane, g.LaneGap())
	if last[0] != wantX || last[1] != 125 {
		t.Fatalf("connection ends at %v, want [%v 125]", last, wantX)
	}
	if len(g.Rows) != 2 {
		t.Fatal("working tree inserted a fake commit into history")
	}
	a.status.Entries = nil
	if _, ok := a.worktreeGraphConnection(commits, g); ok {
		t.Fatal("clean worktree claims an uncommitted edge")
	}
	a.status.Entries = []gitrepo.Entry{{Path: "new", Index: '?', Worktree: '?'}}
	a.showGraph = false
	if _, ok := a.worktreeGraphConnection(commits, g); ok {
		t.Fatal("disabled graph still draws worktree edge")
	}
}

func TestWorkingChangesConnectVerticallyToFirstHead(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "actual"}}
	a := &app{showGraph: true, status: gitrepo.Status{Head: "actual", Entries: []gitrepo.Entry{{Path: "a", Worktree: 'M'}}}}
	graph := newCommitGraph(commits, nil)
	link, _ := a.worktreeGraphConnection(commits, graph)
	graph.Worktree = &link
	for _, gap := range []float32{graphLaneGap, graphLaneGapMin} {
		graph.Gap = gap
		r := ui.Rect{X: 10, Y: 56, H: 50}
		points := link.rowPoints(graph, 0, r, 25)
		wantX := laneXGap(int(r.X+graph.worktreeGutter()), graph.Rows[0].Lane, graph.LaneGap())
		if len(points) < 2 {
			t.Fatal("missing working tree to HEAD connection")
		}
		for _, point := range points {
			if point[0] != wantX {
				t.Fatalf("adjacent HEAD connector bends sideways at %v; want vertical x=%v", point, wantX)
			}
		}
	}
}

func TestWorkingChangesMarkerAlignsWithFirstHeadLane(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "actual"}}
	a := &app{showGraph: true, status: gitrepo.Status{Head: "actual", Entries: []gitrepo.Entry{{Path: "a", Worktree: 'M'}}}}
	graph := newCommitGraph(commits, nil)
	link, _ := a.worktreeGraphConnection(commits, graph)
	graph.Worktree = &link
	a.historyGraph = graph
	tt := pageTester(a, func(c *ui.Context) { a.historyWorktreeRow(c, 0, 2) })
	r := a.worktreeGraphBounds
	x := int(laneXGap(int(r.X+graph.worktreeGutter()), graph.Rows[0].Lane, graph.LaneGap()))
	y := int(r.Y + r.H/2)
	img := tt.Image()
	if img.RGBAAt(x, y) == img.RGBAAt(x+6, y) {
		t.Fatalf("working tree marker is missing from HEAD lane x=%d", x)
	}
}

func TestWorktreeEdgeDoesNotDoubleBackAcrossPinnedGap(t *testing.T) {
	a := &app{showGraph: true, status: gitrepo.Status{Head: "actual", Entries: []gitrepo.Entry{{Path: "a", Worktree: 'M'}}}}
	commits := []gitrepo.Commit{{SHA: "actual"}}
	graph := newCommitGraph(commits, nil)
	link, _ := a.worktreeGraphConnection(commits, graph)
	graph.Worktree = &link
	// 6 DIP padding makes the pinned marker-to-parent distance 56 rather than
	// exactly one 50-DIP row. The old edge builder emitted overlapping bends.
	points := link.rowPoints(graph, 0, ui.Rect{Y: 56, H: 50}, 25)
	if len(points) < 2 {
		t.Fatal("missing connection")
	}
	for i := 1; i < len(points); i++ {
		if points[i][1] < points[i-1][1] {
			t.Fatalf("dashed edge doubles back at %v -> %v", points[i-1], points[i])
		}
	}
}

func TestHiddenHeadDoesNotConnectToFirstVisibleCommit(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "unrelated"}}
	a := &app{showGraph: true, status: gitrepo.Status{Head: "hidden", Entries: []gitrepo.Entry{{Path: "a", Worktree: 'M'}}}}
	g := newCommitGraph(commits, nil)
	link, ok := a.worktreeGraphConnection(commits, g)
	if !ok || link.HeadRow != -1 {
		t.Fatalf("hidden HEAD retargeted: %+v", link)
	}
	if points := link.rowPoints(g, 0, ui.Rect{Y: 50, H: 50}, 25); len(points) != 0 {
		t.Fatal("hidden HEAD connected to visible unrelated commit")
	}
	a.status.Head = ""
	link, ok = a.worktreeGraphConnection(nil, nil)
	if !ok || link.HasHead {
		t.Fatal("unborn worktree fabricated a parent")
	}
}

// The rendered lane needs actual alternating stroke/gap pixels. A blank graph
// placeholder or a solid edge must fail this test independently of the model.
func TestWorkingChangesPaintDashedLane(t *testing.T) {
	a := &app{showGraph: true, status: gitrepo.Status{Head: "hidden", Entries: []gitrepo.Entry{{Path: "a", Worktree: 'M'}}}}
	tt := pageTester(a, func(c *ui.Context) { a.historyWorktreeRow(c, 0, 1) })
	row, ok := tt.AccessByRole(ui.RoleOption, a.changesLabel(1))
	if !ok {
		t.Fatal("missing worktree row")
	}
	img := tt.Image()
	x := int(row.Bounds.X + 16)
	y0 := int(row.Bounds.Y + row.Bounds.H/2 + 8)
	y1 := int(row.Bounds.Y + row.Bounds.H - 1)
	changes := 0
	for y := y0 + 1; y < y1; y++ {
		if img.RGBAAt(x, y) != img.RGBAAt(x, y-1) {
			changes++
		}
	}
	if changes < 2 {
		t.Fatalf("working lane has no dashed stroke/gap pattern: %d transitions", changes)
	}
}
