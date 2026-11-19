package lobedesk

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestRepositoryTabDragPersistsOrderWithoutSwitching(t *testing.T) {
	tabs, a, b, tt, _ := repositoryTabsSafetyFixture(t)
	path := filepath.Join(t.TempDir(), "session.json")
	tabs.sessionState = repositorySessionState{path: path}
	saved := tabs.repositorySession()
	unavailable := filepath.Join(t.TempDir(), "unavailable")
	saved.Roots = slices.Insert(saved.Roots, 1, unavailable)
	if err := writeRepositorySession(path, saved); err != nil {
		t.Fatal(err)
	}
	b.commitText = "Keep my draft while moving tabs"
	aName, bName := filepath.Base(a.repo.Root), filepath.Base(b.repo.Root)
	center := func(name string) (float32, float32) {
		t.Helper()
		node, ok := tt.AccessByRole(ui.RoleTab, name)
		if !ok || node.Offscreen {
			t.Fatalf("tab %q is not visible", name)
		}
		return node.Bounds.X + node.Bounds.W/2, node.Bounds.Y + node.Bounds.H/2
	}
	drag := func(from, to string, cancel bool) {
		x, y := center(from)
		destX, destY := center(to)
		tt.Press(x, y)
		tt.Move(destX, destY)
		tt.Frame()
		if cancel {
			tt.Key(0, ui.KeyEscape)
		}
		tt.Release(destX, destY)
		tt.Frame()
	}

	drag(aName, bName, true)
	if tabs.sessions[0] != a || tabs.active != b {
		t.Fatal("Escape changed the order or active repository")
	}
	drag(aName, bName, false)
	xA, _ := center(aName)
	xB, _ := center(bName)
	if xA <= xB || tabs.active != b || b.commitText != "Keep my draft while moving tabs" || b.loading {
		t.Fatalf("drag should move the inactive tab without switching or reloading the active draft: first=%s, active=%s", tabs.sessions[0].repo.Root, tabs.active.repo.Root)
	}
	tabs.saveSession()
	if saved := readRepositorySession(path); !slices.Equal(saved.Roots, []string{b.repo.Root, unavailable, a.repo.Root}) || saved.Active != b.repo.Root {
		t.Fatalf("reordered workspace was not saved: %+v", saved)
	}
}

func TestRepositoryTabDragRevealsActiveCheckoutAtOverflowEdge(t *testing.T) {
	tabs, _, active, tt, _ := repositoryTabsSafetyFixture(t)
	root := t.TempDir()
	active.commitText = "Draft in the moved repository"
	for i := range 8 {
		session := tabs.newSession(active)
		session.repo = &gitrepo.Repository{Root: filepath.Join(root, fmt.Sprintf("Long release engineering repository %d", i))}
		tabs.sessions = append(tabs.sessions, session)
	}
	settle := func() {
		for range 4 {
			tt.Frame()
		}
	}
	node := func(session *app) ui.AccessInfo {
		t.Helper()
		item, ok := tt.AccessByRole(ui.RoleTab, filepath.Base(session.repo.Root))
		if !ok {
			t.Fatalf("tab for %s is absent", session.repo.Root)
		}
		return item
	}
	var edge, dropX float32
	var target *app
	for width := minWindowWidth; width <= defaultWindowWidth && target == nil; width += 4 {
		tt.SetSize(width, minWindowHeight)
		settle()
		arrow, ok := tt.AccessByRole(ui.RoleButton, active.t("scrollRepositoryTabsLeft"))
		if !ok || arrow.Offscreen {
			continue
		}
		edge = arrow.Bounds.X - 4 // gap between the viewport and first overflow control
		for _, session := range tabs.sessions[2:] {
			item := node(session)
			visible := edge - item.Bounds.X
			if !item.Offscreen && visible >= 12 && visible <= 26 {
				target, dropX = session, item.Bounds.X+visible/2
				break
			}
		}
	}
	if target == nil {
		t.Fatal("no partly visible drop target found")
	}
	before := node(active).Bounds
	y := before.Y + before.H/2
	wantIndex := slices.Index(tabs.sessions, target)
	tt.Press(before.X+before.W/2, y)
	tt.Move(dropX, y)
	tt.Frame()
	tt.Release(dropX, y)
	settle()
	if tabs.sessions[wantIndex] != active || tabs.active != active || active.commitText != "Draft in the moved repository" {
		t.Fatal("dropping the active tab should reorder it and keep its draft")
	}
	after := node(active)
	if after.Offscreen || !after.Selected || after.Bounds.X+after.Bounds.W > edge+.5 {
		t.Fatalf("activity moved but its label is clipped: tab=%+v viewport end=%.1f", after.Bounds, edge)
	}
}
