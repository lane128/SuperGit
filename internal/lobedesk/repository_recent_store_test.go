package lobedesk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestRecentRepositoriesSurviveClosingAllTabsAndRestarting(t *testing.T) {
	_, first, _, _, _ := repositoryTabsSafetyFixture(t)
	_, _, other := worktreeApp(t)
	path := filepath.Join(t.TempDir(), "repository-session.json")
	tabs, callbacks := startRepositorySession(t, first, path, "")
	for _, root := range []string{first.repo.Root, other} {
		tabs.active.openRepository(root, true)
		for tabs.opening != nil || tabs.active.loading {
			runContinuation(t, callbacks)
		}
		if tabs.active.repo == nil || tabs.active.repo.Root != root {
			t.Fatalf("failed to open fixture %q", root)
		}
	}
	for tabs.active.repo != nil {
		tabs.close(tabs.active)
		for tabs.active.loading {
			runContinuation(t, callbacks)
		}
	}
	tabs.saveSession()
	if len(readRepositorySession(path).Roots) != 0 {
		t.Fatal("closed checkouts were saved as open")
	}
	resumed, callbacks := startRepositorySession(t, first, path, "")
	if resumed.active.repo != nil || len(resumed.sessions) != 1 {
		t.Fatal("recent checkouts reopened as tabs without being selected")
	}
	if !slices.Equal(resumed.active.recentRepositories, []string{first.repo.Root, other}) {
		t.Fatalf("wrong recent ordering after restart: %v", resumed.active.recentRepositories)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		resumed.active.welcomeRecentRepositories(c)
	}, 980, 640)
	tt.Frame()
	entry, ok := tt.AccessByRole(ui.RoleButton, filepath.Base(other))
	if !ok || entry.Description != other {
		t.Fatal("Welcome does not offer the stored checkout")
	}
	tt.ClickAt(entry.Bounds.X+entry.Bounds.W/2, entry.Bounds.Y+entry.Bounds.H/2)
	for resumed.active.loading {
		runContinuation(t, callbacks)
	}
	if resumed.active.repo == nil || resumed.active.repo.Root != other || len(resumed.sessions) != 1 {
		t.Fatal("the selected recent checkout did not fill Welcome")
	}
}

func TestRecentRepositoriesPersistOnReopeningRestoredCheckout(t *testing.T) {
	_, first, _, _, _ := repositoryTabsSafetyFixture(t)
	root := first.repo.Root
	path := filepath.Join(t.TempDir(), "repository-session.json")
	if err := writeRepositorySession(path, repositorySession{Roots: []string{root}, Active: root}); err != nil {
		t.Fatal(err)
	}
	tabs, callbacks := startRepositorySession(t, first, path, "")
	store := filepath.Join(filepath.Dir(path), "repository-recents.json")
	if _, err := os.Stat(store); !os.IsNotExist(err) {
		t.Fatalf("passive restore must not write recents: %v", err)
	}
	// Simulate an unwritable record without relying on platform permissions.
	if err := os.Mkdir(store, 0700); err != nil {
		t.Fatal(err)
	}
	for _, fail := range []bool{true, false} {
		if !fail {
			if err := os.Remove(store); err != nil {
				t.Fatal(err)
			}
		}
		tabs.close(tabs.active)
		tabs.active.openRepository(root, true)
		for tabs.active.loading {
			runContinuation(t, callbacks)
		}
		if tabs.active.repo == nil || tabs.active.repo.Root != root {
			t.Fatal("recent file failure prevented a successful repository open")
		}
	}
	data, err := os.ReadFile(store)
	if err != nil {
		t.Fatalf("same-root open did not retry persistence: %v", err)
	}
	var saved struct{ Roots []string }
	if err := json.Unmarshal(data, &saved); err != nil || !slices.Equal(saved.Roots, []string{root}) {
		t.Fatalf("wrong saved roots: %v (%v)", saved.Roots, err)
	}
}

func TestRecentRepositoriesLoadsLegacyPathsWithoutReopeningTabs(t *testing.T) {
	_, first, _, _, _ := repositoryTabsSafetyFixture(t)
	profile := t.TempDir()
	legacy := filepath.Join(profile, "SuperGit", "preferences.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0700); err != nil {
		t.Fatal(err)
	}
	original, err := json.Marshal(map[string]any{"Dark": true, "FontSize": 15, "Recent": []string{first.repo.Root, "relative", first.repo.Root}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, original, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(profile, "supergit", "repository-session.json")
	tabs, callbacks := startRepositorySession(t, first, path, "")
	if tabs.active.repo != nil || !slices.Equal(tabs.active.recentRepositories, []string{first.repo.Root}) {
		t.Fatal("legacy path was not imported independently of the open tabs")
	}
	tabs.active.openRecentRepository(first.repo.Root)
	for tabs.active.loading {
		runContinuation(t, callbacks)
	}
	if tabs.active.repo == nil || tabs.active.repo.Root != first.repo.Root {
		t.Fatal("the legacy checkout did not open from recents")
	}
	if data, err := os.ReadFile(legacy); err != nil || string(data) != string(original) {
		t.Fatalf("legacy preferences changed: %v", err)
	}
	store := filepath.Join(filepath.Dir(path), "repository-recents.json")
	if err := os.WriteFile(store, []byte(`{"roots":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	resumed, _ := startRepositorySession(t, first, path, "")
	if len(resumed.active.recentRepositories) != 0 {
		t.Fatal("newer Lobe recent list must take precedence even when empty")
	}
}

func TestRecentRepositoriesDoesNotOverwriteAnotherWritersList(t *testing.T) {
	store := filepath.Join(t.TempDir(), "repository-recents.json")
	first := filepath.Join(filepath.Dir(store), "one")
	second := filepath.Join(filepath.Dir(store), "two")
	if _, err := updateStoredRecentRepositories(store, first, nil); err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(store+".lock", os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := lockRepositoryRecents(lock); err != nil {
		t.Fatal(err)
	}
	if _, err := updateStoredRecentRepositories(store, second, nil); err == nil {
		t.Fatal("another instance's update was allowed into the locked list")
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	roots, err := updateStoredRecentRepositories(store, second, nil)
	if err != nil || !slices.Equal(roots, []string{second, first}) {
		t.Fatalf("retry should preserve existing roots: %v (%v)", roots, err)
	}
}

func TestRecentRepositoriesExplicitLaunchLeavesNormalRecentsUntouched(t *testing.T) {
	_, first, _, _, _ := repositoryTabsSafetyFixture(t)
	path := filepath.Join(t.TempDir(), "repository-session.json")
	_, _, explicit := worktreeApp(t)
	store := filepath.Join(filepath.Dir(path), "repository-recents.json")
	original, err := json.Marshal(map[string][]string{"roots": {first.repo.Root}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store, original, 0600); err != nil {
		t.Fatal(err)
	}
	reopened, _ := startRepositorySession(t, first, path, explicit)
	if reopened.active.repo == nil || reopened.active.repo.Root != explicit {
		t.Fatal("one-off checkout was not opened")
	}
	if got, err := os.ReadFile(store); err != nil || string(got) != string(original) {
		t.Fatalf("one-off startup changed the normal recent list: %v", err)
	}
}
