package lobedesk

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestUndoCommitMenuKeepsOriginallySelectedHead(t *testing.T) {
	a, _, root := worktreeApp(t)
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v: %s", err, out)
		}
		return string(out)
	}
	run("commit", "--allow-empty", "-m", "selected latest")
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	commit := gitrepo.Commit{SHA: a.status.Head, Subject: "selected latest", Refs: "HEAD -> main"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.commitMenu(c, display.Text(c, "selected latest", display.TextProps{}), commit)
	}, 960, 900)
	row, ok := tt.Find("selected latest")
	if !ok {
		t.Fatal("commit menu row missing")
	}
	tt.RightClickAt(row.X+row.W/2, row.Y+row.H/2)
	item, ok := tt.Find(a.t("undoCommit"))
	if !ok {
		t.Fatal("Undo action missing from selected HEAD menu")
	}
	tt.ClickAt(item.X+item.W/2, item.Y+item.H/2)
	if !a.confirm.open {
		t.Fatal("Undo confirmation missing")
	}
	if !strings.Contains(a.confirm.message, shortSHA(commit.SHA)) {
		t.Fatal("Undo confirmation does not identify original commit")
	}
	run("commit", "--allow-empty", "-m", "later latest")
	head, index := run("rev-parse", "HEAD"), run("ls-files", "--stage")
	// Even an intervening UI refresh must retain the original confirmation target.
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan func(), 1)
	a.runOnMain = func(fn func()) {
		release := make(chan struct{})
		done <- func() { defer close(release); fn() }
		<-release
	}
	a.confirm.action()
	runContinuation(t, done)
	if run("rev-parse", "HEAD") != head || run("ls-files", "--stage") != index {
		t.Fatal("old Undo confirmation removed a later commit")
	}
	found := false
	for _, n := range a.notices[toastMain] {
		if strings.HasPrefix(n.description, a.t("undoTargetChanged")) {
			found = true
		}
	}
	if !found {
		t.Fatal("Undo refusal lacks localized recovery")
	}
}
