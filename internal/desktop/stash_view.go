package desktop

import (
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func (a *app) stashes(c *ui.Context) {
	if a.stashListReset {
		a.stashList = ui.ListState{Handle: a.stashList.Handle}
		a.stashListReset = false
	}
	t := c.Theme()
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.Row(c).FillWidth().Gap(12).Children(func() {
			ui.Text(c, "Stash").FontSize(22).Bold().Grow(1)
			if ui.Button(c, "保存 Stash…").Disabled(a.busy()).Clicked() {
				a.startForm(actionDraft{Kind: actionSaveStash, Flag: true})
			}
		})
		ui.Text(c, "恢复前要求干净工作区；应用时恢复 Index，冲突时保留条目。").FontSize(12).TextColor(t.TextMuted)
		ui.Scroll(c).Height(230).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				if len(a.data.stashes) == 0 {
					ui.Text(c, "没有 Stash。").Padding(16)
				}
				for _, s := range a.data.stashes {
					ui.Row(c.Key(s.SHA+s.Ref)).FillWidth().Padding(8).Gap(8).Border(1, t.Border).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Children(func() {
							if ui.Button(c, s.Ref+"  "+s.Message).FillWidth().Disabled(a.guard.writing || a.opening || a.closed).Clicked() {
								a.selectStash(s.SHA)
							}
							ui.Text(c, short(s.SHA)+"  "+s.Date).FontSize(11).TextColor(t.TextMuted)
						})
						if ui.Button(c, "应用").Disabled(a.busy()).Clicked() {
							a.ask(actionDraft{Kind: actionApplyStash, First: s.SHA, Second: s.Ref})
						}
						if ui.Button(c, "应用并移除").Disabled(a.busy()).Clicked() {
							a.ask(actionDraft{Kind: actionPopStash, First: s.SHA, Second: s.Ref})
						}
						if ui.Button(c, "删除").Disabled(a.busy()).Clicked() {
							a.ask(actionDraft{Kind: actionDropStash, First: s.SHA, Second: s.Ref})
						}
					})
				}
			})
		})
		if a.stashSHA == "" {
			ui.Text(c, "选择 Stash 查看差异。").Padding(12).TextColor(t.TextMuted)
			return
		}
		ui.Text(c, "差异 · "+short(a.stashSHA)).Bold().Tooltip(a.stashSHA)
		switch {
		case a.stashLoading:
			ui.Text(c, "正在读取 Stash…").Padding(12)
		case !a.stashLoaded || a.stashErr != nil:
			if a.stashErr != nil {
				ui.Text(c, "Stash 差异读取失败："+a.stashErr.Error()).TextColor(t.Danger).FillWidth()
			} else {
				ui.Text(c, "Stash 差异尚未读取完成。").TextColor(t.TextMuted)
			}
			if ui.Button(c, "重试读取 Stash 差异").Disabled(a.guard.writing || a.opening || a.closed).Clicked() {
				a.selectStash(a.stashSHA)
			}
		case a.stashDiff.Text == "":
			ui.Text(c, "此 Stash 没有可显示的差异。").Padding(12).TextColor(t.TextMuted)
		default:
			readOnlyPatch(c, a.stashDiff, &a.stashList, a.fontSize)
		}
	})
}
func readOnlyPatch(c *ui.Context, d gitrepo.Diff, state *ui.ListState, size float32) {
	ui.ScrollHorizontal(c).Grow(1).FillWidth().Children(func() {
		width := float32(800)
		for _, line := range d.Lines {
			width = max(width, float32(len([]rune(line.Text)))*size*.65+40)
		}
		ui.List(c, state, len(d.Lines), func(i int) {
			line := d.Lines[i]
			bg := c.Theme().Background
			switch line.Kind {
			case "add":
				bg = c.Theme().Success.Alpha(.10).Over(bg)
			case "del":
				bg = c.Theme().Danger.Alpha(.10).Over(bg)
			}
			ui.Text(c, line.Text).Font("Menlo").FontSize(size).Selectable().FillWidth().Height(26).Padding(3, 10).Background(bg)
		}).Width(min(width, 24000)).FillHeight()
	})
}
