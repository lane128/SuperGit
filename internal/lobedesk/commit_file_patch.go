package lobedesk

import (
	"context"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"supergit/internal/gitrepo"
)

type commitFilePatchTarget struct {
	Root        string
	Generation  uint64
	SHA, Parent string
	File        gitrepo.ChangedFile
}

func (a *app) commitFilePatchTarget(path string) commitFilePatchTarget {
	if a.repo == nil {
		return commitFilePatchTarget{}
	}
	commit, ok := a.selectedCommit()
	if !ok {
		return commitFilePatchTarget{}
	}
	for _, file := range a.commitFiles {
		if file.Path == path {
			target := commitFilePatchTarget{Root: a.repo.Root, Generation: a.loadSeq, SHA: a.selectedSHA, File: file}
			if len(commit.Parents) > 0 {
				target.Parent = commit.Parents[0]
			}
			return target
		}
	}
	return commitFilePatchTarget{}
}
func (a *app) commitFilePatchReady(target commitFilePatchTarget) bool {
	return target.Root != "" && target.SHA != "" && a.repo != nil && !a.worktreeSelected &&
		!a.busy && !a.loading && !a.tabSnapshotUnavailable() && !a.repositoryChooserOpen && !a.hasPendingDialog(toastMain) &&
		a.detailNavigationAllowed() && a.commitFilesLoaded && !a.commitFilesLoading && a.commitFilesErr == "" &&
		a.commitFilePatchTarget(target.File.Path) == target
}
func (a *app) exportCommitFilePatch(target commitFilePatchTarget) {
	if !a.commitFilePatchReady(target) || a.window == nil && a.saveFileDialog == nil {
		return
	}
	save := a.saveFileDialog
	if save == nil {
		save = mygo.Dialog.Save
	}
	base := a.t("patchEmptyTree")
	if target.Parent != "" {
		base = a.tf("patchFirstParent", shortSHA(target.Parent))
	}
	path := target.File.Path
	if target.File.OriginalPath != "" {
		path = target.File.OriginalPath + " → " + path
	}
	a.repositoryChooserOpen = true
	destination, err := func() (string, error) {
		defer func() { a.repositoryChooserOpen = false }()
		return save(mygo.SaveDialogOptions{Parent: a.window, Title: a.t("exportFilePatchTitle"),
			Message:     a.tf("exportFilePatchMessage", path, shortSHA(target.SHA), base),
			DefaultPath: filepath.Base(target.File.Path) + "-" + shortSHA(target.SHA) + ".patch", ButtonLabel: a.t("save"),
			Filters: []mygo.FileFilter{{Name: a.t("patchFile"), Extensions: []string{"patch", "diff"}}}, CreateDirectories: true})
	}()
	if !a.commitFilePatchReady(target) {
		return
	}
	if err != nil {
		a.queue("error", a.t("exportFilePatchFailed"), err.Error())
		a.invalidate()
		return
	}
	if destination == "" {
		return
	}
	// Capture immutable inputs before the worker starts. No repository reload is
	// needed: exporting a historical object does not change Git state.
	repo := &gitrepo.Repository{Root: target.Root}
	ctx, cancel := a.beginOperation(busyLabel("busyExportFilePatch"))
	go func() {
		defer cancel()
		err := saveCommitFilePatch(ctx, repo, target.SHA, target.File.Path, destination)
		cancelled := operationCancelled(ctx, err)
		a.hopToMain(func() {
			a.finishOperation()
			// Feedback belongs to the originating root, even if the app moved away.
			notice := &toastNotice{root: target.Root}
			switch {
			case cancelled:
				notice.kind = "info"
				notice.title = a.t("operationCancelled")
			case err != nil:
				notice.kind = "error"
				notice.title = a.t("exportFilePatchFailed")
				notice.description = a.operationErrorDescription(err)
			default:
				notice.kind = "success"
				notice.title = a.t("patchSaved")
				notice.description = destination
			}
			a.notices[toastMain] = append(a.notices[toastMain], notice)
			a.invalidate()
		})
	}()
}

// Generate beside the destination and only replace it after the complete patch
// has been written and closed. Failures only remove our own temporary file.
func saveCommitFilePatch(ctx context.Context, repo *gitrepo.Repository, sha, path, destination string) error {
	file, err := os.CreateTemp(filepath.Dir(destination), ".supergit-patch-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	err = repo.WriteCommitFilePatch(ctx, sha, path, file)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(file.Name(), destination)
}
