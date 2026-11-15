package desktop

import (
	"context"
	"github.com/egoist/mygo"
	"supergit/internal/gitrepo"
)

type stashPreviewRequest struct {
	ticket ticket
	sha    string
}

func (a *app) cancelStashPreview() {
	if a.stashCancel != nil {
		a.stashCancel()
		a.stashCancel = nil
	}
	a.stashLoading = false
	a.guard.Begin("stash")
}

func (a *app) beginStashPreview(sha string) (stashPreviewRequest, context.Context, bool) {
	if a.closed || a.repo == nil || a.guard.writing || a.opening || sha == "" {
		return stashPreviewRequest{}, nil, false
	}
	a.cancelStashPreview()
	a.stashSHA = sha
	a.stashDiff = gitrepo.Diff{}
	a.stashLoaded, a.stashLoading = false, true
	a.stashErr = nil
	// Committed accessibility state borrows ListState until the next build.
	a.stashListReset = true
	ctx, cancel := context.WithCancel(a.ctx)
	a.stashCancel = cancel
	return stashPreviewRequest{ticket: a.guard.Begin("stash"), sha: sha}, ctx, true
}

func (a *app) finishStashPreview(req stashPreviewRequest, d gitrepo.Diff, err error) bool {
	if a.closed || a.repo == nil || !a.stashLoading || !a.guard.Accept(req.ticket) || req.sha != a.stashSHA {
		return false
	}
	if a.stashCancel != nil {
		a.stashCancel()
		a.stashCancel = nil
	}
	a.stashLoading, a.stashLoaded = false, true
	a.stashErr = err
	if err != nil {
		a.stashDiff = gitrepo.Diff{}
		a.record("读取 Stash 失败", err)
	} else {
		a.stashDiff = d
	}
	return true
}

func (a *app) selectStash(sha string) {
	req, ctx, ok := a.beginStashPreview(sha)
	if !ok {
		return
	}
	r, cancel := a.repo, a.stashCancel
	go func() {
		defer cancel()
		d, err := r.StashDiff(ctx, req.sha)
		mygo.RunOnMain(func() {
			if a.finishStashPreview(req, d, err) {
				a.invalidate()
			}
		})
	}()
}
