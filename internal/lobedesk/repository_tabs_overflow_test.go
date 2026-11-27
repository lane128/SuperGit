package lobedesk

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestRepositoryTabsOverflowCannotCloseClippedTab(t *testing.T) {
	tabs, a, b, view, _ := repositoryTabsSafetyFixture(t)
	prefix := t.TempDir()
	tabs.sessions = []*app{a}
	for i := range 8 {
		other := tabs.newSession(b)
		other.repo = &gitrepo.Repository{Root: filepath.Join(prefix, fmt.Sprintf("Example repository with long folder name %d", i))}
		tabs.sessions = append(tabs.sessions, other)
	}
	tabs.sessions = append(tabs.sessions, b)
	for range 3 {
		view.Frame()
	}
	view.Scroll(120, 17, -50000, 0)
	view.Frame()
	const left = 80
	name := b.t("close") + " " + a.repo.Root
	close, ok := view.AccessByRole(ui.RoleButton, name)
	if !ok || close.Disabled || close.Offscreen {
		t.Fatal("the fully visible first tab cannot be closed")
	}
	// Put half of the close control past the left edge, with the tab body
	// entirely outside the viewport, as when horizontally scrolling.
	tabs.scroll.X = close.Bounds.X + close.Bounds.W/2 - left
	for range 3 {
		view.Frame()
	}
	view.ClickAt(left+2, close.Bounds.Y+close.Bounds.H/2)
	view.Frame()
	if len(tabs.sessions) != 10 || tabs.active != b {
		t.Fatal("the clipped edge of a tab closed or switched a repository")
	}
}

func TestRepositoryTabRevealsWiderCheckoutThatFillsWelcome(t *testing.T) {
	tabs, a, b, view, callbacks := repositoryTabsSafetyFixture(t)
	_, _, original := worktreeApp(t)
	root := filepath.Join(filepath.Dir(original), "An independent repository with a much longer name")
	if err := os.Rename(original, root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Rename(root, original) })
	prefix := t.TempDir()
	tabs.sessions = []*app{a}
	for i := range 7 {
		other := tabs.newSession(b)
		other.repo = &gitrepo.Repository{Root: filepath.Join(prefix, fmt.Sprintf("Example repository with long folder name %d", i))}
		tabs.sessions = append(tabs.sessions, other)
	}
	tabs.sessions = append(tabs.sessions, b)
	b.rememberRepository(root)
	for range 5 {
		view.Frame()
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, b.t("newRepositoryTab"))
	welcome := tabs.active
	for range 5 {
		view.Frame()
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, filepath.Base(root))
	for welcome.loading {
		runContinuation(t, callbacks)
	}
	for range 5 {
		view.Frame()
	}
	if tabs.active != welcome || welcome.repo == nil || welcome.repo.Root != root {
		t.Fatal("the selected checkout did not fill the same Welcome tab")
	}
	close, ok := view.AccessByRole(ui.RoleButton, b.t("close")+" "+root)
	if !ok || close.Offscreen || close.Disabled {
		t.Fatalf("the active tab did not reveal its new width: close=%+v scroll=%+v", close, tabs.scroll)
	}
}

func TestRepositoryTabsOverflowArrowsPanWithoutSwitching(t *testing.T) {
	tabs, a, b, view, callbacks := repositoryTabsSafetyFixture(t)
	b.commitText = "Draft in the active checkout"
	prefix := t.TempDir()
	tabs.sessions = []*app{a}
	for i := range 8 {
		extra := tabs.newSession(b)
		extra.repo = &gitrepo.Repository{Root: filepath.Join(prefix, fmt.Sprintf("Long project name %d", i))}
		tabs.sessions = append(tabs.sessions, extra)
	}
	tabs.sessions = append(tabs.sessions, b)
	frame := func() {
		for range 5 {
			view.Frame()
		}
	}
	left, right := "Scroll tabs left", "Scroll tabs right"
	view.SetSize(980, 680)
	frame()
	assertArrow := func(label string, enabled bool) {
		t.Helper()
		node, ok := view.AccessByRole(ui.RoleButton, label)
		if !ok || node.Offscreen || node.Disabled == enabled || node.Bounds.Y+node.Bounds.H > 52 {
			t.Fatalf("expected visible arrow %q (enabled=%v), got %+v", label, enabled, node)
		}
	}
	assertArrow(left, true)
	assertArrow(right, false)
	x := tabs.scroll.X
	clickRepositoryTabsControl(t, view, ui.RoleButton, left)
	frame()
	if tabs.scroll.X >= x || tabs.active != b || len(tabs.sessions) != 10 || tabs.sessions[0] != a || tabs.sessions[9] != b || b.commitText != "Draft in the active checkout" {
		t.Fatalf("panning left changed the checkout, draft or order, or did not scroll: x=%g -> %g", x, tabs.scroll.X)
	}
	assertArrow(right, true)
	clickRepositoryTabsControl(t, view, ui.RoleButton, right)
	frame()
	if tabs.scroll.X != x || tabs.active != b {
		t.Fatalf("panning right did not return to the end: x=%g, want %g", tabs.scroll.X, x)
	}
	assertArrow(right, false)
	for range 10 {
		if tabs.scroll.X <= 0 {
			break
		}
		clickRepositoryTabsControl(t, view, ui.RoleButton, left)
		frame()
	}
	assertArrow(left, false)
	clickRepositoryTabsControl(t, view, ui.RoleTab, filepath.Base(a.repo.Root))
	for a.loading {
		runContinuation(t, callbacks)
	}
	frame()
	if tabs.active != a {
		t.Fatal("the exposed tab could not be selected")
	}
	view.SetSize(3000, 680)
	frame()
	for _, label := range []string{left, right} {
		if _, ok := view.AccessByRole(ui.RoleButton, label); ok {
			t.Fatalf("%q is visible when all tabs fit", label)
		}
	}
}

func TestRepositoryTabsOverflowKeepsLastTabAndOpenReachable(t *testing.T) {
	tabs, a, b, tt, callbacks := repositoryTabsSafetyFixture(t)
	prefix := t.TempDir()
	const extra = 8
	tabs.sessions = []*app{a}
	for i := 0; i < extra; i++ {
		other := tabs.newSession(b)
		other.repo = &gitrepo.Repository{Root: filepath.Join(prefix, fmt.Sprintf("Example repository with long folder name %d", i))}
		tabs.sessions = append(tabs.sessions, other)
	}
	tabs.sessions = append(tabs.sessions, b)
	tt.SetSize(560, 680)
	tt.Frame()
	tt.Frame()
	activeName := filepath.Base(b.repo.Root)
	focus, ok := tt.AccessByRole(ui.RoleTab, activeName)
	if !ok || focus.Offscreen {
		t.Errorf("after resize, active tab %q has no visible bounds: %+v scroll=%+v", activeName, focus, tabs.scroll)
	}
	tt.Scroll(120, 17, -50000, 0)
	tt.Frame()
	tt.Frame()
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(a.repo.Root))
	for a.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if tabs.active != b {
		t.Fatalf("End then Enter did not select the last tab; active=%q scroll=%+v", tabs.active.repo.Root, tabs.scroll)
	}
	for b.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	previousRight := float32(0)
	for _, label := range []string{b.t("switchRepositoryTab"), b.t("newRepositoryTab")} {
		node, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok || node.Offscreen || node.Disabled || node.Bounds.Y < 0 || node.Bounds.Y+node.Bounds.H > 52 ||
			node.Bounds.X < previousRight || node.Bounds.X+node.Bounds.W > 560 {
			t.Fatalf("%q escaped the narrow title strip or overlapped an action: %+v", label, node)
		}
		previousRight = node.Bounds.X + node.Bounds.W
	}
}
