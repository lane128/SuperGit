package lobedesk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestRepositoryColorPreferenceValidation(t *testing.T) {
	isolateSettings(t)
	root := t.TempDir()
	if got := readRepositoryColor(root); got != "default" {
		t.Fatalf("missing preference = %q", got)
	}
	path, err := repositoryColorPath(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"default", "blue", "green", "amber", "rose", "violet", "gray"} {
		if err := writeLocalJSON(path, "color-*.json", value); err != nil {
			t.Fatal(err)
		}
		if got := readRepositoryColor(root); got != value {
			t.Fatalf("read %q, want %q", got, value)
		}
	}
	for _, data := range []string{`"unknown"`, `null`, `{`, `42`, `""`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if got := readRepositoryColor(root); got != "default" {
			t.Fatalf("invalid %s => %q", data, got)
		}
	}
}

func TestRepositoryColorInactivePersistenceAndGuards(t *testing.T) {
	tabs, target, active, view, callbacks := repositoryTabsSafetyFixture(t)
	root := target.repo.Root
	target.commitText, active.commitText = "target draft", "active draft"
	tabs.setRepositoryColor(target, root, "green")
	if target.repositoryColor != "green" || tabs.active != active || target.commitText != "target draft" || active.commitText != "active draft" {
		t.Fatal("assignment changed workbench state")
	}
	if readRepositoryColor(root) != "green" || readRepositoryColor(active.repo.Root) != "default" {
		t.Fatal("preferences leaked across checkouts")
	}
	for _, blocked := range []string{"busy", "loading", "stale-root", "unknown"} {
		target.busy, target.loading = blocked == "busy", blocked == "loading"
		candidate, color := root, "rose"
		if blocked == "stale-root" {
			candidate = active.repo.Root
		}
		if blocked == "unknown" {
			color = "invalid"
		}
		tabs.setRepositoryColor(target, candidate, color)
		if readRepositoryColor(root) != "green" || target.repositoryColor != "green" {
			t.Fatalf("unguarded %s", blocked)
		}
	}
	target.busy, target.loading = false, false
	path, _ := repositoryColorPath(root)
	// A directory at the destination makes atomic rename fail without chmod/root assumptions.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	tabs.setRepositoryColor(target, root, "rose")
	if target.repositoryColor != "green" || tabs.active != active {
		t.Fatal("failed save changed assignment/navigation")
	}
	view.Frame()
	if !view.HasText(active.t("saveRepositoryColorFailed")) {
		t.Fatal("save failure not visible on active workbench")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	tabs.setRepositoryColor(target, root, "blue")
	if readRepositoryColor(root) != "blue" {
		t.Fatal("explicit blue was not persisted")
	}
	sessionPath := filepath.Join(t.TempDir(), "session.json")
	if err := writeRepositorySession(sessionPath, tabs.repositorySession()); err != nil {
		t.Fatal(err)
	}
	restarted, _ := startRepositorySession(t, active, sessionPath, "")
	if got := restarted.findCheckout(root); got == nil || got.repositoryColor != "blue" {
		t.Fatal("session did not restore color")
	}
	target.commitText = ""
	tabs.close(target)
	tabs.setRepositoryColor(target, root, "rose")
	if readRepositoryColor(root) != "blue" {
		t.Fatal("closed target still wrote preferences")
	}
	tabs.open(active, root)
	for tabs.opening != nil || tabs.active.loading {
		runContinuation(t, callbacks)
	}
	if tabs.active.repositoryColor != "blue" {
		t.Fatal("reopen lost color")
	}
	tabs.setRepositoryColor(tabs.active, root, "default")
	if readRepositoryColor(root) != "default" {
		t.Fatal("reset did not persist")
	}
}

func TestRepositoryColorMenuTargetsInactiveTab(t *testing.T) {
	tabs, target, active, view, _ := repositoryTabsSafetyFixture(t)
	target.commitText, active.commitText = "target draft", "active draft"
	tab, _ := view.AccessByRole(ui.RoleTab, filepath.Base(target.repo.Root))
	view.RightClickAt(tab.Bounds.X+tab.Bounds.W/2, tab.Bounds.Y+tab.Bounds.H/2)
	view.Frame()
	var y float32
	for i, label := range []string{"Default", "Blue", "Green", "Amber", "Rose", "Violet", "Gray"} {
		dot, ok := view.AccessByRole(ui.RoleMenuItemRadio, label)
		if !ok || dot.Offscreen {
			t.Fatalf("missing color dot %s", label)
		}
		if i == 0 {
			y = dot.Bounds.Y
		} else if dot.Bounds.Y != y {
			t.Fatal("palette is not one row")
		}
	}
	clickRepositoryTabsControl(t, view, ui.RoleMenuItemRadio, "Green")
	if tabs.active != active || target.repositoryColor != "green" || target.commitText != "target draft" || active.commitText != "active draft" {
		t.Fatal("inactive recolor changed repository/drafts or failed")
	}
	if _, ok := view.AccessByRole(ui.RoleMenuItemRadio, "Green"); ok {
		t.Fatal("menu stayed open after choosing")
	}
	// Keyboard focus alone must not activate the target checkout.
	tabs.focusTab = target
	view.Frame()
	view.Key(ui.Shift, ui.KeyF10)
	view.Frame()
	green, ok := view.AccessByRole(ui.RoleMenuItemRadio, "Green")
	if !ok || !green.Checked {
		t.Fatal("keyboard menu lost current choice")
	}
	view.Key(0, ui.KeyRight)
	view.Key(0, ui.KeyEnter)
	view.Frame()
	if target.repositoryColor != "amber" || tabs.active != active {
		t.Fatal("arrow selection failed or navigated away")
	}
	view.Key(ui.Shift, ui.KeyF10)
	view.Key(0, ui.KeyEscape)
	view.Frame()
	focused, _ := view.AccessByRole(ui.RoleTab, filepath.Base(target.repo.Root))
	if !focused.Focused || tabs.active != active {
		t.Fatalf("Escape lost focus or activated the target: target=%+v activeChanged=%t controls=%+v", focused, tabs.active != active, view.AccessAll())
	}
	view.Key(ui.Shift, ui.KeyF10)
	view.ClickAt(900, 650)
	view.Frame()
	if _, ok := view.AccessByRole(ui.RoleMenuItemRadio, "Green"); ok {
		t.Fatal("outside click did not dismiss menu")
	}
	if target.repositoryColor != "amber" {
		t.Fatal("dismissal changed color")
	}
}

func TestRepositoryColorCanonicalCheckoutIdentity(t *testing.T) {
	tabs, target, active, _, callbacks := repositoryTabsSafetyFixture(t)
	tabs.setRepositoryColor(target, target.repo.Root, "violet")
	link := filepath.Join(t.TempDir(), "checkout-link")
	if err := os.Symlink(target.repo.Root, link); err != nil {
		t.Fatal(err)
	}
	tabs.open(active, link)
	for tabs.opening != nil || tabs.active.loading {
		runContinuation(t, callbacks)
	}
	if tabs.active != target || tabs.active.repositoryColor != "violet" || len(tabs.sessions) != 2 {
		t.Fatal("canonical symlink did not reuse colored checkout")
	}
	path, _ := repositoryColorPath(target.repo.Root)
	aliasPath, _ := repositoryAliasPath(target.repo.Root)
	if path == aliasPath {
		t.Fatal("color storage overwrites alias")
	}
	if err := writeLocalJSON(aliasPath, "alias-*.json", "renamed"); err != nil {
		t.Fatal(err)
	}
	if readRepositoryColor(target.repo.Root) != "violet" {
		t.Fatal("alias edit lost color")
	}
}
