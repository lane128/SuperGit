package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func (a *app) branchTrackingStatus() string {
	s := a.status
	switch {
	case s.Branch == "":
		return ""
	case s.Upstream == "":
		return a.t("branchNoUpstream")
	case s.Ahead > 0 && s.Behind > 0:
		return a.tf("aheadBehind", s.Ahead, s.Behind)
	case s.Ahead > 0:
		return a.tf("branchAhead", s.Ahead)
	case s.Behind > 0:
		return a.tf("branchBehind", s.Behind)
	default:
		return ""
	}
}

func (a *app) syncMenu(c *ui.Context) {
	tk := theme.TokensOf(c)
	blocked := a.repo == nil || a.busy || a.loading || a.repositoryOpenPending() ||
		a.tabSnapshotUnavailable() || a.hasPendingDialog(toastMain)
	if blocked {
		a.syncOpen = false
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
			a.syncMenuHeader(c)
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

func (a *app) syncMenuHeader(c *ui.Context) {
	tk := theme.TokensOf(c)
	tracking := ""
	header := a.status.Upstream
	switch {
	case header != "":
		tracking = a.branchTrackingStatus()
		if tracking == "" {
			tracking = a.t("syncUpToDate")
		}
	case a.status.Branch == "":
		header = a.t("detached")
	default:
		header = a.t("branchNoUpstream")
	}
	ui.Column(c).AlignSelf(ui.Stretch).MinWidth(0).Gap(3).Padding(8, 10).Children(func() {
		ui.Text(c, header).FontSize(scaledFont(12)).FontWeight(600).SingleLine().Tooltip(header)
		if tracking != "" {
			ui.Text(c, tracking).FontSize(scaledFont(11)).TextColor(tk.TextSecondary)
		}
	})
	ui.Box(c).Height(1).Margin(2, 0).AlignSelf(ui.Stretch).Background(tk.Border)
}
