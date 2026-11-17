package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"fmt"
	"slices"
	"supergit/internal/gitrepo"
)

// The history page keeps Sublime Merge's three columns after the locations
// sidebar: the commit list, the files the selection changed, and the patch of
// the chosen file. The commit message lives on the diff pane's Summary tab, so
// the files column stays a list of files rather than a stack of blocks.
const (
	// historyFilesPaneWidth is the files column's width when the details have
	// the room for it. Below that it gives width back to the patch, down to
	// historyFilesPaneMin: a fixed column would leave the patch a sliver on a
	// narrow window.
	historyFilesPaneWidth float32 = 300
	historyFilesPaneMin   float32 = 180
	historyDiffPaneMin    float32 = 300
)

// historyFilesWidth is the files column's width this frame, read from the
// share the row was laid out with.
func (a *app) historyFilesWidth() float32 {
	row := a.filesSplit.rowWidth
	if row <= 0 && a.historySplit.rowWidth > 0 {
		row = (a.historySplit.rowWidth - splitterWidth) * (1 - a.historyShare())
	}
	if row <= splitterWidth {
		return historyFilesPaneWidth
	}
	state := a.filesSplit
	state.rowWidth = row
	def := historyFilesPaneWidth / (row - splitterWidth)
	minimum := historyFilesPaneMin
	if a.page == "changes" {
		minimum = changesFilesPaneMin
	}
	return max(minimum, (row-splitterWidth)*state.fraction(def, minimum, historyDiffPaneMin))
}

const (
	detailTabSummary = "summary"
	detailTabFile    = "file"
)

// normalizeDetailVisibility maps legacy preferences to content inside the
// permanent first tab and drops selections when their content is hidden.
func (a *app) normalizeDetailVisibility() {
	if a.detailsInfoHidden && a.detailsDiffHidden {
		a.detailsInfoHidden = false
	}
	if !a.worktreeSelected && a.detailsDiffHidden {
		if !a.detailTabs().Active.Summary {
			a.activateDetailTab(detailTabKey{Summary: true})
		}
		a.clearLineSelection()
		a.clearCommitSelection()
		a.commitCollapsedSel = nil
	}
	if a.detailsChangesHidden {
		a.detailFileFocus = nil
	}
	if a.historyDetailsHidden {
		a.detailFocusPending = false
		a.detailFileFocus = nil
		a.clearLineSelection()
		a.clearCommitSelection()
		a.commitCollapsedSel = nil
	}
}

func (a *app) setHistoryDetailsHidden(hidden bool) {
	a.saveDetailScroll()
	a.historyDetailsHidden = hidden
	a.normalizeDetailVisibility()
	a.invalidate()
}

func (a *app) detailSummaryLabel() string {
	if a.worktreeSelected {
		return a.t("commit")
	}
	return a.t("summaryTab")
}

func (a *app) saveDetailScroll() {
	tabs := a.detailTabs()
	tabs.FilesScroll = detailScrollPosition{X: a.commitFilesScroll.X, Y: a.commitFilesScroll.Y}
	if tabs.Scroll == nil {
		tabs.Scroll = make(map[detailTabKey]detailScrollPosition)
	}
	scroll := &a.commitInfoScroll
	if !tabs.Active.Summary {
		scroll = &a.commitDiffScroll
		if a.worktreeSelected {
			scroll = &a.diffScroll
		}
	}
	tabs.Scroll[tabs.Active] = detailScrollPosition{X: scroll.X, Y: scroll.Y}
}

func (a *app) openDetailFile(file detailFileKey) {
	if file.Path == "" || (!a.worktreeSelected && a.detailsDiffHidden) {
		return
	}
	a.saveDetailScroll()
	old := a.detailTabs().Active
	a.detailTabs().Open(file)
	a.applyDetailTab(old)
}

func (a *app) activateDetailTab(tab detailTabKey) {
	if !tab.Summary && !a.worktreeSelected && a.detailsDiffHidden {
		return
	}
	a.saveDetailScroll()
	old := a.detailTabs().Active
	if a.detailTabs().Activate(tab) {
		a.applyDetailTab(old)
	}
}

func (a *app) closeDetailFile(file detailFileKey) {
	a.saveDetailScroll()
	old := a.detailTabs().Active
	a.detailTabs().Close(file)
	delete(a.detailTabs().Scroll, detailTabKey{File: file})
	a.applyDetailTab(old)
}

func (a *app) applyDetailTab(previous detailTabKey) {
	tab := a.detailTabs().Active
	a.detailReveal = true
	if tab == previous {
		a.invalidate()
		return
	}
	a.detailFileFocus = nil
	a.clearLineSelection()
	a.clearCommitSelection()
	a.commitCollapsedSel = nil
	pos := a.detailTabs().Scroll[tab]
	if tab.Summary {
		a.commitInfoScroll.X, a.commitInfoScroll.Y = pos.X, pos.Y
	} else if a.worktreeSelected {
		file := tab.File
		changed := a.selectedPath != file.Path || a.selectedStaged != file.Staged
		a.selectedPath, a.selectedStaged = file.Path, file.Staged
		a.diffScroll = ui.ScrollState{X: pos.X, Y: pos.Y}
		if a.repo != nil {
			if cached, ok := a.diffCache.get(a.repo.Root, a.loadSeq, file); ok {
				a.diffSeq++ // Reject a pending read of a previous projection.
				a.diff, a.diffErr, a.diffLoading = cached, "", false
			} else if changed || (a.diff.Text == "" && !a.diffLoading) {
				a.diff = gitrepo.Diff{}
				a.diffErr = ""
				a.loadDiff()
			}
		}
	} else {
		a.commitPath = tab.File.Path
		a.commitDiffScroll = ui.ScrollState{X: pos.X, Y: pos.Y}
	}
	a.invalidate()
}

// historyDetailsPane is the share right of the commit list: the changed files
// and the patch side by side. The details toggle in the page header collapses
// both columns at once.
func (a *app) historyDetailsPane(c *ui.Context) {
	row := ui.Row(c).Grow(1 - a.historyShare()).Basis(0).MinWidth(0).AlignSelf(ui.Stretch)
	a.filesSplit.rowWidth = row.Bounds().W
	row.Children(func() {
		a.historyFilesPane(c)
		if !a.detailsChangesHidden {
			def := shareForWidth(a.filesSplit.rowWidth-splitterWidth, historyFilesPaneWidth)
			a.splitHandle(c, &a.filesSplit, def, historyFilesPaneMin, historyDiffPaneMin, a.t("resizeFilesPane"))
		}
		a.historyDiffPane(c)
	})
}

// detailFilePath is the file the details are pointing at: the selected file of
// the selected commit, or the working tree's file when its row is selected.
func (a *app) detailFilePath() string {
	if a.worktreeSelected {
		return a.selectedPath
	}
	return a.commitPath
}

// detailTabShown classifies the active target for existing pane/shortcut
// callers. The context's tab key is the sole source of active-file identity;
// a hidden historical patch falls back to the permanent first tab.
func (a *app) detailTabShown() string {
	if !a.detailTabs().Active.Summary && (a.worktreeSelected || !a.detailsDiffHidden) {
		return detailTabFile
	}
	return detailTabSummary
}

// The first tab remains available even when metadata is hidden.
func (a *app) detailSummaryShown() bool {
	return true
}

// detailFileShown reports whether the file tab is one of the pane's tabs.
func (a *app) detailFileShown() bool {
	return len(a.detailTabs().OpenFiles) > 0 && (a.worktreeSelected || !a.detailsDiffHidden)
}

// historyFilesPane lists the selection's changed files: the working tree's
// when its row is selected, otherwise the commit's. Sublime Merge puts this
// list in its own column between the commits and the patch.
func (a *app) historyFilesPane(c *ui.Context) {
	if a.detailsChangesHidden && a.page != "changes" {
		return
	}
	tk := theme.TokensOf(c)
	count := len(a.commitFiles)
	if a.worktreeSelected {
		count = len(a.status.Entries)
	}
	ui.Column(c).Width(a.historyFilesWidth()).Shrink(0).MinHeight(0).AlignSelf(ui.Stretch).
		Children(func() {
			ui.Row(c).Height(38).Shrink(0).Padding(0, 12).Gap(8).AlignItems(ui.Center).
				BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
				display.Text(c, a.t("filesTab"), display.TextProps{Strong: true, FontSize: scaledFont(12)})
				if !a.worktreeSelected && (a.commitFilesLoading || a.commitFilesErr != "") {
					return
				}
				display.Tag(c, fmt.Sprint(count), display.TagProps{Variant: display.TagFilled, Size: theme.Small, Round: true})
				// The working tree keeps the two bulk stage controls the
				// workspace header used to carry; a commit's file list has no
				// index to move files in or out of.
				if a.worktreeSelected {
					ui.Row(c).Grow(1).Basis(0)
					a.filesToolbar(c)
				}
			})
			filesScroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitFilesScroll}, func() {
				ui.Column(c).FillWidth().Padding(7).Gap(1).FocusGroup(ui.Horizontal | ui.Vertical).Children(func() {
					if a.worktreeSelected {
						a.workingFileSections(c)
						return
					}
					a.commitFileList(c)
				})
			}).Grow(1).Basis(0).MinHeight(0)
			a.commitFilesViewport = filesScroll.Bounds()
			if a.page == "changes" {
				a.changesCommitPane(c)
			}
		})
}

// historyDiffPane keeps the permanent Commit/Summary tab beside the open file
// tabs. The first tab combines context information and all relevant patches;
// each file tab projects exactly one path and, in the worktree, stage version.
func (a *app) historyDiffPane(c *ui.Context) {
	tk := theme.TokensOf(c)
	// The strip wears a quiet fill so the active tab, painted in the page's
	// own ground, reads as one of the pane and the others as put away.
	// The files pane draws the divider between the two details columns; a
	// border here would sit on the window's outer edge and leave the columns
	// themselves separated by nothing.
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).AlignSelf(ui.Stretch).
		Children(func() {
			// The active tab runs the full height of the strip and shares the
			// page's ground, so it meets the body below with no gray seam; a
			// bottom border here would cut that join.
			strip := layout.ScrollArea(c, layout.ScrollAreaProps{Direction: layout.Horizontal, State: &a.detailTabStrip}, func() {
				ui.Row(c).Height(38).Gap(2).Padding(0, 8).AlignItems(ui.Center).
					Background(tk.FillTertiary).Role(ui.RoleTabList).Label(a.t("detailTabs")).FocusGroup(ui.Horizontal).Children(func() {
					a.detailTabButton(c, tk, detailTabKey{Summary: true}, a.detailSummaryLabel(), "")
					if a.detailFileShown() {
						// A close click mutates OpenFiles during this render pass.
						// Keep the current frame's iteration stable until the next.
						for _, file := range slices.Clone(a.detailTabs().OpenFiles) {
							name, _ := splitRepoPath(file.Path)
							if a.worktreeSelected {
								stage := "unstaged"
								if file.Staged {
									stage = "staged"
								}
								name += " · " + a.t(stage)
							}
							a.detailTabButton(c, tk, detailTabKey{File: file}, name, file.Path)
						}
					}
				})
			}).Height(38).Shrink(0).FillWidth()
			previousWidth := ui.Local(strip, "detail-tab-strip-width", func() float32 { return 0 })
			width := strip.Bounds().W
			if width > 0 && *previousWidth > 0 && width < *previousWidth-0.5 {
				// Existing horizontal offset can leave the active tab outside a
				// narrower viewport. Reveal it again once new bounds are known.
				a.detailReveal = true
				a.invalidate()
			}
			*previousWidth = width
			a.worktreeSelectionBar(c)
			a.historyDiffBody(c, tk)
		})
}

// detailTabButton draws one tab. It fills the strip and wears the page's
// surface when active, so it reads as the pane's edge rather than a floating
// pill; the tooltip carries the full file path the label shortens.
func (a *app) detailTabButton(c *ui.Context, tk *theme.Tokens, key detailTabKey, label, tooltip string) {
	c = c.Key(key)
	active := a.detailTabs().Active == key
	wrapper := ui.Row(c).Height(38).Shrink(0).MaxWidth(260).AlignItems(ui.Center).
		Radius(tk.BorderRadius, tk.BorderRadius, 0, 0)
	if active {
		wrapper.Background(tk.BgContainer)
	}
	wrapper.Children(func() {
		tab := ui.ButtonBase(c).Label(label).Row().Height(38).Padding(0, 10).MinWidth(0).
			Role(ui.RoleTab).AccessSelected(active)
		a.wireDetailTabKeys(tab, key)
		if tooltip != "" {
			tab.Tooltip(tooltip)
		}
		style := display.TextSecondary
		if active {
			style = display.TextDefault
		} else if tab.Hovered() {
			tab.Background(tk.FillSecondary)
		}
		tab.Children(func() {
			display.Text(c, label, display.TextProps{FontSize: scaledFont(11.5), Strong: active, Type: style}).SingleLine().MaxWidth(204)
		})
		if tab.Clicked() {
			a.activateDetailTab(key)
		}
		if !key.Summary {
			close := ui.ButtonBase(c).Label(a.t("close") + " " + tooltip).Width(26).Height(30).Shrink(0).Center()
			a.wireDetailTabKeys(close, key)
			close.Children(func() { ui.Icon(c, lucide.X()).FontSize(scaledFont(12)).TextColor(tk.TextSecondary) })
			if close.Clicked() {
				a.closeDetailFile(key.File)
				a.requestDetailTabFocus()
			}
		}
		a.focusRequestedDetailTab(tab, key)
	})
	if active && a.detailReveal {
		wrapper.ScrollIntoView()
		a.detailReveal = false
	}
}

// historyDiffBody draws the active tab: the commit message, the working tree's
// commit box, or the selected file's patch.
func (a *app) historyDiffBody(c *ui.Context, tk *theme.Tokens) {
	switch a.detailTabShown() {
	case detailTabFile:
		a.historyDiffFile(c, tk)
	case detailTabSummary:
		if !a.detailSummaryShown() {
			// Settings can hide every pane; a hand-edited file is the only way
			// here, since the page keeps one visible. Say so instead of
			// drawing a column with nothing in it.
			a.detailPlaceholder(c)
			return
		}
		a.detailOverview(c)
	}
}

// detailPlaceholder fills the diff pane when there is no commit to describe.
func (a *app) detailPlaceholder(c *ui.Context) {
	tk := theme.TokensOf(c)
	ui.Column(c).Fill().Center().Gap(8).Padding(24).Children(func() {
		ui.Icon(c, lucide.GitCommitHorizontal()).FontSize(scaledFont(22)).TextColor(tk.TextTertiary)
		display.Text(c, a.t(a.detailsPlaceholderKey()), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)})
	})
}

// The header names the original path and the actual comparison alongside its
// guarded file actions, independently of a truncated or scrolled file tab.
func (a *app) worktreeDiffHeader(c *ui.Context, tk *theme.Tokens) {
	header := ui.Row(c).Height(scaledFont(48)).Shrink(0).Padding(0, 12).Gap(8).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary)
	header.Children(func() {
		baseline := a.t("indexToWorktree")
		if a.selectedStaged {
			baseline = a.t("headToIndex")
		}
		if entry, ok := a.selectedEntry(); ok && entry.Conflict {
			baseline = a.t("conflict")
		}
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			display.Text(c, a.selectedPath, display.TextProps{FontSize: scaledFont(11)}).FillWidth().SingleLine().Tooltip(a.selectedPath)
			display.Text(c, baseline, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(10)}).FillWidth().SingleLine().Tooltip(baseline)
		})
		entry, ok := a.selectedEntry()
		if !ok {
			return
		}
		if entry.Conflict {
			a.statusTag(c, a.t("conflict"), display.TagProps{
				Variant: display.TagFilled, Color: display.TagError, Size: theme.Small, Round: true,
			})
			button := workbenchButton(c, a.t("markResolved"), buttonNormal, a.busy || a.loading)
			if button.Clicked() {
				a.resolveConflict(entry.Path)
			}
			return
		}
		target, _ := a.fileActionTarget(entry, a.selectedStaged)
		a.fileActionButtons(c.Key("file-actions"), target)
	})
	a.diffHeaderMenu(c, header)
}

func (a *app) historyDiffFile(c *ui.Context, tk *theme.Tokens) {
	if a.worktreeSelected {
		if a.selectedPath == "" {
			a.diffPlaceholder(c, lucide.GitCommitHorizontal(), a.t("selectFile"))
			return
		}
		a.worktreeDiffHeader(c, tk)
		// diffBody owns the two-axis viewport and binds diffScroll itself.
		a.diffBody(c)
		return
	}
	if a.detailsDiffHidden {
		// detailTabShown never resolves here while the patch is hidden; this
		// keeps the body honest if a caller asks for the file directly.
		display.Text(c, a.t("diffHiddenNote"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).
			Padding(14)
		return
	}
	scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitDiffScroll}, func() {
		ui.Column(c).FillWidth().Padding(4).Children(func() {
			a.commitDiffView(c)
		})
	}).Grow(1).Basis(0).MinHeight(0)
	// A line drag that leaves the pane scrolls the patch; the box it is
	// measured against comes from this layout.
	a.commitDiffViewport = scroll.Bounds()
}
