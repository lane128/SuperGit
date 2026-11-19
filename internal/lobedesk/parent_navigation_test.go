package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func parentShortcut(t *testing.T) appShortcut {
	t.Helper()
	for _, s := range shortcutTable() {
		if s.ID == "history.parent" {
			return s
		}
	}
	t.Fatal("missing first-parent shortcut")
	return appShortcut{}
}

func parentNavigationApp(t *testing.T) *app {
	a := headShortcutApp(t)
	a.worktreeSelected = false
	a.selectedSHA = "merge"
	a.commits = []gitrepo.Commit{{SHA: "merge", Parents: []string{"first", "second"}}, {SHA: "unrelated"}, {SHA: "second"}, {SHA: "first"}}
	return a
}

func TestParentNavigationSelectsFirstParent(t *testing.T) {
	a := parentNavigationApp(t)
	s := parentShortcut(t)
	tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
	tt.Key(s.Mod, s.Key)
	if a.selectedSHA != "first" || a.worktreeSelected || !a.revealSelection || a.pendingJumpSHA != "" || a.commitText != "retained draft" {
		t.Fatal("shortcut did not reveal actual first parent with draft preserved")
	}
}

func TestParentNavigationPagesOutOfFileHistoryWithoutGitWrites(t *testing.T) {
	a, callbacks, _ := automaticHistoryFixture(t, 420)
	all, err := a.repo.ReadHistory(a.ctx, 1000)
	if err != nil || len(all.Commits) < 302 {
		t.Fatalf("history fixture: %v", err)
	}
	source := all.Commits[300]
	if len(source.Parents) != 1 {
		t.Fatal("fixture source lacks first parent")
	}
	a.locale, a.page = localeEnglish, "history"
	a.worktreeSelected, a.selectedSHA = false, source.SHA
	a.fileFilter, a.historyPaused = "only-in-source.txt", true
	a.fileCommits = []gitrepo.Commit{source}
	a.commitText, a.amend = "draft\n\nbody\n\n", true
	fingerprint := func() string {
		t.Helper()
		var result string
		for _, args := range [][]string{{"rev-parse", "HEAD"}, {"symbolic-ref", "HEAD"}, {"for-each-ref"}, {"ls-files", "--stage", "-z"}, {"status", "--porcelain=v1", "-z"}} {
			cmd := exec.Command("git", append([]string{"-C", a.repo.Root}, args...)...)
			cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("fingerprint %v: %v %s", args, err, out)
			}
			result += string(out) + "\x00"
		}
		data, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
		if err != nil {
			t.Fatal(err)
		}
		return result + string(data)
	}
	before := fingerprint()
	s := parentShortcut(t)
	tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
	tt.Key(s.Mod, s.Key)
	for a.pendingJumpSHA != "" || a.historyLoadingMore || a.commitFilesLoading || a.commitDiffLoading {
		historyCompletion(t, callbacks)()
	}
	if a.selectedSHA != source.Parents[0] || !a.revealSelection || a.fileFilter != "" || a.historyPaused || a.commitText != "draft\n\nbody\n\n" || !a.amend {
		t.Fatal("parent navigation failed to complete beyond file history or changed draft")
	}
	if fingerprint() != before {
		t.Fatal("read-only navigation changed Git or working bytes")
	}
}

func TestParentNavigationGuards(t *testing.T) {
	for _, reason := range []string{"root", "hidden", "worktree", "page", "busy", "loading", "reload", "chooser", "reference", "dialog", "nil"} {
		t.Run(reason, func(t *testing.T) {
			a := parentNavigationApp(t)
			s := parentShortcut(t)
			switch reason {
			case "root":
				a.commits[0].Parents = nil
			case "hidden":
				a.historyQuery = "no matching commit"
			case "worktree":
				a.worktreeSelected = true
			case "page":
				a.page = "changes"
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "reload":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "reference":
				a.historyRefOpen = true
			case "dialog":
				a.confirm.open = true
			case "nil":
				a.repo = nil
			}
			before := a.selectedSHA
			if s.Available == nil || s.Available(a) {
				t.Fatal("unavailable parent shortcut offered")
			}
			tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
			tt.Key(s.Mod, s.Key)
			s.Run(a)
			if a.selectedSHA != before || a.pendingJumpSHA != "" || a.revealSelection {
				t.Fatal("guarded shortcut changed navigation")
			}
		})
	}
}

func TestPendingJumpMissingTargetReportsBlockedRead(t *testing.T) {
	for _, reason := range []string{"paused", "page error", "read error"} {
		t.Run(reason, func(t *testing.T) {
			a := parentNavigationApp(t)
			a.historyMore = true
			switch reason {
			case "paused":
				a.historyPaused = true
			case "page error":
				a.historyPageErr = "failed"
			case "read error":
				a.historyErr = "failed"
			}
			a.jumpToCommit("not loaded")
			if a.pendingJumpSHA != "" || a.historyLoadingMore || len(a.notices[toastMain]) != 1 {
				t.Fatal("blocked read left an inert jump without retry feedback")
			}
		})
	}
}

func TestPendingJumpClearsAfterPagingFailure(t *testing.T) {
	a := parentNavigationApp(t)
	callbacks := make(chan func(), 4)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	a.historyMore = true
	a.jumpToCommit("not loaded")
	historyCompletion(t, callbacks)()
	if a.pendingJumpSHA != "" || a.historyPageErr == "" || len(a.notices[toastMain]) != 1 {
		t.Fatal("failed page retained parent jump or omitted retry feedback")
	}
}

func TestPendingJumpCancelledByWorktreeSelection(t *testing.T) {
	a := parentNavigationApp(t)
	a.historyLoadingMore = true
	a.jumpToCommit("not loaded")
	a.selectWorktree()
	if a.pendingJumpSHA != "" {
		t.Fatal("pending parent navigation survived explicit worktree selection")
	}
}

func TestParentNavigationLeavesActualHistorySearchEditing(t *testing.T) {
	a, _ := overviewFixture(t)
	a.locale, a.page = localeEnglish, "history"
	a.worktreeSelected = false
	a.selectedSHA = "merge"
	a.commits = []gitrepo.Commit{{SHA: "merge", Subject: "merge title", Parents: []string{"parent"}}, {SHA: "parent", Subject: "parent title"}}
	tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.historyPage(c) })
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("merge")
	tt.Frame()
	if a.selectedSHA != "merge" {
		t.Fatal("search fixture lost selected commit")
	}
	tt.Key(ui.Alt, ui.KeyDown)
	if a.selectedSHA != "merge" || a.historyQuery != "merge" || a.pendingJumpSHA != "" {
		t.Fatal("parent shortcut escaped actual search editor")
	}
	tt.Type("X")
	if a.historyQuery != "mergeX" {
		t.Fatal("parent shortcut stole search focus")
	}
}

func TestPendingJumpCancelledByNewSelection(t *testing.T) {
	a := parentNavigationApp(t)
	a.historyLoadingMore = true
	a.jumpToCommit("not loaded")
	if a.pendingJumpSHA != "not loaded" {
		t.Fatal("fixture did not queue navigation")
	}
	a.selectCommit("second")
	if a.pendingJumpSHA != "" {
		t.Fatal("older pending jump survived newer user selection")
	}
}
