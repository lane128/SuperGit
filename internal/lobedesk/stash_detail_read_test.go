package lobedesk

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func stashDetailReadFixture(t *testing.T) (*app, <-chan func(), gitrepo.Stash) {
	t.Helper()
	a, done, root := worktreeApp(t)
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("saved change to inspect\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := a.repo.SaveStash(a.ctx, "Inspect saved change", false); err != nil {
		t.Fatal(err)
	}
	list, err := a.repo.Stashes(a.ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("stash fixture: %v %v", list, err)
	}
	a.stashes, a.page, a.locale = list, "stash", localeEnglish
	return a, done, list[0]
}

func receiveStashDetailRead(t *testing.T, done <-chan func()) func() {
	t.Helper()
	select {
	case fn := <-done:
		return fn
	case <-time.After(2 * time.Second):
		t.Fatal("Stash detail completion was not delivered through the UI dispatcher")
	}
	return nil
}

func TestStashDetailLoadsSavedPatchThroughWorkbenchControls(t *testing.T) {
	a, done, stash := stashDetailReadFixture(t)
	tt := pageTester(a, a.stashPage)
	row, ok := tt.AccessByRole(ui.RoleOption, stashTitle(stash))
	if !ok || row.Offscreen {
		t.Fatal("missing stash row")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	receiveStashDetailRead(t, done)()
	tt.Frame()
	if a.stashDiffLoading || a.stashDiffErr != "" || a.stashDiff.Text == "" {
		t.Fatalf("stash patch unreadable: loading=%v error=%q", a.stashDiffLoading, a.stashDiffErr)
	}
	if _, ok := tt.Find("+saved change to inspect"); !ok {
		t.Fatal("saved patch not rendered")
	}
	if got := splitCommitGit(t, a.repo.Root, "show", "HEAD:base.txt"); got != "base\n" {
		t.Fatal("reading stash changed HEAD")
	}
	if got := splitCommitGit(t, a.repo.Root, "status", "--porcelain=v1"); got != "" {
		t.Fatalf("reading stash changed work: %q", got)
	}
}

func TestStashDetailLateReadCannotReplaceAnotherRepository(t *testing.T) {
	a, done, stash := stashDetailReadFixture(t)
	a.selectStash(stash.SHA)
	late := receiveStashDetailRead(t, done)
	other, _, _ := worktreeApp(t)
	a.repo = other.repo
	// Navigation can replace the repository before its next list read clears the
	// old selection. That intermediate state must not accept the previous read.
	a.stashDiff = gitrepo.Diff{Text: "other repository placeholder"}
	late()
	if a.stashDiff.Text != "other repository placeholder" {
		t.Fatal("old repository read replaced current detail")
	}
}
