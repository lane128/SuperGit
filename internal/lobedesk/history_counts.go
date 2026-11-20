package lobedesk

import (
	"strings"
	"supergit/internal/gitrepo"
)

// Count only rows requested by the virtual list. A large repository can
// finish loading its history without diffing every tree in the repository.
type historyFileCounts struct {
	root      string
	counts    map[string]int
	loading   bool
	failed    bool
	failedSeq uint64
}

// prepareHistoryCounts returns the repository cache only when a read can start.
func (a *app) prepareHistoryCounts() *historyFileCounts {
	if a.window == nil || a.repo == nil {
		return nil
	}
	if a.historyCounts == nil || a.historyCounts.root != a.repo.Root {
		a.historyCounts = &historyFileCounts{root: a.repo.Root, counts: make(map[string]int)}
	}
	cache := a.historyCounts
	if cache.loading || cache.failed && cache.failedSeq == a.loadSeq {
		return nil
	}
	return cache
}

func (cache *historyFileCounts) queue(batch *[]gitrepo.Commit, row gitrepo.Commit) {
	if cache == nil || len(*batch) >= 64 {
		return
	}
	if _, known := cache.counts[row.SHA]; !known {
		*batch = append(*batch, row)
	}
}

// A stash's third parent is a root commit containing its saved untracked files.
func stashUntrackedSnapshot(s *gitrepo.Stash) string {
	if s != nil {
		parents := strings.Fields(s.ParentSHAs)
		if len(parents) == 3 {
			return parents[2]
		}
	}
	return ""
}

func (cache *historyFileCounts) fileCount(sha string, stash *gitrepo.Stash) (int, bool) {
	count, known := cache.counts[sha]
	if extra := stashUntrackedSnapshot(stash); extra != "" {
		files, ok := cache.counts[extra]
		return count + files, known && ok
	}
	return count, known
}

func (a *app) loadHistoryFileCounts(cache *historyFileCounts, batch []gitrepo.Commit) {
	if len(batch) == 0 {
		return
	}
	cache.loading = true
	repo, ctx, seq := a.repo, a.ctx, a.loadSeq
	go func() {
		result, err := repo.ChangedFileCounts(ctx, batch)
		a.hopToMain(func() {
			cache.loading = false
			for sha, n := range result {
				cache.counts[sha] = n
			}
			if err != nil {
				cache.failed, cache.failedSeq = true, seq
			} else {
				cache.failed = false
			}
			if a.historyCounts == cache {
				a.invalidate()
			}
		})
	}()
}
