package desktop

import "github.com/egoist/mygo/ui"

type pageShortcut struct {
	Page, Label string
	Key         ui.Key
}

var pageShortcuts = [...]pageShortcut{
	{"workspace", "工作区", ui.Key1},
	{"history", "历史线图", ui.Key2},
	{"modules", "子模块", ui.Key3},
	{"branches", "分支", ui.Key4},
	{"remotes", "远端与同步", ui.Key5},
	{"stash", "Stash", ui.Key6},
	{"conflicts", "冲突处理", ui.Key7},
	{"repositories", "仓库", ui.Key8},
	{"worktrees", "Worktrees", ui.Key9},
}

func (a *app) navigateShortcut(page string) bool {
	if a.page == "action" || a.guard.writing || a.opening {
		return false
	}
	for _, item := range pageShortcuts {
		if item.Page == page && (a.repo != nil || page == "repositories") {
			a.page = page
			if page == "history" {
				a.focusHistoryList = true
			}
			if page == "workspace" {
				a.focusWorkspaceList = true
			}
			return true
		}
	}
	return false
}
func (a *app) cancelAction() bool {
	if a.page != "action" || a.busy() {
		return false
	}
	a.confirmation = nil
	a.page = a.returnPage
	if a.page == "" {
		a.page = "repositories"
		if a.repo != nil {
			a.page = "workspace"
		}
	}
	return true
}

func (a *app) keyboard(c *ui.Context) {
	// Action forms/confirmations own the workflow until the user cancels it.
	// Escape never executes the target, starts a commit, or cancels a Git job.
	if a.page == "action" {
		if c.Shortcut(0, ui.KeyEscape) && !c.ShortcutRepeated() {
			a.cancelAction()
		}
		return
	}
	if a.opening || a.guard.writing {
		return
	}
	if c.Shortcut(ui.Cmd, ui.KeyO) && !c.ShortcutRepeated() {
		a.choose()
	}
	if c.Shortcut(ui.Cmd, ui.KeyR) && !c.ShortcutRepeated() {
		a.refresh()
	}
	if c.Shortcut(ui.Cmd, ui.KeyEnter) && !c.ShortcutRepeated() {
		a.commit()
	}
	for _, item := range pageShortcuts {
		if c.Shortcut(ui.Cmd, item.Key) && !c.ShortcutRepeated() {
			a.navigateShortcut(item.Page)
		}
	}
	if a.page == "history" && c.Shortcut(ui.Cmd, ui.KeyF) && !c.ShortcutRepeated() {
		a.focusHistorySearch = true
	}
}
