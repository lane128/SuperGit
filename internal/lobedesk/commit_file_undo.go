package lobedesk

import (
	"context"
	"errors"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

type commitFileUndoTarget struct {
	Patch        commitFilePatchTarget
	Head, Branch string
}

func (a *app) commitFileUndoTarget(path string) commitFileUndoTarget {
	return commitFileUndoTarget{Patch: a.commitFilePatchTarget(path), Head: a.status.Head, Branch: a.status.Branch}
}
func (a *app) commitFileUndoReady(target commitFileUndoTarget) bool {
	commit, ok := a.selectedCommit()
	return ok && len(commit.Parents) == 1 && target.Patch.Parent != "" && target.Patch.File.Status == "M" && target.Patch.File.OriginalPath == "" && len(a.status.Entries) == 0 && a.operation.Kind == "" && a.commitFilePatchReady(target.Patch) && a.commitFileUndoTarget(target.Patch.File.Path) == target
}
func (a *app) requestCommitFileUndo(target commitFileUndoTarget) {
	if !a.commitFileUndoReady(target) {
		return
	}
	repo := a.repo
	expected := gitrepo.Status{Head: target.Head, Branch: target.Branch}
	ctx, cancel := a.beginOperation(busyLabel("busyCheckUndoFile"))
	go func() {
		defer cancel()
		err := repo.CheckCommitFileUndo(ctx, target.Patch.SHA, target.Patch.File.Path, expected)
		if err == nil {
			err = ctx.Err()
		}
		a.hopToMain(func() {
			a.finishOperation()
			if err != nil {
				a.commitFileUndoNotice(target, err)
				return
			}
			if !a.commitFileUndoReady(target) {
				a.commitFileUndoStale(target)
				return
			}
			branch := target.Branch
			if branch == "" {
				branch = a.t("undoFileDetached")
			}
			a.askConfirm(a.t("undoFileTitle"), a.tf("undoFileMessage", target.Patch.File.Path, target.Patch.SHA, target.Patch.Parent, branch, target.Head), a.t("undoFileAction"), true, func() { a.applyCommitFileUndo(target) })
			a.confirm.undoFile = true
		})
	}()
}
func (a *app) applyCommitFileUndo(target commitFileUndoTarget) {
	if !a.commitFileUndoReady(target) {
		a.commitFileUndoStale(target)
		return
	}
	repo := a.repo
	expected := gitrepo.Status{Head: target.Head, Branch: target.Branch}
	ctx, cancel := a.beginOperation(busyLabel("busyUndoFile"))
	go func() {
		defer cancel()
		err, cleanupErr := applyCommitFileUndo(ctx, repo, target, expected)
		// Successful replacement wins over a cancellation arriving afterward.
		a.hopToMain(func() {
			a.finishOperation()
			a.commitFileUndoNotice(target, err)
			if cleanupErr != nil {
				a.notices[toastMain] = append(a.notices[toastMain], &toastNotice{root: target.Patch.Root, kind: "warning", title: a.t("undoFileCleanup"), description: cleanupErr.Error()})
			}
			if a.repo != nil && a.repo.Root == target.Patch.Root {
				a.reload()
			}
			a.invalidate()
		})
	}()
}
func applyCommitFileUndo(ctx context.Context, repo *gitrepo.Repository, target commitFileUndoTarget, expected gitrepo.Status) (error, error) {
	prepared, err := repo.PrepareCommitFileUndo(ctx, target.Patch.SHA, target.Patch.File.Path, expected)
	if err != nil {
		return err, nil
	}
	err = repo.ApplyPreparedCommitFileUndo(ctx, prepared)
	return err, prepared.Close()
}
func (a *app) commitFileUndoStale(target commitFileUndoTarget) {
	a.notices[toastMain] = append(a.notices[toastMain], &toastNotice{root: target.Patch.Root, kind: "warning", title: a.t("undoFileTargetChanged")})
	a.invalidate()
}
func (a *app) commitFileUndoNotice(target commitFileUndoTarget, err error) {
	notice := &toastNotice{root: target.Patch.Root, kind: "success", title: a.t("undoFileSuccess"), description: target.Patch.File.Path}
	if err != nil {
		notice.kind = "error"
		notice.title = a.t("undoFileFailed")
		notice.description = a.operationErrorDescription(err)
		if errors.Is(err, gitrepo.ErrUnsupportedCommitFileUndo) {
			notice.description = a.t("undoFileUnsupported")
		}
		if errors.Is(err, gitrepo.ErrIntegrationTargetChanged) {
			notice.kind = "warning"
			notice.title = a.t("undoFileTargetChanged")
		}
		// The returned error records the outcome. A context cancelled after a
		// permission or I/O failure must not relabel that failure as cancellation.
		if errors.Is(err, context.Canceled) {
			notice.kind = "info"
			notice.title = a.t("operationCancelled")
		}
	}
	a.notices[toastMain] = append(a.notices[toastMain], notice)
	a.invalidate()
}
func (a *app) commitFileUndoDialog(c *ui.Context, blocked bool) bool {
	confirmed := false
	tk := theme.TokensOf(c)
	workbenchModal(c, &a.confirm.open, func() {
		ui.Column(c).Width(600).MaxWidthPercent(90).Height(420).MaxHeightPercent(90).Padding(20).Gap(12).Background(tk.BgElevated).Radius(12).Border(1, tk.Border).Children(func() {
			display.Text(c, a.confirm.title, display.TextProps{Strong: true, FontSize: scaledFont(14), Rows: 2}).FillWidth().Shrink(0)
			layout.ScrollArea(c, layout.ScrollAreaProps{State: &a.confirm.patchScroll}, func() {
				display.Text(c, a.confirm.message, display.TextProps{FontSize: scaledFont(12)}).FillWidth().MinWidth(0)
			}).Grow(1).Basis(0).MinHeight(0)
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Shrink(0).Children(func() {
				if workbenchButton(c, a.t("cancel"), buttonNormal, false).AutoFocus().Clicked() {
					a.confirm.open = false
				}
				if workbenchButton(c, a.confirm.actionName, buttonDanger, blocked).Clicked() && !blocked {
					confirmed = true
					a.confirm.open = false
				}
			})
		})
	})
	return confirmed
}
