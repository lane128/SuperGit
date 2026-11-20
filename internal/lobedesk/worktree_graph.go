package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type worktreeGraphLink struct {
	HeadRow int // -1 when hidden, len(rows) when HEAD is beyond the loaded page.
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

// dashBottom clips the connection to this row, ending at the centre of HEAD.
func (link worktreeGraphLink) dashBottom(row int, r ui.Rect) float32 {
	switch {
	case row > link.HeadRow:
		return r.Y
	case row == link.HeadRow:
		return r.Y + r.H/2
	default:
		return r.Y + r.H
	}
}

func (a *app) strokeWorktreeDash(p *ui.Painter, x, top, bottom, width float32, ink ui.Color) {
	if top >= bottom {
		return
	}
	// Keep the dash phase continuous across row and stash boundaries.
	path := (&ui.Path{}).MoveTo(x, top).LineTo(x, bottom)
	p.StrokePath(path.Dash(top+a.historyScroll.Y, 3, 3), width, ink)
}

// Stashes and their error banner sit between Changes and the first commit.
// Leave room beside their content for the connection through the entire row.
func (a *app) continueWorktreeThrough(row ui.Element, tk *theme.Tokens, padX, padY float32) {
	graph := a.historyGraph
	if graph == nil || graph.Worktree == nil || graph.Worktree.HeadRow < 0 {
		return
	}
	row.Padding(padY, padX, padY, historyRowPadding+graphLaneGap+8)
	row.Draw(func(p *ui.Painter, r ui.Rect) {
		left := r.X + historyRowPadding
		x := laneX(int(left-a.historyGraphScroll.X), 0)
		if x >= left {
			line, _ := a.graphStyle()
			a.strokeWorktreeDash(p, x, r.Y, r.Y+r.H, line, graphInk(tk, graph.LaneColor(graph.Worktree.Color)))
		}
	})
}

func (a *app) drawWorktreeNode(p *ui.Painter, r ui.Rect, tk *theme.Tokens, link worktreeGraphLink) {
	ink := graphInk(tk, laneColor(a.lanePalette(tk), link.Color))
	if a.historyGraph != nil {
		ink = graphInk(tk, a.historyGraph.LaneColor(link.Color))
	}
	line, _ := a.graphStyle()
	x, y := laneX(int(r.X), 0), r.Y+r.H/2
	// An unborn repository has a standalone square and no invented parent.
	if link.HasHead {
		a.strokeWorktreeDash(p, x, y+5, r.Y+r.H, line, ink)
	}
	p.Fill(ui.Rect{X: x - 4, Y: y - 4, W: 8, H: 8}, ink, 1)
}
