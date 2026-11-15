package desktop

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *app) loadConflict(path string) {
	if a.repo == nil || a.guard.writing {
		return
	}
	a.conflict = nil
	a.conflictError = ""
	a.conflictText = ""
	a.conflictLoading = true
	r, ctx := a.repo, a.ctx
	ticket := a.guard.Begin("conflict")
	go func() {
		conflict, err := r.Conflict(ctx, path)
		mygo.RunOnMain(func() {
			if !a.guard.Accept(ticket) {
				return
			}
			a.conflictLoading = false
			if err != nil {
				a.conflictError = err.Error()
			} else {
				a.conflict = &conflict
				a.conflictText = conflict.Working
			}
			a.invalidate()
		})
	}()
}
func (a *app) conflicts(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Padding(20).Gap(14).Children(func() {
			ui.Text(c, "冲突处理").FontSize(22).Bold()
			ui.Text(c, "Base 为共同祖先；Ours / Theirs 是 Index 阶段 2 / 3，变基期间双方含义与普通合并不同。").FontSize(12).TextColor(t.TextMuted)
			count := 0
			for _, e := range a.data.status.Entries {
				if e.Conflict {
					count++
					if ui.Button(c, oneLine(e.Path)).FillWidth().Disabled(a.busy()).Clicked() {
						a.loadConflict(e.Path)
					}
				}
			}
			if count == 0 {
				ui.Text(c, "没有未解决的冲突。").PaddingY(12)
				return
			}
			if a.conflictLoading {
				ui.Text(c, "正在读取冲突版本…")
				return
			}
			if a.conflictError != "" {
				ui.Text(c, a.conflictError).TextColor(t.Danger).Selectable()
				ui.Text(c, "使用外部工具完成文件级处理后刷新；应用不会把二进制或链接当作文本写入。").TextColor(t.TextMuted)
				return
			}
			if a.conflict == nil {
				ui.Text(c, "选择一个冲突文件。")
				return
			}
			original := *a.conflict
			valid := false
			for _, e := range a.data.status.Entries {
				if e.Path == original.Path && e.Conflict {
					valid = true
				}
			}
			if !valid {
				ui.Text(c, "该文件已解决或发生变化，请重新选择。")
				return
			}
			ui.Text(c, oneLine(original.Path)).Bold().Selectable()
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				conflictVersion(c, "Base", original.Base)
				conflictVersion(c, "Ours · 阶段 2", original.Ours)
				conflictVersion(c, "Theirs · 阶段 3", original.Theirs)
			})
			ui.Row(c).Gap(10).Children(func() {
				if ui.Button(c, "使用 Ours 作为结果").Disabled(original.OIDs[1] == "").Clicked() {
					a.conflictText = original.Ours
				}
				if ui.Button(c, "使用 Theirs 作为结果").Disabled(original.OIDs[2] == "").Clicked() {
					a.conflictText = original.Theirs
				}
				if ui.Button(c, "重新载入").Disabled(a.busy()).Clicked() {
					a.loadConflict(original.Path)
				}
			})
			if original.OIDs[1] == "" || original.OIDs[2] == "" {
				ui.Text(c, "这是删除／修改冲突；可保留编辑结果，或明确接受删除。").TextColor(t.TextMuted)
				if ui.Button(c, "接受删除…").Disabled(a.busy()).Clicked() {
					a.ask(actionDraft{Kind: actionResolveDeletion, First: original.Path, Conflict: original})
				}
			}
			ui.Text(c, "最终结果").Bold()
			ui.TextArea(c, &a.conflictText).Label("冲突最终结果").Font("Menlo").FontSize(a.fontSize).FillWidth().Height(260)
			if ui.PrimaryButton(c, "保存并标记解决…").Disabled(a.busy()).Clicked() {
				a.ask(actionDraft{Kind: actionResolve, First: original.Path, Conflict: original, Result: a.conflictText})
			}
			ui.Text(c, "保存前核对外部修改，拒绝剩余冲突标记。所有文件解决后使用顶部的继续操作。").FontSize(12).TextColor(t.TextMuted)
		})
	})
}
func conflictVersion(c *ui.Context, label, text string) {
	ui.Column(c.Key(label)).Grow(1).Basis(0).MinWidth(0).Gap(6).Children(func() {
		ui.Text(c, label).Bold().FontSize(12)
		ui.ScrollBoth(c).FillWidth().Height(200).Border(1, c.Theme().Border).Children(func() { ui.Text(c, text).Font("Menlo").FontSize(12).Selectable().Padding(10) })
	})
}
