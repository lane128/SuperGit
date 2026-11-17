package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func selectedLineSpan(low, high int) string {
	if low == 0 {
		return "—"
	}
	if low == high {
		return itoa(low)
	}
	return itoa(low) + "–" + itoa(high)
}

func (a *app) selectedLineRangeLabel(diff gitrepo.Diff, indexes []int) string {
	oldLow, oldHigh, newLow, newHigh := 0, 0, 0, 0
	for _, index := range indexes {
		line := diff.Lines[index]
		if line.Old > 0 {
			if oldLow == 0 || line.Old < oldLow {
				oldLow = line.Old
			}
			oldHigh = max(oldHigh, line.Old)
		}
		if line.New > 0 {
			if newLow == 0 || line.New < newLow {
				newLow = line.New
			}
			newHigh = max(newHigh, line.New)
		}
	}
	return a.tf("selectedLinesRange", len(indexes), selectedLineSpan(oldLow, oldHigh), selectedLineSpan(newLow, newHigh))
}

// Only the visible patch owns the bar. Overview selections carry their own
// root/generation/path/version, independently of the single-file cache.
func (a *app) worktreeSelectionBar(c *ui.Context) {
	if len(a.lineSel) == 0 || !a.worktreeDiffShown() {
		return
	}
	diff := a.activeWorktreeDiff()
	var indexes []int
	for _, index := range sortedSelection(a.lineSel) {
		if index >= 0 && index < len(diff.Lines) && (diff.Lines[index].Kind == "add" || diff.Lines[index].Kind == "del") {
			indexes = append(indexes, index)
		}
	}
	if len(indexes) == 0 {
		return
	}
	path, staged := a.selectedPath, a.selectedStaged
	allowed := a.detailNavigationAllowed() && a.lineActionsAllowed() && !a.diffLoading && a.diffErr == ""
	var overview *overviewPatchTarget
	if a.overviewLineTarget != nil {
		target := *a.overviewLineTarget
		overview = &target
		path, staged = target.File.Path, target.File.Staged
		allowed = a.overviewLineAction(target, indexes) != nil
	}
	label := a.selectedLineRangeLabel(diff, indexes)
	action, description := "stageSelection", "stageLines"
	if staged {
		action, description = "unstageSelection", "unstageLines"
	}
	tk := theme.TokensOf(c)
	ui.Column(c).FillWidth().MinWidth(0).Shrink(0).Padding(6, 10).Gap(5).
		Background(tk.FillTertiary).BorderWidth(0, 0, 1, 0).BorderColor(tk.BorderSecondary).
		Role(ui.RoleGroup).Label(label).Description(path).Children(func() {
		if overview != nil {
			display.Text(c, path, display.TextProps{FontSize: scaledFont(11)}).FillWidth().SingleLine().Tooltip(path)
		}
		display.Text(c, label, display.TextProps{FontSize: scaledFont(11), Type: display.TextSecondary}).FillWidth().SingleLine().Tooltip(label)
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
			apply := workbenchButton(c, a.t(action), buttonPrimary, !allowed).Description(a.tf(description, len(indexes))).Tooltip(a.tf(description, len(indexes)))
			if apply.Clicked() {
				if overview != nil {
					a.applyOverviewLines(*overview, indexes)
				} else {
					a.applyLines(indexes)
				}
			}
			if workbenchIconButton(c, lucide.Copy(), a.copyLinesLabel(len(indexes)), buttonNormal, false).Clicked() {
				a.copyText(diffLinesText(diff, indexes))
			}
			if workbenchIconButton(c, lucide.X(), a.t("shortcutClearSelection"), buttonNormal, false).Clicked() {
				a.clearLineSelection()
				a.invalidate()
			}
		})
	})
}
