package lobedesk

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestRepositoryTabsKeepDraftsAndReuseCanonicalRoot(t *testing.T) {
	first, _, firstRoot := worktreeApp(t)
	_, _, secondRoot := worktreeApp(t)
	first.locale, first.page, first.worktreeSelected = localeEnglish, "changes", true
	first.commitText = "First draft\n\nFirst body"
	first.historyQuery = "first query"
	if err := os.WriteFile(filepath.Join(firstRoot, "base.txt"), []byte("first repository patch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	callbacks := make(chan func(), 128)
	first.runOnMain = func(fn func()) { callbacks <- fn }
	tabs := newRepositoryTabs(first.ctx, first)
	t.Cleanup(tabs.shutdown)
	var picked string
	chooserCalls := 0
	first.openDirectoryDialog = func(mygo.OpenDialogOptions) ([]string, error) {
		chooserCalls++
		if picked == "" {
			return nil, nil
		}
		return []string{picked}, nil
	}
	view := ui.NewTester(tabs.view, 980, 760)
	view.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
	click := func(role ui.Role, name string) {
		t.Helper()
		node, ok := view.AccessByRole(role, name)
		if !ok || node.Disabled || node.Offscreen || node.Bounds.W <= 0 || node.Bounds.H <= 0 {
			t.Fatalf("unusable %q: %+v", name, node)
		}
		view.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
		view.Frame()
	}
	wait := func() {
		t.Helper()
		for tabs.opening != nil || tabs.active.loading {
			runContinuation(t, callbacks)
			view.Frame()
		}
	}
	first.selectedPath = "base.txt"
	first.loadDiff()
	var delayedFirstRead func()
	select {
	case delayedFirstRead = <-callbacks:
	case <-time.After(10 * time.Second):
		t.Fatal("first repository Diff did not complete")
	}
	openLabel := first.t("chooseFolder")
	opener, ok := view.AccessByRole(ui.RoleButton, openLabel)
	if !ok || opener.Bounds.Y >= 38 || opener.Offscreen {
		t.Fatalf("the open repository action is missing from the top strip: %+v", opener)
	}
	click(ui.RoleButton, openLabel)
	if chooserCalls != 1 || tabs.active != first || len(tabs.sessions) != 1 || first.commitText != "First draft\n\nFirst body" {
		t.Fatal("cancelling from the top strip changed the checkout or draft")
	}
	picked = secondRoot
	click(ui.RoleButton, openLabel)
	wait()
	second := tabs.active
	if second == first || second.repo.Root != secondRoot || second.commitText != "" || len(tabs.sessions) != 2 {
		t.Fatalf("opening second repository replaced or inherited first session")
	}
	delayedFirstRead()
	view.Frame()
	if !strings.Contains(first.diff.Text, "first repository patch") || strings.Contains(second.diff.Text, "first repository patch") || tabs.active != second {
		t.Fatal("late Diff completion crossed repository sessions")
	}
	selectWorktreeRow(t, view, second)
	click(ui.RoleTextField, second.t("commitSummary"))
	view.Type("Second draft")
	view.Frame()
	click(ui.RoleTab, filepath.Base(firstRoot))
	wait()
	if tabs.active != first || first.commitText != "First draft\n\nFirst body" || first.historyQuery != "first query" {
		t.Fatal("switching back did not preserve first session")
	}
	nested := filepath.Join(secondRoot, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	picked = nested
	click(ui.RoleButton, openLabel)
	wait()
	if tabs.active != second || len(tabs.sessions) != 2 || second.commitText != "Second draft" || chooserCalls != 3 {
		t.Fatalf("nested open: activeIsSecond=%v tabs=%d chooserCalls=%d draft=%q", tabs.active == second, len(tabs.sessions), chooserCalls, second.commitText)
	}
	if output := os.Getenv("SUPERGIT_REPOSITORY_TABS_SCREENSHOT"); output != "" {
		view.Advance(150 * time.Millisecond)
		file, err := os.Create(output)
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(file, view.Image())
		if err := file.Close(); err != nil || encodeErr != nil {
			t.Fatalf("capture tabs: encode=%v close=%v", encodeErr, err)
		}
	}
	picked = t.TempDir()
	click(ui.RoleButton, second.t("openRepo"))
	wait()
	if tabs.active != second || len(tabs.sessions) != 2 || second.loadErr != "" || second.commitText != "Second draft" {
		t.Fatal("failed open replaced or reset the active workbench")
	}
	closeFirst := first.t("close") + " " + firstRoot
	click(ui.RoleButton, closeFirst)
	click(ui.RoleButton, second.t("cancel"))
	if len(tabs.sessions) != 2 || first.commitText != "First draft\n\nFirst body" {
		t.Fatal("cancel did not keep the inactive tab and draft")
	}
	click(ui.RoleButton, closeFirst)
	click(ui.RoleButton, second.t("close"))
	if tabs.active != second || len(tabs.sessions) != 1 || second.commitText != "Second draft" {
		t.Fatal("closing the inactive tab changed the active workbench")
	}
	click(ui.RoleButton, second.t("close")+" "+secondRoot)
	click(ui.RoleButton, second.t("close"))
	if tabs.active.repo != nil || len(tabs.sessions) != 1 || tabs.active.commitText != "" {
		t.Fatal("last close did not return to an empty workbench")
	}

	// The top New Tab action remains visible after the last checkout closes.
	add, found := view.AccessByRole(ui.RoleButton, first.t("newRepositoryTab"))
	if !found || add.Offscreen || add.Disabled || add.Bounds.Y >= 38 || add.Bounds.X+add.Bounds.W < 960 || add.Bounds.X+add.Bounds.W > 980 {
		t.Fatalf("missing usable top-right open button after closing the last tab: %+v", add)
	}
	picked = firstRoot
	click(ui.RoleButton, first.t("openRepo"))
	wait()
	if tabs.active.repo == nil || tabs.active.repo.Root != firstRoot || len(tabs.sessions) != 1 {
		t.Fatal("opening from the empty strip did not select the first checkout")
	}
	picked = secondRoot
	click(ui.RoleButton, first.t("openRepo"))
	wait()
	if tabs.active.repo == nil || tabs.active.repo.Root != secondRoot || len(tabs.sessions) != 2 {
		t.Fatal("the empty-to-two-tab journey did not select the second checkout")
	}
}
