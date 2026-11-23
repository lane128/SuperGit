package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestHistoryReservesLeftLaneForCheckedOutLine(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "side-merge", Parents: []string{"side", "head"}},
		{SHA: "side", Parents: []string{"head"}},
		{SHA: "sibling", Parents: []string{"head"}},
		{SHA: "head", Refs: "HEAD -> main", Parents: []string{"base", "topic"}},
		{SHA: "topic", Parents: []string{"base"}},
		{SHA: "base", Parents: []string{"root"}},
		{SHA: "root"},
		{SHA: "unrelated"},
	}
	a := &app{showGraph: true, commits: commits, status: gitrepo.Status{Head: "head", Entries: []gitrepo.Entry{{Path: "dirty", Index: '.', Worktree: 'M'}}}}
	graph := a.cachedCommitGraph(commits, lanePalette, 80)
	for i, row := range graph.Rows {
		mainline := i == 3 || i == 5 || i == 6
		if (row.Lane == 0) != mainline {
			t.Errorf("%s uses lane %d (mainline %t)", commits[i].SHA, row.Lane, mainline)
		}
	}
	for _, edge := range graph.edges {
		if edge.ViaLane == 0 && (edge.SourceLane != 0 || edge.ParentLane != 0 || edge.Merge) {
			t.Errorf("branch edge occupies mainline: %+v", edge)
		}
	}
	if len(graph.edges) != 8 {
		t.Fatalf("lost parent links: got %d", len(graph.edges))
	}
	if graph.Worktree == nil || graph.Worktree.HeadRow != 3 || graph.Worktree.Color != graph.Rows[3].Color {
		t.Fatalf("worktree does not share HEAD's lane color and endpoint: %+v", graph.Worktree)
	}
	rect := ui.Rect{Y: 100, H: 44}
	for row, want := range []float32{144, 144, 144, 122, 100} {
		if bottom := graph.Worktree.dashBottom(row, rect); bottom != want {
			t.Errorf("row %d dash ends at %g, want %g", row, bottom, want)
		}
	}
}

func TestWorkingChangesUseDashedHeadConnection(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "other", Parents: []string{"actual"}}, {SHA: "actual"}}
	a := &app{showGraph: true, status: gitrepo.Status{Head: "actual", Entries: []gitrepo.Entry{{Path: "a", Worktree: 'M'}}}}
	g := newCommitGraph(a.status.Head, commits, nil)
	link, ok := a.worktreeGraphConnection(commits, g)
	if !ok || link.HeadRow != 1 {
		t.Fatalf("worktree does not target actual HEAD: %+v visible=%v", link, ok)
	}
	if g.Rows[0].Lane <= g.Rows[1].Lane || g.Rows[1].Lane != 0 {
		t.Fatalf("child did not fork to the right of HEAD: %+v", g.Rows)
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

func TestStashesJoinHistoryAtTheirBase(t *testing.T) {
	a := &app{stashes: []gitrepo.Stash{
		{SHA: "stash2", Ref: "stash@{0}", Date: "2026-10-05T12:00:00Z", ParentSHAs: "base index untracked"},
		{SHA: "stash1", Ref: "stash@{1}", Date: "2026-10-01T12:00:00Z", ParentSHAs: "base older-index"},
	}}
	commits := []gitrepo.Commit{
		{SHA: "tip", Date: "2026-10-07T12:00:00Z", Parents: []string{"base"}},
		{SHA: "base", Date: "2026-10-03T12:00:00Z"},
	}
	rows := a.historyWithStashes(commits)
	if len(rows) != 4 || rows[0].SHA != "tip" || rows[1].SHA != "stash2" || rows[2].SHA != "stash1" || rows[3].SHA != "base" {
		t.Fatalf("stash insertion reordered history or placed a stash below its base: %+v", rows)
	}
	g := newCommitGraph("tip", rows, nil)
	for i := 1; i <= 2; i++ {
		if len(rows[i].Parents) != 1 || rows[i].Parents[0] != "base" || g.Rows[i].Lane <= g.Rows[0].Lane {
			t.Fatalf("invalid stash lane or graph parents: %+v in %+v", rows[i], g.Rows)
		}
		connected := false
		for edge := range g.spans(i) {
			if edge.Start == i && edge.End == 3 {
				connected = true
			}
		}
		if !connected {
			t.Fatalf("stash %s has no edge to base", rows[i].SHA)
		}
	}
}

// Check the native painter as well as the lane numbers. The column through
// a stash and a newer side tip must show both ink and gaps at the same x.
func TestWorkingChangesPaintStraightThroughStashAndSideTip(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "child", Subject: "Side", Parents: []string{"actual"}}, {SHA: "actual", Subject: "Main"}}
	a := &app{showGraph: true, commits: commits, stashes: []gitrepo.Stash{{SHA: "stash", Message: "Saved work"}},
		status: gitrepo.Status{Head: "actual", Entries: []gitrepo.Entry{{Path: "dirty", Index: '.', Worktree: 'M'}}}}
	tt := pageTester(a, func(c *ui.Context) { a.commitList(c) })
	img := tt.Image()
	for _, name := range []string{"Saved work", "Side"} {
		item, ok := tt.AccessByRole(ui.RoleOption, name)
		if !ok {
			t.Fatalf("missing row %s", name)
		}
		r := item.Bounds
		x := int(laneX(int(r.X+historyRowPadding), 0))
		y0, y1 := int(r.Y)+2, int(r.Y+r.H)-2
		ink, gaps := 0, 0
		for y := y0; y < y1; y++ {
			if img.RGBAAt(x, y) != img.RGBAAt(x-6, y) {
				ink++
			} else {
				gaps++
			}
		}
		if ink < 2 || gaps < 2 {
			t.Errorf("mainline through %q: %d ink and %d gap pixels", name, ink, gaps)
		}
	}
}

func TestWorkingChangesMarkerAlignsWithFirstHeadLane(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "actual"}}
	a := &app{showGraph: true, status: gitrepo.Status{Head: "actual", Entries: []gitrepo.Entry{{Path: "a", Worktree: 'M'}}}}
	graph := newCommitGraph(a.status.Head, commits, nil)
	link, _ := a.worktreeGraphConnection(commits, graph)
	graph.Worktree = &link
	a.historyGraph = graph
	tt := pageTester(a, func(c *ui.Context) { a.historyWorktreeRow(c, 0, 2) })
	row, ok := tt.AccessByRole(ui.RoleOption, a.changesLabel(1))
	if !ok {
		t.Fatal("missing worktree row")
	}
	r := row.Bounds
	x := int(laneXGap(int(r.X+historyRowPadding), graph.Rows[0].Lane, graph.LaneGap()))
	y := int(r.Y + r.H/2)
	img := tt.Image()
	if img.RGBAAt(x, y) == img.RGBAAt(x+6, y) {
		t.Fatalf("working tree marker is missing from HEAD lane x=%d", x)
	}
}

func TestHiddenHeadDoesNotConnectToFirstVisibleCommit(t *testing.T) {
	commits := []gitrepo.Commit{{SHA: "unrelated"}}
	a := &app{showGraph: true, status: gitrepo.Status{Head: "hidden", Entries: []gitrepo.Entry{{Path: "a", Worktree: 'M'}}}}
	g := newCommitGraph(a.status.Head, commits, nil)
	link, ok := a.worktreeGraphConnection(commits, g)
	if !ok || link.HeadRow != -1 {
		t.Fatalf("hidden HEAD retargeted: %+v", link)
	}
	if link.dashBottom(0, ui.Rect{Y: 50, H: 50}) != 50 {
		t.Fatal("hidden HEAD connected to visible unrelated commit")
	}
	a.status.Head = ""
	link, ok = a.worktreeGraphConnection(nil, nil)
	if !ok || link.HasHead {
		t.Fatal("unborn worktree fabricated a parent")
	}
}

func TestLeftLaneContinuesAcrossPageUntilHeadArrives(t *testing.T) {
	a := &app{showGraph: true, historyMore: true, historyTips: []string{"main", "child"},
		status: gitrepo.Status{Head: "main", Branch: "main", Entries: []gitrepo.Entry{{Path: "dirty", Index: '.', Worktree: 'M'}}}}
	page := []gitrepo.Commit{{SHA: "child", Parents: []string{"main"}}}
	rect := ui.Rect{Y: 50, H: 44}
	draw := func() *commitGraph { return a.cachedCommitGraph(page, lanePalette, 80) }
	if g := draw(); g.Rows[0].Lane == 0 || g.Worktree.dashBottom(0, rect) != 94 {
		t.Fatal("the pending HEAD lost its lane at the page boundary")
	}
	a.historyQuery = "child"
	if link := draw().Worktree; link.HeadRow >= 0 || link.dashBottom(0, rect) != 50 {
		t.Fatal("filtered history drew a connection to a missing HEAD")
	}
	a.historyQuery, a.historyMore = "", false
	page = append(page, gitrepo.Commit{SHA: "main"})
	if g := draw(); g.Rows[0].Lane == 0 || g.Rows[1].Lane != 0 || g.Worktree.dashBottom(1, rect) != 72 {
		t.Fatal("the next page did not end the same straight line at HEAD")
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
	x := int(laneX(int(row.Bounds.X+historyRowPadding), 0))
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
