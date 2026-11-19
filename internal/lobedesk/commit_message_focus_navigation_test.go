package lobedesk

import (
	"os"
	"os/exec"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCommitMessageShortcutFromHistoricalContentPreservesNavigation(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.locale = localeEnglish
	a.page = "history"
	history, err := a.repo.HistoryPage(a.ctx, nil, 0, 200)
	if err != nil || len(history.Commits) == 0 {
		t.Fatalf("history fixture: %v", err)
	}
	a.commits = history.Commits
	sha := a.commits[0].SHA
	a.worktreeSelected = false
	a.selectedSHA = sha
	a.commitFilesLoaded = true
	a.historyDetailsHidden = true
	a.detailTabs().Open(detailFileKey{Path: "same.go"})
	a.commitDiffScroll = ui.ScrollState{Y: 87}
	a.commitFilesScroll = ui.ScrollState{Y: 45}
	draft := "标题abc\n\nbody\n\n"
	a.commitText = draft
	a.amend = true
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", a.repo.Root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, data)
		}
		return string(data)
	}
	head, index, refs := git("rev-parse", "HEAD"), git("ls-files", "--stage", "-z"), git("for-each-ref")
	tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.contentCard(c) })
	tt.Key(ui.Cmd, ui.Key9)
	waitOverview(t, a, callbacks)
	tt.Frame()
	if !a.worktreeSelected || a.page != "changes" || !tt.Focused(a.t("commitSummary")) || a.commitText != draft || !a.amend || !a.historyDetailsHidden {
		t.Fatal("real Changes editor navigation lost focus or preserved state")
	}
	// Repeated navigation must not reset a caret within existing text.
	tt.Key(0, ui.KeyEnd)
	tt.Key(0, ui.KeyLeft)
	tt.Key(ui.Cmd, ui.Key9)
	tt.Type("X")
	if a.commitFields.Summary != "标题abXc" || a.commitFields.Body != "body\n\n" {
		t.Fatalf("repeat focus reset caret or draft: %+v", a.commitFields)
	}
	// A later render must permit normal focus movement away from the title.
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	tt.Frame()
	if !tt.Focused(a.t("commitDescription")) {
		t.Fatal("focus was not one-shot in the actual page")
	}
	a.openPage("history")
	if a.worktreeSelected || a.selectedSHA != sha || a.detailTabs().Active.File.Path != "same.go" || a.commitDiffScroll.Y != 87 || a.commitFilesScroll.Y != 45 || !a.historyDetailsHidden {
		t.Fatal("shortcut lost previous historical context")
	}
	if git("rev-parse", "HEAD") != head || git("ls-files", "--stage", "-z") != index || git("for-each-ref") != refs {
		t.Fatal("navigation changed Git state")
	}
}
