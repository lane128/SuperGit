package lobedesk

import (
	"sort"
	"time"

	"supergit/internal/gitrepo"
)

// Stashes are separate reflog entries. Only their first parent belongs in
// the visible graph; their index and optional untracked snapshots are metadata.
type stashTimelineCache struct {
	source, rows []gitrepo.Commit
	stashes      []gitrepo.Stash
	bySHA        map[string]*gitrepo.Stash
	query        string
}

func (a *app) historyWithStashes(source []gitrepo.Commit) []gitrepo.Commit {
	if len(a.stashes) == 0 || a.stashesErr != "" || a.fileFilter != "" || a.historyRef.Kind != "" {
		a.stashTimeline = stashTimelineCache{}
		return source
	}
	cached := &a.stashTimeline
	if cached.query == a.historyQuery && len(cached.stashes) == len(a.stashes) && &cached.stashes[0] == &a.stashes[0] && sameHistorySnapshot(cached.source, source) {
		return cached.rows
	}
	// The ordinary walk is topological and can contain skewed timestamps.
	// Search a non-increasing sequence without reordering existing commits.
	dates := make([]time.Time, len(source))
	positions := make(map[string]int, len(source))
	for i, commit := range source {
		dates[i], _ = time.Parse(time.RFC3339, commit.Date)
		if i > 0 && dates[i].After(dates[i-1]) {
			dates[i] = dates[i-1]
		}
		positions[commit.SHA] = i
	}
	bySHA := make(map[string]*gitrepo.Stash, len(a.stashes))
	slots := make(map[int][]gitrepo.Commit)
	for i := range a.stashes {
		s := &a.stashes[i]
		commit := stashCommit(*s)
		commit.Refs = s.Ref
		if a.historyQuery != "" && len(filterCommits([]gitrepo.Commit{commit}, a.historyQuery)) == 0 {
			continue
		}
		bySHA[s.SHA] = s
		// A branch can independently point to the same Git object.
		if _, exists := positions[s.SHA]; exists {
			continue
		}
		at := 0
		if t, err := time.Parse(time.RFC3339, s.Date); err == nil {
			at = sort.Search(len(dates), func(i int) bool { return !dates[i].After(t) })
		}
		if len(commit.Parents) > 0 {
			commit.Parents = commit.Parents[:1]
			if base, ok := positions[commit.Parents[0]]; ok {
				at = min(at, base)
			}
		}
		slots[at] = append(slots[at], commit)
	}
	rows := make([]gitrepo.Commit, 0, len(source)+len(a.stashes))
	for i, commit := range source {
		rows = append(rows, slots[i]...)
		if stash := bySHA[commit.SHA]; stash != nil && len(commit.Parents) > 1 {
			commit.Parents = commit.Parents[:1]
		}
		rows = append(rows, commit)
	}
	rows = append(rows, slots[len(source)]...)
	*cached = stashTimelineCache{query: a.historyQuery, source: source, stashes: a.stashes, rows: rows, bySHA: bySHA}
	return rows
}
