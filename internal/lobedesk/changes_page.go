package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const changesFilesPaneMin float32 = 280

// Changes uses the existing file/version and patch models, with the commit
// editor beside the patch rather than mounted only on its first tab.
func (a *app) changesPage(c *ui.Context) {
	row := ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch)
	a.filesSplit.rowWidth = row.Bounds().W
	row.Children(func() {
		a.historyFilesPane(c)
		a.splitHandle(c, &a.filesSplit, shareForWidth(a.filesSplit.rowWidth-splitterWidth, historyFilesPaneWidth), changesFilesPaneMin, historyDiffPaneMin, a.t("resizeFilesPane"))
		a.historyDiffPane(c)
	})
}

func (a *app) changesCommitPane(c *ui.Context) {
	tk := theme.TokensOf(c)
	// Files and the editor each have their own viewport. The action footer is
	// outside both so long body/identity/operation content cannot push it away.
	ui.Column(c).FillWidth().Height(scaledFont(300)).MaxHeightPercent(60).Shrink(0).
		BorderWidth(1, 0, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
		layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.changesCommitScroll}, func() {
			ui.Column(c).FillWidth().Padding(10).Gap(7).Children(func() {
				a.operationCard(c)
				branch := a.status.Branch
				if branch == "" {
					branch = a.t("detached")
				}
				display.Text(c, a.tf("commitTo", branch), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).FillWidth().SingleLine().Tooltip(branch)
				a.commitMessageFields(c)
				a.commitAmendControl(c)
				a.commitIdentityView(c)
			})
		}).Grow(1).Basis(0).MinHeight(0)
		ui.Column(c).FillWidth().Padding(8, 10).Gap(5).Shrink(0).Children(func() { a.commitAction(c, true) })
	})
}

func (a *app) workbenchNavigation(c *ui.Context) {
	tk := theme.TokensOf(c)
	ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(6).
		Background(tk.BgLayout).Role(ui.RoleTabList).Label(a.t("workbenchPages")).FocusGroup(ui.Horizontal).Children(func() {
		for _, page := range []string{"changes", "history", "stash"} {
			current := a.page == page || page == "history" && a.page == ""
			button := workbenchButton(c, a.t(page), buttonNormal, a.repo == nil)
			styleWorkbenchButton(c, button, buttonNormal, a.repo == nil, current).Role(ui.RoleTab).AccessSelected(current)
			if button.Clicked() {
				a.openPage(page)
			}
		}
	})
}

// Snapshot the context before its selected SHA is cleared by selectWorktree.
// All targets still pass through the normal selection/loading guards.
func (a *app) openChanges() {
	if a.worktreeSelected {
		a.saveDetailScroll()
		return
	}
	a.selectWorktree()
	a.commitFilesScroll = ui.ScrollState{X: a.detailTabs().FilesScroll.X, Y: a.detailTabs().FilesScroll.Y}
}

func (a *app) restoreHistorySelection() {
	if !a.worktreeSelected {
		return
	}
	sha := ""
	if a.repo != nil && a.detailContexts.Root == a.repo.Root {
		sha = a.detailContexts.HistorySHA
	}
	if sha == "" && len(a.historySource()) > 0 {
		sha = a.historySource()[0].SHA
	}
	if sha != "" {
		a.selectCommit(sha)
		pos := a.detailTabs().FilesScroll
		a.commitFilesScroll = ui.ScrollState{X: pos.X, Y: pos.Y}
	}
}
