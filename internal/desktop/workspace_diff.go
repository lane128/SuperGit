package desktop

import (
	"context"
	"supergit/internal/gitrepo"
)

// Index and worktree previews of the same raw path are different targets.
type workspaceDiffRequest struct {
	ticket ticket
	path   string
	staged bool
}

func (a *app) beginWorkspaceDiff(path string, staged, preserve bool) (workspaceDiffRequest, context.Context, bool) {
	if a.closed || a.repo == nil || a.guard.writing || a.opening || path == "" {
		return workspaceDiffRequest{}, nil, false
	}
	if !preserve || a.path != path || a.staged != staged {
		a.clearLineSelection()
		a.diff = gitrepo.Diff{}
		a.diffLoaded = false
		a.workspaceDiffReset = true
	}
	a.path, a.staged = path, staged
	// A previous failure completed with no usable preview. Clearing its error
	// starts recovery; only a successful result can make that preview loaded.
	if a.diffErr != nil {
		a.diffLoaded = false
	}
	a.diffErr = nil
	a.diffLoading = true
	ctx, _ := a.newWorkspaceRead()
	return workspaceDiffRequest{ticket: a.guard.Begin("diff"), path: path, staged: staged}, ctx, true
}

func (a *app) failWorkspaceDiff(err error) {
	a.clearLineSelection()
	a.diff = gitrepo.Diff{}
	a.diffLoaded, a.diffErr = true, err
}

func (a *app) finishWorkspaceDiff(req workspaceDiffRequest, d gitrepo.Diff, err error) bool {
	if a.closed || a.repo == nil || !a.diffLoading || !a.guard.Accept(req.ticket) || req.path != a.path || req.staged != a.staged {
		return false
	}
	if a.diffCancel != nil {
		a.diffCancel()
		a.diffCancel = nil
	}
	a.diffLoading = false
	a.diffLoaded, a.diffErr = true, err
	if err != nil {
		a.failWorkspaceDiff(err)
		a.record("读取 Diff 失败", err)
	} else {
		a.applyDiffResult(req.ticket, d, false)
	}
	return true
}
