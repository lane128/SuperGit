package lobedesk

import (
	"fmt"
	"image/png"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func manyLaneHistory(n int) []gitrepo.Commit {
	commits := make([]gitrepo.Commit, 2*n)
	for i := 0; i < n; i++ {
		commits[i] = gitrepo.Commit{SHA: fmt.Sprint(i), Subject: "A readable commit title", Author: "Fixture", Parents: []string{fmt.Sprint(i + n)}}
		commits[i+n] = gitrepo.Commit{SHA: fmt.Sprint(i + n)}
	}
	return commits
}

func TestGraphViewportKeepsSelectedNodeVisibleWhenNarrowed(t *testing.T) {
	a := &app{showGraph: true, commits: manyLaneHistory(64), locale: localeEnglish, historyNode: graphNodeLarge}
	const selected = 8
	a.selectedSHA = a.commits[selected].SHA
	rowH := a.historyRowHeight()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		windowWidth, _ := c.Size()
		budget := 80 + (windowWidth-980)*0.17
		a.historyGraph = a.cachedCommitGraph(a.commits, lanePalette, budget)
		ui.Column(c).Fill().Children(func() {
			a.historyWorktreeRow(c, 0, len(a.commits)+1)
			a.commitRow(c, a.commits[selected], selected, 1, len(a.commits)+1, a.historyGraph)
		})
	}, 1320, int(rowH*2))
	frame := func() {
		for range 4 {
			tt.Frame()
		}
	}
	nodeVisible := func() bool {
		graph := a.historyGraph
		x := laneXGap(0, graph.Rows[selected].Lane, graph.LaneGap()) - a.historyGraphScroll.X
		line, node := a.graphStyle()
		_, halo := graphNodeRing(node, line)
		return x >= halo && x+halo <= graph.Width
	}
	frame()
	if !nodeVisible() {
		t.Fatal("selected node must start fully visible")
	}
	tt.SetSize(980, int(rowH*2))
	frame()
	if !nodeVisible() || !tt.HasText(a.commits[selected].Subject) {
		t.Fatal("narrowing hid the selected node or commit subject")
	}
	if a.selectedSHA != a.commits[selected].SHA || a.revealSelection {
		t.Fatal("resizing unexpectedly changed or reselected the commit")
	}

	tt.Scroll(40, rowH/2, 210, 0)
	frame()
	pan := a.historyGraphScroll.X
	if nodeVisible() || pan <= 0 {
		t.Fatal("the user must be able to pan away from the selection")
	}
	tt.SetSize(1320, int(rowH*2))
	frame()
	if a.historyGraphScroll.X != pan {
		t.Fatal("resizing overrode the user's horizontal graph position")
	}
}

func TestHistoryGraphRetainsPanWhenRepositoryViewReturns(t *testing.T) {
	a := &app{showGraph: true, commits: manyLaneHistory(64), selectedSHA: "0", locale: localeEnglish}
	other := &app{}
	active := a
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		c = c.Key(active)
		if active == a {
			a.historyPage(c)
		} else {
			ui.Text(c, "Other repository")
		}
	}, 980, 640)
	frame := func() {
		for range 4 {
			tt.Frame()
		}
	}
	frame()
	rail, ok := tt.AccessByRole(ui.RoleScroll, a.t("scrollCommitGraph"))
	if !ok {
		t.Fatal("expected an overflowing Graph")
	}
	bounds := rail.Bounds
	tt.ClickAt(bounds.X+bounds.W-13, bounds.Y+bounds.H-6)
	frame()
	pan := a.historyGraphScroll.X
	if pan <= 0 {
		t.Fatal("expected the Graph to pan away from the selected node")
	}

	active = other
	frame()
	active = a
	frame()
	if got := a.historyGraphScroll.X; got != pan || a.selectedSHA != "0" {
		t.Fatalf("returning to the repository changed the Graph position from %g to %g or lost the selected commit (%q)", pan, got, a.selectedSHA)
	}
}

func TestHistoryClickRevealsSelectedCommitNode(t *testing.T) {
	a := &app{showGraph: true, commits: manyLaneHistory(64), locale: localeEnglish, historyNode: graphNodeLarge}
	const selected = 8
	const subject = "Target merge with a long title that should remain in its row"
	a.commits[selected].Subject = subject
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.historyGraph = a.cachedCommitGraph(a.commits, lanePalette, 80)
		ui.Column(c).Fill().Children(func() {
			a.historyWorktreeRow(c, 0, len(a.commits)+1)
			a.commitRow(c, a.commits[selected], selected, 1, len(a.commits)+1, a.historyGraph)
		})
	}, 310, int(a.historyRowHeight()*2))
	nodeVisible := func() bool {
		graph := a.historyGraph
		x := laneXGap(0, graph.Rows[selected].Lane, graph.LaneGap()) - a.historyGraphScroll.X
		line, node := a.graphStyle()
		_, halo := graphNodeRing(node, line)
		return x >= halo && x+halo <= graph.Width
	}
	tt.Frame()
	tt.Frame()
	tt.Scroll(40, a.historyRowHeight()/2, 600, 0)
	tt.Frame()
	if nodeVisible() || a.historyGraphScroll.X <= 0 {
		t.Fatal("fixture must first pan the target node out of view")
	}
	clickRepositoryTabsControl(t, tt, ui.RoleOption, subject)
	tt.Frame()
	if a.selectedSHA != a.commits[selected].SHA || !nodeVisible() {
		t.Fatalf("mouse selection did not reveal the target node: selected=%q, pan=%g", a.selectedSHA, a.historyGraphScroll.X)
	}
	tt.Scroll(40, a.historyRowHeight()/2, 600, 0)
	tt.Frame()
	if nodeVisible() {
		t.Fatal("graph must still allow deliberate panning away from the selected node")
	}
	clickRepositoryTabsControl(t, tt, ui.RoleOption, subject)
	tt.Frame()
	if !nodeVisible() || a.selectedSHA != a.commits[selected].SHA || a.revealSelection {
		t.Fatal("clicking the selected row should reveal only its graph node")
	}
}

func TestHistoryClickRevealsWorktreeNodeAfterGraphPan(t *testing.T) {
	a := &app{page: "history", showGraph: true, locale: localeEnglish, commits: manyLaneHistory(48)}
	a.status = gitrepo.Status{Head: a.commits[0].SHA, Entries: []gitrepo.Entry{{Path: "a.txt", Worktree: 'M'}}}
	a.selectedSHA = a.commits[0].SHA
	a.historySplit.rowWidth = 790
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Fill().Children(func() {
			a.commitList(c)
			ui.Box(c).Grow(1 - a.historyShare()).Basis(0)
		})
	}, 790, 570)
	view.Frame()
	view.Frame()
	list := a.historyViewport
	row, ok := view.AccessByRole(ui.RoleOption, a.changesLabel(1))
	if !ok || row.Offscreen {
		t.Fatal("working tree must start at the top of History")
	}
	view.Scroll(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2, 0, 220)
	view.Frame()
	row, ok = view.AccessByRole(ui.RoleOption, a.changesLabel(1))
	if a.historyScroll.Y < row.Bounds.H || ok && !row.Offscreen {
		t.Fatalf("wheel over Changes should scroll it away with the commits: offset=%g row=%+v", a.historyScroll.Y, row)
	}
	view.Scroll(list.X+list.W/2, list.Y+list.H/2, 0, -220)
	view.Frame()
	listOffset := a.historyScroll.Y
	for range 2 {
		rail, ok := view.AccessByRole(ui.RoleScroll, a.t("scrollCommitGraph"))
		if !ok || rail.Offscreen {
			t.Fatal("graph scroll control is not visible")
		}
		view.Scroll(rail.Bounds.X+rail.Bounds.W/2, rail.Bounds.Y+rail.Bounds.H/2, 250, 0)
		view.Frame()
		nodeLeft := laneX(0, 0) - 4
		if a.historyGraphScroll.X <= nodeLeft {
			t.Fatal("fixture must pan the worktree square out of view")
		}
		row, ok := view.AccessByRole(ui.RoleOption, a.changesLabel(1))
		if !ok || row.Offscreen {
			t.Fatal("worktree row did not return to the top")
		}
		view.ClickAt(row.Bounds.X+row.Bounds.W-20, row.Bounds.Y+row.Bounds.H/2)
		view.Frame()
		if !a.worktreeSelected || a.historyGraphScroll.X > nodeLeft {
			t.Fatalf("click did not reveal the worktree node: selected=%v pan=%g", a.worktreeSelected, a.historyGraphScroll.X)
		}
		if a.historyScroll.Y != listOffset {
			t.Fatalf("revealing the graph node changed the list offset from %g to %g", listOffset, a.historyScroll.Y)
		}
	}
}

func TestHistoryUpRevealsWorktreeNodeAfterGraphPan(t *testing.T) {
	a := &app{page: "history", showGraph: true, locale: localeEnglish, themeMode: "light",
		repo: &gitrepo.Repository{Root: t.TempDir()}, commits: manyLaneHistory(48), commitFilesLoaded: true}
	a.commits[0].Subject = "First commit"
	a.selectedSHA = a.commits[0].SHA
	a.selectedPath = "a.txt"
	a.status = gitrepo.Status{Head: a.selectedSHA, Entries: []gitrepo.Entry{{Path: a.selectedPath, Worktree: 'M'}}}
	view := ui.NewTester(a.root, 980, 640)
	view.Frame()
	clickRepositoryTabsControl(t, view, ui.RoleOption, a.commits[0].Subject)
	rail, ok := view.AccessByRole(ui.RoleScroll, a.t("scrollCommitGraph"))
	if !ok || rail.Offscreen || a.historyGraph.Worktree == nil {
		t.Fatal("fixture needs a scrollable graph with the worktree node")
	}
	view.Scroll(rail.Bounds.X+rail.Bounds.W/2, rail.Bounds.Y+rail.Bounds.H/2, 250, 0)
	view.Frame()
	nodeLeft := laneX(0, 0) - 4
	if a.historyGraphScroll.X <= nodeLeft {
		t.Fatal("fixture must pan the worktree square out of view")
	}
	list, listOffset := a.historyViewport, a.historyScroll.Y
	view.Key(0, ui.KeyUp)
	view.Frame()
	if !a.worktreeSelected || a.historyGraphScroll.X > nodeLeft {
		t.Fatalf("Up did not reveal the worktree square: selected=%v pan=%g", a.worktreeSelected, a.historyGraphScroll.X)
	}
	if a.historyViewport != list || a.historyScroll.Y != listOffset {
		t.Fatal("revealing the worktree node moved the commit list")
	}
}

func TestGraphHorizontalTrackShowsOnlyWithOverflow(t *testing.T) {
	a := &app{showGraph: true, locale: localeEnglish, selectedSHA: "selected"}
	wide := manyLaneHistory(64)
	rowH := a.historyRowHeight()
	for _, width := range []int{980, 1320} {
		for _, dark := range []bool{false, true} {
			commits := wide
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.historyGraph = newCommitGraph("", commits, lanePalette)
				a.historyGraph.Width = 80 + float32(width-980)*.12
				a.historyWorktreeRow(c, 0, len(commits)+1)
			}, width, int(rowH)+2)
			tt.SetDark(dark)
			tt.Frame()
			scroll, ok := tt.AccessByRole(ui.RoleScroll, a.t("scrollCommitGraph"))
			if !ok || a.historyGraphScroll.MaxX <= 0 {
				t.Fatal("overflow fixture has no graph scroller")
			}
			x := int(scroll.Bounds.X + scroll.Bounds.W - 13)
			y := int(scroll.Bounds.Y + scroll.Bounds.H - 6)
			image := tt.Image()
			if image.At(x, y) == image.At(x, y-9) {
				t.Fatalf("%d points, dark=%v: track beyond the thumb has no visible rail", width, dark)
			}
			tt.ClickAt(float32(x), float32(y))
			if a.historyGraphScroll.X <= 0 || a.worktreeSelected || a.selectedSHA != "selected" {
				t.Fatal("clicking the rail should pan the graph while preserving the selected commit")
			}
			commits = []gitrepo.Commit{{SHA: "selected"}}
			tt.Frame()
			if _, ok := tt.AccessByRole(ui.RoleScroll, a.t("scrollCommitGraph")); ok {
				t.Fatal("a graph that fits has no scroll control")
			}
		}
	}
}

func TestGraphViewportPansWithoutMovingTitle(t *testing.T) {
	a := &app{showGraph: true, commits: manyLaneHistory(64), locale: localeEnglish}
	rowH := a.historyRowHeight()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.historyGraph = a.cachedCommitGraph(a.commits, lanePalette, 80.6)
		ui.Column(c).Fill().Children(func() {
			a.historyWorktreeRow(c, 0, len(a.commits)+1)
			a.commitRow(c, a.commits[63], 63, 1, len(a.commits)+1, a.historyGraph)
		})
	}, 310, int(rowH*2))
	tt.Frame()
	tt.Frame()
	tt.Frame()
	if !tt.HasText("A readable commit title") {
		t.Fatal("graph displaced the commit title")
	}
	if a.historyGraphScroll.MaxX <= 0 {
		t.Fatal("overflow graph has no horizontal scroll range")
	}
	before := tt.Image()
	tt.Scroll(40, rowH/2, 600, 0)
	tt.Frame()
	tt.Frame()
	if a.historyGraphScroll.X <= 0 {
		t.Fatal("horizontal input did not pan graph lanes")
	}
	after := tt.Image()
	// Panning must change the graph pixels but leave the title and metadata still.
	graphChanged := false
	for y := int(rowH); y < int(rowH*2); y++ {
		for x := 0; x < 310; x++ {
			changed := before.RGBAAt(x, y) != after.RGBAAt(x, y)
			if x < 89 {
				graphChanged = graphChanged || changed
			} else if changed {
				t.Fatalf("graph panning changed text pixels at %d,%d", x, y)
			}
		}
	}
	if !graphChanged {
		t.Fatal("scroll state changed but rendered graph did not move")
	}
	if target := os.Getenv("SUPERGIT_GRAPH_SCREENSHOT"); target != "" {
		f, err := os.Create(target)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, after); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	a.selectedSHA, a.revealSelection = "63", true
	tt.Frame()
	tt.Frame()
	graph := a.historyGraph
	x := laneXGap(0, graph.Rows[63].Lane, graph.LaneGap()) - a.historyGraphScroll.X
	if x < 0 || x > graph.Width {
		t.Fatalf("selected node x=%v lies outside graph viewport", x)
	}
}
