package lobedesk

const sidebarPaneMin float32 = 160

// Leave room for both inner dividers as well as the History/Files/patch panes.
// Only the effective share is clamped: saved shares survive a narrow window.
func (a *app) workbenchContentMin() float32 {
	if a.page == "changes" {
		return changesFilesPaneMin + splitterWidth + historyDiffPaneMin
	}
	if a.page != "history" && a.page != "" {
		return 480
	}
	if a.historyDetailsHidden {
		return historySplitMinList
	}
	if a.detailsChangesHidden {
		return historySplitMinList + splitterWidth + historyDiffPaneMin
	}
	return historySplitMinList + splitterWidth + historySplitMinDetail
}

func (a *app) workbenchSidebarWidth() float32 {
	row := a.sidebarSplit.rowWidth
	if row <= splitterWidth {
		return sidebarWidth
	}
	def := float32(sidebarWidth) / (row - splitterWidth)
	return (row - splitterWidth) * a.sidebarSplit.fraction(def, sidebarPaneMin, a.workbenchContentMin())
}
