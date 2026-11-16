package lobedesk

import "supergit/internal/gitrepo"

const worktreeDiffCacheLimit = 16 << 20

// UI-thread owned. Working patches belong to one status refresh, never merely
// to status letters: bytes can change without changing a file's status code.
// Errors are never inserted, and explicit Retry always reads Git again.
type worktreeDiffCache struct {
	root         string
	epoch        uint64
	entries      map[detailFileKey]gitrepo.Diff
	order        []detailFileKey
	bytes, limit int
}

func (c *worktreeDiffCache) get(root string, epoch uint64, key detailFileKey) (gitrepo.Diff, bool) {
	if c.root != root || c.epoch != epoch {
		return gitrepo.Diff{}, false
	}
	diff, ok := c.entries[key]
	return diff, ok
}

// Count text, parsed strings and conservative per-entry/slice/map overhead.
// Shared substrings may be counted twice, keeping this an upper budget.
func cachedDiffBytes(key detailFileKey, diff gitrepo.Diff) int {
	size := 256 + len(key.Path) + len(diff.Text)
	for _, line := range diff.Lines {
		size += 80 + len(line.Kind) + len(line.Text)
	}
	for _, hunk := range diff.Hunks {
		size += 48 + len(hunk.Header) + len(hunk.Patch)
	}
	return size
}

func (c *worktreeDiffCache) put(root string, epoch uint64, key detailFileKey, diff gitrepo.Diff) {
	if c.root != root || c.epoch != epoch || c.entries == nil {
		c.root, c.epoch = root, epoch
		c.entries = make(map[detailFileKey]gitrepo.Diff)
		c.order, c.bytes = nil, 0
	}
	limit := c.limit
	if limit <= 0 {
		limit = worktreeDiffCacheLimit
	}
	size := cachedDiffBytes(key, diff)
	if old, ok := c.entries[key]; ok {
		c.bytes -= cachedDiffBytes(key, old)
		delete(c.entries, key)
		for i, existing := range c.order {
			if existing == key {
				c.order = append(c.order[:i], c.order[i+1:]...)
				break
			}
		}
	}
	if size > limit {
		return
	}
	for c.bytes+size > limit && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.bytes -= cachedDiffBytes(oldest, c.entries[oldest])
		delete(c.entries, oldest)
	}
	c.entries[key] = diff
	c.order = append(c.order, key)
	c.bytes += size
}
