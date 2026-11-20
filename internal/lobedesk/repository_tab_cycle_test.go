package lobedesk

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestRepositoryTabCycleFromTextEditor(t *testing.T) {
	tabs, a, b, tt, callbacks := repositoryTabsSafetyFixture(t)
	_, _, thirdRoot := worktreeApp(t)
	b.openRepository(thirdRoot, true)
	for tabs.opening != nil || tabs.active.loading {
		runContinuation(t, callbacks)
	}
	c := tabs.active
	if c == b || c.repo == nil {
		t.Fatal("third checkout did not open")
	}
	tt.Frame()
	clickRepositoryTabsControl(t, tt, ui.RoleTab, filepath.Base(b.repo.Root))
	for b.loading {
		runContinuation(t, callbacks)
	}
	tt.Frame()
	selectWorktreeRow(t, tt, b)
	label := b.t("commitMessage")
	clickRepositoryTabsControl(t, tt, ui.RoleTextField, label)
	tt.Type("Draft belongs to the second workspace")
	if !tt.Focused(label) || b.commitText != "Draft belongs to the second workspace" {
		t.Fatal("editor did not receive the fixture draft")
	}
	step := func(mod ui.Modifiers, want *app) {
		t.Helper()
		tt.Key(mod, ui.KeyTab)
		tt.Frame()
		if tabs.active != want {
			t.Fatalf("did not select %v", want.repo.Root)
		}
		for want.loading {
			runContinuation(t, callbacks)
		}
		tt.Frame()
	}
	step(ui.Ctrl, c)
	step(ui.Ctrl, a)
	step(ui.Ctrl|ui.Shift, c)
	step(ui.Ctrl|ui.Shift, b)
	if b.commitText != "Draft belongs to the second workspace" {
		t.Fatal("switching tabs changed the draft")
	}
}
