package lobedesk

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type noticeDetailsState struct {
	deferred bool
	open     bool
	notice   *toastNotice
	scroll   ui.ScrollState
}

func longNoticeTitle(n *toastNotice) bool {
	return utf8.RuneCountInString(n.title) > 100 || strings.Contains(n.title, "\n")
}

func needsNoticeDetails(n *toastNotice) bool {
	return n.kind == "error" && n.onAction == nil && (longNoticeTitle(n) || utf8.RuneCountInString(n.description) > 160 || strings.Count(n.description, "\n") > 2)
}

func noticeDetailText(n *toastNotice) string {
	if longNoticeTitle(n) {
		if n.description == "" {
			return n.title
		}
		return n.title + "\n\n" + n.description
	}
	return n.description
}

func (a *app) openNoticeDetails(owner toastOwner, notice *toastNotice) {
	if notice == nil || (notice.root != "" && (a.repo == nil || a.repo.Root != notice.root)) {
		return
	}
	a.discardStaleNoticeDetails(owner)
	if state := a.noticeDetails[owner]; state.open || state.deferred {
		// A toast action closes its toast. Keep a later notice available without
		// replacing the detail surface the reader is already using or awaiting.
		if state.notice != notice {
			a.pushNotice(owner, notice)
		}
		return
	}
	if a.rejectPendingDialog(owner) {
		a.noticeDetails[owner] = noticeDetailsState{deferred: true, notice: notice}
		return
	}
	a.noticeDetails[owner] = noticeDetailsState{open: true, notice: notice}
	a.invalidate()
}

func (a *app) discardStaleNoticeDetails(owner toastOwner) {
	// A newly opened Settings window must not inherit its predecessor's modal.
	if w := a.settingsWin.Load(); w == nil || w.IsDestroyed() {
		a.noticeDetails[toastSettings] = noticeDetailsState{}
	}
	state := &a.noticeDetails[owner]
	if state.notice == nil {
		return
	}
	n := state.notice
	if n.root != "" && (a.repo == nil || a.repo.Root != n.root) {
		*state = noticeDetailsState{}
	}
}

// Each window owns its detail surface. The original notice remains intact so
// neither the compact toast nor copying drops any of the output.
func (a *app) noticeDetailsDialog(c *ui.Context, owner toastOwner) {
	a.discardStaleNoticeDetails(owner)
	state := &a.noticeDetails[owner]
	if state.deferred && !a.hasPendingDialog(owner) {
		state.deferred = false
		state.open = true
	}
	if !state.open || state.notice == nil {
		return
	}
	n := state.notice
	title := n.title
	if longNoticeTitle(n) {
		title = a.t("errorDetailsTitle")
	}
	text := noticeDetailText(n)
	tk := theme.TokensOf(c)
	ui.Modal(c, &state.open, func() {
		ui.Column(c).Width(600).MaxWidthPercent(90).Height(420).MaxHeightPercent(90).Padding(16).Gap(12).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).Children(func() {
			display.Text(c, title, display.TextProps{Strong: true, FontSize: scaledFont(14), Rows: 2}).FillWidth().Shrink(0)
			layout.ScrollArea(c, layout.ScrollAreaProps{State: &state.scroll}, func() {
				display.Text(c, text, display.TextProps{FontSize: scaledFont(12)}).FillWidth().MinWidth(0)
			}).Grow(1).Basis(0).MinHeight(0)
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Shrink(0).Children(func() {
				if action.Button(c, a.t("copyErrorDetails"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).Clicked() {
					a.copyText(text)
				}
				if action.Button(c, a.t("close"), action.ButtonProps{Variant: action.Default, Size: theme.Small}).Clicked() {
					state.open = false
				}
			})
		})
	})
	if !state.open {
		*state = noticeDetailsState{}
	}
}
