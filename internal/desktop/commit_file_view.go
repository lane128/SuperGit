package desktop

import (
	"fmt"
	"github.com/egoist/mygo/ui"
)

func (a *app) commitFileListView(c *ui.Context) {
	if a.selectedSHA != "" {
		switch {
		case a.commitLoading && a.commitPath == "":
			ui.Text(c, "正在读取提交文件…").FontSize(12).TextColor(c.Theme().TextMuted)
		case !a.commitFilesLoaded || a.commitFilesErr != nil:
			if a.commitFilesErr != nil {
				ui.Text(c, "提交文件读取失败："+a.commitFilesErr.Error()).FontSize(12).TextColor(c.Theme().Danger).FillWidth()
			} else {
				ui.Text(c, "提交文件尚未读取完成。").FontSize(12).TextColor(c.Theme().TextMuted)
			}
			if ui.Button(c, "重试读取提交文件").Disabled(a.guard.writing || a.opening || a.closed).Clicked() {
				a.loadCommitFiles()
			}
		case len(a.commitFiles) == 0:
			ui.Text(c, "此提交没有变更文件。").FontSize(12).TextColor(c.Theme().TextMuted)
		}
	}
	model := a.commitFileModel()
	reset := a.commitFileCursor.model != model
	if reset {
		if a.commitFileList.Handle.FocusWithin(c) {
			a.focusCommitFiles = true
		}
		// A committed accessibility tree may borrow the old frame until now.
		a.commitFileList = ui.ListState{Handle: a.commitFileList.Handle}
	}
	a.commitFileCursor.sync(model, a.commitPath)
	if reset && a.commitFileCursor.row >= 0 {
		a.commitFileList.ScrollIntoView(a.commitFileCursor.row)
	}
	a.commitFileList.Selected = &a.commitFileCursor.row
	a.commitFileList.Key = func(i int) any { return model.files[i].Path }
	a.commitFileList.Label = func(i int) string { return oneLine(model.files[i].Path) }
	list := ui.List(c.Key("history-files"), &a.commitFileList, len(model.files), func(i int) {
		file := model.files[i]
		tip := file.Path
		if file.OriginalPath != "" {
			tip = file.OriginalPath + " → " + file.Path
		}
		bg := c.Theme().Background
		if file.Path == a.commitPath {
			bg = c.Theme().Accent.Alpha(.13).Over(bg)
		}
		ui.Row(c).FillWidth().Padding(4).Background(bg).Children(func() {
			if ui.Button(c.Key(file.Path), fmt.Sprintf("%s   %s", file.Status, oneLine(file.Path))).Grow(1).MinWidth(0).Tooltip(tip).Clicked() {
				if current, ok := a.commitFileSelection(model, i); ok {
					a.showCommitFile(current.Path)
					a.focusCommitFiles = true
				}
			}
		})
	}).Grow(1).FillWidth().MinHeight(44).Label("提交变更文件").Tooltip("↑↓ 选择文件，Home / End 跳转首尾，Enter 查看 Diff，⌘← 返回提交列表").Disabled(a.guard.writing || a.opening)
	if a.focusCommitFiles && !a.guard.writing && !a.opening {
		list.Focus()
		a.focusCommitFiles = false
	}
	if list.Shortcut(ui.Cmd, ui.KeyLeft) {
		a.historyList.Focus(c)
	}
	changed, submitted := list.Changed(), list.Submitted()
	if changed || submitted {
		if file, ok := a.commitFileSelection(model, a.commitFileCursor.row); ok && (submitted || file.Path != a.commitPath) {
			a.showCommitFile(file.Path)
		}
	}
}
