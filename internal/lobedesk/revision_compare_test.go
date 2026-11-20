package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestRevisionComparisonFromHistoryTagMenus(t *testing.T) {
	a, callbacks := overviewFixture(t)
	root := a.repo.Root
	splitCommitGit(t, root, "tag", "-a", "v0.1.23", "-m", "base tag")
	if err := os.WriteFile(filepath.Join(root, "added.txt"), []byte("release note\n"), 0600); err != nil {
		t.Fatal(err)
	}
	splitCommitGit(t, root, "add", "--", "added.txt")
	splitCommitGit(t, root, "commit", "-m", "fixture target")
	splitCommitGit(t, root, "tag", "v0.1.25")
	var err error
	a.commits, err = a.repo.History(a.ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	a.tags, err = a.repo.Tags(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.page, a.locale, a.historyQuery, a.sidebarHidden = "history", localeEnglish, "fixture", true
	tt := ui.NewTester(a.root, 1480, 860)
	choose := func(tag, action string) {
		t.Helper()
		r, ok := tt.Find(tag)
		if !ok {
			t.Fatalf("missing tag %s", tag)
		}
		tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
		item, ok := tt.Find(action)
		if !ok {
			var menu []string
			for _, node := range tt.AccessAll() {
				if node.Role == ui.RoleMenuItem {
					menu = append(menu, node.Name)
				}
			}
			t.Fatalf("missing %q on tag %s; base=%+v; items=%v", action, tag, a.comparison.base, menu)
		}
		tt.ClickAt(item.X+item.W/2, item.Y+item.H/2)
		tt.Advance(200 * time.Millisecond)
	}
	choose("v0.1.23", "Set comparison base")
	choose("v0.1.25", "Compare with base")
	waitForText := func(text string) {
		t.Helper()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		for {
			tt.Frame()
			if _, ok := tt.Find(text); ok {
				return
			}
			select {
			case fn := <-callbacks:
				fn()
			case <-timer.C:
				t.Fatalf("comparison missing %q", text)
			}
		}
	}
	waitForText("+release note")
	for _, text := range []string{"Base", "v0.1.23 · " + shortSHA(a.commits[1].SHA), "Target", "v0.1.25 · " + shortSHA(a.commits[0].SHA)} {
		if _, ok := tt.Find(text); !ok {
			t.Errorf("missing comparison endpoint %q", text)
		}
	}
	row, ok := tt.AccessByRole(ui.RoleOption, "same.go")
	if !ok {
		t.Fatal("missing second changed file")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	waitForText("+index version")
	// From the selected file, the next tab stop is its read-only patch.
	tt.Key(0, ui.KeyTab)
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("index version")
	if _, ok := tt.Find("1/1"); !ok {
		t.Fatal("comparison patch search did not find the selected file’s content")
	}
	tt.Key(0, ui.KeyEscape)
	if _, ok := tt.Find(a.t("compareRevisions")); !ok || a.comparison.view == nil || a.comparison.view.find != nil {
		t.Fatal("closing search dismissed comparison or retained the search")
	}
	if _, ok := tt.Find("+working version"); ok {
		t.Fatal("showing the working tree")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(200 * time.Millisecond)
	if a.hasPendingDialog(toastMain) || !a.worktreeSelected || a.page != "history" || a.historyQuery != "fixture" {
		t.Fatalf("dismissal changed history selection or filter")
	}
	if got := strings.TrimSpace(splitCommitGit(t, root, "status", "--porcelain")); got != "M same.go" {
		t.Fatalf("checkout changed: %q", got)
	}
}

func TestRevisionComparisonEmptyErrorAndScope(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.locale = localeChinese
	sha := a.status.Head
	base := a.comparisonRevision(sha, "release")
	a.comparisonMenuAction(compareBaseKey, base)
	receive := func() func() {
		t.Helper()
		select {
		case fn := <-callbacks:
			return fn
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for a comparison read")
		}
		return nil
	}
	// An absent comparison object is a read error, not an empty diff.
	a.comparisonMenuAction(compareTargetKey, a.comparisonRevision(strings.Repeat("f", 40), "missing"))
	receive()()
	tt := pageTester(a, a.comparisonDialog)
	if _, ok := tt.Find(a.t("retry")); !ok {
		t.Fatal("missing retry for unreadable revision")
	}
	if _, ok := tt.Find(a.t("comparisonIdentical")); ok {
		t.Fatal("read failure presented as identical trees")
	}
	a.closeComparison()
	a.comparisonMenuAction(compareTargetKey, base)
	receive()()
	tt.Frame()
	if _, ok := tt.Find(a.t("comparisonIdentical")); !ok {
		t.Fatal("identical trees lack an explicit empty state")
	}
	a.closeComparison()
	// Pin one callback, dismiss and reopen. Even comparing the same pair
	// again must not let that callback complete the new dialog's read.
	a.comparisonMenuAction(compareTargetKey, base)
	stale := receive()
	a.closeComparison()
	a.comparisonMenuAction(compareTargetKey, base)
	stale()
	if !a.comparison.view.loading {
		t.Fatal("a closed comparison populated its replacement")
	}
	fresh := receive()
	other := &app{repo: &gitrepo.Repository{Root: t.TempDir()}}
	a.repositoryTabs = &repositoryTabs{active: other, sessions: []*app{a, other}}
	other.repositoryTabs = a.repositoryTabs
	if items := other.comparisonMenuItems(other.comparisonRevision(sha, "")); len(items) != 1 {
		t.Fatal("other tab inherited comparison base")
	}
	other.comparisonMenuAction(compareTargetKey, base)
	if other.comparison.view != nil {
		t.Fatal("other tab accepted old target")
	}
	a.repo = other.repo
	a.discardStaleComparison()
	fresh()
	if a.comparison.base.sha != "" || a.comparison.view != nil {
		t.Fatal("old comparison leaked across checkout replacement")
	}
}
