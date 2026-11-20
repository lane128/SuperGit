package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
	"testing"
)

func TestDetailTabsKeepKeyboardOwnership(t *testing.T) {
	a, _ := overviewFixture(t)
	a.locale = localeEnglish
	a.page = "history"
	a.commits = []gitrepo.Commit{{SHA: "aaaaaaaa", Subject: "historical commit"}}
	a.status.Entries = append(a.status.Entries, gitrepo.Entry{Path: "b.go", Index: ' ', Worktree: 'M'})
	first, second := detailFileKey{Path: "same.go"}, detailFileKey{Path: "b.go"}
	a.openDetailFile(first)
	a.openDetailFile(second)
	a.activateDetailTab(detailTabKey{Summary: true})
	tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.historyPage(c) })
	click := func(role ui.Role, label string) {
		t.Helper()
		node, ok := tt.AccessByRole(role, label)
		if !ok {
			t.Fatalf("missing %q", label)
		}
		tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
	}
	// Editing the commit message must not navigate tabs or the history list.
	foundInput := false
	for _, info := range tt.AccessAll() {
		if info.Placeholder == a.t("commitMessage") {
			tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
			foundInput = true
			break
		}
	}
	if !foundInput {
		t.Fatal("missing commit input")
	}
	for _, key := range []ui.Key{ui.KeyRight, ui.KeyDown, ui.KeyHome, ui.KeyEnd} {
		tt.Key(0, key)
	}
	if !a.detailTabs().Active.Summary || !a.worktreeSelected {
		t.Fatal("editing keys escaped the commit input")
	}
	click(ui.RoleTab, a.t("commit"))
	tt.Key(0, ui.KeyRight)
	if a.detailTabs().Active.File != first {
		t.Fatal("Right on tab strip did not activate the next tab")
	}
	tt.Key(0, ui.KeyEnd)
	if a.detailTabs().Active.File != second {
		t.Fatal("End on tab strip did not reach final tab")
	}
	tt.Key(0, ui.KeyHome)
	if !a.detailTabs().Active.Summary {
		t.Fatal("Home on tab strip did not return to first tab")
	}
	click(ui.RoleOption, "same.go")
	tt.Key(0, ui.KeyDown)
	if a.selectedPath != "b.go" || !a.worktreeSelected || a.selectedSHA != "" {
		t.Fatal("Down from file list escaped into history")
	}
	node, ok := tt.AccessByRole(ui.RoleOption, "b.go")
	if !ok || !node.Focused {
		t.Fatal("file navigation did not keep focus on the new row")
	}
	tt.Key(0, ui.KeyUp)
	if a.selectedPath != "same.go" || a.selectedStaged {
		t.Fatal("Up from file list did not navigate file versions")
	}
	click(ui.RoleTab, "b.go · "+a.t("unstaged"))
	click(ui.RoleButton, a.t("close")+" b.go")
	tt.Frame()
	node, ok = tt.AccessByRole(ui.RoleTab, "same.go · "+a.t("unstaged"))
	if !ok || !node.Focused {
		t.Fatal("closing active tab did not restore focus to neighbor")
	}
	click(ui.RoleButton, a.t("close")+" same.go")
	tt.Frame()
	node, ok = tt.AccessByRole(ui.RoleTab, a.t("commit"))
	if !ok || !node.Focused {
		t.Fatal("closing final file did not focus permanent first tab")
	}
}

func TestDetailNavigationStaysBehindOverlays(t *testing.T) {
	for _, blocked := range []string{"reference popup", "hidden details", "dialog", "repository loading"} {
		t.Run(blocked, func(t *testing.T) {
			a := &app{page: "history", worktreeSelected: true}
			first, second := detailFileKey{Path: "a.go"}, detailFileKey{Path: "b.go"}
			a.status.Entries = []gitrepo.Entry{{Path: first.Path, Worktree: 'M'}, {Path: second.Path, Worktree: 'M'}}
			a.openDetailFile(first)
			a.openDetailFile(second)
			a.activateDetailTab(detailTabKey{Summary: true})
			switch blocked {
			case "reference popup":
				a.historyRefOpen = true
			case "hidden details":
				a.setHistoryDetailsHidden(true)
			case "dialog":
				a.confirm.open = true
			case "repository loading":
				a.loading = true
			}
			a.navigateDetailTab(detailTabKey{Summary: true}, ui.KeyRight)
			a.navigateHistoryFile(first, 1)
			if !a.detailTabs().Active.Summary || a.detailFocusPending || a.detailFileFocus != nil {
				t.Fatal("blocked detail navigation changed target or requested background focus")
			}
		})
	}
}
