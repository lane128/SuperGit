package lobedesk

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"supergit/internal/gitrepo"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

type cloneFormState struct {
	open                         bool
	address, path, err           string
	paletteHandoff, paletteFocus bool
}

func (a *app) canShowCloneRepository() bool {
	return !a.busy && !a.loading && !a.repositoryChooserOpen && !a.hasPendingDialog(toastMain)
}

func (a *app) showCloneRepository() {
	if !a.canShowCloneRepository() {
		return
	}
	a.cloneForm = cloneFormState{open: true}
	a.invalidate()
}

func cloneSourceName(source string) string {
	source = strings.TrimSpace(source)
	if strings.Contains(source, "://") {
		u, err := url.Parse(source)
		if err != nil {
			return ""
		}
		// Decode only the last component, so an escaped separator cannot
		// turn part of a remote name into a local directory path.
		escaped := strings.TrimRight(u.EscapedPath(), "/")
		source, err = url.PathUnescape(escaped[strings.LastIndexByte(escaped, '/')+1:])
		if err != nil {
			return ""
		}
	} else {
		source = filepath.ToSlash(source[len(filepath.VolumeName(source)):])
		// Git also accepts SSH addresses in host:path form. A colon after
		// the first slash belongs to the path rather than the SSH host.
		host := strings.SplitN(source, "/", 2)[0]
		if colon := strings.LastIndexByte(host, ':'); colon >= 0 {
			source = source[colon+1:]
		}
		source = strings.TrimRight(source, "/")
		source = source[strings.LastIndexByte(source, '/')+1:]
	}
	return strings.TrimSpace(strings.TrimSuffix(source, ".git"))
}

func (a *app) chooseCloneParent() {
	if !a.cloneForm.open || a.busy || a.loading || a.repositoryChooserOpen || (a.window == nil && a.openDirectoryDialog == nil) {
		return
	}
	a.repositoryChooserOpen = true
	defer func() {
		a.repositoryChooserOpen = false
		a.invalidate()
	}()
	open := a.openDirectoryDialog
	if open == nil {
		open = mygo.Dialog.Open
	}
	paths, err := open(mygo.OpenDialogOptions{
		Parent:    a.window,
		Title:     a.t("chooseParentDir"),
		Directory: true,
	})
	if !a.cloneForm.open || a.busy || a.loading {
		return
	}
	if err != nil {
		a.cloneForm.err = a.t("chooseParentDirFailed") + ": " + err.Error()
		return
	}
	if len(paths) == 0 || !filepath.IsAbs(paths[0]) {
		return
	}
	validName := func(name string) bool {
		return name != "." && filepath.IsLocal(name) && !strings.ContainsAny(name, "/\\:\x00")
	}
	name := filepath.Base(strings.TrimSpace(a.cloneForm.path))
	if !validName(name) {
		name = cloneSourceName(a.cloneForm.address)
		if !validName(name) {
			name = "repository"
		}
	}
	a.cloneForm.path = filepath.Join(paths[0], name)
	a.cloneForm.err = ""
}

func (a *app) submitCloneRepository() {
	if !a.cloneForm.open || a.busy || a.loading || a.repositoryChooserOpen {
		return
	}
	address, path := strings.TrimSpace(a.cloneForm.address), strings.TrimSpace(a.cloneForm.path)
	switch {
	case address == "" || path == "":
		a.cloneForm.err = a.t("cloneRequired")
	case !filepath.IsAbs(path):
		a.cloneForm.err = a.t("cloneAbsolutePath")
	default:
		// Match the backend's Lstat contract, including existing empty directories
		// and dangling symlinks. Git rechecks the destination when it starts.
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			a.cloneForm.err = a.t("cloneDestinationExists")
			a.invalidate()
			return
		}
		a.cloneForm.err = ""
		repo, seq := a.repo, a.loadSeq
		ctx, cancel := a.beginOperation("busyClone")
		go func() {
			cloned, err := gitrepo.Clone(ctx, address, path)
			a.hopToMain(func() {
				defer cancel()
				a.finishOperation()
				// Cancel still wins if Git completed just before the cancel request.
				if ctx.Err() != nil && err == nil {
					err = ctx.Err()
				}
				if repo != a.repo || seq != a.loadSeq || !a.cloneForm.open {
					a.invalidate()
					return
				}
				if err != nil {
					if operationCancelled(ctx, err) {
						a.cloneForm.err = a.t("operationCancelled")
					} else {
						a.cloneForm.err = a.operationErrorDescription(err)
					}
					if _, statErr := os.Lstat(path); statErr == nil {
						a.cloneForm.err += "\n\n" + a.tf("cloneDestinationRemains", path)
					}
					a.invalidate()
					return
				}
				a.cloneForm = cloneFormState{}
				a.cloneOpeningRoot = cloned.Root
				a.openRepository(cloned.Root, true)
			})
		}()
	}
	a.invalidate()
}

func (a *app) cloneRepositoryDialog(c *ui.Context) {
	if !a.cloneForm.open || a.cloneForm.paletteHandoff {
		return
	}
	a.renderCloneRepositoryDialog(c, false)
}

func (a *app) paletteCloneDialog(c *ui.Context) {
	if a.cloneForm.open && a.cloneForm.paletteHandoff {
		a.renderCloneRepositoryDialog(c, true)
	}
}

func (a *app) renderCloneRepositoryDialog(c *ui.Context, palette bool) {
	tk := theme.TokensOf(c)
	// A running clone retains its modal and its cancellation control until Git
	// exits. Escape/outside clicks cannot leave a hidden job that later switches.
	open := a.cloneForm.open
	var source ui.Element
	content := func(panel ui.Element) {
		panel.Padding(18).Gap(10).
			Background(tk.BgElevated).Radius(tk.BorderRadiusLG).Border(1, tk.Border).Children(func() {
			display.Text(c, a.t("cloneRepoTitle"), display.TextProps{Strong: true, FontSize: scaledFont(14)})
			display.Text(c, a.t("cloneSource"), display.TextProps{FontSize: scaledFont(12)})
			address := field.Input(c, &a.cloneForm.address, field.InputProps{Variant: field.InputFilled, Size: theme.Middle}).
				Label(a.t("cloneSource")).FillWidth().Disabled(a.busy)
			source = address
			if !palette {
				address.AutoFocus()
			}
			display.Text(c, a.t("cloneDestination"), display.TextProps{FontSize: scaledFont(12)})
			path := field.Input(c, &a.cloneForm.path, field.InputProps{Variant: field.InputFilled, Size: theme.Middle}).
				Label(a.t("cloneDestination")).FillWidth().Disabled(a.busy)
			display.Text(c, a.t("cloneHint"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)})
			if a.cloneForm.err != "" {
				errorHeight := float32(100)
				if palette {
					_, height := c.Size()
					errorHeight = min(100, max(40, height-320))
				}
				layout.ScrollArea(c, layout.ScrollAreaProps{}, func() {
					ui.Text(c, a.cloneForm.err).FillWidth().TextColor(tk.Red).FontSize(scaledFont(12))
				}).FillWidth().Height(errorHeight)
			}
			ready := !a.busy && !a.repositoryChooserOpen && strings.TrimSpace(a.cloneForm.address) != "" && strings.TrimSpace(a.cloneForm.path) != ""
			if ready && (address.Submitted() || path.Submitted()) {
				a.submitCloneRepository()
			}
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				if workbenchButton(c, a.t("chooseParentDir"), buttonNormal, a.busy || a.loading || a.repositoryChooserOpen).Clicked() {
					a.chooseCloneParent()
				}
				ui.Box(c).Grow(1)
				if workbenchButton(c, a.t("cancel"), buttonNormal, a.repositoryChooserOpen).Clicked() {
					if a.busy {
						a.cancelOperation()
					} else {
						open = false
					}
				}
				label := a.t("cloneAction")
				if a.busy {
					label = a.t("busyClone")
				}
				if workbenchButton(c, label, buttonPrimary, !ready).Clicked() && ready {
					a.submitCloneRepository()
				}
			})
		})
	}
	if palette {
		width, height := c.Size()
		c.Key(commandPickerOverlayKey)
		ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
			backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
			content(panel.Column().Label(a.t("cloneRepoTitle")).Width(min(480, max(0, width-32))).MaxHeight(max(0, height-32)))
		})
		// Capture the original opener before focusing the replacement form.
		if a.cloneForm.paletteFocus && !a.busy {
			source.Focus()
			a.cloneForm.paletteFocus = false
		}
	} else {
		ui.Modal(c, &open, func() { content(ui.Column(c).Width(480).MaxWidthPercent(95)) })
	}
	if !open && !a.busy && !a.repositoryChooserOpen {
		a.cloneForm = cloneFormState{}
	}
}
