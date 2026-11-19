package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestRepositoryWelcomeDropOpensAndReusesCheckoutTabs(t *testing.T) {
	tabs, first, second, view, callbacks := repositoryTabsSafetyFixture(t)
	_, _, thirdRoot := worktreeApp(t)
	first.commitText = "draft stays in the first tab"
	waitOpen := func() {
		for tabs.opening != nil || tabs.active.loading {
			runContinuation(t, callbacks)
		}
		view.Frame()
	}
	if view.DropFiles(700, 300, thirdRoot) {
		t.Fatal("dropping onto the workbench should not open a repository")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, second.t("newRepositoryTab"))
	welcome := tabs.active
	if welcome.repo != nil || len(tabs.sessions) != 3 {
		t.Fatal("new tab did not open Welcome")
	}
	if !view.DropFiles(700, 300, t.TempDir()) {
		t.Fatal("Welcome did not accept a folder drop")
	}
	waitOpen()
	if tabs.active != welcome || welcome.repo != nil || welcome.loadErr == "" {
		t.Fatal("invalid folder did not leave Welcome available for retry")
	}
	if !view.DropFiles(700, 300, thirdRoot) {
		t.Fatal("Welcome did not accept a repository folder on retry")
	}
	waitOpen()
	if tabs.active != welcome || welcome.repo == nil || welcome.repo.Root != thirdRoot || len(tabs.sessions) != 3 {
		t.Fatal("successful drop should fill the current Welcome tab")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, second.t("newRepositoryTab"))
	if !view.DropFiles(700, 300, first.repo.Root) {
		t.Fatal("Welcome did not accept an already-open repository folder")
	}
	waitOpen()
	if tabs.active != first || len(tabs.sessions) != 4 || first.commitText != "draft stays in the first tab" {
		t.Fatal("dropping an existing checkout should activate its unique tab and preserve the draft")
	}
}
