package lobedesk

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestRepositoryTabPickerFindsClippedWelcomeAndExcludesClosedRecents(t *testing.T) {
	tabs, a, b, view, _ := repositoryTabsSafetyFixture(t)
	welcome := tabs.newSession(b)
	tabs.sessions = []*app{welcome, a}
	prefix := t.TempDir()
	for i := range 8 {
		extra := tabs.newSession(b)
		extra.repo = &gitrepo.Repository{Root: filepath.Join(prefix, fmt.Sprintf("Independent checkout %d", i))}
		tabs.sessions = append(tabs.sessions, extra)
	}
	tabs.sessions = append(tabs.sessions, b)
	closed := filepath.Join(prefix, "Closed recent checkout")
	b.rememberRepository(closed)
	view.SetSize(700, 760)
	for range 5 {
		view.Frame()
	}
	blank, present := view.AccessByRole(ui.RoleTab, b.t("welcomeTab"))
	if !present || !blank.Offscreen {
		t.Fatal("expected the Welcome tab to be outside the viewport")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, b.t("switchRepositoryTab"))
	view.Frame()
	if _, ok := view.AccessByRole(ui.RoleOption, closed); ok {
		t.Error("the tab picker offered to open a recent-only checkout")
	}
	if _, ok := view.AccessByRole(ui.RoleOption, b.t("welcomeTab")); !ok {
		t.Fatal("the tab picker cannot reach the clipped Welcome tab")
	}
	clickRepositoryTabsControl(t, view, ui.RoleOption, b.t("welcomeTab"))
	view.Frame()
	if tabs.active != welcome || len(tabs.sessions) != 11 {
		t.Fatal("the tab picker should activate the existing Welcome session")
	}
	welcome.openQuickRepositorySwitch()
	view.Frame()
	if _, ok := view.AccessByRole(ui.RoleOption, closed); !ok {
		t.Fatal("regular Quick Switch lost its recent repository results")
	}
	view.Key(0, ui.KeyEscape)
}

func TestRepositoryTabsQuickSwitchFindsOpenCheckoutOutsideRecents(t *testing.T) {
	tabs, a, b, tt, callbacks := repositoryTabsSafetyFixture(t)
	a.commitText = "Draft in the earliest checkout"
	prefix := t.TempDir()
	for i := 0; i < 8; i++ {
		session := tabs.newSession(b)
		session.repo = &gitrepo.Repository{Root: filepath.Join(prefix, fmt.Sprintf("Newer checkout %d", i))}
		tabs.sessions = append(tabs.sessions, session)
		b.rememberRepository(session.repo.Root)
	}
	b.rememberRepository(b.repo.Root)
	frame := func() {
		for range 5 {
			tt.Frame()
		}
	}
	switchLabel := b.t("switchRepositoryTab")
	const narrow, wide = 700, 2200
	assertSwitcher := func(visible bool) {
		t.Helper()
		button, found := tt.AccessByRole(ui.RoleButton, switchLabel)
		if found != visible || (visible && (button.Offscreen || button.Disabled || button.Bounds.Y >= 38)) {
			t.Fatalf("switcher visibility want=%v, found=%v, button=%+v, scroll=%+v", visible, found, button, tabs.scroll)
		}
	}
	tt.SetSize(wide, 760)
	frame()
	assertSwitcher(false)
	tt.SetSize(narrow, 760)
	frame()
	assertSwitcher(true)
	tt.SetSize(wide, 760)
	frame()
	assertSwitcher(false)
	tt.SetSize(narrow, 760)
	frame()
	assertSwitcher(true)
	clickRepositoryTabsControl(t, tt, ui.RoleButton, switchLabel)
	tt.Key(0, ui.KeyEscape)
	frame()
	if b.quickRepositorySwitch.open || !tt.Focused(switchLabel) {
		t.Fatal("Escape did not close the switcher and return focus to its button")
	}
	clickRepositoryTabsControl(t, tt, ui.RoleButton, b.t("newRepositoryTab"))
	frame()
	if tabs.active.repo != nil {
		t.Fatal("expected Welcome to be active")
	}
	assertSwitcher(true)
	clickRepositoryTabsControl(t, tt, ui.RoleButton, switchLabel)
	frame()
	search, ok := tt.AccessByRole(ui.RoleTextField, b.t("searchRepositoryTabs"))
	if !ok || search.Disabled {
		t.Fatal("Quick Switch search missing")
	}
	tt.Type(a.repo.Root)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleOption, a.repo.Root); !ok {
		t.Fatal("the earliest open checkout disappeared from Quick Switch after recent-list eviction")
	}
	tt.Key(0, ui.KeyEnter)
	for tabs.active.loading || tabs.opening != nil {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	if tabs.active != a || len(tabs.sessions) != 11 || a.commitText != "Draft in the earliest checkout" {
		t.Fatalf("Quick Switch did not return to the existing session: active=%v tabs=%d draft=%q", tabs.active == a, len(tabs.sessions), a.commitText)
	}
}
