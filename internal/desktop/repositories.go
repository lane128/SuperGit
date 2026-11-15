package desktop

import (
	"github.com/egoist/mygo/ui"
	"path/filepath"
)

func (a *app) modules(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(22).Gap(16).Children(func() {
			ui.Text(c, "子模块").FontSize(22).Bold()
			ui.Text(c, "父仓库提交只记录子模块的提交指针。内部改动需进入子模块处理。").TextColor(t.TextMuted)
			if len(a.data.modules) == 0 {
				ui.Text(c, "当前仓库没有配置子模块。").PaddingY(16)
			}
			for _, m := range a.data.modules {
				ui.Column(c.Key(m.Path)).FillWidth().Padding(16).Gap(10).Border(1, t.Border).Radius(6).Children(func() {
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, oneLine(m.Path)).Bold().FontSize(16).Grow(1)
						state := "未初始化"
						if m.Initialized {
							state = "已初始化"
						}
						if m.Dirty {
							state += " / 内部有改动"
						}
						ui.Text(c, state).TextColor(t.TextMuted)
						if ui.Button(c, "初始化／更新").Disabled(a.busy() || m.Dirty || m.Error != "").Clicked() {
							a.ask(actionDraft{Kind: actionSubmodule, First: m.Path})
						}
						if ui.Button(c, "进入子模块").Disabled(!m.Initialized || m.Error != "" || a.guard.writing || a.opening).Clicked() {
							a.open(filepath.Join(a.repo.Root, m.Path), true)
						}
					})
					ui.Row(c).Gap(30).Children(func() {
						pointer(c, "父仓库 HEAD", m.Recorded)
						pointer(c, "暂存区 Index", m.Index)
						pointer(c, "子模块当前 HEAD", m.Current)
					})
					if m.Initialized && m.Recorded != m.Current {
						ui.Text(c, "子模块当前提交与父仓库记录不同；可在工作区暂存指针。").FontSize(12).TextColor(t.TextMuted)
					}
					if m.Error != "" {
						ui.Text(c, m.Error).TextColor(t.Danger).Selectable()
					}
				})
			}
		})
	})
}
func pointer(c *ui.Context, label, sha string) {
	ui.Column(c).Gap(5).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Text(c, short(sha)).Font("Menlo").Selectable().Tooltip(sha)
	})
}
func (a *app) branches(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(22).Gap(14).Children(func() {
			ui.Text(c, "本地分支").FontSize(22).Bold()
			ui.Row(c).Gap(10).Children(func() {
				if ui.Button(c, "创建分支…").Disabled(a.busy()).Clicked() {
					a.startForm(actionDraft{Kind: actionCreateBranch, Second: "HEAD", Flag: true})
				}
				if ui.Button(c, "跟踪远端分支…").Disabled(a.busy()).Clicked() {
					a.startForm(actionDraft{Kind: actionTrack})
				}
			})
			for _, b := range a.data.branches {
				ui.Row(c).FillWidth().Padding(14).Gap(16).Border(1, t.Border).Children(func() {
					name := b.Name
					if b.Current {
						name = "●  " + name
					}
					ui.Text(c, name).Bold().Grow(1)
					ui.Text(c, b.Upstream).TextColor(t.TextMuted)
					ui.Text(c, short(b.SHA)).Font("Menlo").Selectable()
					if ui.Button(c, "切换").Disabled(a.busy() || b.Current).Clicked() {
						a.ask(actionDraft{Kind: actionSwitch, First: b.Name})
					}
					if ui.Button(c, "重命名…").Disabled(a.busy()).Clicked() {
						a.startForm(actionDraft{Kind: actionRename, First: b.Name})
					}
					if ui.Button(c, "删除").Disabled(a.busy() || b.Current).Clicked() {
						a.ask(actionDraft{Kind: actionDeleteBranch, First: b.Name})
					}
					if ui.Button(c, "Merge").Disabled(a.busy() || b.Current).Clicked() {
						a.ask(actionDraft{Kind: actionMerge, First: b.SHA, Second: b.Name})
					}
					if ui.Button(c, "Rebase onto").Disabled(a.busy() || b.Current).Clicked() {
						a.ask(actionDraft{Kind: actionRebase, First: b.SHA, Second: b.Name})
					}
				})
			}
			a.tagsView(c)
		})
	})
}
func (a *app) worktrees(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(22).Gap(14).Children(func() {
			ui.Text(c, "Worktrees").FontSize(22).Bold()
			ui.Text(c, "每个工作区有独立文件和暂存区，点击进入查看。").TextColor(t.TextMuted)
			if ui.Button(c, "新建 Worktree…").Disabled(a.busy()).Clicked() {
				a.startForm(actionDraft{Kind: actionAddWorktree, Third: "HEAD"})
			}
			for _, w := range a.data.worktrees {
				ui.Row(c).FillWidth().Padding(14).Gap(16).Border(1, t.Border).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Grow(1).Gap(6).Children(func() {
						name := w.Branch
						if name == "" {
							name = "detached HEAD"
						}
						if w.Locked {
							name += " / 已锁定"
						}
						ui.Text(c, name).Bold()
						ui.Text(c, oneLine(w.Path)).TextColor(t.TextMuted).Selectable()
					})
					ui.Text(c, short(w.HEAD)).Font("Menlo")
					if ui.Button(c, "移除").Disabled(a.busy() || w.Locked || w.Bare || filepath.Clean(w.Path) == filepath.Clean(a.repo.Root)).Clicked() {
						a.ask(actionDraft{Kind: actionRemoveWorktree, First: w.Path})
					}
					if ui.Button(c, "进入工作区").Disabled(w.Bare || a.guard.writing || a.opening).Clicked() {
						a.open(w.Path, false)
					}
				})
			}
		})
	})
}
func (a *app) logView(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(22).Gap(14).Children(func() {
			ui.Text(c, "操作记录").FontSize(22).Bold()
			ui.Text(c, "本次会话最近 40 条操作与错误。").TextColor(t.TextMuted)
			for i, line := range a.logs {
				ui.Text(c.Key(i), line).Font("Menlo").FontSize(12).Selectable().FillWidth().Padding(10).Border(1, t.Border)
			}
		})
	})
}
