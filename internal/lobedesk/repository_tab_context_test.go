package lobedesk

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestRepositoryTabAliasTargetsInactiveCheckoutAndSurvivesReopen(t *testing.T) {
	tabs, alpha, bravo, view, callbacks := repositoryTabsSafetyFixture(t)
	root := alpha.repo.Root
	original := filepath.Base(root)
	bravo.commitText = "Bravo draft"
	setAlias := func(current, value string) {
		t.Helper()
		node, ok := view.AccessByRole(ui.RoleTab, current)
		if !ok || node.Offscreen || node.Description != root {
			t.Fatalf("cannot identify checkout %q: %+v", current, node)
		}
		previous := tabs.active
		view.RightClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
		view.Frame()
		if tabs.active != previous || previous.prompt.open {
			t.Fatal("opening the context menu changed the active workbench")
		}
		clickRepositoryTabsControl(t, view, ui.RoleMenuItem, "Set Alias")
		for tabs.active.loading {
			runContinuation(t, callbacks)
		}
		view.Frame()
		if tabs.active.repo.Root != root || !tabs.active.prompt.open {
			t.Fatal("Set Alias did not open the editor in the target checkout")
		}
		if value != "" {
			view.Type(value)
		} else {
			if got := tabs.active.prompt.value; got != current {
				t.Fatalf("alias editor prefilled %q, want %q", got, current)
			}
			view.Key(ui.Cmd, ui.KeyA)
			view.Key(0, ui.KeyBackspace)
		}
		if got := tabs.active.prompt.value; got != value {
			t.Fatalf("editing the alias left %q, want %q", got, value)
		}
		clickRepositoryTabsControl(t, view, ui.RoleButton, "Save Alias")
	}
	const alias = "Release worktree"
	setAlias(original, alias)
	if tabs.active != alpha || bravo.commitText != "Bravo draft" {
		t.Fatal("setting an inactive alias must activate the target and preserve the other draft")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, bravo.t("close")+" "+root)
	for tabs.active.loading {
		runContinuation(t, callbacks)
	}
	tabs.open(bravo, root)
	for tabs.opening != nil || tabs.active.loading {
		runContinuation(t, callbacks)
	}
	view.Frame()
	if node, ok := view.AccessByRole(ui.RoleTab, alias); !ok || node.Description != root || !node.Selected {
		t.Fatalf("reopened repository should have its saved alias and exact path: %+v", node)
	}
	path := filepath.Join(t.TempDir(), "repository-session.json")
	if err := writeRepositorySession(path, tabs.repositorySession()); err != nil {
		t.Fatal(err)
	}
	assertLaunchName := func(initialPath, want string) {
		t.Helper()
		restarted, _ := startRepositorySession(t, bravo, path, initialPath)
		restoredView := ui.NewTester(restarted.view, 980, 760)
		restoredView.Frame()
		if node, ok := restoredView.AccessByRole(ui.RoleTab, want); !ok || node.Description != root || !node.Selected {
			t.Fatalf("launch with path %q did not restore name %q: %+v", initialPath, want, node)
		}
	}
	assertLaunchName("", alias)
	assertLaunchName(root, alias)
	setAlias(alias, "Edited alias")
	setAlias("Edited alias", "")
	if _, ok := view.AccessByRole(ui.RoleTab, original); !ok {
		t.Fatal("clearing the alias did not restore the original tab name")
	}
	assertLaunchName("", original)
}

func TestRepositoryTabsDisambiguateIdenticalAliases(t *testing.T) {
	_, alpha, bravo, view, _ := repositoryTabsSafetyFixture(t)
	const alias = "A long alias shared by two separate repository checkouts"
	alpha.repositoryAlias, bravo.repositoryAlias = alias, alias
	view.Frame()
	labels := make(map[string]string)
	for _, node := range view.AccessAll() {
		if node.Role == ui.RoleTab && (node.Description == alpha.repo.Root || node.Description == bravo.repo.Root) {
			labels[node.Description] = node.Name
		}
	}
	if len(labels) != 2 || labels[alpha.repo.Root] == labels[bravo.repo.Root] {
		t.Fatalf("tab names could not distinguish the two checkout paths: %+v", labels)
	}
	node, ok := view.AccessByRole(ui.RoleTab, labels[alpha.repo.Root])
	if !ok || node.Offscreen {
		t.Fatal("tab is unavailable for pointer hover")
	}
	view.Move(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
	view.Advance(time.Second)
	if !view.HasText(alias) {
		t.Fatal("hovering the truncated tab did not reveal its full alias")
	}
}

func TestRepositoryTabMiddleClickRespectsDraftAndPointerCapture(t *testing.T) {
	tabs, alpha, bravo, view, _ := repositoryTabsSafetyFixture(t)
	alpha.commitText = "Alpha draft"
	bravo.commitText = "Bravo draft"
	center := func(session *app) (float32, float32) {
		t.Helper()
		tab, ok := view.AccessByRole(ui.RoleTab, filepath.Base(session.repo.Root))
		if !ok || tab.Offscreen {
			t.Fatalf("tab unavailable: %+v", tab)
		}
		return tab.Bounds.X + tab.Bounds.W/2, tab.Bounds.Y + tab.Bounds.H/2
	}
	ax, ay := center(alpha)
	bx, by := center(bravo)
	view.PressButton(2, ax, ay)
	view.ReleaseButton(2, bx, by)
	view.Frame()
	if tabs.active != bravo || len(tabs.sessions) != 2 || bravo.confirm.open {
		t.Fatal("releasing the middle button on another tab closed or selected a checkout")
	}
	view.PressButton(2, ax, ay)
	view.CancelPress()
	view.Frame()
	if tabs.active != bravo || len(tabs.sessions) != 2 || bravo.confirm.open {
		t.Fatal("canceled middle press acted on the tab")
	}
	view.PressButton(2, ax, ay)
	if bravo.confirm.open || len(tabs.sessions) != 2 {
		t.Fatal("middle press acted before release")
	}
	view.ReleaseButton(2, ax, ay)
	view.Frame()
	if !bravo.confirm.open || tabs.active != bravo || len(tabs.sessions) != 2 {
		t.Fatal("middle click must confirm the inactive target's draft before closing")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, bravo.t("close"))
	if tabs.active != bravo || len(tabs.sessions) != 1 || bravo.commitText != "Bravo draft" {
		t.Fatal("middle click closed or modified the wrong checkout")
	}
}

func TestRepositoryTabContextClosesTargetWithDraftProtection(t *testing.T) {
	tabs, alpha, bravo, view, _ := repositoryTabsSafetyFixture(t)
	alpha.commitText = "Alpha draft"
	bravo.commitText = "Bravo draft"
	root := alpha.repo.Root
	openMenu := func() {
		t.Helper()
		tab, ok := view.AccessByRole(ui.RoleTab, filepath.Base(root))
		if !ok || tab.Offscreen {
			t.Fatalf("inactive tab is unavailable: %+v", tab)
		}
		view.RightClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
		view.Frame()
		if tabs.active != bravo {
			t.Fatal("opening an inactive tab's menu changed the active repository")
		}
	}
	closeFromMenu := func() {
		t.Helper()
		openMenu()
		clickRepositoryTabsControl(t, view, ui.RoleMenuItem, "Close Tab")
		if tabs.active != bravo || len(tabs.sessions) != 2 || !bravo.confirm.open {
			t.Fatal("closing an inactive tab must confirm its draft before closing")
		}
	}
	closeFromMenu()
	clickRepositoryTabsControl(t, view, ui.RoleButton, bravo.t("cancel"))
	if tabs.findCheckout(root) != alpha || alpha.commitText != "Alpha draft" {
		t.Fatal("cancel lost the inactive tab or its draft")
	}
	closeFromMenu()
	clickRepositoryTabsControl(t, view, ui.RoleButton, bravo.t("close"))
	if tabs.active != bravo || tabs.findCheckout(root) != nil || len(tabs.sessions) != 1 || bravo.commitText != "Bravo draft" {
		t.Fatal("context action closed or modified the wrong repository")
	}
}
