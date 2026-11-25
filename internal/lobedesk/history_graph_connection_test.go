package lobedesk

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestSelectedHistoryNodeKeepsItsConnectingLine(t *testing.T) {
	for _, shape := range []string{"dot", "head", "merge", "fork", "stash"} {
		for _, node := range []float32{graphNodeSmall, graphNodeStandard, graphNodeLarge} {
			for _, line := range []float32{graphLineThin, graphLineStandard, graphLineThick} {
				t.Run(fmt.Sprintf("%s/%g/%g", shape, node, line), func(t *testing.T) {
					commits := []gitrepo.Commit{{SHA: "a", Subject: "Selected", Parents: []string{"b"}}, {SHA: "b", Subject: "Parent"}}
					a := &app{showGraph: true, selectedSHA: "a", historyDetailsHidden: true, historyNode: node, historyLine: line}
					if shape == "head" {
						a.status.Head = "a"
					}
					if shape == "merge" {
						commits[0].Parents = append(commits[0].Parents, "c")
						commits = append(commits, gitrepo.Commit{SHA: "c", Subject: "Other parent"})
					}
					var stash *gitrepo.Stash
					if shape == "stash" {
						stash = &gitrepo.Stash{SHA: "a", Ref: "stash@{0}"}
						a.selectedStash = "a"
					}
					var ground ui.Color
					view := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						tk := theme.TokensOf(c)
						ground = graphNodeGround(tk, true, false)
						g := newCommitGraph("", commits, a.lanePalette(tk))
						if shape == "fork" {
							g.Rows[0].Fork = true
						}
						a.historySplit.rowWidth, _ = c.Size()
						ui.Column(c).FillWidth().Children(func() {
							a.historyEntryRow(c, commits[0], stash, 0, 0, len(commits), g)
							for i := 1; i < len(commits); i++ {
								a.commitRow(c, commits[i], i, i, len(commits), g)
							}
						})
					}, 600, 180)
					view.SetScale(2)
					view.Frame()
					row, _ := view.AccessByRole(ui.RoleOption, "Selected")
					x := int((row.Bounds.X + historyRowPadding + laneXGap(0, 0, 20)) * 2)
					radius := graphNodeDot(node, line)
					if shape != "dot" {
						radius = graphNodeMerge(node, line)
					}
					y := int(math.Ceil(float64((row.Bounds.Y + row.Bounds.H/2 + radius + .5) * 2)))
					pixel := view.Image().RGBAAt(x, y)
					if got := ui.RGB(pixel.R, pixel.G, pixel.B); contrastRatio(got, ground) < 2 {
						t.Fatalf("selected halo erased line between node and ring: pixel=%v ground=%v at %d,%d", got, ground, x, y)
					}
				})
			}
		}
	}
}
