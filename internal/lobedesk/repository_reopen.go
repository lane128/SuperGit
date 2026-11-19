package lobedesk

import "slices"

// Keep just enough information to reopen a checkout as a fresh tab.
// Drafts and Git snapshots belong to the closed session only.
type closedRepositoryTab struct {
	root  string
	index int
}

func (tabs *repositoryTabs) rememberClosedRepository(root string, index int) {
	const limit = 20
	tabs.forgetClosedRepository(root)
	if len(tabs.closed) == limit {
		tabs.closed = slices.Delete(tabs.closed, 0, 1)
	}
	tabs.closed = append(tabs.closed, closedRepositoryTab{root: root, index: index})
	tabs.updateReopenMenu()
}

func (tabs *repositoryTabs) forgetClosedRepository(root string) {
	tabs.closed = slices.DeleteFunc(tabs.closed, func(entry closedRepositoryTab) bool { return entry.root == root })
	tabs.updateReopenMenu()
}

func (tabs *repositoryTabs) updateReopenMenu() {
	if item := tabs.reopenMenuItem; item != nil && item.Disabled != (len(tabs.closed) == 0) {
		item.SetEnabled(len(tabs.closed) > 0)
	}
}

func (tabs *repositoryTabs) reopenClosedRepository() {
	if len(tabs.closed) == 0 || !tabs.canChange() {
		return
	}
	i := len(tabs.closed) - 1
	entry := tabs.closed[i]
	// Consume this attempt even if the checkout is unavailable, so an old
	// missing checkout cannot block older entries. Open Recent remains usable.
	tabs.closed = slices.Delete(tabs.closed, i, i+1)
	tabs.updateReopenMenu()
	source := tabs.active
	next := tabs.newSession(source)
	source.cancelLocalProbe()
	tabs.opening, tabs.reopening = next, entry
	next.readRepositoryAtRoot(entry.root, true, entry.root)
	source.invalidate()
}
