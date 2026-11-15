package desktop

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"supergit/internal/gitrepo"
)

func (a *app) diffView(c *ui.Context, d gitrepo.Diff, path string, staged, history bool) {
	if history && a.historyDiffReset {
		a.historyDiffList = ui.ListState{}
		a.historyDiffReset = false
	}
	if !history && a.workspaceDiffReset {
		a.diffList = ui.ListState{}
		a.workspaceDiffReset = false
	}
	t := c.Theme()
	ui.Column(c).Fill().Clip().Children(func() {
		ui.Row(c).FillWidth().Height(51).Shrink(0).Padding(12).Gap(12).AlignItems(ui.Center).Border(1, t.Border).Children(func() {
			title := "选择文件查看 Diff"
			if path != "" {
				title = oneLine(path)
			}
			ui.Text(c, title).Bold().Grow(1).MinWidth(0).Tooltip(path)
			group := "Index → 工作区"
			if staged {
				group = "HEAD → Index"
			}
			if history {
				group = "提交差异 · 只读"
			}
			if path != "" {
				ui.Text(c, group).FontSize(11).TextColor(t.TextMuted)
			}
		})
		ui.Row(c).FillWidth().Shrink(0).Padding(6, 12).Gap(8).Wrap().AlignItems(ui.Center).Border(1, t.Border).Children(func() {
			if ui.Button(c, "统一视图").Disabled(!a.splitDiff).Clicked() {
				a.setDiffPresentation(false)
			}
			if ui.Button(c, "左右视图").Disabled(a.splitDiff).Clicked() {
				a.setDiffPresentation(true)
			}
			if !history && d.CanStageHunks && a.hunksAllowed() {
				count := 0
				for _, on := range a.selectedLines {
					if on {
						count++
					}
				}
				ui.Textf(c, "已选 %d 行", count).FontSize(12).TextColor(t.TextMuted)
				label := "暂存选中行"
				if staged {
					label = "取消暂存选中行"
				}
				if ui.PrimaryButton(c, label).Disabled(!a.linesAllowed(history) || count == 0).Clicked() {
					if target, ok := a.captureLines(history); ok {
						a.mutate(label, func(ctx context.Context, r *gitrepo.Repository) error {
							return r.ApplyLines(ctx, target.Path, target.Diff, target.Selection, target.Reverse)
						}, false)
					}
				}
				if ui.Button(c, "清除选择").Disabled(count == 0).Clicked() {
					a.clearLineSelection()
				}
			}
		})
		waiting := a.diffLoading
		if history {
			waiting = a.commitLoading
		}
		if waiting && (history || d.Text == "") {
			ui.Text(c, "正在读取差异…").Padding(24).TextColor(t.TextMuted)
			return
		}
		if path == "" {
			ui.Text(c, "从文件列表选择一项，查看内容与片段。").Padding(24).TextColor(t.TextMuted)
			return
		}
		if !history && (!a.diffLoaded || a.diffErr != nil) {
			if a.diffErr != nil {
				ui.Text(c, "工作区 Diff 读取失败："+a.diffErr.Error()).Padding(12).TextColor(t.Danger).FillWidth()
			} else {
				ui.Text(c, "工作区 Diff 尚未读取完成。").Padding(12).TextColor(t.TextMuted)
			}
			if ui.Button(c, "重试读取工作区 Diff").Disabled(a.guard.writing || a.opening || a.closed).Clicked() {
				a.loadDiff(path, staged)
			}
			return
		}
		if history && (!a.commitDiffLoaded || a.commitDiffErr != nil) {
			if a.commitDiffErr != nil {
				ui.Text(c, "提交 Diff 读取失败："+a.commitDiffErr.Error()).Padding(12).TextColor(t.Danger).FillWidth()
			} else {
				ui.Text(c, "提交 Diff 尚未读取完成。").Padding(12).TextColor(t.TextMuted)
			}
			if ui.Button(c, "重试读取提交 Diff").Disabled(a.guard.writing || a.opening || a.closed).Clicked() {
				a.showCommitFile(path)
			}
			return
		}
		if d.Text == "" {
			ui.Text(c, "没有文本差异。子模块内部的改动可进入子模块查看。").Padding(24).TextColor(t.TextMuted)
			return
		}
		if !history && !d.CanStageHunks {
			ui.Text(c, "此差异仅支持文件级操作。").Padding(6, 12).FontSize(11).TextColor(t.TextMuted)
		}
		list := &a.diffList
		if history {
			list = &a.historyDiffList
		}
		ui.ScrollHorizontal(c.Key(path + fmt.Sprint(staged, history, a.splitDiff))).Grow(1).FillWidth().Children(func() {
			width := float32(800)
			for _, line := range d.Lines {
				width = max(width, float32(len([]rune(line.Text)))*a.fontSize*.65+170)
			}
			width = min(width, 24000)
			if !a.splitDiff {
				ui.List(c.Key("unified"), list, len(d.Lines), func(i int) {
					if d.Lines[i].Kind == "hunk" {
						a.diffSharedRow(c, d, i, staged, history)
					} else {
						a.diffCell(c, d, i, false, true, history, width)
					}
				}).Width(width).FillHeight()
			} else {
				rows := splitDiffRows(d)
				// Both halves share one vertical list and horizontal scroll position.
				half := max(float32(420), width/2)
				for _, line := range d.Lines {
					if line.Kind == "add" || line.Kind == "del" || line.Kind == "context" {
						half = max(half, min(float32(12000), float32(len([]rune(line.Text)))*a.fontSize*.65+100))
					}
				}
				ui.List(c.Key("split"), list, len(rows), func(i int) {
					row := rows[i]
					if row.Shared >= 0 {
						a.diffSharedRow(c, d, row.Shared, staged, history)
						return
					}
					ui.Row(c).Width(half * 2).Height(27).Children(func() {
						a.diffCell(c.Key("left"), d, row.Left, true, false, history, half)
						a.diffCell(c.Key("right"), d, row.Right, false, false, history, half)
					})
				}).Width(half * 2).FillHeight()
			}
		})
	})
}
func (a *app) setDiffPresentation(split bool) {
	a.splitDiff = split
	a.diffList = ui.ListState{}
	a.historyDiffList = ui.ListState{}
}
func (a *app) diffSharedRow(c *ui.Context, d gitrepo.Diff, i int, staged, history bool) {
	t := c.Theme()
	line := d.Lines[i]
	ui.Row(c).Height(27).FillWidth().Padding(2, 8).Gap(8).Background(t.Accent.Alpha(.08).Over(t.Background)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, line.Text).Font("Menlo").FontSize(a.fontSize).Grow(1).Selectable()
		if line.Kind == "hunk" && !history && d.CanStageHunks && a.hunksAllowed() {
			label := "暂存片段"
			if staged {
				label = "取消暂存片段"
			}
			if ui.Button(c.Key(i), label).FontSize(11).Disabled(a.busy()).Clicked() {
				// Hunk selection uses the same stale-Diff guard as individual lines.
				var selection []int
				for idx, l := range d.Lines {
					if l.Hunk == line.Hunk && (l.Kind == "add" || l.Kind == "del") {
						selection = append(selection, idx)
					}
				}
				path := a.path
				a.mutate(label, func(ctx context.Context, r *gitrepo.Repository) error {
					return r.ApplyLines(ctx, path, d, selection, staged)
				}, false)
			}
		}
	})
}
func (a *app) diffCell(c *ui.Context, d gitrepo.Diff, i int, left, unified, history bool, width float32) {
	t := c.Theme()
	bg := t.Background
	line := gitrepo.DiffLine{}
	if i >= 0 {
		line = d.Lines[i]
		switch line.Kind {
		case "add":
			bg = t.Success.Alpha(.10).Over(bg)
		case "del":
			bg = t.Danger.Alpha(.10).Over(bg)
		}
		if !history && a.selectedLines[i] {
			bg = t.Accent.Alpha(.19).Over(bg)
		}
	}
	ui.Row(c).Width(width).Height(27).Shrink(0).Padding(2, 8).Gap(6).Background(bg).AlignItems(ui.Center).Children(func() {
		if i < 0 {
			return
		}
		if !history && d.CanStageHunks && a.hunksAllowed() {
			if line.Kind == "add" || line.Kind == "del" {
				on := a.selectedLines[i]
				side, num := "新", line.New
				if line.Kind == "del" {
					side, num = "旧", line.Old
				}
				if ui.Checkbox(c.Key(i), &on, "").Label(fmt.Sprintf("选择%s行 %d", side, num)).Width(24).Disabled(!a.linesAllowed(history)).Changed() {
					if a.selectedLines == nil {
						a.selectedLines = make(map[int]bool)
					}
					if on {
						a.selectedLines[i] = true
					} else {
						delete(a.selectedLines, i)
					}
				}
			} else {
				ui.Box(c).Width(24).Shrink(0)
			}
		}
		nums := ""
		numberWidth := float32(40)
		if unified {
			nums = fmt.Sprintf("%4s %4s", diffNumber(line.Old), diffNumber(line.New))
			numberWidth = 78
		} else if left {
			nums = diffNumber(line.Old)
		} else {
			nums = diffNumber(line.New)
		}
		ui.Text(c, nums).Font("Menlo").FontSize(11).Width(numberWidth).Shrink(0).TextColor(t.TextMuted)
		ui.Text(c, strings.ReplaceAll(line.Text, "\t", "    ")).Font("Menlo").FontSize(a.fontSize).Selectable().Grow(1)
	})
}
func diffNumber(n int) string {
	if n > 0 {
		return fmt.Sprint(n)
	}
	return ""
}
