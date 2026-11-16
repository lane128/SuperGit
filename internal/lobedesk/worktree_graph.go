package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// A separate temporary lane prevents an edge to an older HEAD from passing
// through newer commits on the same real lane. When HEAD is the first commit,
// the working tree shares its lane instead. It is never a fake Commit row.
const worktreeGraphGutter float32 = 20

type worktreeGraphLink struct {
	HeadRow int
	HasHead bool
	Color   int
}

func (a *app) worktreeGraphConnection(commits []gitrepo.Commit, graph *commitGraph) (worktreeGraphLink, bool) {
	link := worktreeGraphLink{HeadRow: -1, HasHead: a.status.Head != ""}
	if !a.showGraph || len(a.status.Entries) == 0 {
		return link, false
	}
	if graph != nil && link.HasHead {
		for i, commit := range commits {
			if commit.SHA == a.status.Head && i < len(graph.Rows) {
				link.HeadRow = i
				link.Color = graph.Rows[i].Color
				break
			}
		}
	}
	return link, true
}

func (g *commitGraph) worktreeGutter() float32 {
	if g != nil && g.Worktree != nil {
		return worktreeGraphGutter
	}
	return 0
}

func (link worktreeGraphLink) nodeX(graph *commitGraph, x float32) float32 {
	if link.HeadRow == 0 && graph != nil && len(graph.Rows) > 0 {
		return laneXGap(int(x+graph.worktreeGutter()), graph.Rows[0].Lane, graph.LaneGap())
	}
	return x + 8
}

func (link worktreeGraphLink) rowPoints(graph *commitGraph, row int, r ui.Rect, sourceY float32) [][2]float32 {
	if graph == nil || link.HeadRow < 0 || row > link.HeadRow || link.HeadRow >= len(graph.Rows) {
		return nil
	}
	parent := graph.Rows[link.HeadRow]
	parentX := laneXGap(int(r.X+worktreeGraphGutter), parent.Lane, graph.LaneGap())
	parentY := r.Y + r.H/2 + float32(link.HeadRow-row)*r.H
	if r.H <= 0 || parentY <= sourceY {
		return nil
	}
	sourceX := link.nodeX(graph, r.X)
	// Stay in the dedicated lane until the parent's upper half. The pinned
	// row and scrolling list have padding between them, so two full-row bends
	// can overlap and double back when their centers are just over a row apart.
	bendY := max(sourceY, parentY-r.H/2)
	var points [][2]float32
	if lo, hi := max(sourceY, r.Y), min(bendY, r.Y+r.H); lo < hi {
		points = append(points, [2]float32{sourceX, lo}, [2]float32{sourceX, hi})
	}
	return append(points, graphSwoopRange(sourceX, bendY, parentX, parentY, r.Y, r.Y+r.H)...)
}

func strokeWorktreeDash(p *ui.Painter, points [][2]float32, width float32, ink ui.Color) {
	if len(points) < 2 {
		return
	}
	path := (&ui.Path{}).MoveTo(points[0][0], points[0][1])
	for _, point := range points[1:] {
		path.LineTo(point[0], point[1])
	}
	p.StrokePath(path.Dash(0, 3, 3), width, ink)
}

func (a *app) drawWorktreeNode(p *ui.Painter, r ui.Rect, tk *theme.Tokens, link worktreeGraphLink) {
	a.worktreeGraphBounds = r
	ink := graphInk(tk, laneColor(a.lanePalette(tk), link.Color))
	if a.historyGraph != nil {
		ink = graphInk(tk, a.historyGraph.LaneColor(link.Color))
	}
	line, _ := a.graphStyle()
	x, y := link.nodeX(a.historyGraph, r.X), r.Y+r.H/2
	// An unborn repository has a standalone square and no invented parent.
	if link.HasHead {
		strokeWorktreeDash(p, [][2]float32{{x, y + 5}, {x, r.Y + r.H}}, line, ink)
	}
	p.Fill(ui.Rect{X: x - 4, Y: y - 4, W: 8, H: 8}, ink, 1)
}
