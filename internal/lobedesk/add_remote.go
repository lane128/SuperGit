package lobedesk

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type addRemoteFormState struct {
	root          string
	name, address string
	open          bool
	err           string
}

func (a *app) discardStaleAddRemoteForm() {
	if a.addRemoteForm.open && (a.repo == nil || a.addRemoteForm.root != a.repo.Root) {
		a.addRemoteForm = addRemoteFormState{}
	}
}

func (a *app) canAddRemote() bool {
	return a.repo != nil && !a.busy && !a.loading && !a.tabSnapshotUnavailable() &&
		(a.repositoryTabs == nil || (a.repositoryTabs.active == a && a.repositoryTabs.canChange())) && !a.hasPendingDialog(toastMain)
}

func (a *app) openAddRemote() {
	if !a.canAddRemote() {
		return
	}
	a.addRemoteForm = addRemoteFormState{root: a.repo.Root, open: true}
	a.invalidate()
}

func (a *app) addRemoteDialog(c *ui.Context) {
	a.discardStaleAddRemoteForm()
	form := &a.addRemoteForm
	if !form.open {
		return
	}
	tk := theme.TokensOf(c)
	open, submit := true, false
	ui.Modal(c, &open, func() {
		ui.Column(c).Width(400).Padding(18).Gap(12).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).
			Children(func() {
				display.Text(c, a.t("addRemote"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
				input := func(label string, value *string) (el ui.Element) {
					ui.Column(c).FillWidth().Gap(4).Children(func() {
						display.Text(c, label, display.TextProps{FontSize: scaledFont(12)})
						el = field.Input(c, value, field.InputProps{Variant: field.InputFilled, Size: theme.Middle}).Label(label).Disabled(a.busy)
					})
					return el
				}
				name := input(a.t("remoteName"), &form.name).AutoFocus()
				address := input(a.t("remoteAddress"), &form.address)
				if form.err != "" {
					display.Text(c, a.t("addRemoteFailed"), display.TextProps{Type: display.TextDanger, Strong: true, FontSize: scaledFont(12)})
					layout.ScrollArea(c, layout.ScrollAreaProps{}, func() {
						ui.Text(c, form.err).FillWidth().FontSize(scaledFont(12)).TextColor(tk.TextSecondary).Selectable()
					}).MaxHeight(96)
				}
				ready := !a.busy && !a.loading && strings.TrimSpace(form.name) != "" && strings.TrimSpace(form.address) != ""
				submit = ready && (name.Submitted() || address.Submitted())
				ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
					if workbenchButton(c, a.t("cancel"), buttonNormal, a.busy).Clicked() {
						open = false
					}
					if workbenchButton(c, a.t("addRemote"), buttonPrimary, !ready).Clicked() && ready {
						submit = true
					}
				})
			})
	})
	if !open && !a.busy {
		a.addRemoteForm = addRemoteFormState{}
	} else if submit {
		a.submitAddRemote()
	}
}

func (a *app) submitAddRemote() {
	form := &a.addRemoteForm
	if !form.open || a.repo == nil || a.repo.Root != form.root || a.activeRepositorySession() != a {
		return
	}
	name, address := strings.TrimSpace(form.name), strings.TrimSpace(form.address)
	if name == "" || address == "" {
		return
	}
	root := form.root
	form.err = ""
	// Own the failure inside the form so users can read it while correcting the inputs.
	a.performDone("", a.t("remoteAdded"), func(err error) {
		if a.repo == nil || a.repo.Root != root || !form.open || form.root != root {
			return
		}
		if err == nil {
			a.addRemoteForm = addRemoteFormState{}
		} else {
			var parseError *url.Error
			if errors.As(err, &parseError) {
				// A parser error includes the entire address, potentially with credentials.
				form.err = a.t("invalidRemoteAddress")
			} else {
				form.err = a.operationErrorDescription(err)
			}
		}
	}, func(ctx context.Context, repo *gitrepo.Repository) error {
		return repo.AddRemote(ctx, name, address)
	})
}
