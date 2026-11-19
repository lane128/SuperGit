package lobedesk

import (
	"os"
	"path/filepath"
	"strings"

	"supergit/internal/gitrepo"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type newRepositoryFormState struct {
	open      bool
	path, err string
}

func (a *app) showNewRepository() {
	if !a.canShowCloneRepository() {
		return
	}
	a.newRepoForm = newRepositoryFormState{open: true}
	a.invalidate()
}

func (a *app) submitNewRepository() {
	if !a.newRepoForm.open || a.busy || a.loading || a.repositoryChooserOpen {
		return
	}
	path := strings.TrimSpace(a.newRepoForm.path)
	switch {
	case path == "":
		a.newRepoForm.err = a.t("newRepoRequired")
	case !filepath.IsAbs(path):
		a.newRepoForm.err = a.t("cloneAbsolutePath")
	default:
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			a.newRepoForm.err = a.t("cloneDestinationExists")
			a.invalidate()
			return
		}
		a.newRepoForm.err = ""
		repo, seq := a.repo, a.loadSeq
		ctx, cancel := a.beginOperation("busyCreateRepo")
		go func() {
			created, err := gitrepo.Init(ctx, path)
			a.hopToMain(func() {
				defer cancel()
				a.finishOperation()
				if ctx.Err() != nil && err == nil {
					err = ctx.Err()
				}
				if repo != a.repo || seq != a.loadSeq || !a.newRepoForm.open {
					a.invalidate()
					return
				}
				if err != nil {
					if operationCancelled(ctx, err) {
						a.newRepoForm.err = a.t("operationCancelled")
					} else {
						a.newRepoForm.err = a.operationErrorDescription(err)
					}
					if _, statErr := os.Lstat(path); statErr == nil {
						a.newRepoForm.err += "\n\n" + a.tf("cloneDestinationRemains", path)
					}
					a.invalidate()
					return
				}
				a.newRepoForm = newRepositoryFormState{}
				a.newRepoOpeningRoot = created.Root
				a.openRepository(created.Root, true)
			})
		}()
	}
	a.invalidate()
}

func (a *app) newRepositoryDialog(c *ui.Context) {
	if !a.newRepoForm.open {
		return
	}
	tk := theme.TokensOf(c)
	open := a.newRepoForm.open
	ui.Modal(c, &open, func() {
		ui.Column(c).Width(480).MaxWidthPercent(95).Padding(18).Gap(10).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).Children(func() {
			display.Text(c, a.t("newRepoTitle"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
			display.Text(c, a.t("cloneDestination"), display.TextProps{FontSize: scaledFont(12)})
			path := field.Input(c, &a.newRepoForm.path, field.InputProps{Variant: field.InputFilled, Size: theme.Middle}).Label(a.t("cloneDestination")).FillWidth().Disabled(a.busy).AutoFocus()
			display.Text(c, a.t("newRepoHint"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
			if a.newRepoForm.err != "" {
				layout.ScrollArea(c, layout.ScrollAreaProps{}, func() { ui.Text(c, a.newRepoForm.err).FillWidth().TextColor(tk.Red).FontSize(scaledFont(12)) }).FillWidth().Height(100)
			}
			ready := !a.busy && strings.TrimSpace(a.newRepoForm.path) != ""
			if ready && path.Submitted() {
				a.submitNewRepository()
			}
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.End).Children(func() {
				if workbenchButton(c, a.t("cancel"), buttonNormal, false).Clicked() {
					if a.busy {
						a.cancelOperation()
					} else {
						open = false
					}
				}
				label := a.t("create")
				if a.busy {
					label = a.t("busyCreateRepo")
				}
				if workbenchButton(c, label, buttonPrimary, !ready).Clicked() && ready {
					a.submitNewRepository()
				}
			})
		})
	})
	if !open && !a.busy {
		a.newRepoForm = newRepositoryFormState{}
	}
}
