package lobedesk

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRepositorySessionRestoresOrderedCheckouts(t *testing.T) {
	tabs, a, b, _, _ := repositoryTabsSafetyFixture(t)
	path := filepath.Join(t.TempDir(), "repository-session.json")
	if err := writeRepositorySession(path, tabs.repositorySession()); err != nil {
		t.Fatal(err)
	}
	storage := readRepositorySession(path)
	if !slices.Equal(storage.Roots, []string{a.repo.Root, b.repo.Root}) || storage.Active != b.repo.Root {
		t.Fatalf("wrong saved session: %+v", storage)
	}

	// A saved nested root that no longer refers to a checkout cannot turn into
	// another tab for its enclosing repository, even if it still exists.
	nested := filepath.Join(a.repo.Root, "former-checkout")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	storage.Roots = []string{a.repo.Root, nested, b.repo.Root, a.repo.Root}
	storage.Active = nested
	if err := writeRepositorySession(path, storage); err != nil {
		t.Fatal(err)
	}
	reopened, callbacks := startRepositorySession(t, a, path, "")
	got := reopened.repositorySession()
	if want := []string{a.repo.Root, b.repo.Root}; !slices.Equal(got.Roots, want) || got.Active != b.repo.Root {
		t.Fatalf("restored %+v, want ordered roots %v and active %q", got, want, b.repo.Root)
	}
	// A restored checkout opens on its newest commit, the same as a freshly
	// opened repository.
	if reopened.active.commitText != "" || reopened.active.loadErr != "" || reopened.active.selectedSHA == "" {
		t.Fatal("active checkout was not loaded into a clean session")
	}
	reopened.saveSession()
	if saved := readRepositorySession(path); !slices.Equal(saved.Roots, storage.Roots) || saved.Active != storage.Active {
		t.Fatal("passive restore overwrote a temporarily unavailable checkout")
	}

	reopened.activate(reopened.sessions[0], false)
	for reopened.active.loading {
		runContinuation(t, callbacks)
	}
	if reopened.active.status.Head == "" || reopened.active.selectedSHA == "" {
		t.Fatal("restored inactive checkout was not loaded on selection")
	}
	reopened.saveSession()
	if saved := readRepositorySession(path); !slices.Equal(saved.Roots, storage.Roots) || saved.Active != a.repo.Root {
		t.Fatalf("selection should preserve other entries: %+v", saved)
	}
	reopened.close(reopened.sessions[1])
	reopened.close(reopened.active)
	reopened.saveSession()
	if saved := readRepositorySession(path); len(saved.Roots) != 0 || saved.Active != "" {
		t.Fatalf("explicitly closing the last tab did not save an empty workspace: %+v", saved)
	}
}

func TestRepositorySessionRestoresIndependentCommitDrafts(t *testing.T) {
	tabs, a, b, _, _ := repositoryTabsSafetyFixture(t)
	path := filepath.Join(t.TempDir(), "repository-session.json")
	if err := writeRepositorySession(path, tabs.repositorySession()); err != nil {
		t.Fatal(err)
	}
	first, callbacks := startRepositorySession(t, a, path, "")
	betaDraft := "Beta title\n\nBeta body\n"
	first.active.commitText = betaDraft
	first.activate(first.sessions[0], false)
	for first.active.loading {
		runContinuation(t, callbacks)
	}
	alphaDraft := "Alpha title\n\n中文正文\n\n"
	first.active.commitText = alphaDraft
	first.saveSession()

	second, callbacks := startRepositorySession(t, a, path, "")
	if second.active.repo.Root != a.repo.Root || second.active.commitText != alphaDraft || second.active.amend {
		t.Fatal("the active checkout did not restore its exact ordinary draft")
	}
	second.activate(second.sessions[1], false)
	for second.active.loading {
		runContinuation(t, callbacks)
	}
	if second.active.repo.Root != b.repo.Root || second.active.commitText != betaDraft {
		t.Fatal("the inactive checkout lost its own draft during lazy restoration")
	}
	// Clear one editor and explicitly discard the other tab. No saved draft
	// should resurface when the app reopens either checkout.
	second.active.commitText = ""
	second.close(second.sessions[0])
	second.saveSession()
	third, _ := startRepositorySession(t, a, path, "")
	if len(third.sessions) != 1 || third.active.repo.Root != b.repo.Root || third.active.commitText != "" {
		t.Fatal("closing or clearing a draft did not update the saved session")
	}
}

func TestRepositorySessionKeepsOrdinaryDraftSeparateFromMerge(t *testing.T) {
	a, _, root := worktreeApp(t)
	status, err := a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	const ordinary = "Ordinary checkout draft\n\nKeep after merge"
	const mergeMessage = "Git merge fixture"
	path := filepath.Join(t.TempDir(), "repository-session.json")
	if err := writeRepositorySession(path, repositorySession{Roots: []string{root}, Active: root, Drafts: map[string]string{root: ordinary}}); err != nil {
		t.Fatal(err)
	}
	mergeHead := filepath.Join(root, ".git", "MERGE_HEAD")
	if err := os.WriteFile(mergeHead, []byte(status.Head+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "MERGE_MSG"), []byte(mergeMessage), 0o600); err != nil {
		t.Fatal(err)
	}
	merging, _ := startRepositorySession(t, a, path, "")
	if merging.active.operation.Kind != "merge" || merging.active.commitText != mergeMessage {
		t.Fatal("the ordinary saved draft displaced the message from Git")
	}
	merging.active.commitText = "User-edited merge message"
	merging.saveSession()
	if saved := readRepositorySession(path); saved.Drafts[root] != ordinary {
		t.Fatal("the merge editor overwrote the ordinary draft")
	}
	if err := os.Remove(mergeHead); err != nil {
		t.Fatal(err)
	}
	restored, _ := startRepositorySession(t, a, path, "")
	if restored.active.commitText != ordinary {
		t.Fatal("the ordinary draft did not return after the merge stopped")
	}
}

func TestRepositorySessionPreservesNewerRepositoryAndDraft(t *testing.T) {
	tabs, a, b, _, _ := repositoryTabsSafetyFixture(t)
	path := filepath.Join(t.TempDir(), "repository-session.json")
	roots := []string{a.repo.Root, b.repo.Root}
	stale := repositorySession{Roots: roots[:1], Active: roots[0]}
	if err := writeRepositorySession(path, stale); err != nil {
		t.Fatal(err)
	}
	_, _, addedRoot := worktreeApp(t)
	const draft = "Newer workspace draft"
	newer := repositorySession{Roots: append(slices.Clone(stale.Roots), addedRoot), Active: addedRoot, Drafts: map[string]string{addedRoot: draft}}
	if err := writeRepositorySession(path, newer); err != nil {
		t.Fatal(err)
	}
	tabs.sessionState = repositorySessionState{path: path, rootsBaseline: stale.Roots, tabsChanged: true}
	tabs.saveSession()
	if saved := readRepositorySession(path); !slices.Equal(saved.Roots, append(roots, addedRoot)) || saved.Drafts[addedRoot] != draft {
		t.Fatal("saving local tabs erased a repository and draft saved by another instance")
	}
}

func TestRepositorySessionExplicitPathPreservesSavedWorkspace(t *testing.T) {
	tabs, a, _, _, _ := repositoryTabsSafetyFixture(t)
	path := filepath.Join(t.TempDir(), "repository-session.json")
	storage := tabs.repositorySession()
	if err := writeRepositorySession(path, storage); err != nil {
		t.Fatal(err)
	}
	_, _, explicitRoot := worktreeApp(t)
	reopened, _ := startRepositorySession(t, a, path, explicitRoot)
	if got := reopened.repositorySession(); !slices.Equal(got.Roots, []string{explicitRoot}) || got.Active != explicitRoot {
		t.Fatalf("explicit launch opened %+v", got)
	}
	reopened.saveSession()
	if saved := readRepositorySession(path); !slices.Equal(saved.Roots, storage.Roots) || saved.Active != storage.Active {
		t.Fatal("explicit launch replaced the saved workspace")
	}
}

func TestRepositorySessionSelectionPreservesNewerTabs(t *testing.T) {
	tabs, a, _, _, _ := repositoryTabsSafetyFixture(t)
	_, _, addedRoot := worktreeApp(t)
	path := filepath.Join(t.TempDir(), "repository-session.json")
	storage := tabs.repositorySession()
	if err := writeRepositorySession(path, storage); err != nil {
		t.Fatal(err)
	}
	reopened, callbacks := startRepositorySession(t, a, path, "")
	storage.Roots = append(storage.Roots, addedRoot)
	if err := writeRepositorySession(path, storage); err != nil {
		t.Fatal(err)
	}
	reopened.activate(reopened.sessions[0], false)
	for reopened.active.loading {
		runContinuation(t, callbacks)
	}
	reopened.saveSession()
	if saved := readRepositorySession(path); !slices.Equal(saved.Roots, storage.Roots) || saved.Active != a.repo.Root {
		t.Fatalf("saving selection removed a checkout added by another process: %+v", saved)
	}
}

func TestRepositorySessionSkipsUnreadableActiveCheckout(t *testing.T) {
	tabs, a, b, _, _ := repositoryTabsSafetyFixture(t)
	_, _, rightRoot := worktreeApp(t)
	path := filepath.Join(t.TempDir(), "repository-session.json")
	storage := tabs.repositorySession()
	storage.Roots = append(storage.Roots, rightRoot)
	if err := writeRepositorySession(path, storage); err != nil {
		t.Fatal(err)
	}
	// Git still resolves the exact root, but cannot read working-tree status.
	if err := os.WriteFile(filepath.Join(b.repo.Root, ".git", "index"), []byte("invalid index"), 0o600); err != nil {
		t.Fatal(err)
	}
	reopened, _ := startRepositorySession(t, a, path, "")
	if got := reopened.repositorySession(); !slices.Equal(got.Roots, []string{a.repo.Root, rightRoot}) || got.Active != rightRoot {
		t.Fatalf("unreadable active checkout blocked a usable one: %+v", got)
	}
	reopened.saveSession()
	if saved := readRepositorySession(path); !slices.Equal(saved.Roots, storage.Roots) || saved.Active != storage.Active {
		t.Fatal("passive restore removed an unreadable checkout from the saved workspace")
	}
}

func TestRepositorySessionDoesNotPersistUnreadableInactiveCheckoutAsActive(t *testing.T) {
	_, first, second, _, _ := repositoryTabsSafetyFixture(t)
	firstRoot, secondRoot := first.repo.Root, second.repo.Root
	path := filepath.Join(t.TempDir(), "repository-session.json")
	roots := []string{firstRoot, secondRoot}
	if err := writeRepositorySession(path, repositorySession{Roots: roots, Active: firstRoot}); err != nil {
		t.Fatal(err)
	}
	if _, err := updateStoredRecentRepositories(recentRepositoriesPath(path), firstRoot, nil); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(secondRoot, ".git", "index")
	original, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(index, []byte("invalid index"), 0o600); err != nil {
		t.Fatal(err)
	}
	tabs, callbacks := startRepositorySession(t, first, path, "")
	target := tabs.findCheckout(secondRoot)
	if target == nil {
		t.Fatal("fixture checkout should be available for lazy loading")
	}
	tabs.activate(target, false)
	for tabs.active.loading {
		runContinuation(t, callbacks)
	}
	if tabs.active != target || target.loadErr == "" {
		t.Fatal("failed checkout should show the load error and allow retry")
	}
	tabs.saveSession()
	saved := readRepositorySession(path)
	recents, err := readRecentRepositories(recentRepositoriesPath(path))
	if saved.Active != firstRoot || !slices.Equal(saved.Roots, roots) || err != nil || !slices.Equal(recents, []string{firstRoot}) {
		t.Fatalf("failed snapshot must preserve the last verified checkout: session=%+v recents=%v error=%v", saved, recents, err)
	}
	if err := os.WriteFile(index, original, 0o600); err != nil {
		t.Fatal(err)
	}
	target.reload()
	for target.loading {
		runContinuation(t, callbacks)
	}
	tabs.saveSession()
	saved = readRepositorySession(path)
	recents, err = readRecentRepositories(recentRepositoriesPath(path))
	if target.loadErr != "" || saved.Active != secondRoot || err != nil || !slices.Equal(recents, []string{secondRoot, firstRoot}) {
		t.Fatalf("a successful retry must accept the restored checkout: session=%+v recents=%v error=%v", saved, recents, err)
	}
}

func startRepositorySession(t *testing.T, source *app, path, initialPath string) (*repositoryTabs, <-chan func()) {
	t.Helper()
	first := &app{ctx: source.ctx, page: "changes", locale: localeEnglish}
	callbacks := make(chan func(), 128)
	first.runOnMain = func(fn func()) { callbacks <- fn }
	tabs := newRepositoryTabs(source.ctx, first)
	t.Cleanup(tabs.shutdown)
	tabs.startSession(path, initialPath)
	for tabs.active.loading {
		runContinuation(t, callbacks)
	}
	return tabs, callbacks
}
