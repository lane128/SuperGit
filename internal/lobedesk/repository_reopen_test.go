package lobedesk

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestReopenClosedRepositoryRestoresPositionWithFreshState(t *testing.T) {
	isolateSettings(t)
	a, _, aRoot := worktreeApp(t)
	_, _, bRoot := worktreeApp(t)
	_, _, cRoot := worktreeApp(t)
	a.locale, a.page = localeEnglish, "changes"
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	tabs := newRepositoryTabs(a.ctx, a)
	t.Cleanup(tabs.shutdown)
	view := ui.NewTester(tabs.view, 980, 640)
	view.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
	wait := func() {
		t.Helper()
		for tabs.opening != nil || tabs.active.loading {
			runContinuation(t, callbacks)
			view.Frame()
		}
		view.Frame()
		view.Frame()
	}
	open := func(root string) *app {
		t.Helper()
		tabs.active.openRepository(root, true)
		wait()
		if tabs.active.repo == nil || tabs.active.repo.Root != root {
			t.Fatalf("setup could not open %q", root)
		}
		return tabs.active
	}
	click := func(role ui.Role, label string) {
		t.Helper()
		clickRepositoryTabsControl(t, view, role, label)
		wait()
	}
	a.reload()
	wait()
	b := open(bRoot)
	focusCommitField(t, view, b.t("commitSummary"))
	view.Type("Discarded Bravo draft")
	c := open(cRoot)
	focusCommitField(t, view, c.t("commitSummary"))
	view.Type("Charlie draft")
	click(ui.RoleButton, c.t("close")+" "+bRoot)
	click(ui.RoleButton, c.t("cancel"))
	if !slices.Contains(tabs.sessions, b) || b.commitText != "Discarded Bravo draft" {
		t.Fatal("cancelled close removed Bravo or its draft")
	}
	click(ui.RoleButton, c.t("close")+" "+bRoot)
	click(ui.RoleButton, c.t("close"))
	if len(tabs.sessions) != 2 || tabs.active != c {
		t.Fatal("closing Bravo did not leave Alpha and Charlie")
	}
	splitCommitGit(t, bRoot, "commit", "--allow-empty", "-m", "New HEAD while closed")
	newHead := strings.TrimSpace(splitCommitGit(t, bRoot, "rev-parse", "HEAD"))
	fingerprint := repositoryOpenFingerprint(t, bRoot)
	view.Key(ui.Cmd, ui.KeyP)
	view.Frame()
	view.Type("Reopen Closed Tab")
	view.Frame()
	click(ui.RoleOption, "Reopen Closed Tab")
	reopened := tabs.active
	if reopened == b || reopened.repo == nil || reopened.repo.Root != bRoot || reopened.status.Head != newHead || reopened.commitText != "" {
		t.Fatal("reopen did not select a fresh Bravo session with current HEAD and an empty draft")
	}
	if !slices.Equal(tabs.sessions, []*app{a, reopened, c}) || c.commitText != "Charlie draft" {
		t.Fatal("reopen changed tab order or the other repository's draft")
	}
	if _, ok := view.AccessByRole(ui.RoleTab, filepath.Base(bRoot)); !ok || repositoryOpenFingerprint(t, bRoot) != fingerprint {
		t.Fatal("reopened tab is absent or Git state changed")
	}
	tabs.sessionState.path = filepath.Join(t.TempDir(), "session.json")
	tabs.saveSession()
	if saved := readRepositorySession(tabs.sessionState.path); !slices.Equal(saved.Roots, []string{aRoot, bRoot, cRoot}) || saved.Active != bRoot {
		t.Fatalf("reopened tab not saved correctly: %+v", saved)
	}
}
