package lobedesk

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type patchRows struct {
	lines   []gitrepo.DiffLine
	visible []int
	split   []gitrepo.SplitDiffRow
}

func newPatchRows(lines []gitrepo.DiffLine, visible []int, split bool) patchRows {
	rows := patchRows{lines: lines, visible: visible}
	if split {
		rows.split = gitrepo.SplitDiffRows(lines, visible)
	}
	return rows
}

func (rows patchRows) len() int {
	if rows.split != nil {
		return len(rows.split)
	}
	if rows.visible != nil {
		return len(rows.visible)
	}
	return len(rows.lines)
}

func (rows patchRows) at(i int) gitrepo.SplitDiffRow {
	if rows.split != nil {
		return rows.split[i]
	}
	if rows.visible != nil {
		i = rows.visible[i]
	}
	return gitrepo.SplitDiffRow{Shared: i, Left: -1, Right: -1}
}

func (rows patchRows) height(a *app, i int) float32 {
	if index := rows.at(i).Shared; index >= 0 && rows.lines[index].Kind == "hunk" {
		return diffHunkH
	}
	return a.patchRowHeight()
}

// appendSelectionOrder visits the visible cells in reading order while
// keeping their original patch indexes. Context appears once in a range.
func (rows patchRows) appendSelectionOrder(out []int, selectable func(string) bool) []int {
	for i := range rows.len() {
		row := rows.at(i)
		for _, index := range [...]int{row.Shared, row.Left, row.Right} {
			if index >= 0 && selectable(rows.lines[index].Kind) && (len(out) == 0 || out[len(out)-1] != index) {
				out = append(out, index)
			}
		}
	}
	return out
}

func patchSelectionRange(order []int, anchor, target int) map[int]bool {
	first, last := slices.Index(order, anchor), slices.Index(order, target)
	if first < 0 || last < 0 {
		return map[int]bool{target: true}
	}
	return selectionSet(order[min(first, last) : max(first, last)+1])
}

func rowContainsLine(row gitrepo.SplitDiffRow, index int) bool {
	return index >= 0 && (index == row.Shared || index == row.Left || index == row.Right)
}

type patchSide uint8

const (
	patchBoth patchSide = iota
	patchOld
	patchNew
)

func (side patchSide) gutterWidth() float32 {
	if side == patchBoth {
		return 92
	}
	return 50
}

func (a *app) patchGutter(c *ui.Context, line gitrepo.DiffLine, side patchSide, ink ui.Color) ui.Element {
	label := a.lineGutterLabel(line)
	if line.Kind == "context" && side != patchBoth {
		if side == patchOld {
			label = a.tf("oldDiffLine", line.Old)
		} else {
			label = a.tf("newDiffLine", line.New)
		}
	}
	button := ui.ButtonBase(c).Label(label).Row().Width(side.gutterWidth()).Height(a.patchRowHeight()).Shrink(0).
		Padding(0, 8).Gap(8).AlignItems(ui.Center)
	button.Children(func() {
		font := a.patchFontFamily()
		drawNumber := func(number int) {
			ui.Text(c, lineNumber(number)).Font(font).FontSize(scaledFont(10)).Width(34).Shrink(0).
				TextAlign(ui.End).TextColor(ink)
		}
		if side != patchNew {
			drawNumber(line.Old)
		}
		if side != patchOld {
			drawNumber(line.New)
		}
	})
	return button
}

type patchWidths struct {
	canvas, columns float32
}

func (a *app) paintPatchRows(c *ui.Context, rows patchRows, widths patchWidths, viewport ui.Rect, scroll *ui.ScrollState,
	heights []float32, reveal, pressed int, draw func(*ui.Context, int, float32, patchSide)) {
	if widths.columns == 0 {
		virtualPatchRows(c, viewport, scroll, heights, reveal, func(row *ui.Context, i int) {
			draw(row, rows.at(i).Shared, widths.canvas, patchBoth)
		}, pressed)
		return
	}
	tk := theme.TokensOf(c)
	half := widths.columns / 2
	empty := tk.FillQuaternary.Over(tk.BgContainer)
	// Keep each side in its own text scope. The columns share row heights and
	// the viewport, so dragging down one side selects only that side's text.
	ui.Row(c).Width(widths.canvas).Shrink(0).Children(func() {
		for _, side := range []patchSide{patchOld, patchNew} {
			column := c.Key(side)
			ui.Column(column).Width(half).Shrink(0).Selectable().Children(func() {
				virtualPatchRows(column, viewport, scroll, heights, reveal, func(cell *ui.Context, i int) {
					row := rows.at(i)
					if row.Shared >= 0 {
						// Shared headers span the left column's bounds. The right
						// column leaves a transparent placeholder at that height.
						if side == patchOld {
							ui.Box(cell).Width(widths.canvas).Height(heights[i]).Shrink(0).Selectable().Children(func() {
								draw(cell, row.Shared, widths.canvas, patchBoth)
							})
						} else {
							ui.Box(cell).Height(heights[i]).Shrink(0)
						}
						return
					}
					index, width := row.Left, half
					box := ui.Box(cell).Width(half).Height(heights[i]).Shrink(0).Clip().Background(empty)
					if side == patchNew {
						index, width = row.Right, half-1
						box.Padding(0, 0, 0, 1).Background(tk.BorderSecondary)
					}
					if index >= 0 {
						box.Children(func() { draw(cell, index, width, side) })
					} else if side == patchNew {
						box.Children(func() { ui.Box(cell).Fill().Background(empty) })
					}
				}, pressed)
			})
		}
	})
}

func splitPatchCodeLine(kind string) bool {
	return kind == "add" || kind == "del" || kind == "context"
}

const splitPatchReserve = diffCodeReserve - 92 + 50 + 1

func (a *app) splitPatchWidths(lines []gitrepo.DiffLine, viewport float32) patchWidths {
	var code, shared float32
	for _, line := range lines {
		advance := diffEstimatedAdvance(line.Text)
		if splitPatchCodeLine(line.Kind) {
			code = max(code, advance)
		} else {
			shared = max(shared, advance)
		}
	}
	font := a.patchFontSize()
	limit := 20000 * font / diffCodeSize
	half := max(140, min(code*font+splitPatchReserve, limit))
	columns := max(2*half, viewport)
	return patchWidths{canvas: max(columns, min(shared*font+diffCodeReserve, limit)), columns: columns}
}

// patchOffset keeps the source line at the top of the pane when paired
// replacements change the height of the rows above it.
func (a *app) patchOffset(from, to patchRows, y float32) float32 {
	if y <= 0 {
		return y
	}
	top := float32(0)
	for i := range from.len() {
		height := from.height(a, i)
		if y < top+height {
			row := from.at(i)
			index := max(row.Shared, row.Left)
			if index < 0 {
				index = row.Right
			}
			newTop := float32(0)
			for j := range to.len() {
				if rowContainsLine(to.at(j), index) {
					return newTop + y - top
				}
				newTop += to.height(a, j)
			}
			break
		}
		top += height
	}
	return y
}

func (a *app) patchLayoutButton(c *ui.Context, preservePosition func()) {
	icon, label := lucide.Columns2(), a.t("showSplitDiff")
	if a.splitDiff {
		icon, label = lucide.Rows2(), a.t("showUnifiedDiff")
	}
	if workbenchIconButton(c, icon, label, buttonNormal, false).Shrink(0).Clicked() {
		if preservePosition != nil {
			preservePosition()
		}
		a.splitDiff = !a.splitDiff
		a.lineReleasePress()
		a.commitReleasePress()
		a.diffScroll.X = 0
		if patch := a.detailTabs().PatchScroll[a.detailTabs().Active]; patch != nil {
			patch.X = 0
		}
		a.revealLine = a.lineCursor >= 0 && a.lineSel[a.lineCursor]
		a.commitReveal = a.commitCursor >= 0 && a.commitSel[a.commitCursor]
		a.invalidate()
	}
}
