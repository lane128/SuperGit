package desktop

import (
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"strings"
)

func (a *app) repositoryEntry(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(28).Gap(18).Children(func() {
			ui.Text(c, "仓库").FontSize(26).Bold()
			ui.Text(c, "打开工作区，或从远端克隆到一个新目录。").TextColor(c.Theme().TextMuted)
			ui.Row(c).Gap(12).Children(func() {
				if ui.PrimaryButton(c, "选择仓库文件夹").Disabled(a.busy()).Clicked() {
					a.choose()
				}
				if ui.Button(c, "克隆仓库…").Disabled(a.busy()).Clicked() {
					a.startForm(actionDraft{Kind: actionClone})
				}
				if ui.Button(c, "初始化仓库…").Disabled(a.busy()).Clicked() {
					a.startForm(actionDraft{Kind: actionInit})
				}
			})
			ui.Text(c, "最近打开").Bold().PaddingY(8)
			if len(a.prefs.Recent) == 0 {
				ui.Text(c, "打开的仓库会显示在这里。").TextColor(c.Theme().TextMuted)
			}
			for _, path := range a.prefs.Recent {
				ui.Row(c.Key(path)).FillWidth().Padding(12).Gap(12).Border(1, c.Theme().Border).Children(func() {
					ui.Column(c).Grow(1).Gap(5).Children(func() {
						ui.Text(c, filepath.Base(path)).Bold()
						ui.Text(c, oneLine(path)).FontSize(12).TextColor(c.Theme().TextMuted)
					})
					if ui.Button(c, "打开").Disabled(a.busy()).Clicked() {
						a.parents = nil
						a.open(path, false)
					}
				})
			}
		})
	})
}
func (a *app) remotes(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(22).Gap(16).Children(func() {
			ui.Text(c, "远端与同步").FontSize(22).Bold()
			ui.Text(c, "Pull 仅快进；Push 使用明确的源和目标分支，并检查子模块提交。").TextColor(t.TextMuted)
			ui.Row(c).Gap(10).Children(func() {
				if ui.Button(c, "添加远端…").Disabled(a.busy()).Clicked() {
					a.startForm(actionDraft{Kind: actionAddRemote, First: "origin"})
				}
				if ui.Button(c, "Pull（仅快进）").Disabled(a.busy() || a.data.status.Upstream == "").Clicked() {
					a.ask(actionDraft{Kind: actionPull})
				}
			})
			if len(a.data.remotes) == 0 {
				ui.Text(c, "没有远端。添加后可 Fetch 与 Push。").PaddingY(14)
			}
			for _, remote := range a.data.remotes {
				ui.Column(c.Key(remote.Name)).FillWidth().Padding(16).Gap(10).Border(1, t.Border).Radius(6).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, remote.Name).Bold().FontSize(16).Grow(1)
						if ui.Button(c, "Fetch").Disabled(a.busy()).Clicked() {
							a.ask(actionDraft{Kind: actionFetch, First: remote.Name})
						}
						if ui.Button(c, "Push…").Disabled(a.busy() || a.data.status.Branch == "").Clicked() {
							a.startForm(actionDraft{Kind: actionPush, First: remote.Name, Second: a.data.status.Branch, Third: a.data.status.Branch, Flag: a.data.status.Upstream == ""})
						}
					})
					ui.Text(c, "Fetch  "+displayURL(remote.URL)).Selectable().FontSize(12)
					ui.Text(c, "Push   "+displayURL(remote.PushURL)).Selectable().FontSize(12)
					for _, b := range a.data.remoteBranches {
						if strings.HasPrefix(b.Name, remote.Name+"/") {
							ui.Row(c.Key(b.Name)).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
								ui.Text(c, b.Name).Grow(1)
								ui.Text(c, short(b.SHA)).Font("Menlo").FontSize(12)
								if ui.Button(c, "创建跟踪分支…").Disabled(a.busy()).Clicked() {
									a.startForm(actionDraft{Kind: actionTrack, First: strings.TrimPrefix(b.Name, remote.Name+"/"), Second: b.Name})
								}
							})
						}
					}
				})
			}
		})
	})
}
