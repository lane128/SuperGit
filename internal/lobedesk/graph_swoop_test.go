package lobedesk

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// graphSwoop leaves and arrives vertically, so a branch line bends out of the
// node it leaves instead of stepping sideways at it.
func TestGraphSwoopKeepsBothEndsVertical(t *testing.T) {
	points := graphSwoop(10, 0, 24, 40)
	if first := points[0]; first[0] != 10 || first[1] != 0 {
		t.Fatalf("the swoop starts at %v; want the node (10, 0)", first)
	}
	if last := points[len(points)-1]; last[0] != 24 || last[1] != 40 {
		t.Fatalf("the swoop ends at %v; want the lane (24, 40)", last)
	}
	// The first and last steps move down without crossing much ground: that
	// is the vertical tangent. The middle of the swoop does the crossing.
	if step := points[1][0] - points[0][0]; step > 1 {
		t.Errorf("the swoop starts with a %.1f DIP sideways step; want it to drop first", step)
	}
	if step := points[len(points)-1][0] - points[len(points)-2][0]; step > 1 {
		t.Errorf("the swoop ends with a %.1f DIP sideways step; want it to arrive vertically", step)
	}
	if mid := points[len(points)/2]; mid[0] < 12 || mid[0] > 22 {
		t.Errorf("the swoop is at x=%.1f halfway down; want it between the lanes", mid[0])
	}
	for i := 1; i < len(points); i++ {
		if points[i][1] < points[i-1][1] {
			t.Fatalf("the swoop goes back up at point %d", i)
		}
	}
}

// An edge leaves its dot, bends into its lane over one whole row, and then
// runs straight in that lane, the way the reference graph's branch arc does.
// Spreading the change over the whole distance flattens it into a diagonal.
func TestAnEdgeReachesItsLaneWithinOneRow(t *testing.T) {
	// A five-row edge from lane 0 to lane 1 (one lane gap across, 250 DIP
	// down). 50 DIP is one row: by the middle row the line is in its lane.
	gap := float32(graphLaneGap)
	points := graphEdgePoints(0, 25, gap, gap, 275, 50, 125, 175)
	if x := xAtY(points, 150); x < gap-2 || x > gap+2 {
		t.Errorf("the edge is at x=%.1f halfway down; want about %.1f, in its lane", x, gap)
	}
	// It gets there through a bend: right below the dot it is still beside it.
	if x := xAtY(graphEdgePoints(0, 25, gap, gap, 275, 50, 0, 50), 30); x > 4 {
		t.Errorf("the edge is at x=%.1f just below its source; want it beside the dot", x)
	}
	// And it has arrived by the end of that one row, not later.
	if x := xAtY(graphEdgePoints(0, 25, gap, gap, 275, 50, 0, 75), 74); x < gap-2 {
		t.Errorf("the edge is at x=%.1f at the end of the first row; want its lane", x)
	}
}

// xAtY is the edge's x at a given y, interpolated between the sampled points.
func xAtY(points [][2]float32, y float32) float32 {
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		if b[1] < y {
			continue
		}
		if b[1] == a[1] {
			return b[0]
		}
		return a[0] + (b[0]-a[0])*(y-a[1])/(b[1]-a[1])
	}
	return points[len(points)-1][0]
}

// The default R1 treatment turns beside the node and reaches the vertical
// lane within the source row, matching the approved local-turn preview.
func TestHistoryDefaultBranchLineTurnsWithinSourceRow(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "a", Subject: "merge", Parents: []string{"b", "c"}},
		{SHA: "b", Subject: "side", Parents: []string{"d"}},
		{SHA: "c", Subject: "main", Parents: []string{"d"}},
		{SHA: "d", Subject: "base", Parents: []string{"e"}},
	}
	graph := newCommitGraph("", commits, lanePalette)
	var span commitGraphEdge
	found := false
	for edge := range graph.spans(0) {
		if edge.SourceLane != edge.ViaLane {
			span, found = edge, true
			break
		}
	}
	if !found {
		t.Fatal("the merge row has no branch span to test")
	}
	a := &app{locale: localeChinese, page: "history", commits: commits, showGraph: true}
	rowH := a.historyRowHeight()
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(400).Height(rowH * float32(len(commits))).Children(func() {
			for i, commit := range commits {
				a.commitRow(c, commit, i, 0, len(commits), graph)
			}
		})
	}, 400, int(rowH)*len(commits))
	tt.Frame()
	row, ok := tt.AccessByRole(ui.RoleOption, "merge")
	if !ok {
		t.Fatal("the merge row is not laid out")
	}
	// commitRow pads the row by 8 on the left; the graph column starts there.
	x0 := int(row.Bounds.X) + 8
	center := int(row.Bounds.Y + row.Bounds.H/2)
	img := tt.Image()
	background := img.RGBAAt(x0+2, center+3)
	if !lineAt(img, x0+int(laneX(0, span.SourceLane)), center+3, background) {
		t.Fatal("the test reads no line under the node; it cannot tell a bend from a step")
	}
	viaX := x0 + int(laneX(0, span.ViaLane))
	bottomEdge := int(row.Bounds.Y+row.Bounds.H) - 2
	if !lineAt(img, viaX, bottomEdge, background) {
		t.Errorf("the local turn has not reached lane %d by the bottom of the source row", span.ViaLane)
	}
}

func TestHistorySelectionPreservesUnrelatedLaneStroke(t *testing.T) {
	commits := []gitrepo.Commit{
		{SHA: "main-tip", Subject: "Main tip", Parents: []string{"main-base"}},
		{SHA: "topic-tip", Subject: "Topic tip", Parents: []string{"topic-base"}},
		{SHA: "topic-base", Subject: "Topic base"},
		{SHA: "main-base", Subject: "Main base"},
	}
	dense := append([]gitrepo.Commit{}, commits[:2]...)
	for lane := range 22 {
		dense = append(dense, gitrepo.Commit{SHA: fmt.Sprintf("tip-%d", lane), Parents: []string{fmt.Sprintf("base-%d", lane)}})
	}
	for lane := range 22 {
		dense = append(dense, gitrepo.Commit{SHA: fmt.Sprintf("base-%d", lane)})
	}
	dense = append(dense, commits[2:]...)
	for _, mode := range []struct {
		name  string
		dark  bool
		width int
		dense bool
	}{
		{"light", false, 980, false},
		{"dark", true, 980, false},
		{"dense-wide", false, 2000, true},
		{"dense-narrow-light", false, 980, true},
		{"dense-narrow-dark", true, 980, true},
	} {
		t.Run(mode.name, func(t *testing.T) {
			a := &app{page: "history", locale: localeEnglish, showGraph: true, commits: commits, historyDetailsHidden: true}
			lineWidth := graphLineStandard
			if mode.dense {
				a.commits, a.historyNode, a.historyLine = dense, graphNodeLarge, graphLineThick
				lineWidth = graphLineThick
			}
			a.historySplit.rowWidth = float32(mode.width)
			var mainInk ui.Color
			view := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.commitList(c)
				mainInk = graphInk(theme.TokensOf(c), a.historyGraph.LaneColor(a.historyGraph.Rows[0].Color))
			}, mode.width, 350)
			const scale = 2
			view.SetScale(scale)
			view.SetDark(mode.dark)
			view.Frame()
			row, ok := view.AccessByRole(ui.RoleOption, "Topic tip")
			if !ok || row.Offscreen {
				t.Fatal("topic row is not visible")
			}
			rows := a.historyGraph.Rows
			if rows[0].Lane != 0 || rows[1].Lane != 1 || rows[len(rows)-1].Lane != 0 {
				t.Fatal("expected the main edge to pass through the topic row on a separate lane")
			}
			bounds := row.Bounds
			x := int((bounds.X + historyRowPadding + laneX(0, 0)) * scale)
			centerY := bounds.Y + bounds.H/2
			y := int(centerY * scale)
			ink := color.RGBA{R: mainInk.R, G: mainInk.G, B: mainInk.B, A: mainInk.A}
			strokeWidth := func() int {
				img := view.Image()
				width := 0
				for px := x - 8; px <= x+8; px++ {
					if img.RGBAAt(px, y) == ink {
						width++
					}
				}
				return width
			}
			before := strokeWidth()
			if before != int(lineWidth*scale) {
				t.Fatalf("expected to sample the unselected stroke, got %d ink pixels", before)
			}
			view.ClickAt(bounds.X+bounds.W-30, centerY)
			view.Move(float32(mode.width-22), 340)
			view.Frame()
			if a.selectedSHA != commits[1].SHA {
				t.Fatalf("row click selected %q, want topic tip", a.selectedSHA)
			}
			if after := strokeWidth(); before != after {
				t.Fatalf("selecting the topic row changed the unrelated main stroke from %d to %d pixels", before, after)
			}
		})
	}
}

// lineAt reports whether a pixel is drawn ink rather than the ground behind
// it, so the test reads a line without knowing which palette slot it wears.
func lineAt(img *image.RGBA, x, y int, background color.RGBA) bool {
	got := img.RGBAAt(x, y)
	diff := absInt(int(got.R)-int(background.R)) + absInt(int(got.G)-int(background.G)) + absInt(int(got.B)-int(background.B))
	return diff > 30
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
