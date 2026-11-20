package lobedesk

import (
	"context"
	"errors"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

// All data needed to apply a patch travels with the displayed block. The
// workspace's preloaded selectedPath/selectedStaged never determines a write.
type overviewPatchTarget struct {
	Root       string
	Generation uint64
	File       detailFileKey
	Diff       gitrepo.Diff
}

func (a *app) overviewTargetVisible(target overviewPatchTarget) bool {
	if a.repo == nil || a.repo.Root != target.Root || a.overviewRoot != target.Root || a.overviewSeq != target.Generation ||
		!a.detailPaneVisible() || !a.worktreeSelected || !a.detailTabs().Active.Summary || a.detailContexts.Worktree.Collapsed[target.File] {
		return false
	}
	for _, entry := range a.overviewEntries {
		if entry.Key == target.File {
			return !entry.Loading && entry.Err == "" && entry.Diff.Text == target.Diff.Text
		}
	}
	return false
}

func (a *app) overviewLineAction(target overviewPatchTarget, indexes []int) func(context.Context, *gitrepo.Repository) error {
	if !a.overviewTargetVisible(target) || a.loading || a.busy || a.historyRefOpen ||
		a.confirm.open || a.prompt.open || a.blameOpen || a.worktreeForm.open || a.reset.open || !target.Diff.CanStageHunks {
		return nil
	}
	for _, entry := range a.status.Entries {
		if entry.Path == target.File.Path && entry.Conflict {
			return nil
		}
	}
	choice := make([]int, 0, len(indexes))
	for _, index := range indexes {
		if index >= 0 && index < len(target.Diff.Lines) {
			kind := target.Diff.Lines[index].Kind
			if kind == "add" || kind == "del" {
				choice = append(choice, index)
			}
		}
	}
	if len(choice) == 0 {
		return nil
	}
	stale := a.t("patchTargetChanged")
	return func(ctx context.Context, repo *gitrepo.Repository) error {
		if repo == nil || repo.Root != target.Root {
			return errors.New(stale)
		}
		// ApplyLines rereads and compares the exact patch before any Index write.
		return repo.ApplyLines(ctx, target.File.Path, target.Diff, choice, target.File.Staged)
	}
}

func (a *app) applyOverviewLines(target overviewPatchTarget, indexes []int) {
	action := a.overviewLineAction(target, indexes)
	if action == nil {
		return
	}
	label, success := a.t("stageLinesFailed"), a.t("stageLinesSuccess")
	if target.File.Staged {
		label, success = a.t("unstageLinesFailed"), a.t("unstageLinesSuccess")
	}
	a.clearLineSelection()
	a.perform(label, success, action)
}

func (a *app) selectOverviewLine(target overviewPatchTarget, index int, mods ui.Modifiers) {
	if !a.overviewTargetVisible(target) {
		return
	}
	if a.overviewLineTarget == nil || a.overviewLineTarget.Root != target.Root ||
		a.overviewLineTarget.Generation != target.Generation || a.overviewLineTarget.File != target.File {
		a.clearLineSelection()
	}
	snapshot := target
	a.overviewLineTarget = &snapshot
	a.clickLine(index, mods)
}

func (a *app) overviewSelectionMatches(target overviewPatchTarget) bool {
	selected := a.overviewLineTarget
	return selected != nil && selected.Root == target.Root && selected.Generation == target.Generation && selected.File == target.File
}

func (a *app) overviewPatchLine(c *ui.Context, target overviewPatchTarget, index int, width float32) {
	a.overviewPatchLineForSide(c, target, index, width, patchBoth)
}

func (a *app) overviewPatchLineForSide(c *ui.Context, target overviewPatchTarget, index int, width float32, side patchSide) {
	line := target.Diff.Lines[index]
	if line.Kind == "hunk" {
		a.overviewHunkLine(c, target, line, width, index)
		return
	}
	tk := theme.TokensOf(c)
	bg := tk.BgContainer
	if line.Kind == "add" {
		bg = tk.Success.Alpha(0.075).Over(bg)
	}
	if line.Kind == "del" {
		bg = tk.Error.Alpha(0.065).Over(bg)
	}
	changed := line.Kind == "add" || line.Kind == "del"
	selected := a.overviewSelectionMatches(target)
	if selected && a.lineSel[index] {
		bg = tk.Blue.Alpha(0.16).Over(bg)
	}
	row := ui.Row(c.Key(index)).Width(width).Height(a.patchRowHeight()).Shrink(0).Background(bg)
	row.Children(func() {
		numbers := a.patchGutter(c, line, side, gutterInk(tk, bg))
		markDiffGutter(numbers, changed && selected && a.lineSel[index], gutterInk(tk, bg))
		if changed && numbers.Clicked() {
			a.selectOverviewLine(target, index, numbers.ClickModifiers())
		}
		a.diffFind.lineText(c, diffFindLine{target.File, index}, diffFindText(line), a.patchFontSize(), side == patchNew && line.Kind == "context").TextColor(lineTextColor(tk, line.Kind)).Role(diffCodeRole(line.Kind))
	})
	choice := []int{index}
	if selected && a.lineSel[index] {
		choice = sortedSelection(a.lineSel)
	}
	copyLabel := a.t("copyLine")
	if len(choice) > 1 {
		copyLabel = a.copyLinesLabel(len(choice))
	}
	items := []overlay.MenuItem{{Key: "copy", Label: copyLabel, Icon: lucide.Copy()}}
	if changed {
		key := "stageLine"
		if target.File.Staged {
			key = "unstageLine"
		}
		label := a.t(key)
		if len(choice) > 1 {
			key = "stageLines"
			if target.File.Staged {
				key = "unstageLines"
			}
			label = a.tf(key, len(choice))
		}
		items = append(items, overlay.MenuItem{Key: "apply", Label: label, Disabled: !target.Diff.CanStageHunks || a.busy || a.loading})
	}
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: items}) {
	case "copy":
		a.copyText(diffLinesText(target.Diff, choice))
	case "apply":
		a.applyOverviewLines(target, choice)
	}
	if selected && a.lineCursor == index && a.revealLine {
		row.ScrollIntoView()
		a.revealLine = false
	}
}

func (a *app) overviewHunkLine(c *ui.Context, target overviewPatchTarget, line gitrepo.DiffLine, width float32, index int) {
	tk := theme.TokensOf(c)
	row := ui.Row(c).Width(width).Height(diffHunkH).Shrink(0).Padding(0, 12).AlignItems(ui.Center).Background(tk.Blue.Alpha(0.05).Over(tk.BgContainer))
	row.Children(func() {
		a.diffFind.lineText(c, diffFindLine{target.File, index}, line.Text, scaledFont(10)).TextColor(tk.TextSecondary)
	})
	key := "stageHunk"
	if target.File.Staged {
		key = "unstageHunk"
	}
	switch overlay.ContextMenu(c, row, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "copy", Label: a.t("copyHunk"), Icon: lucide.Copy()},
		{Key: "apply", Label: a.t(key), Disabled: !target.Diff.CanStageHunks || a.busy || a.loading},
	}}) {
	case "copy":
		a.copyText(diffHunkText(target.Diff, line.Hunk))
	case "apply":
		a.applyOverviewLines(target, hunkChangedLines(target.Diff, line.Hunk))
	}
}

// activeWorktreeDiff is also used by keyboard navigation and copy. A summary
// selection must never read the single-file cache behind the active overview.
func (a *app) activeWorktreeDiff() gitrepo.Diff {
	if a.overviewLineTarget != nil {
		return a.overviewLineTarget.Diff
	}
	return a.diff
}
