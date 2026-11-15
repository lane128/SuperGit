package desktop

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"strings"
	"supergit/internal/gitrepo"
)

func (a *app) canSelectCommit(index int) bool {
	return !a.closed && a.repo != nil && !a.guard.writing && !a.opening && index >= 0 && index < len(a.data.history)
}

func (a *app) selectCommit(index int) {
	if !a.canSelectCommit(index) {
		return
	}
	commit := a.data.history[index]
	a.setSelectedCommit(commit)
	a.loadCommitMessage(commit.SHA)
	a.loadCommitFiles()
}
func (a *app) resetCommitDiff(path string) {
	a.commitPath = path
	a.commitDiff = gitrepo.Diff{}
	a.commitDiffLoaded, a.commitDiffErr = false, nil
	a.historyDiffReset = true
}
func (a *app) showCommitFile(path string) {
	if path == "" {
		return
	}
	req, ctx, ok := a.beginCommitPreview(path)
	if !ok {
		return
	}
	r, cancel := a.repo, a.commitCancel
	go func() {
		defer cancel()
		d, err := r.Show(ctx, req.sha, req.path)
		mygo.RunOnMain(func() {
			if a.finishCommitDiff(req, d, err) {
				a.invalidate()
			}
		})
	}()
}
func filterHistory(commits []gitrepo.Commit, query string) ([]gitrepo.Commit, []int) {
	query = strings.ToLower(strings.TrimSpace(query))
	var matches []gitrepo.Commit
	var indices []int
	for i, commit := range commits {
		if query == "" || strings.Contains(strings.ToLower(commit.SHA+" "+commit.Subject+" "+commit.Author+" "+commit.Refs), query) {
			matches = append(matches, commit)
			indices = append(indices, i)
		}
	}
	return matches, indices
}
func (a *app) historySearchChanged() { a.historyListReset = true }

func (a *app) historyLoadLabel() string {
	if a.historyPageLoading {
		return "正在加载历史…"
	}
	if a.data.historyErr != nil {
		return "历史读取失败，请刷新"
	}
	if !a.data.historyMore {
		return "已到历史末尾"
	}
	return "继续加载 300 条"
}

func (a *app) history(c *ui.Context) {
	if a.historyListReset {
		a.historyList = ui.ListState{}
		a.historyListReset = false
	}
	t := c.Theme()
	model := a.historyModel()
	commits := model.Commits
	a.historyCursor.sync(model, a.selectedSHA)
	ui.Split(c, &a.historyWidth, func() {
		ui.Column(c).Fill().Clip().Children(func() {
			ui.Column(c).FillWidth().Padding(16).Gap(7).Children(func() {
				ui.Text(c, "历史线图").FontSize(20).Bold()
				ui.Textf(c, "所有引用及 HEAD，最近 %d 条提交", len(a.data.history)).FontSize(12).TextColor(t.TextMuted)
				ui.Text(c, "● 提交    ◉ 合并    ┆ 父提交未载入或被过滤").FontSize(11).TextColor(t.TextMuted)
				search := ui.TextInput(c, &a.historyQuery).Label("搜索已加载提交").Placeholder("搜索 SHA、标题、作者、引用").FillWidth().Tooltip("搜索已加载提交 (⌘F)").KeyShortcuts(ui.Cmd, ui.KeyF).OnChange(a.historySearchChanged)
				if a.focusHistorySearch {
					search.Focus()
					a.focusHistorySearch = false
				}
				if search.Submitted() || search.Shortcut(0, ui.KeyEscape) {
					a.focusHistoryList = true
				}
				if ui.Button(c, a.historyLoadLabel()).Disabled(!a.canLoadHistory()).Clicked() {
					a.loadMoreHistory()
				}
			})
			if len(a.data.history) == 0 {
				emptyLabel := "还没有提交。先在工作区暂存文件并提交。"
				if a.data.historyErr != nil {
					emptyLabel = "提交历史读取失败。刷新后重试；其他仓库状态仍可用。"
				}
				ui.Text(c, emptyLabel).Padding(16).TextColor(t.TextMuted)
			}
			a.historyList.Selected = &a.historyCursor.row
			a.historyList.Key = func(i int) any { return model.Commits[i].SHA }
			list := ui.List(c.Key("history-commits"), &a.historyList, len(commits), func(i int) {
				commit := commits[i]
				bg := t.Background
				if commit.SHA == a.selectedSHA {
					bg = t.Accent.Alpha(.12).Over(t.Background)
				}
				ui.Row(c.Key(commit.SHA)).Height(62).FillWidth().Background(bg).AlignItems(ui.Center).Children(func() {
					a.graphCell(c, i, model)
					ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
						if ui.Button(c, commit.Subject).FillWidth().Tooltip(commit.Subject).Clicked() {
							if index, ok := a.historySelectionIndex(model, i); ok {
								a.selectCommit(index)
								a.focusHistoryList = true
							}
						}
						ui.Text(c, short(commit.SHA)+"  "+oneLine(commit.Author)+"  "+commit.Refs).FontSize(11).TextColor(t.TextMuted)
					})
				})
			}).Grow(1).FillWidth().Label("提交历史").Tooltip("↑↓ 选择提交，Home / End 跳转首尾，Enter 查看详情").Disabled(a.guard.writing || a.opening)
			if a.focusHistoryList && !a.guard.writing && !a.opening {
				list.Focus()
				a.focusHistoryList = false
			}
			if list.Shortcut(ui.Cmd, ui.KeyRight) {
				a.focusCommitFiles = true
			}
			changed, submitted := list.Changed(), list.Submitted()
			if changed || submitted {
				if index, ok := a.historySelectionIndex(model, a.historyCursor.row); ok && (submitted || a.data.history[index].SHA != a.selectedSHA) {
					a.selectCommit(index)
				}
			}
		})
	}, func() {
		ui.SplitVertical(c, &a.detailsHeight, func() { a.commitDetails(c) }, func() { a.diffView(c, a.commitDiff, a.commitPath, false, true) }).Fill().SplitReset(340)
	}).Fill().SplitReset(530)
}
func (a *app) graphCell(c *ui.Context, index int, model *historyPresentation) {
	t := c.Theme()
	commits, rows := model.Commits, model.Rows
	if index >= len(rows) {
		return
	}
	ui.Box(c).Width(model.Width).Height(62).Shrink(0).Tooltip("连线对应真实父 SHA；提交详情中可查看父提交").Draw(func(p *ui.Painter, r ui.Rect) {
		laneX := func(lane int) float32 { return r.X + 14 + float32(lane)*18 }
		blue := graphColor(t, 0)
		orange := graphColor(t, 1)
		model.Edges.eachAt(index, func(edge graphSpan) {
			x0, x1 := laneX(edge.SourceLane), laneX(edge.Lane)
			top, bottom := r.Y, r.Y+r.H
			if index == edge.Start {
				top = r.Y + 31
			}
			if index == edge.End {
				bottom = r.Y + 31
			}
			via := laneX(edge.ViaLane)
			if index == edge.Start {
				p.Line(x0, top, via, top, 2, blue)
			}
			p.Line(via, top, via, bottom, 2, blue)
			if index == edge.End {
				p.Line(via, bottom, x1, bottom, 2, blue)
			}
		})
		row := rows[index]
		x, y := laneX(row.Lane), r.Y+31
		col := blue
		if len(commits[index].Parents) > 1 {
			col = orange
		}
		p.Fill(ui.Rect{X: x - 6, Y: y - 6, W: 12, H: 12}, t.Background, 6)
		if len(commits[index].Parents) > 1 {
			p.Stroke(ui.Rect{X: x - 4, Y: y - 4, W: 8, H: 8}, col, 4, 2)
		} else {
			p.Fill(ui.Rect{X: x - 4, Y: y - 4, W: 8, H: 8}, col, 4)
		}
		if row.Boundary {
			for n := float32(9); n < 30; n += 6 {
				p.Line(x, y+n, x, y+n+3, 2, t.TextMuted)
			}
		}
	})
}
func (a *app) commitDetails(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Clip().Padding(16).Gap(8).Children(func() {
		ui.Scroll(c).FillWidth().Height(180).Shrink(0).Children(func() {
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				selected := a.selectedCommit()
				if selected == nil {
					ui.Text(c, "选择提交查看详情").TextColor(t.TextMuted)
					return
				}
				if pin := a.tagInspection; pin != nil && pin.commit.SHA == selected.SHA {
					ui.Text(c, "标签来源："+oneLine(pin.name)).FontSize(12).TextColor(t.TextMuted).Tooltip("标签对象：" + pin.objectSHA)
					if a.historyIndex < 0 {
						ui.Text(c, "此提交尚未载入历史列表；详情按固定 SHA 显示。").FontSize(12).TextColor(t.TextMuted)
					}
				}
				title, body := a.commitText.parts(selected.SHA, selected.Subject)
				ui.Text(c, title).Bold().FontSize(16).Selectable()
				ui.Text(c, selected.SHA).Font("Menlo").FontSize(11).Selectable()
				ui.Text(c, selected.Author+"  "+selected.Date).TextColor(t.TextMuted).FontSize(12)
				ui.Row(c).Gap(6).Wrap().Children(func() {
					ui.Text(c, "父提交").FontSize(12)
					for _, parent := range selected.Parents {
						idx := -1
						for i, commit := range a.data.history {
							if commit.SHA == parent {
								idx = i
								break
							}
						}
						if ui.Button(c, short(parent)).Disabled(!a.canSelectCommit(idx)).Tooltip(parent).Clicked() {
							a.selectCommit(idx)
						}
					}
				})
				ui.Row(c).Gap(8).Wrap().Children(func() {
					if ui.Button(c, "Cherry-pick…").Disabled(a.busy() || len(selected.Parents) > 1).Clicked() {
						a.ask(actionDraft{Kind: actionCherryPick, First: selected.SHA, Second: selected.Subject})
					}
					if ui.Button(c, "Revert…").Disabled(a.busy() || len(selected.Parents) > 1).Clicked() {
						a.ask(actionDraft{Kind: actionRevert, First: selected.SHA, Second: selected.Subject})
					}
					if ui.Button(c, "从此创建分支…").Disabled(a.busy()).Clicked() {
						a.startForm(actionDraft{Kind: actionCreateBranch, Second: selected.SHA})
					}
					if ui.Button(c, "创建轻量标签…").Disabled(a.busy()).Clicked() {
						a.startForm(actionDraft{Kind: actionCreateTag, Second: selected.SHA})
					}
				})
				a.commitMessageView(c, selected.SHA, body)
			})
		})
		ui.Text(c, "变更文件（与第一父提交比较）").Bold().FontSize(12)
		a.commitFileListView(c)
	})
}
