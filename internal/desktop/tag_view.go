package desktop

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func tagKind(tag gitrepo.Tag) string {
	if tag.SymbolicRef != "" {
		return "符号引用"
	}
	if tag.ObjectType == "tag" {
		return "附注标签"
	}
	return "轻量标签"
}

func (a *app) tagsView(c *ui.Context) {
	if a.tagListReset {
		a.tagList = ui.ListState{Handle: a.tagList.Handle}
		a.tagListReset = false
	}
	t := c.Theme()
	ui.Text(c, "本地标签").FontSize(20).Bold()
	ui.Text(c, "只读浏览可解析的本地标签；悬空符号引用不会列出。查看提交不切换工作区。").FontSize(12).TextColor(t.TextMuted).FillWidth()
	if a.data.tagsErr != nil {
		ui.Text(c, "标签读取失败："+a.data.tagsErr.Error()).TextColor(t.Danger).Selectable().FillWidth()
		if ui.Button(c, "重试读取标签").Disabled(a.busy() || a.closed).Clicked() {
			a.refresh()
		}
		return
	}
	if len(a.data.tags) == 0 {
		ui.Text(c, "当前仓库没有可解析的本地标签。").TextColor(t.TextMuted)
		return
	}
	ui.TextInput(c.Key("tag-search"), &a.tagQuery).Label("搜索本地标签").Placeholder("搜索标签名称或 SHA").FillWidth().OnChange(a.tagSearchChanged)
	tags := filterTags(a.data.tags, a.tagQuery)
	ui.Text(c, fmt.Sprintf("匹配 %d / %d 个标签", len(tags), len(a.data.tags))).FontSize(12).TextColor(t.TextMuted)
	if len(tags) == 0 {
		ui.Text(c, "没有匹配的标签。").TextColor(t.TextMuted)
	}
	a.tagList.Key = func(i int) any { return tags[i].Name }
	ui.List(c.Key("local-tags"), &a.tagList, len(tags), func(i int) {
		tag := tags[i]
		ui.Column(c.Key(tag.Name)).FillWidth().Padding(12).Gap(6).Border(1, t.Border).Children(func() {
			ui.Row(c).FillWidth().Gap(10).Wrap().AlignItems(ui.Center).Children(func() {
				ui.Text(c, oneLine(tag.Name)).Bold().Selectable().Tooltip(tag.Name)
				ui.Text(c, tagKind(tag)).FontSize(12).TextColor(t.TextMuted)
				if ui.Button(c, "查看提交").Label("查看标签提交：" + tag.Name).Disabled(tag.Commit.SHA == "" || a.closed || a.guard.writing || a.opening).Clicked() {
					a.showTagCommit(tag.Name, tag.ObjectSHA)
				}
			})
			ui.Text(c, "标签对象："+tag.ObjectSHA+" · "+tag.ObjectType).Font("Menlo").FontSize(11).Selectable().FillWidth()
			ui.Text(c, "目标对象："+tag.TargetSHA+" · "+tag.TargetType).Font("Menlo").FontSize(11).Selectable().FillWidth()
			if tag.SymbolicRef != "" {
				ui.Text(c, "引用目标："+tag.SymbolicRef).FontSize(12).TextColor(t.TextMuted).Selectable().FillWidth()
			}
			if tag.Commit.SHA == "" {
				ui.Text(c, "目标不是提交对象，无法显示提交详情。").FontSize(12).TextColor(t.TextMuted)
			}
		})
	}).FillWidth().Height(360).Label("本地标签列表")
}
