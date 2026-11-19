package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func repositoryTabsSafetyFixture(t *testing.T) (*repositoryTabs, *app, *app, *ui.Tester, <-chan func()) {
	t.Helper()
	isolateSettings(t)
	a, _, aRoot := worktreeApp(t)
	_, _, bRoot := worktreeApp(t)
	a.locale, a.page = localeEnglish, "changes"
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	tabs := newRepositoryTabs(a.ctx, a)
	t.Cleanup(tabs.shutdown)
	a.openRepository(aRoot, false)
	for a.loading {
		runContinuation(t, callbacks)
	}
	tt := ui.NewTester(tabs.view, 980, 760)
	tt.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
	tt.Frame()
	a.openRepository(bRoot, true)
	for tabs.opening != nil || tabs.active.loading {
		runContinuation(t, callbacks)
	}
	b := tabs.active
	tt.Frame()
	if b == a || b.loadErr != "" || a.loadErr != "" {
		t.Fatal("could not load both repositories")
	}
	return tabs, a, b, tt, callbacks
}

func clickRepositoryTabsControl(t *testing.T, tt *ui.Tester, role ui.Role, label string) {
	t.Helper()
	node, ok := tt.AccessByRole(role, label)
	if !ok || node.Disabled || node.Offscreen || node.Bounds.W <= 0 || node.Bounds.H <= 0 {
		t.Fatalf("unusable %q: %+v", label, node)
	}
	tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
	tt.Frame()
}

func TestRepositoryTabsReportUncommittedChangesOnInactiveCheckout(t *testing.T) {
	tabs, clean, dirty, view, callbacks := repositoryTabsSafetyFixture(t)
	clean.commitFields.Summary = "An unsent draft"
	path := filepath.Join(dirty.repo.Root, "untracked.txt")
	if err := os.WriteFile(path, []byte("fixture change\n"), 0600); err != nil {
		t.Fatal(err)
	}
	refresh := func() {
		t.Helper()
		dirty.reload()
		for dirty.loading {
			runContinuation(t, callbacks)
		}
		view.Frame()
	}
	status := func(session *app, want string) ui.AccessInfo {
		t.Helper()
		label := "Close " + session.repo.Root
		close, ok := view.AccessByRole(ui.RoleButton, label)
		if !ok || close.Offscreen || strings.TrimSpace(strings.TrimPrefix(close.Description, label)) != want {
			t.Fatalf("expected accessible checkout state %q, got %+v", want, close)
		}
		return close
	}
	refresh()
	clickRepositoryTabsControl(t, view, ui.RoleTab, filepath.Base(clean.repo.Root))
	for clean.loading {
		runContinuation(t, callbacks)
	}
	view.Frame()
	close := status(dirty, "Uncommitted changes")
	status(clean, "")
	view.Move(close.Bounds.X+close.Bounds.W/2, close.Bounds.Y+close.Bounds.H/2)
	view.Frame()
	if hovered := status(dirty, "Uncommitted changes"); hovered.Bounds != close.Bounds || hovered.Disabled || tabs.active != clean {
		t.Fatal("hovering changed the close hit target or selected a different checkout")
	}
	clickRepositoryTabsControl(t, view, ui.RoleTab, filepath.Base(dirty.repo.Root))
	for dirty.loading {
		runContinuation(t, callbacks)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	refresh()
	status(dirty, "")
}

func TestRepositoryTabsDismissOtherRepositoryUndo(t *testing.T) {
	tabs, a, b, tt, callbacks := repositoryTabsSafetyFixture(t)
	path := filepath.Join(b.repo.Root, "base.txt")
	if err := os.WriteFile(path, []byte("base\nrestore only in this repository\n"), 0644); err != nil {
		t.Fatal(err)
	}
	diff, err := b.repo.Diff(b.ctx, "base.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	var changed []int
	for i, line := range diff.Lines {
		if line.Kind == "add" {
			changed = append(changed, i)
		}
	}
	patch, err := b.repo.DiscardLines(b.ctx, "base.txt", diff, changed)
	if err != nil {
		t.Fatal(err)
	}
	// Use the production toast with a synchronous Undo callback so Tester can
	// observe whether the window dispatched it to the other checkout.
	var undoErr error
	b.pushNotice(toastMain, &toastNotice{kind: "success", title: b.t("discardLinesSuccess"), action: b.t("undo"), onAction: func() { undoErr = b.repo.UndoDiscard(b.ctx, "base.txt", patch) }, timeout: 8 * time.Second})
	tt.Frame()
	tt.Advance(400 * time.Millisecond)
	action, ok := tt.AccessByRole(ui.RoleButton, b.t("undo"))
	if !ok || action.Disabled || action.Offscreen {
		t.Fatalf("Undo was unavailable in the originating repository: %+v", action)
	}
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(a.repo.Root))
	for tabs.active.loading {
		runContinuation(t, callbacks)
	}
	tt.Advance(400 * time.Millisecond)
	_, remains := tt.AccessByRole(ui.RoleButton, b.t("undo"))
	if remains {
		clickRepositoryTabsControl(t, tt, ui.RoleButton, b.t("undo"))
	}
	content, err := os.ReadFile(path)
	if undoErr != nil || err != nil || string(content) != "base\n" || remains {
		t.Fatalf("after switching: old Undo visible=%v, inactive file=%q, undo error=%v, read error=%v", remains, content, undoErr, err)
	}
}

func TestRepositoryTabsFailedRefreshKeepsAmendDisabled(t *testing.T) {
	tabs, a, b, tt, callbacks := repositoryTabsSafetyFixture(t)
	root := b.repo.Root
	clickRepositoryTabsControl(t, tt, ui.RoleCheckBox, b.t("amendLast"))
	b.commitText = "Draft to keep on read failure"
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(a.repo.Root))
	for tabs.active.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	metadata := filepath.Join(root, ".git")
	if err := os.Rename(metadata, metadata+"-off"); err != nil {
		t.Fatal(err)
	}
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(root))
	for b.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	action, ok := tt.AccessByRole(ui.RoleButton, b.t("amend"))
	if !ok || !action.Disabled || b.commitText != "Draft to keep on read failure" {
		t.Fatalf("failed activation refresh leaves unsafe commit state: action=%+v draft=%q error=%q", action, b.commitText, b.loadErr)
	}
	if err := os.Rename(metadata+"-off", metadata); err != nil {
		t.Fatal(err)
	}
	clickRepositoryTabsControl(t, tt, ui.RoleButton, b.t("retry"))
	for b.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	action, ok = tt.AccessByRole(ui.RoleButton, b.t("amend"))
	if !ok || action.Disabled || b.commitText != "Draft to keep on read failure" {
		t.Fatalf("successful retry did not restore Amend on the same HEAD: action=%+v draft=%q error=%q", action, b.commitText, b.loadErr)
	}
}

func TestRepositoryTabsPreventPullIntoParentAfterFailedRefresh(t *testing.T) {
	isolateSettings(t)
	a, _, parent := worktreeApp(t)
	a.locale, a.page = localeEnglish, "changes"
	git := func(root string, args ...string) string { return pullFixtureGit(t, root, args...) }
	bare, nested := filepath.Join(t.TempDir(), "upstream.git"), filepath.Join(parent, "nested")
	git(parent, "clone", "--bare", parent, bare)
	git(parent, "remote", "add", "origin", bare)
	git(parent, "fetch", "origin")
	git(parent, "branch", "--set-upstream-to=origin/main")
	if err := os.WriteFile(filepath.Join(parent, ".git", "info", "exclude"), []byte("/nested/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(parent, "clone", bare, nested)
	git(nested, "config", "user.name", "Fixture")
	git(nested, "config", "user.email", "fixture@example.invalid")
	git(nested, "commit", "--allow-empty", "-m", "upstream advance")
	git(nested, "push", "origin", "main")
	parentHead := git(parent, "rev-parse", "HEAD")
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	tabs := newRepositoryTabs(a.ctx, a)
	t.Cleanup(tabs.shutdown)
	a.openRepository(nested, true)
	for tabs.opening != nil {
		runContinuation(t, callbacks)
	}
	b := tabs.active
	tt := ui.NewTester(tabs.view, 980, 760)
	tt.Frame()
	button, ok := tt.AccessByRole(ui.RoleButton, b.t("pull"))
	if b == a || !ok || button.Disabled {
		t.Fatal("nested checkout did not expose its configured Pull")
	}
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(parent))
	for a.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	if err := os.Rename(filepath.Join(nested, ".git"), filepath.Join(t.TempDir(), "nested-metadata")); err != nil {
		t.Fatal(err)
	}
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(nested))
	for b.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	button, ok = tt.AccessByRole(ui.RoleButton, b.t("pull"))
	if ok && !button.Disabled {
		clickRepositoryTabsControl(t, tt, ui.RoleButton, b.t("pull"))
	} else {
		b.pullFF() // A stale dispatch must be rejected even without the toolbar.
	}
	for b.busy || b.loading {
		runContinuation(t, callbacks)
	}
	if head := git(parent, "rev-parse", "HEAD"); head != parentHead || ok && !button.Disabled {
		t.Fatalf("failed nested checkout refresh left Pull enabled or mutated the parent: enabled=%v parent HEAD=%s want %s", ok && !button.Disabled, head, parentHead)
	}
}

func TestRepositoryTabsPreserveUnchangedHistoryNavigation(t *testing.T) {
	_, a, b, tt, callbacks := repositoryTabsSafetyFixture(t)
	root := b.repo.Root
	parent := strings.TrimSpace(selectionGit(t, root, "rev-parse", "HEAD"))
	selectionGit(t, root, "commit", "--allow-empty", "-m", "history child")
	child := strings.TrimSpace(selectionGit(t, root, "rev-parse", "HEAD"))
	b.reload()
	for b.loading {
		runContinuation(t, callbacks)
	}
	b.openPage("history")
	b.selectCommit(child)
	b.jumpWithHistory(parent)
	tt.Frame()
	if !b.canNavigateHistory(-1) {
		t.Fatal("fixture lacks a Back destination")
	}
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(a.repo.Root))
	for a.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(root))
	for b.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	if !b.canNavigateHistory(-1) {
		t.Fatal("returning to the unchanged checkout lost Back")
	}
}

func TestRepositoryTabsEndNavigatesWithoutClosing(t *testing.T) {
	tabs, a, b, tt, callbacks := repositoryTabsSafetyFixture(t)
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(a.repo.Root))
	for a.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	tt.Key(0, ui.KeyEnd)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if tabs.active != b || len(tabs.sessions) != 2 {
		t.Fatalf("End then Enter should select the last tab: active=%s tabs=%d", tabs.active.repo.Root, len(tabs.sessions))
	}
}

func TestRepositoryTabsRefuseAmendAfterExternalBranchSwitch(t *testing.T) {
	tabs, a, b, tt, callbacks := repositoryTabsSafetyFixture(t)
	root := b.repo.Root
	selectionGit(t, root, "branch", "other") // Same SHA, different symbolic HEAD.
	originalHead := selectionGit(t, root, "rev-parse", "refs/heads/other")
	clickRepositoryTabsControl(t, tt, ui.RoleCheckBox, b.t("amendLast"))
	b.commitText = "First branch amend draft"
	tt.Frame()
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(a.repo.Root))
	for tabs.active.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	selectionGit(t, root, "switch", "other")
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(root))
	// Do not run any queued probes yet. This is the first frame after a switch.
	if node, ok := tt.AccessByRole(ui.RoleButton, b.t("amend")); ok && !node.Disabled {
		clickRepositoryTabsControl(t, tt, ui.RoleButton, b.t("amend"))
	}
	for b.busy || b.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	head := selectionGit(t, root, "rev-parse", "refs/heads/other")
	if head != originalHead || !strings.HasPrefix(b.commitText, "First branch amend draft") {
		t.Fatalf("external branch changed from %.12s to %.12s, draft=%q", originalHead, head, b.commitText)
	}
	if tabs.active != b || b.status.Branch != "other" || b.amend {
		t.Fatalf("return must show the current branch and disarm old Amend: active=%v branch=%q amend=%v", tabs.active == b, b.status.Branch, b.amend)
	}
}
