package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func (a *app) worktreeOverview(c *ui.Context) {
	tk := theme.TokensOf(c)
	if len(a.overviewEntries) == 0 {
		key := "noChangesShort"
		if len(a.status.Entries) > 0 {
			key = "diffLoading"
		}
		display.Text(c, a.t(key), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).Padding(14)
		return
	}
	root, seq := a.overviewRoot, a.overviewSeq
	heading := ui.Row(c).FillWidth().Padding(8, 12).Gap(6).AlignItems(ui.Center).Children(func() {
		display.Text(c, a.t("overviewChanges"), display.TextProps{Strong: true, FontSize: scaledFont(12)})
		ui.Row(c).Grow(1).Basis(0)
		a.patchLayoutButton(c.Key("patch-layout"), nil)
		if action.ActionIcon(c, lucide.ChevronsDown(), action.ActionIconProps{Title: a.t("expandAllPatches"), Size: theme.Small}).Clicked() {
			a.setOverviewCollapsed(false)
		}
		if action.ActionIcon(c, lucide.ChevronsUp(), action.ActionIconProps{Title: a.t("collapseAllPatches"), Size: theme.Small}).Clicked() {
			a.setOverviewCollapsed(true)
		}
	})
	a.overviewBatchActions(c)
	var visible []detailFileKey
	viewport := a.commitDiffViewport
	draw := func(section *ui.Context, entry worktreeOverviewEntry) {
		block := a.worktreeOverviewFile(section.Key(entry.Key), tk, entry, root, seq)
		bounds := block.Bounds()
		if entry.Loading && !a.detailContexts.Worktree.Collapsed[entry.Key] && viewport.H > 0 && bounds.H > 0 &&
			bounds.Y < viewport.Y+viewport.H && bounds.Y+bounds.H > viewport.Y {
			visible = append(visible, entry.Key)
		}
	}
	for start := 0; start < len(a.overviewEntries); {
		entry := a.overviewEntries[start]
		// The heading shares the section width. Until its first layout gives
		// a real width, wrapping placeholders stay in ordinary layout.
		if _, exact := a.overviewSectionHeightFor(c, entry, heading.Bounds().W); !exact {
			draw(c, entry)
			start++
			continue
		}
		heights := make([]float32, 0)
		reveal, retained := -1, -1
		end := start
		for end < len(a.overviewEntries) {
			height, exact := a.overviewSectionHeightFor(c, a.overviewEntries[end], heading.Bounds().W)
			if !exact {
				break
			}
			heights = append(heights, height)
			if target := a.overviewLineTarget; target != nil && target.Root == root && target.Generation == seq && target.File == a.overviewEntries[end].Key {
				if a.revealLine {
					reveal = end - start
				}
				if a.lineDragHeld {
					retained = end - start
				}
			}
			end++
		}
		virtualPatchRows(c.Key(a.overviewEntries[start].Key).Key("sections"), viewport, &a.commitInfoScroll, heights, reveal, func(section *ui.Context, at int) {
			draw(section, a.overviewEntries[start+at])
		}, retained)
		start = end
	}
	if a.overviewQueue != nil {
		// Publishing priorities only reorders pending work; Git remains on its
		// serial worker, outside drawing and without borrowing mutable UI state.
		a.overviewQueue.prioritize(visible)
	}
}

// overviewSectionHeight reports only heights that do not depend on text
// wrapping. The one-DIP bottom border belongs to the section's layout extent.
func (a *app) overviewSectionHeight(entry worktreeOverviewEntry) (float32, bool) {
	height := float32(35)
	if a.detailContexts.Worktree.Collapsed[entry.Key] {
		return height, true
	}
	if _, conflict := a.conflictPreviewEntry(entry.Key.Path); conflict {
		return 0, false
	}
	if entry.Loading || entry.Err != "" || len(entry.Diff.Lines) == 0 {
		return 0, false
	}
	rows := newPatchRows(entry.Diff.Lines, nil, a.splitDiff)
	for i := range rows.len() {
		height += rows.height(a, i)
	}
	return height, true
}

func (a *app) setOverviewCollapsed(collapsed bool) {
	tabs := &a.detailContexts.Worktree
	if tabs.Collapsed == nil {
		tabs.Collapsed = make(map[detailFileKey]bool)
	}
	for _, entry := range a.overviewEntries {
		tabs.Collapsed[entry.Key] = collapsed
	}
	a.clearLineSelection()
	a.invalidate()
}

func (a *app) worktreeOverviewFile(c *ui.Context, tk *theme.Tokens, entry worktreeOverviewEntry, root string, seq uint64) ui.Element {
	key := entry.Key
	tabs := &a.detailContexts.Worktree
	collapsed := tabs.Collapsed[key]
	return ui.Column(c).FillWidth().Shrink(0).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).Children(func() {
		ui.Row(c).FillWidth().Height(34).AlignItems(ui.Center).Background(tk.FillQuaternary).Children(func() {
			stage := "unstaged"
			if key.Staged {
				stage = "staged"
			}
			label := key.Path + " · " + a.t(stage)
			header := ui.ButtonBase(c).Label(label).Row().Grow(1).Basis(0).MinWidth(0).Height(34).Padding(0, 10).Gap(6).AlignItems(ui.Center).Tooltip(key.Path)
			header.Children(func() {
				icon := lucide.ChevronDown()
				if collapsed {
					icon = lucide.ChevronRight()
				}
				ui.Icon(c, icon).FontSize(scaledFont(12)).TextColor(tk.TextSecondary)
				display.Text(c, key.Path, display.TextProps{FontSize: scaledFont(11.5), Strong: true}).SingleLine().Grow(1).Basis(0).MinWidth(0)
				display.Text(c, a.t(stage), display.TextProps{FontSize: scaledFont(10), Type: display.TextSecondary}).SingleLine().Shrink(0)
				if entry.Loading {
					ui.Icon(c, lucide.LoaderCircle()).FontSize(scaledFont(12)).TextColor(tk.TextTertiary)
				}
				if entry.Err != "" {
					ui.Icon(c, lucide.AlertTriangle()).FontSize(scaledFont(12)).TextColor(tk.Error)
				}
			})
			if header.Clicked() {
				if tabs.Collapsed == nil {
					tabs.Collapsed = make(map[detailFileKey]bool)
				}
				tabs.Collapsed[key] = !collapsed
				a.clearLineSelection()
				a.invalidate()
			}
			if a.repo != nil && a.repo.Root == root && a.overviewRoot == root && a.overviewSeq == seq {
				for _, statusEntry := range a.status.Entries {
					if statusEntry.Path == key.Path {
						target, _ := a.fileActionTarget(statusEntry, key.Staged)
						a.fileActionButtons(c.Key("file-actions"), target)
						break
					}
				}
			}
			if action.ActionIcon(c, lucide.ExternalLink(), action.ActionIconProps{Title: a.t("openFileTab") + " " + key.Path, Size: theme.Small}).Clicked() {
				a.openDetailFile(key)
			}
		})
		if collapsed {
			return
		}
		if statusEntry, conflict := a.conflictPreviewEntry(key.Path); conflict {
			a.conflictPreviewGuidance(c, statusEntry, true)
		}
		if entry.Loading {
			display.Text(c, a.t("diffLoading"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).Padding(12)
			return
		}
		if entry.Err != "" {
			ui.Column(c).FillWidth().Padding(12).Gap(6).Children(func() {
				display.Text(c, entry.Err, display.TextProps{Type: display.TextDanger, FontSize: scaledFont(11), Rows: 4})
				if workbenchButton(c, a.t("retry"), buttonNormal, false).Clicked() {
					a.retryWorktreeOverview(root, seq, key)
				}
			})
			return
		}
		if len(entry.Diff.Lines) == 0 {
			display.Text(c, a.t("diffEmpty"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).Padding(12)
			return
		}
		_, conflict := a.conflictPreviewEntry(key.Path)
		split := a.splitDiff && !conflict
		var widths patchWidths
		if split {
			widths = a.splitPatchWidths(entry.Diff.Lines, a.commitDiffViewport.W)
		} else {
			widths.canvas = a.diffWidth(entry.Diff)
		}
		rows := newPatchRows(entry.Diff.Lines, nil, split)
		target := overviewPatchTarget{Root: root, Generation: seq, File: key, Diff: entry.Diff}
		reveal := -1
		revealTarget := a.revealLine && a.overviewSelectionMatches(target)
		height := float32(0)
		heights := make([]float32, rows.len())
		for i := range heights {
			heights[i] = rows.height(a, i)
			height += heights[i]
			if revealTarget && rowContainsLine(rows.at(i), a.lineCursor) {
				reveal = i
			}
		}
		layout.ScrollArea(c.Key(split), layout.ScrollAreaProps{Direction: layout.Horizontal}, func() {
			ui.Column(c).Width(widths.canvas).Selectable().Children(func() {
				a.paintPatchRows(c, rows, widths, a.commitDiffViewport, &a.commitInfoScroll, heights, reveal, -1, func(cell *ui.Context, index int, cellWidth float32, side patchSide) {
					a.overviewPatchLineForSide(cell, target, index, cellWidth, side)
				})
			})
		}).FillWidth().Height(height).Shrink(0)
	})
}

// detailOverview is the permanent first tab. Its metadata and patches share a
// vertical reading position; opening a file tab keeps that position intact.
func (a *app) detailOverview(c *ui.Context) {
	tk := theme.TokensOf(c)
	scroll := layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.commitInfoScroll}, func() {
		ui.Column(c).FillWidth().Children(func() {
			if a.worktreeSelected {
				if a.page != "changes" {
					ui.Column(c).FillWidth().Padding(14).Gap(12).Children(func() {
						a.commitForm(c)
						a.operationCard(c)
					})
				}
				a.worktreeOverview(c)
				return
			}
			if commit, ok := a.selectedCommit(); ok {
				if !a.detailsInfoHidden {
					a.commitInfoContent(c, commit, tk)
				}
				if !a.detailsDiffHidden {
					ui.Column(c).FillWidth().Padding(historyPatchInset).Children(func() { a.commitDiffView(c) })
				}
			} else {
				a.detailPlaceholder(c)
			}
		})
	}).Grow(1).Basis(0).MinHeight(0)
	a.recordPatchViewport(c, scroll.Bounds())
}
