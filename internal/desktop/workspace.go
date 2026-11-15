package desktop

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func (a *app) workspace(c *ui.Context) {
	ui.Split(c, &a.fileWidth, func() { a.files(c) }, func() { a.diffView(c, a.diff, a.path, a.staged, false) }).Fill().SplitReset(325)
}
func (a *app) files(c *ui.Context) {
	t := c.Theme()
	model := a.workspaceModel()
	reset := a.workspaceSelectionReset
	if reset {
		// Handle reads committed focus before this build, including a row
		// that just disappeared. Restore only focus already in this list.
		if a.workspaceList.Handle.FocusWithin(c) {
			a.focusWorkspaceList = true
		}
		// Reset here, on the UI build stack, rather than in a callback
		// while accessibility can still inspect the previous frame.
		a.workspaceList = ui.ListState{Handle: a.workspaceList.Handle}
		a.workspaceSelectionReset = false
	}
	a.workspaceCursor.sync(model, a.path, a.staged)
	if reset && a.workspaceCursor.row >= 0 {
		a.workspaceList.ScrollIntoView(a.workspaceCursor.row)
	}
	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(16).Gap(5).FillWidth().Children(func() {
			ui.Text(c, "工作区").FontSize(20).Bold()
			ui.Textf(c, "%d 个文件有改动", len(a.data.status.Entries)).TextColor(t.TextMuted).FontSize(12)
		})
		a.workspaceList.Selected = &a.workspaceCursor.row
		a.workspaceList.Key = func(i int) any {
			item := model.rows[i]
			if item.header {
				return workspaceHeaderKey{item.key.staged}
			}
			return item.key
		}
		a.workspaceList.Header = func(i int) bool { return model.rows[i].header }
		a.workspaceList.Label = func(i int) string { return oneLine(model.rows[i].key.path) }
		list := ui.List(c.Key("workspace-files"), &a.workspaceList, len(model.rows), func(i int) {
			item := model.rows[i]
			if item.header {
				title := "未暂存"
				if item.key.staged {
					title = "已暂存"
				}
				ui.Column(c).FillWidth().Padding(8).Gap(3).Background(t.Background).Children(func() {
					ui.Textf(c, "%s  %d", title, item.count).Bold().FontSize(12).Padding(6)
					if item.count == 0 {
						ui.Text(c, "没有文件").TextColor(t.TextMuted).FontSize(12).Padding(6)
					}
				})
			} else {
				a.fileRow(c, model, i)
			}
		}).Grow(1).FillWidth().Label("工作区文件").Tooltip("↑↓ 选择文件，Home / End 跳转首尾，Enter 查看文件或处理冲突").Disabled(a.guard.writing || a.opening)
		if a.focusWorkspaceList && !a.guard.writing && !a.opening {
			list.Focus()
			a.focusWorkspaceList = false
		}
		changed, submitted := list.Changed(), list.Submitted()
		if changed || submitted {
			if item, ok := a.workspaceSelection(model, a.workspaceCursor.row); ok && (submitted || item.key != (workspaceFileKey{a.path, a.staged})) {
				a.chooseWorkspaceFile(model, a.workspaceCursor.row, submitted)
			}
		}
		ui.Column(c).Padding(14).Gap(9).FillWidth().Border(1, t.Border).Children(func() {
			ui.Text(c, "提交暂存内容").Bold()
			ui.TextArea(c, &a.message).Placeholder("提交标题\n\n可选的详细说明").Label("提交说明").Height(112).FillWidth()
			ui.Text(c, "仅提交已暂存内容，子模块内部改动需单独提交。").FontSize(11).TextColor(t.TextMuted)
			if ui.PrimaryButton(c, "提交  ⌘Enter").FillWidth().Disabled(!a.canCommit()).KeyShortcuts(ui.Cmd, ui.KeyEnter).Clicked() {
				a.commit()
			}
		})
	})
}
func (a *app) fileRow(c *ui.Context, model *workspacePresentation, row int) {
	item := model.rows[row]
	e, staged := item.entry, item.key.staged
	t := c.Theme()
	bg := t.Background
	if a.path == e.Path && a.staged == staged {
		bg = t.Accent.Alpha(.13).Over(t.Background)
	}
	code := string(e.Worktree)
	if staged {
		code = string(e.Index)
	}
	if e.Conflict {
		code = "!"
	}
	if e.IsSubmodule() {
		code = "S"
	}
	tip := e.Path
	if e.OriginalPath != "" {
		tip = e.OriginalPath + " → " + e.Path
	}
	ui.Row(c.Key(fmt.Sprintf("%t/%s", staged, e.Path))).FillWidth().Padding(4).Gap(5).Background(bg).AlignItems(ui.Center).Children(func() {
		ui.Text(c, code).Width(18).Bold().TextColor(t.TextMuted)
		if ui.Button(c.Key("file"), oneLine(e.Path)).Grow(1).MinWidth(0).Tooltip(tip).Clicked() {
			a.chooseWorkspaceFile(model, row, true)
			if a.page == "workspace" {
				a.focusWorkspaceList = true
			}
		}
		label := "+"
		action := "暂存文件"
		if staged {
			label = "−"
			action = "取消暂存文件"
		}
		if ui.Button(c.Key("stage"), label).Width(28).Tooltip(action).Disabled(a.guard.writing || a.loading || a.opening || e.Conflict).Clicked() {
			if _, ok := a.workspaceSelection(model, row); !ok {
				return
			}
			path, original := e.Path, e.OriginalPath
			a.mutate(action, func(ctx context.Context, r *gitrepo.Repository) error {
				if original != "" {
					return r.StageRename(ctx, path, original, staged)
				}
				if staged {
					return r.Unstage(ctx, path)
				}
				return r.Stage(ctx, path)
			}, false)
		}
	})
}
func (a *app) hunksAllowed() bool {
	for _, e := range a.data.status.Entries {
		if e.Path == a.path {
			return !e.Conflict && !e.IsSubmodule() && e.Index != '?' && e.OriginalPath == ""
		}
	}
	return false
}
