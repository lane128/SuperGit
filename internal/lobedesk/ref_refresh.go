package lobedesk

import (
	"context"
	"errors"
	"sync"
	"time"

	"supergit/internal/gitrepo"
)

// refreshRefs reconciles remote and external ref updates using the tab lifetime.
// It never advances the local worktree observation or replaces child jobs.
func (a *app) refreshRefs() {
	repo, epoch, expected := a.repo, a.loadSeq, a.checkoutIdentity
	if repo == nil || a.localWatch.closed {
		return
	}
	a.cancelLocalProbe()
	a.loading = true
	a.loadSince = time.Now()
	a.invalidate()
	ctx, cancel := context.WithTimeout(a.ctx, localBackgroundInterval)
	if a.refCancel != nil {
		a.refCancel()
	}
	a.refCancel = cancel
	a.refSeq++
	seq := a.refSeq
	go func() {
		defer cancel()
		stamp, err := repo.RefStamp(ctx)
		var branches, remote []gitrepo.Branch
		var tags []gitrepo.Tag
		var stashes []gitrepo.Stash
		var errs [4]error
		var wg sync.WaitGroup
		if err == nil {
			wg.Go(func() { branches, errs[0] = repo.Branches(ctx) })
			wg.Go(func() { remote, errs[1] = repo.RemoteBranches(ctx) })
			wg.Go(func() { tags, errs[2] = repo.Tags(ctx) })
			wg.Go(func() { stashes, errs[3] = repo.Stashes(ctx) })
			wg.Wait()
			err = errors.Join(errs[:]...)
		}
		id, idErr := repo.CheckoutIdentity(ctx)
		a.hopToMain(func() {
			if seq != a.refSeq || epoch != a.loadSeq || a.repo != repo || a.localWatch.closed {
				return
			}
			a.refCancel = nil
			a.loading = false
			if err != nil || idErr != nil || id != expected || id.Head != a.status.Head {
				a.reload()
				return
			}
			a.activationRefresh = false
			a.branches, a.remoteBranches, a.tags, a.stashes = branches, remote, tags, stashes
			a.branchesErr, a.remoteBranchesErr, a.tagsErr, a.stashesErr = "", "", "", ""
			refsChanged := a.localWatch.refStamp != stamp
			a.localWatch.refStamp = stamp
			for _, b := range branches {
				if b.Current {
					if b.SHA != id.Head || b.Name != a.status.Branch {
						a.reload()
						return
					}
					a.status.Upstream, a.status.UpstreamGone, a.status.Ahead, a.status.Behind = b.Upstream, b.UpstreamGone, b.Ahead, b.Behind
					break
				}
			}
			if a.historyRef.Kind != "" {
				a.historyPinned = a.historyRefTip(a.historyRef)
				if a.historyPinned == "" {
					a.historyRef = historyRef{}
				}
			}
			if refsChanged || a.historyErr != "" {
				if a.fileFilter != "" && a.fileHistoryRevision == "" {
					a.resetFileHistory()
					a.fileCommits = nil
					a.fileHistoryTips = nil
					a.fileHistoryMore = true
					a.fileHistoryErr = ""
					a.loadFileHistory()
				}
				a.reloadHistoryFromTip(a.historyPinned)
			} else {
				a.loadMoreHistory()
			}
			a.invalidate()
		})
	}()
}
