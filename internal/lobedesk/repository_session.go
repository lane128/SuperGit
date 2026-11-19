package lobedesk

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"supergit/internal/gitrepo"
)

// This is window state, independent of preferences and Git operation state.
type repositorySession struct {
	Roots  []string          `json:"roots"`
	Active string            `json:"active"`
	Drafts map[string]string `json:"drafts,omitempty"`
}

type repositorySessionState struct {
	path             string
	savedRecentHead  string
	tabsChanged      bool
	orderChanged     bool
	selectionChanged bool
	draftBaseline    map[string]string
	rootsBaseline    []string
	discardedDrafts  map[string]bool
	restoring        bool
	skipped          int
}

func repositorySessionPath() string {
	path, err := settingsPath()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "repository-session.json")
}

func readRepositorySession(path string) repositorySession {
	var session repositorySession
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &session) != nil {
		return repositorySession{}
	}
	return session
}

func writeRepositorySession(path string, session repositorySession) error {
	return writeLocalJSON(path, "repository-session-*.json", session)
}

func (tabs *repositoryTabs) repositorySession() repositorySession {
	session := repositorySession{Roots: make([]string, 0, len(tabs.sessions))}
	for _, a := range tabs.sessions {
		if a.repo != nil {
			session.Roots = append(session.Roots, a.repo.Root)
		}
	}
	if tabs.active.repo != nil && !tabs.active.restoredCheckout {
		session.Active = tabs.active.repo.Root
	} else if tabs.lastCheckout != nil && slices.Contains(tabs.sessions, tabs.lastCheckout) {
		session.Active = tabs.lastCheckout.repo.Root
	} else {
		for _, a := range tabs.sessions {
			if a.repo != nil && !a.restoredCheckout {
				session.Active = a.repo.Root
				break
			}
		}
	}
	return session
}

func (tabs *repositoryTabs) acceptCheckoutSelection(a *app) {
	tabs.lastCheckout = a
	if !tabs.sessionState.restoring {
		tabs.sessionState.selectionChanged = true
	}
}

func (tabs *repositoryTabs) saveSession() {
	state := &tabs.sessionState
	if state.path == "" {
		return
	}
	draftChanges := make(map[string]string)
	for _, a := range tabs.sessions {
		if a.repo == nil || a.restoredCheckout || a.unrestoredDraft != "" || a.loadErr != "" ||
			a.amend || a.operation.Kind != "" || a.pendingMessage != "" {
			continue
		}
		if root := a.repo.Root; a.commitText != state.draftBaseline[root] || state.discardedDrafts[root] {
			draftChanges[root] = a.commitText
		}
	}
	if !state.tabsChanged && !state.orderChanged && !state.selectionChanged && len(draftChanges) == 0 {
		return
	}
	session := tabs.repositorySession()
	saved := readRepositorySession(state.path)
	if !state.tabsChanged {
		// Preserve unavailable roots and membership changes made elsewhere.
		if state.orderChanged {
			var slots []int
			var roots []string
			for i, root := range saved.Roots {
				if slices.Contains(session.Roots, root) && !slices.Contains(saved.Roots[:i], root) {
					slots = append(slots, i)
				}
			}
			for _, root := range session.Roots {
				if slices.Contains(saved.Roots, root) {
					roots = append(roots, root)
				}
			}
			for i, slot := range slots {
				saved.Roots[slot] = roots[i]
			}
		}
		if slices.Contains(saved.Roots, session.Active) {
			saved.Active = session.Active
		} else if !state.orderChanged && len(draftChanges) == 0 {
			return
		}
		session = saved
	} else {
		// Keep checkouts another process added after we read the session,
		// unless this workbench explicitly closed the same checkout.
		for _, root := range saved.Roots {
			if !slices.Contains(state.rootsBaseline, root) && !slices.Contains(session.Roots, root) && !state.discardedDrafts[root] {
				session.Roots = append(session.Roots, root)
			}
		}
	}
	// Merge only locally edited drafts. Inactive or unchanged tabs may hold
	// older snapshots than another workbench that saved more recently.
	session.Drafts = make(map[string]string)
	for _, root := range session.Roots {
		message, edited := draftChanges[root]
		if !edited && !state.discardedDrafts[root] {
			message = saved.Drafts[root]
		}
		if message != "" {
			session.Drafts[root] = message
		}
	}
	if err := writeRepositorySession(state.path, session); err != nil {
		log.Printf("could not save repository session: %v", err)
		return
	}
	if len(draftChanges) > 0 && state.draftBaseline == nil {
		state.draftBaseline = make(map[string]string)
	}
	for root, message := range draftChanges {
		if message == "" || !slices.Contains(session.Roots, root) {
			delete(state.draftBaseline, root)
		} else {
			state.draftBaseline[root] = message
		}
	}
	for root := range state.discardedDrafts {
		if session.Drafts[root] == "" {
			delete(state.draftBaseline, root)
		}
	}
	clear(state.discardedDrafts)
	state.tabsChanged, state.orderChanged, state.selectionChanged = false, false, false
}

func (tabs *repositoryTabs) startSession(path, initialPath string) {
	a := tabs.active
	defer func() {
		tabs.sessionStarted = true
		if len(tabs.repositoryFolders) > 0 {
			a.invalidate()
		}
	}()
	if initialPath != "" {
		// A one-off launch does not replace the normal workbench session.
		a.openRepository(initialPath, true)
		return
	}
	if path == "" {
		return
	}
	tabs.sessionState.path = path
	a.loading, a.loadSince = true, time.Now()
	a.invalidate()
	go func() {
		saved := readRepositorySession(path)
		recents, err := readRecentRepositories(recentRepositoriesPath(path))
		savedRecentHead := ""
		switch {
		case err != nil:
			log.Printf("could not read recent repositories: %v", err)
		case len(recents) > 0:
			savedRecentHead = recents[0]
		case recents == nil:
			recents = legacyRecentRepositories(path)
		}
		repos, active, skipped := probeRepositorySession(tabs.ctx, saved)
		aliases := make([]string, len(repos))
		for i, repo := range repos {
			aliases[i] = readRepositoryAlias(repo.Root)
		}
		a.hopToMain(func() {
			if a.localWatch.closed {
				return
			}
			a.recentRepositories = recents
			tabs.refreshRepositoryRecents()
			state := &tabs.sessionState
			state.savedRecentHead = savedRecentHead
			state.draftBaseline = saved.Drafts
			state.rootsBaseline = saved.Roots
			state.restoring, state.skipped = true, skipped
			a.loading = false
			if len(repos) > 0 {
				tabs.sessions = make([]*app, len(repos))
				for i, repo := range repos {
					next := a
					if i != active {
						next = tabs.newSession(a)
					}
					next.repo, next.restoredCheckout = repo, true
					next.unrestoredDraft = saved.Drafts[repo.Root]
					next.repositoryAlias = aliases[i]
					tabs.sessions[i] = next
				}
				tabs.reveal = true
				a.reload()
			} else {
				tabs.finishSessionRestore()
			}
		})
	}()
}

func (tabs *repositoryTabs) finishSessionRestore() {
	state := &tabs.sessionState
	state.restoring = false
	a := tabs.active
	if state.skipped > 0 {
		a.queue("warning", a.t("repositoryRestoreSkipped"), a.tf("repositoryRestoreSkippedDetail", state.skipped))
	}
	state.skipped = 0
	a.invalidate()
}

// Use the normal snapshot read to detect a checkout that Git can locate but
// cannot read. Walk right first and then left, just as for a missing root.
func (tabs *repositoryTabs) skipFailedRestore(a *app) bool {
	if !tabs.sessionState.restoring || a != tabs.active {
		return false
	}
	tabs.sessionState.skipped++
	i := slices.Index(tabs.sessions, a)
	tabs.sessions = slices.Delete(tabs.sessions, i, i+1)
	if len(tabs.sessions) == 0 {
		tabs.sessions = append(tabs.sessions, tabs.newSession(a))
	}
	next := tabs.sessions[min(i, len(tabs.sessions)-1)]
	tabs.activate(next, false)
	a.stopRepositorySession()
	if next.repo == nil {
		tabs.finishSessionRestore()
	}
	return true
}

// Probe checkout identity only. The selected tab reads its workbench snapshot
// first; the other checkouts read fresh snapshots when selected.
func probeRepositorySession(ctx context.Context, saved repositorySession) (repos []*gitrepo.Repository, active, skipped int) {
	activeAt := slices.Index(saved.Roots, saved.Active)
	left, right := -1, -1
	seen := make(map[string]bool)
	for i, root := range saved.Roots {
		if ctx.Err() != nil {
			break
		}
		if seen[root] {
			continue
		}
		seen[root] = true
		if !filepath.IsAbs(root) {
			skipped++
			continue
		}
		probe, cancel := context.WithTimeout(ctx, localBackgroundInterval)
		repo, err := gitrepo.Open(probe, root)
		cancel()
		if err != nil || repo.Root != root {
			skipped++
			continue
		}
		if i < activeAt {
			left = len(repos)
		} else if right < 0 {
			right = len(repos)
		}
		repos = append(repos, repo)
	}
	if right >= 0 {
		active = right
	} else if left >= 0 {
		active = left
	}
	return repos, active, skipped
}
