package lobedesk

import (
	"runtime"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func (a *app) requestCommitFilePatch(target commitFilePatchTarget) {
	a.requestCommitFilePatchForPlatform(target, runtime.GOOS)
}

// Windows and Linux native save dialogs do not render the SDK Message field.
// Keep the same immutable target through an app-owned scope review there.
func (a *app) requestCommitFilePatchForPlatform(target commitFilePatchTarget, platform string) {
	if !a.commitFilePatchReady(target) {
		return
	}
	if platform == "darwin" {
		a.exportCommitFilePatch(target)
		return
	}
	base := a.t("patchEmptyTree")
	if target.Parent != "" {
		base = a.tf("patchFirstParent", shortSHA(target.Parent))
	}
	path := target.File.Path
	if target.File.OriginalPath != "" {
		path = target.File.OriginalPath + " → " + path
	}
	a.askConfirm(a.t("exportFilePatchTitle"), a.tf("exportFilePatchMessage", path, shortSHA(target.SHA), base), a.t("choosePatchDestination"), false, func() { a.exportCommitFilePatch(target) })
	a.confirm.patchScope = true
}

// Shares confirmation ownership/cancellation while keeping long scope text in
// a scroll area. Other confirmation dialogs retain their existing rendering.
func (a *app) commitFilePatchScopeDialog(c *ui.Context, blocked bool) bool {
	confirmed := false
	tk := theme.TokensOf(c)
	workbenchModal(c, &a.confirm.open, func() {
		ui.Column(c).Width(600).MaxWidthPercent(90).Height(420).MaxHeightPercent(90).Padding(20).Gap(12).
			Background(tk.BgElevated).Radius(12).Border(1, tk.Border).Children(func() {
			display.Text(c, a.confirm.title, display.TextProps{Strong: true, FontSize: scaledFont(14), Rows: 2}).FillWidth().Shrink(0)
			layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.confirm.patchScroll}, func() {
				display.Text(c, a.confirm.message, display.TextProps{FontSize: scaledFont(12)}).FillWidth().MinWidth(0)
			}).Grow(1).Basis(0).MinHeight(0)
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Shrink(0).Children(func() {
				if workbenchButton(c, a.t("cancel"), buttonNormal, false).Clicked() {
					a.confirm.open = false
				}
				if workbenchButton(c, a.confirm.actionName, buttonNormal, blocked).Clicked() && !blocked {
					confirmed = true
					a.confirm.open = false
				}
			})
		})
	})
	return confirmed
}
