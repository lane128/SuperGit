package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestWelcomeOpensRecentCheckoutInPlaceAndReusesExistingTab(t *testing.T) {
	a, _, alpha := worktreeApp(t)
	_, _, bravo := worktreeApp(t)
	a.repo = nil
	a.locale = localeEnglish
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	tabs := newRepositoryTabs(a.ctx, a)
	t.Cleanup(tabs.shutdown)
	view := ui.NewTester(tabs.view, 980, 640)
	view.SetTitleBar(ui.TitleBar{Height: 38, Left: 72})
	wait := func() {
		t.Helper()
		for tabs.active.loading || tabs.opening != nil || tabs.active.activationRefresh {
			runContinuation(t, callbacks)
		}
		view.Frame()
	}
	a.openRepository(alpha, true)
	wait()
	a.openRepository(bravo, true)
	wait()
	if b := tabs.active; b.repo == nil || b.repo.Root != bravo || b == a {
		t.Fatal("setup did not open Bravo in a second tab")
	}
	tabs.close(tabs.active)
	wait()
	const draft = "Draft belongs to Alpha"
	a.commitText = draft
	clickRepositoryTabsControl(t, view, ui.RoleButton, a.t("newRepositoryTab"))
	blank := tabs.active
	if blank.repo != nil || len(tabs.sessions) != 2 {
		t.Fatal("New Tab did not open a Welcome tab")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, filepath.Base(bravo))
	wait()
	if tabs.active != blank || blank.repo == nil || blank.repo.Root != bravo || len(tabs.sessions) != 2 {
		t.Fatal("clicking a closed recent checkout did not fill Welcome in place")
	}
	clickRepositoryTabsControl(t, view, ui.RoleButton, blank.t("newRepositoryTab"))
	clickRepositoryTabsControl(t, view, ui.RoleButton, filepath.Base(alpha))
	wait()
	if tabs.active != a || len(tabs.sessions) != 3 || a.commitText != draft {
		t.Fatal("clicking an already-open recent checkout duplicated its tab or lost its draft")
	}
}

func TestWelcomeDistinguishesSameNamedRecentCheckouts(t *testing.T) {
	a := &app{locale: localeEnglish}
	prefix := filepath.Join(t.TempDir(), strings.Repeat("shared-parent-", 10))
	desktop := filepath.Join(prefix, "Desktop", "GitGUI")
	server := filepath.Join(prefix, "Server", "GitGUI")
	a.recentRepositories = []string{desktop, server}
	tabs := newRepositoryTabs(t.Context(), a)
	t.Cleanup(tabs.shutdown)
	view := ui.NewTester(tabs.view, 980, 640)
	view.Frame()

	for _, parent := range []string{"Desktop", "Server"} {
		label := "…" + string(filepath.Separator) + parent
		// The button supplies its own accessibility label, so inspect the
		// rendered subtitle independently of the accessible full path.
		if bounds, ok := view.Find(label); !ok || bounds.W < 40 {
			t.Errorf("same-named recent checkout does not show its distinct parent %q: %+v", label, bounds)
		}
	}
	paths := map[string]bool{desktop: false, server: false}
	for _, node := range view.AccessAll() {
		if _, known := paths[node.Description]; node.Role == ui.RoleButton && node.Name == "GitGUI" && known {
			paths[node.Description] = true
		}
	}
	if !paths[desktop] || !paths[server] {
		t.Fatal("same-named recent buttons must still expose their exact checkout paths")
	}
}

func TestWelcomeLocalizesFailedRepositoryOpen(t *testing.T) {
	for _, test := range []struct {
		language locale
		guidance string
	}{
		{localeEnglish, "Check this location or choose the repository’s current folder."},
		{localeChinese, "请检查该目录，或选择仓库当前所在的文件夹。"},
	} {
		t.Run(string(test.language), func(t *testing.T) {
			a, _, _ := worktreeApp(t)
			a.repo, a.locale = nil, test.language
			callbacks := make(chan func(), 128)
			a.runOnMain = func(fn func()) { callbacks <- fn }
			root := t.TempDir()
			a.recentRepositories = []string{root}
			tabs := newRepositoryTabs(a.ctx, a)
			t.Cleanup(tabs.shutdown)
			view := ui.NewTester(tabs.view, 980, 640)
			clickRepositoryTabsControl(t, view, ui.RoleButton, filepath.Base(root))
			for a.loading {
				runContinuation(t, callbacks)
			}
			if a.repo != nil || tabs.active != a || !strings.Contains(a.loadErr, root) ||
				!strings.Contains(a.loadErr, test.guidance) || strings.Contains(a.loadErr, "fatal:") {
				t.Fatalf("Welcome should show the path and localized recovery guidance: %q", a.loadErr)
			}
			notice := a.takeNotice(toastMain)
			if notice == nil || !needsNoticeDetails(notice) || !strings.Contains(noticeDetailText(notice), "fatal: not a git repository") ||
				strings.Contains(noticeDetailText(notice), "无法打开 Git 工作区") {
				t.Fatalf("error details should retain Git's original diagnostic separately: %+v", notice)
			}
		})
	}
}

func TestWelcomeRejectsRecentPathThatNowBelongsToParentCheckout(t *testing.T) {
	a, _, parent := worktreeApp(t)
	a.repo, a.locale = nil, localeEnglish
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	formerRoot := filepath.Join(parent, "Former Checkout")
	if err := os.Mkdir(formerRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	// The recorded nested checkout has been removed, leaving only a directory.
	a.recentRepositories = []string{formerRoot}
	tabs := newRepositoryTabs(a.ctx, a)
	t.Cleanup(tabs.shutdown)
	view := ui.NewTester(tabs.view, 980, 640)
	clickRepositoryTabsControl(t, view, ui.RoleButton, filepath.Base(formerRoot))
	for a.loading {
		runContinuation(t, callbacks)
	}
	view.Frame()
	if a.repo != nil || a.loadErr == "" || tabs.active != a || len(tabs.sessions) != 1 {
		t.Fatal("stale recent entry must leave Welcome open with an error instead of opening its parent checkout")
	}
	if strings.Contains(a.loadErr, "Your current repository is retained") || !strings.Contains(a.loadErr, formerRoot) {
		t.Fatalf("Welcome should identify the failed path without implying a current repository: %q", a.loadErr)
	}
	clickable, ok := view.AccessByRole(ui.RoleButton, a.t("chooseFolder"))
	if !ok || clickable.Disabled || clickable.Offscreen {
		t.Fatal("folder chooser is not reachable after the failed recent open")
	}
	guidance, ok := view.AccessByRole(ui.RoleText, a.loadErr)
	gap := clickable.Bounds.Y - (guidance.Bounds.Y + guidance.Bounds.H)
	if !ok || guidance.Offscreen || gap < 0 || gap > 20 {
		t.Fatalf("recovery guidance should sit immediately above the folder chooser: guidance=%+v button=%+v", guidance.Bounds, clickable.Bounds)
	}
}
