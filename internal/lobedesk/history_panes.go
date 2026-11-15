package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"fmt"
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
	row := a.historySplit.rowWidth
	if row <= 0 {
		return historyFilesPaneWidth
	}
	// The splitter is fixed width, so the two panes share what is left of the
	// row; measuring against the whole row over-subtracted it from the patch.
	details := (row - splitterWidth) * (1 - a.historyShare())
	if details <= 0 {
		return historyFilesPaneWidth
	}
	return min(historyFilesPaneWidth, max(historyFilesPaneMin, details-historyDiffPaneMin))
}

const (
	detailTabSummary = "summary"
	detailTabFile    = "file"
)

// historyDetailsPane is the share right of the commit list: the changed files
// and the patch side by side. The details toggle in the page header collapses
// both columns at once.
func (a *app) historyDetailsPane(c *ui.Context) {
	ui.Row(c).Grow(1 - a.historyShare()).Basis(0).MinWidth(0).AlignSelf(ui.Stretch).Children(func() {
		a.historyFilesPane(c)
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

// detailTabShown resolves the tab to draw. Settings can hide the commit
// message or the commit's patch; the working tree always keeps its Summary,
// because the commit box lives there. A pane with no file open, or with the
// file hidden, falls back to the Summary so it is never blank.
func (a *app) detailTabShown() string {
	showSummary := a.worktreeSelected || !a.detailsInfoHidden
	showFile := a.detailFilePath() != "" && (a.worktreeSelected || !a.detailsDiffHidden)
	if !showSummary && showFile {
		return detailTabFile
	}
	if !showFile {
		return detailTabSummary
	}
	if a.detailTab == detailTabFile {
		return detailTabFile
	}
	return detailTabSummary
}

// detailSummaryShown reports whether the Summary tab is one of the pane's tabs.
func (a *app) detailSummaryShown() bool {
	return a.worktreeSelected || !a.detailsInfoHidden
}

// detailFileShown reports whether the file tab is one of the pane's tabs.
func (a *app) detailFileShown() bool {
	return a.detailFilePath() != "" && (a.worktreeSelected || !a.detailsDiffHidden)
}

// historyFilesPane lists the selection's changed files: the working tree's
// when its row is selected, otherwise the commit's. Sublime Merge puts this
// list in its own column between the commits and the patch.
func (a *app) historyFilesPane(c *ui.Context) {
	if a.detailsChangesHidden {
		return
	}
	tk := theme.TokensOf(c)
	count := len(a.commitFiles)
	if a.worktreeSelected {
		count = len(a.status.Entries)
	}
	ui.Column(c).Width(a.historyFilesWidth()).Shrink(0).MinHeight(0).AlignSelf(ui.Stretch).
		BorderWidth(0, 1, 0, 0).BorderColor(tk.BorderSecondary).Children(func() {
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
		layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitFilesScroll}, func() {
			ui.Column(c).FillWidth().Padding(7).Gap(1).Children(func() {
				if a.worktreeSelected {
					a.workingFileSections(c)
					return
				}
				a.commitFileList(c)
			})
		}).Grow(1).Basis(0).MinHeight(0)
	})
}

// historyDiffPane is the patch column. Its tab strip keeps the commit Summary
// beside the open file, the way Sublime Merge's viewer does; the working tree
// uses the Summary tab for its commit box.
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
			ui.Row(c).Height(38).Shrink(0).Gap(2).Padding(0, 8).AlignItems(ui.Center).
				Background(tk.FillTertiary).Role(ui.RoleTabList).Children(func() {
				if a.detailSummaryShown() {
					a.detailTabButton(c, tk, detailTabSummary, a.t("summaryTab"), "")
				}
				if a.detailFileShown() {
					path := a.detailFilePath()
					name, _ := splitRepoPath(path)
					a.detailTabButton(c, tk, detailTabFile, name, path)
				}
			})
			a.historyDiffBody(c, tk)
		})
}

// detailTabButton draws one tab. It fills the strip and wears the page's
// surface when active, so it reads as the pane's edge rather than a floating
// pill; the tooltip carries the full file path the label shortens.
func (a *app) detailTabButton(c *ui.Context, tk *theme.Tokens, key, label, tooltip string) {
	active := a.detailTabShown() == key
	tab := ui.ButtonBase(c).Label(label).Row().Height(38).Padding(0, 10).Gap(6).
		Radius(tk.BorderRadius, tk.BorderRadius, 0, 0).AlignItems(ui.Center).
		Role(ui.RoleTab).AccessSelected(active)
	if key == detailTabFile {
		// The file tab takes what the Summary leaves, and the text ellipsizes
		// inside it: a fixed 220 cap plus padding overflowed a 300-wide strip,
		// clipping the name at the pane's edge.
		tab.Grow(1).Basis(0).MinWidth(0)
	}
	if tooltip != "" {
		tab.Tooltip(tooltip)
	}
	style := display.TextSecondary
	if active {
		tab.Background(tk.BgContainer)
		style = display.TextDefault
	} else if tab.Hovered() {
		tab.Background(tk.FillSecondary)
	}
	tab.Children(func() {
		text := display.Text(c, label, display.TextProps{
			FontSize: scaledFont(11.5),
			Strong:   active,
			Type:     style,
		}).SingleLine()
		if key == detailTabFile {
			text.Grow(1).Basis(0).MinWidth(0)
		} else {
			text.MaxWidth(220)
		}
	})
	if tab.Clicked() {
		a.detailTab = key
		a.invalidate()
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
		if a.worktreeSelected {
			layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitInfoScroll}, func() {
				ui.Column(c).FillWidth().Padding(14).Gap(12).Children(func() {
					a.commitForm(c)
					a.operationCard(c)
				})
			}).Grow(1).Basis(0).MinHeight(0)
			return
		}
		commit, ok := a.selectedCommit()
		if !ok {
			a.detailPlaceholder(c)
			return
		}
		layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitInfoScroll}, func() {
			a.commitInfoContent(c, commit, tk)
		}).Grow(1).Basis(0).MinHeight(0)
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

// worktreeDiffHeader keeps the stage action and the file menu beside the
// patch they act on, as the workspace's diff pane did. The tab above already
// names the file, so the header adds its folder rather than repeating the
// name; a header would otherwise cost a row of height to say nothing new.
func (a *app) worktreeDiffHeader(c *ui.Context, tk *theme.Tokens) {
	header := ui.Row(c).Height(34).Shrink(0).Padding(0, 12).Gap(8).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary)
	header.Children(func() {
		if _, dir := splitRepoPath(a.selectedPath); dir != "" {
			display.Text(c, dir, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).SingleLine().MaxWidth(260).Tooltip(dir)
		}
		if a.selectedStaged {
			a.statusTag(c, a.t("staged"), display.TagProps{Variant: display.TagFilled, Color: display.TagSuccess, Size: theme.Small, Round: true})
		}
		ui.Row(c).Grow(1).Basis(0)
		entry, ok := a.selectedEntry()
		if !ok {
			return
		}
		if entry.Conflict {
			a.statusTag(c, a.t("conflict"), display.TagProps{
				Variant: display.TagFilled, Color: display.TagError, Size: theme.Small, Round: true,
			})
			button := action.Button(c, a.t("markResolved"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).
				Disabled(a.busy || a.loading)
			if button.Clicked() {
				a.resolveConflict(entry.Path)
			}
			return
		}
		label := a.t("stageFile")
		if a.selectedStaged {
			label = a.t("unstageFile")
		}
		button := action.Button(c, label, action.ButtonProps{Variant: action.Default, Size: theme.Small}).
			Disabled(a.busy || a.loading)
		if button.Clicked() {
			a.toggleStage(entry, a.selectedStaged)
		}
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
		layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.diffScroll}, func() {
			a.diffBody(c)
		}).Grow(1).Basis(0).MinHeight(0)
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
