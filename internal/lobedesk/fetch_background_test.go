package lobedesk

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// A fetch keeps busy set so writes stay gated, but it must not keep the reader
// out of browsing paths.
func TestExclusiveBusyOpensReadsDuringFetch(t *testing.T) {
	a := headShortcutApp(t)
	a.remotes = []gitrepo.Remote{{Name: "origin"}}
	a.busy, a.fetching = true, true
	if a.exclusiveBusy() {
		t.Fatal("a fetch is not an exclusive operation")
	}
	if !a.canOpenCommandPalette() {
		t.Fatal("command palette must open during fetch")
	}
	if why := a.commandPaletteUnavailable("history.head"); why != "" {
		t.Fatalf("navigation command blocked during fetch: %q", why)
	}
	if why := a.commandPaletteUnavailable("repo.push"); why != a.t("commandPaletteWait") {
		t.Fatalf("write command reason=%q; want wait", why)
	}
	if !a.canFocusCommitMessage() {
		t.Fatal("commit message should stay focusable while drafting")
	}
	if a.canRefreshShortcut() {
		t.Fatal("manual refresh must wait for fetch (reload is still busy-gated)")
	}
	if a.performDoneAs("", "", "", nil, func(context.Context, *gitrepo.Repository) error { return nil }) {
		t.Fatal("write operation started during fetch")
	}

	a.fetching = false
	if !a.exclusiveBusy() {
		t.Fatal("an ordinary operation is exclusive")
	}
	if a.canOpenCommandPalette() {
		t.Fatal("command palette must wait for an ordinary operation")
	}
}

func TestFetchUnlocksHistoricalFileSelection(t *testing.T) {
	a, tt := historicalSelectionFixture(t)
	clickHistoricalFile(t, tt, "a.txt", 0)
	a.busy, a.fetching = true, true
	origin, file := a.historicalFileOrigin(), a.commitFiles[2]
	a.selectHistoricalFile(origin, file, ui.Cmd, false)
	if !a.historicalFileSelected("c.txt") || a.commitPath != "c.txt" {
		t.Fatalf("commit file selection blocked during fetch: selected=%v path=%q", a.historicalFileSelected("c.txt"), a.commitPath)
	}
}

func TestFetchUnlocksWorkspaceFileSelection(t *testing.T) {
	a, _ := overviewFixture(t)
	a.page = "history"
	a.worktreeSelected = true
	a.busy, a.fetching = true, true
	path, staged := firstWorkspaceEntry(a.status.Entries)
	if path == "" {
		t.Fatal("fixture has no workspace file")
	}
	a.selectWorkspaceFile(fileRef{path: path, staged: staged}, 0, false)
	if a.selectedPath != path || a.selectedStaged != staged {
		t.Fatalf("workspace selection blocked during fetch: %q/%v", a.selectedPath, a.selectedStaged)
	}
}

func TestFetchKeepsReadsOpenUntilCompletion(t *testing.T) {
	a, done, root := worktreeApp(t)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	bare := filepath.Join(t.TempDir(), "remote.git")
	pullFixtureGit(t, root, "clone", "--bare", root, bare)
	pullFixtureGit(t, root, "remote", "add", "origin", bare)
	var err error
	a.remotes, err = a.repo.Remotes(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.fetchAll()
	if !a.busy || !a.fetching {
		t.Fatalf("fetch did not start: busy=%v fetching=%v", a.busy, a.fetching)
	}
	if a.exclusiveBusy() || !a.canOpenCommandPalette() {
		t.Fatal("fetch closed browsing paths")
	}
	if why := a.commandPaletteUnavailable("repo.terminal"); why != "" {
		t.Fatalf("navigation blocked during real fetch: %q", why)
	}
	if why := a.commandPaletteUnavailable("repo.push"); why != a.t("commandPaletteWait") {
		t.Fatalf("write command reason=%q; want wait", why)
	}
	for a.busy || a.fetching || a.loading {
		runContinuation(t, done)
	}
	if a.busy || a.fetching {
		t.Fatalf("fetch left flags set: busy=%v fetching=%v", a.busy, a.fetching)
	}
}

func TestFetchCompletionOnStoppedSessionSkipsReloadAndToast(t *testing.T) {
	a, done, root := worktreeApp(t)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	bare := filepath.Join(t.TempDir(), "remote.git")
	pullFixtureGit(t, root, "clone", "--bare", root, bare)
	pullFixtureGit(t, root, "remote", "add", "origin", bare)
	var err error
	a.remotes, err = a.repo.Remotes(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.fetchAll()
	a.stopRepositorySession()
	stoppedAt := a.loadSeq
	for a.busy || a.fetching {
		runContinuation(t, done)
	}
	if a.busy || a.fetching {
		t.Fatalf("stopped session kept flags: busy=%v fetching=%v", a.busy, a.fetching)
	}
	if len(a.notices[toastMain]) != 0 {
		t.Fatalf("stopped session queued notices: %+v", a.notices[toastMain])
	}
	if a.loadSeq != stoppedAt || a.loading {
		t.Fatalf("stopped session reloaded: loadSeq=%d want=%d loading=%v", a.loadSeq, stoppedAt, a.loading)
	}
}

func TestFetchCompletionKeepsCommandPaletteOpen(t *testing.T) {
	a, done, root := worktreeApp(t)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	bare := filepath.Join(t.TempDir(), "remote.git")
	pullFixtureGit(t, root, "clone", "--bare", root, bare)
	pullFixtureGit(t, root, "remote", "add", "origin", bare)
	var err error
	a.remotes, err = a.repo.Remotes(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	before := a.loadSeq
	a.fetchAll()
	a.openCommandPalette()
	a.commandPalette.query = "repo.terminal"
	for a.busy || a.fetching || a.loading {
		runContinuation(t, done)
	}
	a.discardStaleCommandPalette()
	if !a.commandPalette.open || a.commandPalette.query != "repo.terminal" {
		t.Fatal("fetch completion discarded the command palette input")
	}
	if a.loadSeq == before {
		t.Fatal("fetch completion did not refresh the repository")
	}
}

func TestFetchCompletionKeepsHistoryRefPickerOpen(t *testing.T) {
	a, done, root := worktreeApp(t)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	bare := filepath.Join(t.TempDir(), "remote.git")
	pullFixtureGit(t, root, "clone", "--bare", root, bare)
	pullFixtureGit(t, root, "remote", "add", "origin", bare)
	var err error
	a.remotes, err = a.repo.Remotes(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	before := a.loadSeq
	a.historyRefPalette = historyRefPaletteState{active: true, root: a.paletteRoot()}
	a.historyRefOpen, a.historyRefQuery = true, "mai"
	a.fetchAll()
	for a.busy || a.fetching || a.loading {
		runContinuation(t, done)
	}
	a.discardStaleHistoryRefPalette()
	if !a.historyRefOpen || a.historyRefQuery != "mai" {
		t.Fatal("fetch completion closed the history ref picker")
	}
	if a.loadSeq == before {
		t.Fatal("fetch completion did not refresh the repository")
	}
}

func TestFetchCompletionClearsAmendWhenHEADMoves(t *testing.T) {
	a, done, root := worktreeApp(t)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	bare := filepath.Join(t.TempDir(), "remote.git")
	pullFixtureGit(t, root, "clone", "--bare", root, bare)
	pullFixtureGit(t, root, "remote", "add", "origin", bare)
	old := pullFixtureGit(t, root, "rev-parse", "HEAD")
	a.status.Head, a.status.Branch = old, pullFixtureGit(t, root, "symbolic-ref", "--short", "HEAD")
	a.amend, a.commitText = true, "draft"
	pullFixtureGit(t, root, "commit", "--allow-empty", "-m", "external")
	var err error
	a.remotes, err = a.repo.Remotes(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.fetchAll()
	for a.busy || a.fetching || a.loading {
		runContinuation(t, done)
	}
	if a.amend {
		t.Fatal("fetch reload kept Amend armed after HEAD moved")
	}
	if a.commitText != "draft" {
		t.Fatalf("fetch reload discarded the draft: %q", a.commitText)
	}
	if a.status.Head == old {
		t.Fatal("fetch reload did not adopt the new HEAD")
	}
}

func TestFetchAllowsTabSwitchButKeepsFetchingTabUnclosable(t *testing.T) {
	tabs, a, b, tt, callbacks := repositoryTabsSafetyFixture(t)
	b.busy, b.fetching = true, true
	b.progressSince = time.Now()
	tt.Frame()
	if !tabs.canSwitch() {
		t.Fatal("cannot switch away from a fetching tab")
	}
	if tabs.canClose(b) {
		t.Fatal("closing the active fetching tab must stay blocked")
	}
	if !tabs.canClose(a) {
		t.Fatal("idle background tab must remain closable")
	}
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(a.repo.Root))
	for tabs.active.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	if tabs.active != a {
		t.Fatal("tab switch during fetch did not activate the other checkout")
	}
	if !tabs.canChange() {
		t.Fatal("structural tab changes should work once the active tab is idle")
	}
	if tabs.canClose(b) {
		t.Fatal("a non-active fetching tab must stay unclosable")
	}
	// Returning to the fetching tab must not leave its activation refresh
	// waiting on a reload that busy refuses to start.
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(b.repo.Root))
	tt.Frame()
	if tabs.active != b || b.loading || !b.canOpenCommandPalette() {
		t.Fatalf("returning to a fetching tab blocked browsing: active=%v loading=%v", tabs.active == b, b.loading)
	}
	if !b.activationRefresh {
		t.Fatal("returning during fetch must still re-check HEAD when the fetch reload runs")
	}
	b.busy, b.fetching = false, false
	b.activationRefresh = false
}

func TestFetchRefRefreshAllowsSwitchAndFinishesOnInactiveTab(t *testing.T) {
	tabs, other, origin, view, callbacks := repositoryTabsSafetyFixture(t)
	origin.refreshRefs()
	view.Frame()
	if !origin.loading || tabs.canChange() || tabs.canClose(origin) {
		t.Fatal("reference reconciliation must still block structural changes")
	}
	clickRepositoryTabsControl(t, view, ui.RoleTab, filepath.Base(other.repo.Root))
	if tabs.active != other {
		t.Fatal("reference reconciliation blocked switching checkouts")
	}
	for origin.loading || other.loading {
		runContinuation(t, callbacks)
	}
	if tabs.active != other || origin.loadErr != "" {
		t.Fatal("background completion disrupted active checkout")
	}
}

func TestSwitchBackWhileDestinationIsStillReadingDuringFetch(t *testing.T) {
	for _, restored := range []bool{false, true} {
		tabs, destination, fetching, view, callbacks := repositoryTabsSafetyFixture(t)
		destination.restoredCheckout = restored
		fetching.busy, fetching.fetching = true, true
		view.Frame()
		clickRepositoryTabsControl(t, view, ui.RoleTab, filepath.Base(destination.repo.Root))
		if tabs.active != destination || !destination.loading || tabs.canChange() {
			t.Fatal("destination should be reading with structural changes blocked")
		}
		clickRepositoryTabsControl(t, view, ui.RoleTab, filepath.Base(fetching.repo.Root))
		for destination.loading {
			runContinuation(t, callbacks)
		}
		if tabs.active != fetching || !fetching.fetching || tabs.canClose(fetching) {
			t.Fatal("background read disrupted the active Fetch")
		}
	}
}

func TestFetchPruneAndTagsCommandsUseTheirMode(t *testing.T) {
	a, done, root := worktreeApp(t)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	bare := filepath.Join(t.TempDir(), "remote.git")
	pullFixtureGit(t, root, "clone", "--bare", root, bare)
	pullFixtureGit(t, root, "remote", "add", "origin", bare)
	pullFixtureGit(t, root, "config", "remote.origin.prune", "false")
	head := pullFixtureGit(t, root, "rev-parse", "HEAD")
	tree := pullFixtureGit(t, root, "rev-parse", head+"^{tree}")
	pullFixtureGit(t, bare, "config", "user.name", "Fetch Fixture")
	pullFixtureGit(t, bare, "config", "user.email", "fetch@example.invalid")
	pullFixtureGit(t, bare, "update-ref", "refs/heads/server-branch", head)
	orphan := pullFixtureGit(t, bare, "commit-tree", tree, "-m", "tag target")
	pullFixtureGit(t, bare, "tag", "server-tag", orphan)
	pullFixtureGit(t, root, "update-ref", "refs/remotes/origin/obsolete", head)
	var err error
	a.remotes, err = a.repo.Remotes(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	refs := func() string { return pullFixtureGit(t, root, "for-each-ref") }

	a.fetchPruneAll()
	for a.busy || a.fetching || a.loading {
		runContinuation(t, done)
	}
	got := refs()
	if strings.Contains(got, "refs/remotes/origin/obsolete") {
		t.Fatal("prune fetch kept a stale remote-tracking ref")
	}
	if !strings.Contains(got, "refs/remotes/origin/server-branch") {
		t.Fatal("prune fetch missed the server branch")
	}
	if strings.Contains(got, "refs/tags/server-tag") {
		t.Fatal("prune fetch pulled tags")
	}

	a.fetchTagsAll()
	for a.busy || a.fetching || a.loading {
		runContinuation(t, done)
	}
	if !strings.Contains(refs(), "refs/tags/server-tag") {
		t.Fatal("tags fetch missed the remote tag")
	}
}

func TestFetchAllowsTabReorder(t *testing.T) {
	tabs, _, b, _, _ := repositoryTabsSafetyFixture(t)
	b.busy, b.fetching = true, true
	if !tabs.moveTab(b, tabs.sessions[0]) {
		t.Fatal("cannot reorder tabs while one fetches")
	}
	b.busy, b.fetching = false, false
}

func TestPaletteFetchAllStartsAfterChoosingMode(t *testing.T) {
	a, done, root := worktreeApp(t)
	t.Setenv("GIT_ALLOW_PROTOCOL", "file")
	bare := filepath.Join(t.TempDir(), "remote.git")
	pullFixtureGit(t, root, "clone", "--bare", root, bare)
	pullFixtureGit(t, root, "remote", "add", "origin", bare)
	var err error
	a.remotes, err = a.repo.Remotes(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	entry := paletteFetchEntry(t, a)
	a.openCommandPalette()
	a.runPaletteCommand(entry.ID, a.commandPalette.generation)
	if !a.commandPalette.open || a.commandPalette.stage != paletteFetchOptions || a.busy || a.fetching || !a.lastAutoFetch.IsZero() {
		t.Fatal("opening the Fetch mode picker should leave Git idle")
	}
	options := a.paletteChoices()
	if len(options) == 0 || options[0].id != entry.ID {
		t.Fatalf("missing the default Fetch option: %v", options)
	}
	a.acceptPaletteChoice(options[0], a.commandPalette.generation)
	if a.commandPalette.open || !a.busy || !a.fetching || a.lastAutoFetch.IsZero() {
		t.Fatal("choosing the default mode should close the picker and start Fetch")
	}
	for a.busy || a.fetching || a.loading {
		runContinuation(t, done)
	}
}

func TestPaletteFetchCommandsRegistered(t *testing.T) {
	a := headShortcutApp(t)
	entries := map[string]appShortcut{}
	for _, entry := range commandPaletteTable() {
		entries[entry.ID] = entry
	}
	ids := []string{"repo.fetch", "repo.fetchPrune", "repo.fetchTags", "repo.fetchRemote"}
	for _, id := range ids {
		entry, ok := entries[id]
		if !ok || entry.Run == nil || a.t(entry.Label) == "" {
			t.Fatalf("missing palette command %q: %+v", id, entry)
		}
		if !paletteCommand(id) {
			t.Fatalf("%q is not a palette command", id)
		}
	}
	a.remotes = nil
	for _, id := range ids {
		if why := a.commandPaletteUnavailable(id); why != a.t("noRemote") {
			t.Fatalf("%s without remotes: %q", id, why)
		}
	}
}

func TestPaletteFetchRemoteCommandOpensRemotePicker(t *testing.T) {
	a := headShortcutApp(t)
	a.remotes = []gitrepo.Remote{{Name: "origin"}, {Name: "upstream"}}
	a.openCommandPalette()
	a.runPaletteCommand("repo.fetchRemote", a.commandPalette.generation)
	if !a.commandPalette.open || a.commandPalette.stage != paletteFetchRemote {
		t.Fatalf("remote command did not open the picker: %+v", a.commandPalette)
	}
	rows := a.paletteChoices()
	if len(rows) != 2 || rows[0].label != "origin" || rows[1].label != "upstream" {
		t.Fatalf("remote picker rows: %+v", rows)
	}
}

func TestFetchKeepsWriteCommandsWaiting(t *testing.T) {
	a := headShortcutApp(t)
	a.remotes = []gitrepo.Remote{{Name: "origin"}}
	a.busy, a.fetching = true, true
	for _, id := range []string{"repo.push", "repo.pull", "branch.create", "stash.saveTracked", "workspace.clean", "repo.open", "repo.reload", "app.checkForUpdates"} {
		if why := a.commandPaletteUnavailable(id); why != a.t("commandPaletteWait") {
			t.Fatalf("%s during fetch: %q; want wait", id, why)
		}
	}
	for _, id := range []string{"history.head", "history.chooseRef", "commit.copySHA", "file.blame", "diff.ignoreWhitespace", "view.toggleSidebar", "window.about", "workspace.focusCommitMessage", "repo.terminal"} {
		if why := a.commandPaletteUnavailable(id); why != "" {
			t.Fatalf("%s during fetch: %q; want available", id, why)
		}
	}
}

func TestCloseInactiveTabWithoutActivatingWhileFetching(t *testing.T) {
	tabs, target, active, view, _ := repositoryTabsSafetyFixture(t)
	active.busy, active.fetching = true, true
	target.commitText = "unsent draft"
	view.Frame()
	clickRepositoryTabsControl(t, view, ui.RoleButton, active.t("close")+" "+target.repo.Root)
	if tabs.active != active || !active.confirm.open {
		t.Fatal("close should confirm the other tab's draft without selecting it")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, active.t("close"))
	if len(tabs.sessions) != 1 || tabs.active != active || !active.fetching {
		t.Fatal("closing inactive tab disrupted active fetch")
	}
}
func TestTabRevalidationDoesNotAdvertiseSync(t *testing.T) {
	_, target, _, view, callbacks := repositoryTabsSafetyFixture(t)
	clickRepositoryTabsControl(t, view, ui.RoleTab, filepath.Base(target.repo.Root))
	target.loadSince = time.Now().Add(-time.Minute)
	if !target.loading || target.progressText() != "" {
		t.Fatal("local tab validation should be silent")
	}
	for target.loading {
		runContinuation(t, callbacks)
	}
}
