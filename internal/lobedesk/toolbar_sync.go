package lobedesk

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Tracking text reserves its own space beside the elided name. Its spoken
// description uses localized words so the arrows do not have to be inferred.
func (a *app) branchTrackingStatus() (visible, spoken string) {
	s := a.status
	switch {
	case s.Branch == "":
		return "", ""
	case s.Upstream == "":
		return "", ""
	case s.UpstreamGone:
		visible = a.t("upstreamGone")
	case s.Ahead == 0 && s.Behind == 0:
		return "", ""
	default:
		return fmt.Sprintf("↑ %d  ↓ %d", s.Ahead, s.Behind), a.tf("aheadBehind", s.Ahead, s.Behind)
	}
	return "(" + visible + ")", visible
}

func (a *app) syncMenu(c *ui.Context) {
	tk := theme.TokensOf(c)
	blocked := a.repo == nil || a.busy || a.loading || a.repositoryOpenPending() ||
		a.tabSnapshotUnavailable() || a.hasPendingDialog(toastMain)
	if blocked {
		a.syncOpen = false
	}
	if p := a.syncCommits; p != nil && !a.syncCommitsCurrent(p) {
		a.dismissSyncCommits()
	}
	menu := ui.MenuBase(c, &a.syncOpen)
	label := a.t("syncMenu")
	styleWorkbenchButton(c, menu.Trigger, buttonNormal, blocked, a.syncOpen).Label(label).Shrink(0).
		Children(func() {
			ui.Text(c, label).SingleLine()
			ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(13)).TextColor(tk.TextSecondary)
		})
	menu.Popup(func(panel ui.Element) {
		panel.Margin(6, 0, 0, 0).Padding(4).Radius(tk.BorderRadius).
			Width(306).MaxWidthPercent(95).Background(tk.BgElevated).AttachedOrigin()
		tk.Elevate(panel, theme.BoxShadowSecondary)
		panel.Children(func() {
			if p := a.syncCommits; p != nil {
				a.syncCommitList(c, menu, p)
				return
			}
			a.syncMenuHeader(c, menu)
			remoteReason, pullReason, pushReason := "", "", ""
			if len(a.remotes) == 0 {
				remoteReason, pushReason = a.t("noRemote"), a.t("noRemote")
			} else if a.status.Branch == "" {
				pushReason = a.t("pushNoBranch")
			}
			if a.status.Upstream == "" {
				pullReason = a.t("noUpstream")
			}
			for i, entry := range []struct {
				label, reason string
				danger        bool
				run           func()
			}{
				{a.t("fetch"), remoteReason, false, a.fetchAll},
				{a.t("pull"), pullReason, false, a.pullFF},
				{a.t("pushBranchAction"), pushReason, false, a.confirmPush},
				{a.t("fetchPrune"), remoteReason, false, a.fetchPruneAll},
				{a.t("forcePushAction"), pushReason, true, a.confirmForcePush},
			} {
				if i == 3 {
					ui.Box(c).Height(1).Margin(4, 0).AlignSelf(ui.Stretch).Background(tk.Border)
				}
				disabled := blocked || entry.reason != ""
				item := menu.Item().Label(entry.label).Disabled(disabled).MinHeight(30).
					AlignSelf(ui.Stretch).Padding(5, 10).Radius(tk.BorderRadiusSM)
				color := tk.Text
				if disabled {
					color = tk.TextDisabled
				} else {
					if entry.danger {
						color = tk.Error
					}
					if item.Highlighted() || item.Pressed() {
						item.Background(tk.FillTertiary)
					}
				}
				if entry.reason != "" {
					item.Description(entry.reason)
				}
				item.TextColor(color).Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
						ui.Text(c, entry.label).FontSize(scaledFont(13))
						if entry.reason != "" {
							ui.Text(c, entry.reason).FontSize(scaledFont(11)).TextColor(tk.TextSecondary)
						}
					})
				})
				if item.Clicked() && !disabled {
					entry.run()
				}
			}
		})
	})
}

func (a *app) syncMenuHeader(c *ui.Context, menu ui.MenuParts) {
	tk := theme.TokensOf(c)
	header := a.status.Upstream
	switch {
	case a.status.Branch == "":
		header = a.t("detached")
	case header == "":
		return
	case a.status.UpstreamGone:
		header += " · " + a.t("upstreamGone")
	}
	ui.Text(c, header).Padding(8, 10).FontSize(scaledFont(12)).FontWeight(600).SingleLine().Tooltip(header)
	if a.status.Branch != "" && a.status.Upstream != "" && !a.status.UpstreamGone {
		menu.Group(func(row ui.Element) {
			row.Gap(4).Padding(0, 6).Label(a.t("syncMenu"))
			for _, incoming := range []bool{false, true} {
				count, arrow := a.status.Ahead, "↑"
				name, detail := a.tf("branchAhead", count), a.t("syncOutgoing")
				if incoming {
					count, arrow = a.status.Behind, "↓"
					name, detail = a.tf("branchBehind", count), a.t("syncIncoming")
				}
				item := syncListItem(c, menu, name, count == 0).Description(detail).Tooltip(name + " · " + detail).KeepOpen()
				item.Children(func() {
					ui.Textf(c, "%s %d", arrow, count).SingleLine()
				})
				if item.Clicked() && count > 0 {
					a.readSyncCommits(incoming, "", 0)
				}
			}
		})
	}
	ui.Box(c).Height(1).Margin(2, 0).AlignSelf(ui.Stretch).Background(tk.Border)
}
