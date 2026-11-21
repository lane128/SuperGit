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
	historyFilesPaneWidth float32 = 240
	historyFilesPaneMin   float32 = 180
	historyDiffPaneMin    float32 = 300
	// Working files also have stage controls and a group selection count.
	worktreeFilesPaneWidth float32 = 300
)

// historyFilesDefaultWidth is the Files column's opening width for the
// current subject.
func (a *app) historyFilesDefaultWidth() float32 {
	if a.worktreeSelected {
		return worktreeFilesPaneWidth
	}
	return historyFilesPaneWidth
}

// historyWidthPressure transitions to the compact Files column at the
// comfortable window width. Base it on the whole window so selection and
// divider drags cannot move the responsive boundary. The boundary stays at
// 980 even though the window can now shrink to 900: these readings were tuned
// there, and widening the range would change them at every size in between.
func (a *app) historyWidthPressure() float32 {
	const normalWidth, compactWidth = 1160, 980
	width := a.sidebarSplit.rowWidth
	if width <= 0 {
		return 0
	}
	return min(1, max(0, (normalWidth-width)/(normalWidth-compactWidth)))
}

// historyNarrowPressure measures how far the window has gone below the
// comfortable 980 points. Only the working tree's row uses it: its commit
// editor lives under the Files column, so below 980 the editor keeps its
// width instead of following the commit list's compact floor. At and above
// 980 the pressure is zero and the established layout is unchanged.
func (a *app) historyNarrowPressure() float32 {
	if !a.worktreeSelected {
		return 0
	}
	const comfortableWidth = 980
	width := a.sidebarSplit.rowWidth
	if width <= 0 {
		return 0
	}
	return min(1, max(0, (comfortableWidth-width)/(comfortableWidth-minWindowWidth)))
}

func (a *app) historyFilesMinWidth() float32 {
	const compactFilesMin = 96
	filesMin := historyFilesPaneMin - a.historyWidthPressure()*(historyFilesPaneMin-compactFilesMin)
	return max(filesMin, historyFilesPaneMin*a.historyNarrowPressure())
}

// historyPatchMinWidth gives wider History windows a more useful reading
// width even when a saved split heavily favors the Files column. Below 980
// the working tree's patch gives room back to the commit editor.
func (a *app) historyPatchMinWidth() float32 {
	const comfortableWidth, wideWindow = 980, 1600
	const extraPatchWidth = 120
	width := a.sidebarSplit.rowWidth
	progress := min(1, max(0, (width-comfortableWidth)/(wideWindow-comfortableWidth)))
	return historyDiffPaneMin - 70*a.historyNarrowPressure() + progress*extraPatchWidth
}

// historyFilesWidth is the files column's width this frame, read from the
// share the row was laid out with.
func (a *app) historyFilesWidth() float32 {
	row := a.filesSplit.rowWidth
	if row <= 0 && a.historySplit.rowWidth > 0 {
		row = (a.historySplit.rowWidth - splitterWidth) * (1 - a.historyShare())
	}
	// One set of columns for every subject: the working tree's files take the
	// same width as a commit's, since the commit editor lives on the Summary
	// tab rather than under the list.
	width, minimum := a.historyFilesDefaultWidth(), a.historyFilesMinWidth()
	patchMin := a.historyPatchMinWidth()
	if row <= splitterWidth {
		return width
	}
	state := a.filesSplit
	state.rowWidth = row
	def := width / (row - splitterWidth)
	return max(minimum, (row-splitterWidth)*state.fraction(def, minimum, patchMin))
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
func (a *app) historyDetailsPane(c *ui.Context, share, width float32) {
	row := ui.Row(c).Grow(share).Basis(0).MinWidth(0).AlignSelf(ui.Stretch)
	a.filesSplit.rowWidth = width
	// The three-way merge editor needs the whole details area: its two source
	// panes and the result cannot share the width with the file list.
	conflictEditing := a.conflictEditorTakesThePane()
	row.Children(func() {
		if !conflictEditing {
			a.historyFilesPane(c)
			if !a.detailsChangesHidden {
				def := shareForWidth(a.filesSplit.rowWidth-splitterWidth, a.historyFilesDefaultWidth())
				a.splitHandle(c, &a.filesSplit, def, a.historyFilesMinWidth(), a.historyPatchMinWidth(), a.t("resizeFilesPane"))
			}
		}
		a.historyDiffPane(c)
	})
}

// conflictEditorTakesThePane reports whether the merge editor is the pane on
// screen: the working tree's conflict file is open in the editor, whether it
// was opened from the Summary overview or from the file's own tab.
func (a *app) conflictEditorTakesThePane() bool {
	if !a.worktreeSelected {
		return false
	}
	if _, conflict := a.conflictPreviewEntry(a.selectedPath); !conflict {
		return false
	}
	editor := a.currentConflictEditor()
	return editor != nil && !editor.Preview
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
	if a.detailsChangesHidden && !a.worktreeSelected {
		return
	}
	ui.Column(c).Width(a.historyFilesWidth()).Shrink(0).MinHeight(0).AlignSelf(ui.Stretch).
		Children(func() {
			a.filesPaneBody(c)
		})
}

// The Files column keeps file groups and their staging actions together.
func (a *app) filesPaneBody(c *ui.Context) {
	tk := theme.TokensOf(c)
	width := a.historyFilesWidth()
	ui.Column(c).FillWidth().Grow(1).Basis(0).MinHeight(0).Children(func() {
		header := ui.Row(c).Height(38).Shrink(0).Padding(0, 12).Gap(8).AlignItems(ui.Center).
			Justify(ui.SpaceBetween).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary)
		if a.worktreeSelected && width < scaledFont(170) {
			// Keep the same controls in the same parent while the heading
			// and actions move onto separate lines.
			header.Column().Height(62).Padding(6, 6).Gap(4).AlignItems(ui.Start).Justify(ui.Center)
		}
		header.Children(func() {
			ui.Row(c.Key("files-heading")).Gap(8).AlignItems(ui.Center).Children(func() {
				display.Text(c, a.t("filesTab"), display.TextProps{Strong: true, FontSize: scaledFont(12)})
				if a.worktreeSelected {
					display.Tag(c, fmt.Sprint(len(a.status.Entries)), display.TagProps{Variant: display.TagFilled, Size: theme.Small, Round: true})
				}
			})
			if a.worktreeSelected {
				ui.Row(c.Key("files-actions")).Gap(4).Children(func() { a.filesToolbar(c) })
			}
		})
		filesScroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitFilesScroll}, func() {
			if a.worktreeSelected {
				ui.Column(c).FillWidth().Gap(0).FocusGroup(ui.Horizontal | ui.Vertical).Children(func() {
					a.workingFileSections(c)
				})
				return
			}
			ui.Column(c).FillWidth().Padding(7).Gap(1).FocusGroup(ui.Horizontal | ui.Vertical).Children(func() {
				a.commitFileList(c)
			})
		}).Grow(1).Basis(0).MinHeight(0)
		a.commitFilesViewport = filesScroll.Bounds()
	})
}

// The worktree composer sits above the patch, leaving the Files column its
// full height. One editor owns the complete message, including pasted bodies.
func (a *app) worktreeCommitPane(c *ui.Context) {
	tk := theme.TokensOf(c)
	pane := ui.Column(c).FillWidth().Shrink(0).Padding(10, 12).Gap(10).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary)
	wideMeta := ui.Local(pane, "composer-wide-meta", func() bool { return false })
	pane.Children(func() {
		a.operationCard(c)
		a.commitMessageField(c)
		ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).Wrap().Children(func() {
			canInline := *wideMeta && !a.commitIdentityHasExtraLines()
			if canInline {
				ui.Row(c).Grow(1).Basis(scaledFont(240)).MinWidth(0).Gap(14).AlignItems(ui.Center).Children(func() {
					a.commitIdentityViewInline(c.Key("identity"), true)
					if a.canShowAmendControl() {
						ui.Box(c).Width(1).Height(scaledFont(13)).Background(tk.Border).Shrink(0)
						ui.Row(c.Key("amend")).Shrink(0).AlignItems(ui.Center).Children(func() {
							a.commitAmendControl(c)
						})
					}
				})
			} else {
				ui.Column(c).Grow(1).Basis(scaledFont(200)).MinWidth(0).Gap(8).Children(func() {
					a.commitIdentityViewInline(c.Key("identity"), false)
					if a.canShowAmendControl() {
						ui.Row(c.Key("amend")).AlignItems(ui.Center).Children(func() {
							a.commitAmendControl(c)
						})
					}
				})
			}
			a.commitAction(c, true)
		})
		a.commitActionReason(c)
	})
	if w := pane.Bounds().W; w > 0 {
		nextWide := w >= scaledFont(520)
		if nextWide != *wideMeta {
			*wideMeta = nextWide
			a.invalidate()
		}
	}
}

// historyDiffPane keeps the permanent Commit/Summary tab beside the open file
// tabs. The first tab combines context information and all relevant patches;
// each file tab projects exactly one path and, in the worktree, stage version.
func (a *app) historyDiffPane(c *ui.Context) {
	tk := theme.TokensOf(c)
	// The files pane draws the divider between the two details columns; a
	// border here would sit on the window's outer edge and leave the columns
	// themselves separated by nothing.
	pane := ui.Column(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).AlignSelf(ui.Stretch)
	a.wireDiffFind(c, pane).
		Children(func() {
			// Keep the thumb visible on overflow so a clipped tab can be reached
			// with a mouse even before the pointer enters the strip.
			strip := ui.ScrollHorizontal(c).Height(32).Shrink(0).FillWidth().Background(tk.BgLayout).
				KeepScrollbars(true).TrackScroll(&a.detailTabStrip)
			strip.Children(func() {
				ui.Row(c).Height(32).AlignItems(ui.Center).
					Role(ui.RoleTabList).Label(a.t("detailTabs")).FocusGroup(ui.Horizontal).Children(func() {
					a.detailTabButton(c, tk, detailTabKey{Summary: true})
					if a.detailFileShown() {
						// A close click mutates OpenFiles during this render pass.
						// Keep the current frame's iteration stable until the next.
						for _, file := range slices.Clone(a.detailTabs().OpenFiles) {
							a.detailTabButton(c, tk, detailTabKey{File: file})
						}
					}
				})
			})
			previousWidth := ui.Local(strip, "detail-tab-strip-width", func() float32 { return 0 })
			width := strip.Bounds().W
			if width > 0 && *previousWidth > 0 && width < *previousWidth-0.5 {
				// Existing horizontal offset can leave the active tab outside a
				// narrower viewport. Reveal it again once new bounds are known.
				a.detailReveal = true
				a.invalidate()
			}
			*previousWidth = width
			if a.worktreeSelected && !a.conflictEditorTakesThePane() {
				a.worktreeCommitPane(c.Key("composer"))
			}
			a.worktreeSelectionBar(c)
			a.drawDiffFindBar(c.Key("find-in-diff"), a.diffFind, a.diffFindSections, a.closeDiffFind)
			a.historyDiffBody(c, tk)
		})
}

func (a *app) detailTabStatus(file detailFileKey) string {
	if a.worktreeSelected {
		for _, entry := range a.status.Entries {
			if entry.Path == file.Path {
				return statusCode(entry, file.Staged)
			}
		}
	} else {
		for _, entry := range a.commitFiles {
			if entry.Path == file.Path {
				return changedFileCode(entry.Status)
			}
		}
	}
	return ""
}

// detailTabButton keeps the full path and worktree version in the tooltip.
func (a *app) detailTabButton(c *ui.Context, tk *theme.Tokens, key detailTabKey) {
	var name, suffix string
	path := key.File.Path
	if key.Summary {
		name = a.detailSummaryLabel()
	} else {
		name, _ = splitRepoPath(path)
		for _, other := range a.detailTabs().OpenFiles {
			if base, _ := splitRepoPath(other.Path); other.Path != path && base == name {
				name = path
				break
			}
		}
		if a.worktreeSelected {
			stage := "unstaged"
			if key.File.Staged {
				stage = "staged"
			}
			suffix = " · " + a.t(stage)
		}
	}
	c = c.Key(key)
	active := a.detailTabs().Active == key
	wrapper := ui.Row(c).Height(32).Shrink(0).MaxWidth(260).AlignItems(ui.Center).Padding(0, 1, 2, 0)
	if active {
		wrapper.Padding(0, 0, 2, 0)
	}
	ink := paintWorkbenchTab(wrapper, tk, active, 2)
	var close ui.Element
	wrapper.Children(func() {
		const tabPadding, captionWidth = 10, 204
		tab := ui.ButtonBase(c).Label(name+suffix).Row().Height(30).Padding(0, tabPadding).
			MinWidth(0).MaxWidth(captionWidth + 2*tabPadding).AlignItems(ui.Center).
			Role(ui.RoleTab).AccessSelected(active)
		a.wireDetailTabKeys(tab, key)
		if path != "" {
			tab.Tooltip(path + suffix)
		}
		tab.Children(func() {
			props := display.TextProps{FontSize: scaledFont(11.5), Strong: active}
			if path != "" {
				if code := a.detailTabStatus(key.File); code != "" {
					display.Text(c, code, display.TextProps{Code: true, FontSize: scaledFont(10), Type: display.TextSecondary}).Font(a.patchFontFamily()).Shrink(0).Padding(0, 4).AccessHidden(true)
				}
			}
			// The file name can shorten, but the two versions must stay distinct.
			display.Text(c, name, props).TextColor(ink).SingleLine().MinWidth(0).Shrink(1)
			if suffix != "" {
				display.Text(c, suffix, props).TextColor(ink).SingleLine().Shrink(0)
			}
		})
		if tab.Clicked() {
			a.activateDetailTab(key)
		}
		if !key.Summary {
			close = ui.ButtonBase(c).Label(a.t("close") + " " + path).Width(22).Height(26).Shrink(0).Center()
			a.wireDetailTabKeys(close, key)
			styleTabClose(close, tk, ink)
			close.Children(func() { ui.Icon(c, lucide.X()).FontSize(scaledFont(12)) })
			if close.Clicked() {
				a.closeDetailFile(key.File)
				a.requestDetailTabFocus()
			}
		}
		a.focusRequestedDetailTab(tab, key)
	})
	// A click inside the wrapper can change the active tab mid-frame.
	// Only the resulting selection may consume the pending reveal. Reveal the
	// Close action, not just the wrapper: on a narrow strip the wrapper can be
	// visible while its trailing Close button is clipped by the viewport.
	if a.detailTabs().Active == key && a.detailReveal {
		if key.Summary {
			wrapper.ScrollIntoView()
		} else {
			close.ScrollIntoView()
		}
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
		if !a.diffLoading && a.diffErr == "" && len(a.diff.Lines) > 0 && (!ok || !entry.Conflict) {
			a.patchLayoutButton(c.Key("patch-layout"), func() {
				lines := a.diff.Lines
				visible := worktreeVisiblePatchRows(lines)
				before := newPatchRows(lines, visible, a.splitDiff)
				after := newPatchRows(lines, visible, !a.splitDiff)
				a.diffScroll.Y = a.patchOffset(before, after, a.diffScroll.Y)
			})
		}
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
		if entry, ok := a.selectedEntry(); ok && !entry.Conflict && a.worktreePreview(c, entry) {
			return
		}
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
	if commit, ok := a.selectedCommit(); ok && a.selectedSHA != "" {
		label := a.t("commitDiffRootBaseline")
		if len(commit.Parents) > 0 {
			label = a.tf("commitDiffParentBaseline", shortSHA(commit.Parents[0]), shortSHA(commit.SHA))
		}
		ui.Column(c).FillWidth().Shrink(0).Padding(6, 10).Children(func() {
			display.Text(c, label, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 2}).FillWidth()
		})
	}
	if a.historyFilePreview(c) {
		return
	}
	scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitDiffScroll}, func() {
		ui.Column(c).FillWidth().Padding(historyPatchInset).Children(func() {
			a.commitDiffView(c)
		})
	}).Grow(1).Basis(0).MinHeight(0)
	// A line drag that leaves the pane scrolls the patch; the box it is
	// measured against comes from this layout.
	a.recordPatchViewport(c, scroll.Bounds())
}

func (a *app) recordPatchViewport(c *ui.Context, bounds ui.Rect) {
	if (!a.worktreeSelected || a.splitDiff) && bounds.W != a.commitDiffViewport.W {
		// Bounds describe the previous layout. Redraw the patch at the new width.
		c.After(0)
	}
	a.commitDiffViewport = bounds
}
