package desktop

import (
	"context"
	"github.com/egoist/mygo"
	"supergit/internal/gitrepo"
)

// Empty path denotes the commit's file list; a nonempty path denotes its Diff.
// These reads share one cancellable lane, independent of commit messages.
type commitPreviewRequest struct {
	ticket    ticket
	sha, path string
}

func (a *app) beginCommitPreview(path string) (commitPreviewRequest, context.Context, bool) {
	if a.closed || a.repo == nil || a.selectedSHA == "" || a.guard.writing || a.opening {
		return commitPreviewRequest{}, nil, false
	}
	if path != "" {
		found := false
		for _, f := range a.commitFiles {
			if f.Path == path {
				found = true
				break
			}
		}
		if !found {
			return commitPreviewRequest{}, nil, false
		}
	} else {
		a.replaceCommitFiles(nil)
		a.commitFilesLoaded = false
		a.commitFilesErr = nil
	}
	a.resetCommitDiff(path)
	a.commitLoading = true
	ctx, _ := a.newCommitRead()
	return commitPreviewRequest{ticket: a.guard.Begin("commit"), sha: a.selectedSHA, path: path}, ctx, true
}

func (a *app) acceptCommitPreview(req commitPreviewRequest) bool {
	return !a.closed && a.repo != nil && a.commitLoading && a.guard.Accept(req.ticket) && req.sha == a.selectedSHA && req.path == a.commitPath
}

func (a *app) finishCommitPreview() {
	if a.commitCancel != nil {
		a.commitCancel()
		a.commitCancel = nil
	}
	a.commitLoading = false
}

func (a *app) finishCommitFiles(req commitPreviewRequest, files []gitrepo.ChangedFile, err error) bool {
	if req.path != "" || !a.acceptCommitPreview(req) {
		return false
	}
	a.finishCommitPreview()
	a.commitFilesLoaded, a.commitFilesErr = true, err
	if err != nil {
		a.replaceCommitFiles(nil)
		a.record("读取提交文件失败", err)
	} else {
		a.replaceCommitFiles(files)
	}
	return true
}

func (a *app) finishCommitDiff(req commitPreviewRequest, d gitrepo.Diff, err error) bool {
	if req.path == "" || !a.acceptCommitPreview(req) {
		return false
	}
	a.finishCommitPreview()
	a.commitDiffLoaded, a.commitDiffErr = true, err
	if err != nil {
		a.commitDiff = gitrepo.Diff{}
		a.record("读取提交 Diff 失败", err)
	} else {
		a.applyDiffResult(req.ticket, d, true)
	}
	return true
}

func (a *app) loadCommitFiles() {
	req, ctx, ok := a.beginCommitPreview("")
	if !ok {
		return
	}
	r, cancel := a.repo, a.commitCancel
	go func() {
		defer cancel()
		files, err := r.CommitFiles(ctx, req.sha)
		mygo.RunOnMain(func() {
			if !a.finishCommitFiles(req, files, err) {
				return
			}
			if err == nil && len(files) > 0 {
				a.showCommitFile(files[0].Path)
			}
			a.invalidate()
		})
	}()
}
