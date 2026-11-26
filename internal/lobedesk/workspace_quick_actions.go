package lobedesk

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type workspaceCleanState struct {
	open, loading, deleteNew bool
	repo                     *gitrepo.Repository
	root                     string
	preview                  gitrepo.WorkspaceCleanPreview
	err                      string
	cancel                   context.CancelFunc
	scroll                   ui.ScrollState
}

func (a *app) workspaceCleanOpen() bool {
	s := a.workspaceClean
	if s == nil {
		return false
	}
	if !s.open || a.repo != s.repo || a.repo == nil || a.repo.Root != s.root || a.localWatch.closed {
		a.closeWorkspaceClean()
		return false
	}
	return true
}

func (a *app) closeWorkspaceClean() {
	if s := a.workspaceClean; s != nil {
		s.open = false
		if s.cancel != nil {
			s.cancel()
		}
	}
	a.workspaceClean = nil
}

func (a *app) workspaceQuickActionsBlocked() bool {
	return a.repo == nil || a.busy || a.loading || a.repositoryOpenPending() || a.tabSnapshotUnavailable() || a.hasPendingDialog(toastMain)
}

func (a *app) workspaceStashBlocked() bool {
	return a.workspaceQuickActionsBlocked() || len(a.status.Entries) == 0
}

func (a *app) titleQuickActions(c *ui.Context) {
	blocked := a.workspaceQuickActionsBlocked()
	stashBlocked := a.workspaceStashBlocked()
	ui.Row(c).Gap(4).Shrink(0).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Gap(0).Shrink(0).AlignItems(ui.Center).Children(func() {
			button := compactActionIcon(c, lucide.Archive(), a.t("workspaceStash"), buttonNormal, stashBlocked).
				Radius(6, 0, 0, 6).Tooltip(a.t("workspaceStashHelp"))
			if button.Clicked() && !stashBlocked {
				a.quickWorkspaceStash()
			}
			a.stashSaveMenu(c)
		})
		if button := compactActionIcon(c, lucide.Eraser(), a.t("workspaceCleanAction"), buttonDangerIcon, blocked).
			Tooltip(a.t("workspaceCleanHelp")); button.Clicked() && !blocked {
			a.openWorkspaceClean()
		}
	})
}

func (a *app) quickWorkspaceStash() {
	if a.workspaceStashBlocked() {
		return
	}
	a.stashSaveOpen = false
	a.performDoneAs(busySaveStash, "", a.t("saveStashSuccess"), func(err error) {
		if err != nil {
			a.queue("error", a.t("saveStashFailed"), a.operationErrorDescription(err))
		}
	}, func(ctx context.Context, r *gitrepo.Repository) error {
		return r.SaveWorkspaceStash(ctx)
	})
}

func (a *app) openWorkspaceClean() {
	if a.workspaceQuickActionsBlocked() {
		return
	}
	a.stashSaveOpen = false
	a.prepareWorkspaceClean(false)
}

func (a *app) prepareWorkspaceClean(deleteNew bool) {
	a.closeWorkspaceClean()
	if a.repo == nil || a.busy || a.loading || a.localWatch.closed {
		return
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	state := &workspaceCleanState{open: true, loading: true, deleteNew: deleteNew, repo: a.repo, root: a.repo.Root, cancel: cancel}
	a.workspaceClean = state
	a.invalidate()
	go func() {
		defer cancel()
		preview, err := state.repo.PrepareWorkspaceClean(ctx, deleteNew)
		a.hopToMain(func() { a.finishWorkspaceCleanPreparation(state, preview, err) })
	}()
}

// Pointer identity distinguishes option changes and reopenings even in one repo.
func (a *app) finishWorkspaceCleanPreparation(state *workspaceCleanState, preview gitrepo.WorkspaceCleanPreview, err error) {
	if a.workspaceClean != state || !a.workspaceCleanOpen() {
		return
	}
	if a.activeRepositorySession() != a {
		a.closeWorkspaceClean()
		return
	}
	state.loading = false
	state.preview = preview
	if err != nil {
		state.err = a.operationErrorDescription(err)
	}
	a.invalidate()
}

func (a *app) confirmWorkspaceClean() {
	if !a.workspaceCleanOpen() {
		return
	}
	s := a.workspaceClean
	if s.loading || s.err != "" || !s.preview.CanExecute || a.busy || a.loading || a.activeRepositorySession() != a {
		return
	}
	preview := s.preview
	started := a.performDoneAs(busyWorkspaceClean, "", a.t("workspaceCleanSuccess"), func(err error) {
		// Cancellation can arrive after mutation. Keep the safety guidance even
		// when the shared operation host also emits its generic cancelled notice.
		if err != nil {
			a.queue("error", a.t("workspaceCleanFailed"), a.operationErrorDescription(err))
		}
	}, func(ctx context.Context, r *gitrepo.Repository) error {
		return r.CleanWorkspaceChecked(ctx, preview)
	})
	if started {
		a.closeWorkspaceClean()
	}
}

func (a *app) workspaceCleanDialog(c *ui.Context) {
	if !a.workspaceCleanOpen() {
		return
	}
	s := a.workspaceClean
	tk := theme.TokensOf(c)
	width, height := c.Size()
	workbenchModal(c.Key("workspace-clean"), &s.open, func() {
		ui.Column(c).Label(a.t("workspaceCleanTitle")).Width(min(float32(520), width-32)).MaxHeight(height-32).
			Padding(18).Gap(10).Background(tk.BgElevated).Radius(12).Border(1, tk.Border).Children(func() {
			display.Text(c, a.t("workspaceCleanTitle"), display.TextProps{Strong: true, FontSize: scaledFont(15)}).Shrink(0)
			switch {
			case s.loading:
				display.Text(c, a.t("workspaceCleanPreparing"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).FillWidth()
			case s.err != "":
				layout.ScrollArea(c, layout.ScrollAreaProps{State: &s.scroll}, func() {
					ui.Text(c, s.err).FillWidth().FontSize(scaledFont(12)).TextColor(tk.Error)
				}).FillWidth().Height(160).MinHeight(0)
			default:
				display.Text(c, a.tf("workspaceCleanSummary", shortSHA(s.preview.Head)), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).FillWidth().Shrink(0)
				count := len(s.preview.RestorePaths) + len(s.preview.NewPaths)
				indexOnly := make(map[string]bool, len(s.preview.IndexOnlyPaths))
				for _, name := range s.preview.IndexOnlyPaths {
					indexOnly[name] = true
				}
				var workingNew []string
				for _, name := range s.preview.NewPaths {
					if !indexOnly[name] {
						workingNew = append(workingNew, name)
					}
				}
				bodyHeight := min(float32(216), float32(count*30+102), max(float32(60), height-360))
				layout.ScrollArea(c, layout.ScrollAreaProps{State: &s.scroll}, func() {
					a.workspaceCleanPaths(c, a.t("workspaceCleanRestore"), s.preview.RestorePaths, false)
					title := a.t("workspaceCleanKeepNew")
					if s.deleteNew {
						title = a.t("workspaceCleanDeleteNew")
					}
					a.workspaceCleanPaths(c, title, workingNew, s.deleteNew)
					a.workspaceCleanPaths(c, a.t("workspaceCleanIndexOnly"), s.preview.IndexOnlyPaths, true)
				}).FillWidth().Height(bodyHeight).MinHeight(0)
				if !s.preview.CanExecute {
					display.Text(c, a.t("workspaceCleanNothing"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(12)}).FillWidth()
				}
			}
			// Keep this available after a blocked preview so deletion can resolve an
			// approved file/directory obstruction, or the user can return to keep mode.
			next := s.deleteNew
			choice.Checkbox(c.Key("delete-new"), &next, a.t("workspaceCleanDeleteOption"), choice.CheckboxProps{Size: 14}).Disabled(s.loading || a.busy)
			if next != s.deleteNew && !s.loading && !a.busy {
				a.prepareWorkspaceClean(next)
			}
			display.Text(c, a.t("workspaceCleanIgnored"), display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).FillWidth().Shrink(0)
			warning := a.t("workspaceCleanWarning")
			if s.deleteNew {
				warning = a.t("workspaceCleanDeleteWarning")
			}
			ui.Text(c, warning).FontSize(scaledFont(12)).TextColor(tk.Error).FillWidth().Shrink(0)
			ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Shrink(0).Children(func() {
				if s.err != "" && workbenchButton(c, a.t("workspaceCleanRetry"), buttonNormal, a.busy || a.loading).Clicked() {
					a.prepareWorkspaceClean(s.deleteNew)
				}
				if workbenchButton(c, a.t("cancel"), buttonNormal, false).AutoFocus().Clicked() {
					s.open = false
				}
				blocked := s.loading || s.err != "" || !s.preview.CanExecute || a.busy || a.loading
				if workbenchButton(c, a.t("workspaceCleanConfirm"), buttonDanger, blocked).Clicked() && !blocked {
					a.confirmWorkspaceClean()
				}
			})
		})
	})
	if !s.open && a.workspaceClean == s {
		a.closeWorkspaceClean()
	}
}

func (a *app) workspaceCleanPaths(c *ui.Context, title string, paths []string, deleting bool) {
	if len(paths) == 0 {
		return
	}
	tk := theme.TokensOf(c)
	ui.Text(c, fmt.Sprintf("%s (%d)", title, len(paths))).FontSize(scaledFont(12)).FontWeight(600).Padding(6, 0).Shrink(0)
	for _, path := range paths {
		label := path
		if strings.ContainsAny(label, "\n\r\t") {
			label = strconv.Quote(label)
		}
		ink := tk.Text
		if deleting {
			ink = tk.Error
		}
		ui.Text(c, label).FontSize(scaledFont(12)).TextColor(ink).SingleLine().FillWidth().MinWidth(0).Height(30).Shrink(0).Tooltip(label)
	}
}
