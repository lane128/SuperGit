package lobedesk

import (
	"context"

	"supergit/internal/gitrepo"
)

type worktreeOverviewEntry struct {
	Key     detailFileKey
	Diff    gitrepo.Diff
	Loading bool
	Err     string
}

// overviewKeys uses the same group order as the file column without depending
// on which groups the reader folded. Every changed version remains a target.
func (a *app) overviewKeys() []detailFileKey {
	var keys []detailFileKey
	add := func(entries []gitrepo.Entry, staged bool) {
		for _, entry := range entries {
			keys = append(keys, detailFileKey{Path: entry.Path, Staged: staged})
		}
	}
	for _, group := range a.workingFileGroups() {
		add(group.Entries, group.Staged)
	}
	return keys
}

func (a *app) invalidateWorktreeOverview() {
	if a.overviewLineTarget != nil {
		a.clearLineSelection()
	}
	if a.overviewCancel != nil {
		a.overviewCancel()
	}
	a.overviewCancel = nil
	a.overviewQueue = nil
	a.overviewEntries = nil
	a.overviewRoot = ""
	a.overviewSeq++
}

// loadWorktreeOverview runs on selection/refresh, never during drawing. Each
// snapshot has one cancellable serial queue, including item-level retries.
func (a *app) loadWorktreeOverview() {
	if a.repo == nil || a.ctx == nil {
		return
	}
	if a.overviewCancel != nil {
		a.overviewCancel()
	}
	if a.overviewLineTarget != nil {
		a.clearLineSelection()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.overviewCancel = cancel
	a.overviewSeq++
	seq, repo := a.overviewSeq, a.repo
	epoch := a.loadSeq
	ignoreWhitespace := a.ignoreDiffWhitespace
	root := repo.Root
	keys := a.overviewKeys()
	a.overviewRoot = root
	a.overviewEntries = make([]worktreeOverviewEntry, len(keys))
	var pending []detailFileKey
	for i, key := range keys {
		if cached, ok := a.diffCache.get(root, epoch, key); ok {
			a.overviewEntries[i] = worktreeOverviewEntry{Key: key, Diff: cached}
		} else {
			a.overviewEntries[i] = worktreeOverviewEntry{Key: key, Loading: true}
			pending = append(pending, key)
		}
	}
	queue := newOverviewReadQueue(pending)
	a.overviewQueue = queue
	a.reconcileWorktreeTabs(keys)
	if len(keys) == 0 {
		cancel()
		return
	}
	go func() {
		for {
			key, ok := queue.next(ctx)
			if !ok {
				return
			}
			diff, err := repo.DisplayDiff(ctx, key.Path, key.Staged, ignoreWhitespace)
			if ctx.Err() != nil {
				return
			}
			a.hopToMain(func() {
				if a.repo == nil || a.loadSeq != epoch || a.repo.Root != root || a.overviewRoot != root || a.overviewSeq != seq {
					return
				}
				for i := range a.overviewEntries {
					entry := &a.overviewEntries[i]
					if entry.Key != key {
						continue
					}
					entry.Loading = false
					entry.Err = ""
					if err != nil {
						entry.Err = err.Error()
					} else {
						entry.Diff = diff
						a.diffCache.put(root, epoch, key, diff)
					}
					break
				}
				a.invalidate()
			})
		}
	}()
	a.invalidate()
}

// Retry is bound to the failed snapshot and exact Index/worktree version. A
// menu left over from a refresh cannot enqueue a read against a newer target.
func (a *app) retryWorktreeOverview(root string, seq uint64, key detailFileKey) {
	if a.repo == nil || a.repo.Root != root || a.overviewRoot != root || a.overviewSeq != seq || a.overviewQueue == nil {
		return
	}
	for i := range a.overviewEntries {
		entry := &a.overviewEntries[i]
		if entry.Key != key || entry.Loading || entry.Err == "" {
			continue
		}
		if a.overviewQueue.push(key) {
			entry.Loading = true
			entry.Err = ""
			a.invalidate()
		}
		return
	}
}

// Call only after a successful status read. A missing version closes its tab;
// its path must never silently change from unstaged to staged or vice versa.
func (a *app) reconcileWorktreeTabs(keys []detailFileKey) {
	a.detailTabs() // Synchronize the repository identity first.
	tabs := &a.detailContexts.Worktree
	valid := make(map[detailFileKey]bool, len(keys))
	for _, key := range keys {
		valid[key] = true
	}
	open := append([]detailFileKey(nil), tabs.OpenFiles...)
	for _, file := range open {
		if valid[file] {
			continue
		}
		if a.worktreeSelected {
			a.closeDetailFile(file)
		} else {
			tabs.Close(file)
			delete(tabs.Scroll, detailTabKey{File: file})
		}
	}
	// A folded file group may move the list cursor during reconciliation.
	// A surviving file tab still owns the patch and every header action.
	if a.worktreeSelected && !tabs.Active.Summary {
		file := tabs.Active.File
		if a.selectedPath != file.Path || a.selectedStaged != file.Staged {
			a.selectedPath, a.selectedStaged = file.Path, file.Staged
			a.diffSeq++
			a.diff = gitrepo.Diff{}
			a.diffErr = ""
			a.diffLoading = false
			a.clearLineSelection()
		}
	}
}
