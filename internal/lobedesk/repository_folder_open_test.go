package lobedesk

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestRepositoryFolderHandoffOpensWithoutWindowFrame(t *testing.T) {
	source, _, firstRoot := worktreeApp(t)
	_, _, nextRoot := worktreeApp(t)
	path := filepath.Join(t.TempDir(), "session.json")
	if err := writeRepositorySession(path, repositorySession{Roots: []string{firstRoot}, Active: firstRoot}); err != nil {
		t.Fatal(err)
	}
	tabs, callbacks := startRepositorySession(t, source, path, "")
	tabs.queueRepositoryFolder(nextRoot)
	if tabs.opening == nil {
		t.Fatal("folder delivery waited for a frame from the background window")
	}
	for tabs.opening != nil || tabs.active.loading {
		runContinuation(t, callbacks)
	}
	if got := tabs.repositorySession(); !slices.Equal(got.Roots, []string{firstRoot, nextRoot}) || got.Active != nextRoot {
		t.Fatalf("folder delivery did not add and select the new checkout: %+v", got)
	}
}

func TestRepositoryFolderHandoffWaitsForRestoreAndBusyTab(t *testing.T) {
	isolateSettings(t)
	source, _, aRoot := worktreeApp(t)
	_, _, bRoot := worktreeApp(t)
	_, _, cRoot := worktreeApp(t)
	_, _, dRoot := worktreeApp(t)
	first := &app{ctx: source.ctx, page: "changes", locale: localeEnglish}
	callbacks := make(chan func(), 128)
	first.runOnMain = func(fn func()) { callbacks <- fn }
	tabs := newRepositoryTabs(first.ctx, first)
	t.Cleanup(tabs.shutdown)
	view := ui.NewTester(tabs.view, 980, 640)
	waitFor := func(want repositorySession) {
		t.Helper()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		for {
			view.Frame()
			got := tabs.repositorySession()
			if slices.Equal(got.Roots, want.Roots) && got.Active == want.Active && tabs.opening == nil && !tabs.active.loading {
				return
			}
			select {
			case fn := <-callbacks:
				fn()
			case <-timer.C:
				t.Fatalf("handoff or restore did not finish: got %+v, want %+v", got, want)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "session.json")
	if err := writeRepositorySession(path, repositorySession{Roots: []string{aRoot, bRoot}, Active: aRoot}); err != nil {
		t.Fatal(err)
	}
	tabs.queueRepositoryFolder(cRoot)
	view.Frame()
	if first.repo != nil || first.loading || tabs.opening != nil {
		t.Fatal("handoff started before session restore")
	}
	tabs.startSession(path, "")
	waitFor(repositorySession{Roots: []string{aRoot, bRoot, cRoot}, Active: cRoot})
	if a := tabs.findCheckout(aRoot); a == nil {
		t.Fatal("cold handoff discarded a saved tab")
	}
	c := tabs.active
	selectWorktreeRow(t, view, c)
	focusCommitField(t, view, c.t("commitMessage"))
	view.Type("Keep the active draft during folder handoff")
	c.busy = true
	alias := filepath.Join(t.TempDir(), "alias-of-B")
	if err := os.Symlink(bRoot, alias); err != nil {
		t.Fatal(err)
	}
	tabs.queueRepositoryFolder(t.TempDir()) // An invalid folder cannot block the batch.
	tabs.queueRepositoryFolder(dRoot)
	tabs.queueRepositoryFolder(alias)
	for range 3 {
		view.Frame()
	}
	if tabs.active != c || tabs.opening != nil || len(tabs.sessions) != 3 {
		t.Fatal("folder handoff bypassed a busy repository")
	}
	c.busy = false
	waitFor(repositorySession{Roots: []string{aRoot, bRoot, cRoot, dRoot}, Active: bRoot})
	if c.commitText != "Keep the active draft during folder handoff" || c.localWatch.closed {
		t.Fatal("incoming folders discarded the draft or closed its tab")
	}
	tabs.saveSession()
	if got := readRepositorySession(path); !slices.Equal(got.Roots, []string{aRoot, bRoot, cRoot, dRoot}) || got.Active != bRoot {
		t.Fatalf("failed to save canonical tabs and focus: %+v", got)
	}
}
