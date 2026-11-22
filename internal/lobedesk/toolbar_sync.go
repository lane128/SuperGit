package lobedesk

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type branchTrackingItem struct {
	text string
	icon *ui.SVG
	ink  ui.Color
}

func (a *app) branchTrackingStatus(tk *theme.Tokens) ([]branchTrackingItem, string) {
	s := a.status
	switch {
	case s.Branch == "" || s.Upstream == "":
		return nil, ""
	case s.UpstreamGone:
		label := a.t("upstreamGone")
		return []branchTrackingItem{{text: "(" + label + ")", ink: tk.TextSecondary}}, label
	case s.Ahead == 0 && s.Behind == 0:
		return nil, ""
	}
	var items []branchTrackingItem
	for _, side := range []struct {
		n        int
		incoming bool
	}{{s.Ahead, false}, {s.Behind, true}} {
		if side.n > 0 {
			icon, ink := trackingInk(tk, side.incoming)
			items = append(items, branchTrackingItem{strconv.Itoa(side.n), icon, ink})
		}
	}
	return items, a.tf("aheadBehind", s.Ahead, s.Behind)
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
		// Two ordinary menu rows share the command rows' height and rhythm;
		// a zero side stays in place, greyed, so the layout never jumps.
		for _, incoming := range []bool{false, true} {
			count, title := a.status.Ahead, a.t("syncToPush")
			name, detail := a.tf("branchAhead", count), a.t("syncOutgoing")
			if incoming {
				count, title = a.status.Behind, a.t("syncToPull")
				name, detail = a.tf("branchBehind", count), a.t("syncIncoming")
			}
			item := syncListItem(c, menu, name, count == 0).AlignSelf(ui.Stretch).MinHeight(30).Gap(8).
				Description(detail).Tooltip(name + " · " + detail).KeepOpen()
			icon, ink := trackingInk(tk, incoming)
			if count == 0 {
				ink = tk.TextDisabled
			} else {
				ink = capsuleInk(ink, tk.BgElevated, 4.5)
			}
			item.Children(func() {
				ui.Icon(c, icon).Size(scaledFont(13), scaledFont(13)).TextColor(ink).Shrink(0)
				ui.Text(c, title).FontSize(scaledFont(13)).SingleLine().Grow(1).Basis(0).MinWidth(0)
				pill := ui.Box(c).Shrink(0).MinWidth(scaledFont(20)).Height(scaledFont(18)).Padding(0, 6).
					Radius(scaledFont(9)).Center().Background(tk.FillSecondary)
				pill.Children(func() {
					ui.Text(c, strconv.Itoa(count)).Font(a.patchFontFamily()).FontSize(scaledFont(11)).SingleLine()
				})
			})
			if item.Clicked() && count > 0 {
				a.readSyncCommits(incoming, "", 0)
			}
		}
	}
	ui.Box(c).Height(1).Margin(2, 0).AlignSelf(ui.Stretch).Background(tk.Border)
}
