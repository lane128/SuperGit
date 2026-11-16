package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func overviewFixture(t *testing.T) (*app, chan func()) {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "same.go"), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-b", "main")
	git("config", "user.name", "Test User")
	git("config", "user.email", "test@example.invalid")
	write("original\n")
	git("add", "--", "same.go")
	git("commit", "--no-gpg-sign", "-m", "fixture")
	write("index version\n")
	git("add", "--", "same.go")
	write("working version\n")
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	repo := &gitrepo.Repository{Root: root}
	status, err := repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	callbacks := make(chan func(), 32)
	a := &app{ctx: ctx, repo: repo, status: status, worktreeSelected: true, runOnMain: func(fn func()) { callbacks <- fn }}
	t.Cleanup(func() {
		if a.overviewCancel != nil {
			a.overviewCancel()
		}
	})
	return a, callbacks
}

func waitOverview(t *testing.T, a *app, callbacks chan func()) {
	t.Helper()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		loading := false
		for _, entry := range a.overviewEntries {
			loading = loading || entry.Loading
		}
		if !loading {
			return
		}
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("overview did not finish")
		}
	}
}

func TestOverviewDrawsBothVersionsAndKeepsDraft(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.locale = localeEnglish
	a.commitText = "draft title\n\nbody"
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	tt := pageTester(a, a.detailOverview)
	for _, text := range []string{"+index version", "+working version"} {
		if _, ok := tt.Find(text); !ok {
			t.Errorf("overview omitted %q", text)
		}
	}
	label := "same.go · " + a.t("unstaged")
	header, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatal("overview has no per-version collapse control")
	}
	tt.ClickAt(header.Bounds.X+header.Bounds.W/2, header.Bounds.Y+header.Bounds.H/2)
	if _, ok := tt.Find("+working version"); ok {
		t.Fatal("collapsed patch still draws its lines")
	}
	if _, ok := tt.Find("+index version"); !ok {
		t.Fatal("collapsing worktree also hid Index patch")
	}
	if a.commitText != "draft title\n\nbody" {
		t.Fatal("overview rendering lost draft")
	}
}

func TestOverviewSeparatesIndexAndWorktree(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	if len(a.overviewEntries) != 2 {
		t.Fatalf("want two versions: %+v", a.overviewEntries)
	}
	for _, entry := range a.overviewEntries {
		if entry.Err != "" {
			t.Fatal(entry.Err)
		}
		if entry.Key.Staged {
			if !strings.Contains(entry.Diff.Text, "+index version") || strings.Contains(entry.Diff.Text, "+working version") {
				t.Fatal("Index overview contains wrong version")
			}
		} else if !strings.Contains(entry.Diff.Text, "+working version") {
			t.Fatal("worktree overview lost working version")
		}
	}
}

func TestOverviewReadFailureIsLocalAndRetryable(t *testing.T) {
	a, callbacks := overviewFixture(t)
	path := filepath.Join(a.repo.Root, "too-large.txt")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.repo.Stage(a.ctx, "too-large.txt"); err != nil {
		t.Fatal(err)
	}
	// Oversized untracked files deliberately return a file-level fallback.
	// A tracked patch exceeding the 4 MiB command limit is a real read error.
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 5<<20)), 0600); err != nil {
		t.Fatal(err)
	}
	a.status.Entries = append(a.status.Entries, gitrepo.Entry{Path: "too-large.txt", Index: ' ', Worktree: 'M'})
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	failed := 0
	for _, entry := range a.overviewEntries {
		if entry.Err != "" {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("failure erased or contaminated other entries: %+v", a.overviewEntries)
	}
	if err := os.WriteFile(path, []byte("small retry\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.retryWorktreeOverview(a.repo.Root, a.overviewSeq, detailFileKey{Path: "too-large.txt"})
	waitOverview(t, a, callbacks)
	for _, entry := range a.overviewEntries {
		if entry.Err != "" {
			t.Fatalf("retry failed: %s", entry.Err)
		}
	}
}

func TestStaleOverviewCannotReplaceTheCurrentTarget(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.loadWorktreeOverview()
	var old func()
	select {
	case old = <-callbacks:
	case <-time.After(10 * time.Second):
		t.Fatal("read never completed")
	}
	a.loadWorktreeOverview()
	old()
	for _, entry := range a.overviewEntries {
		if !entry.Loading {
			t.Fatal("old generation replaced current data")
		}
	}
	waitOverview(t, a, callbacks)
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	before := a.overviewEntries[0].Diff.Text
	old()
	if a.overviewEntries[0].Diff.Text != before {
		t.Fatal("another repository's callback replaced overview")
	}
}

func TestRefreshedTabsKeepOnlyTheirOriginalVersion(t *testing.T) {
	a := &app{worktreeSelected: true}
	staged, unstaged := detailFileKey{Path: "same.go", Staged: true}, detailFileKey{Path: "same.go"}
	a.openDetailFile(staged)
	a.openDetailFile(unstaged)
	a.reconcileWorktreeTabs([]detailFileKey{staged})
	tabs := a.detailTabs()
	if len(tabs.OpenFiles) != 1 || tabs.OpenFiles[0] != staged || tabs.Active.File != staged {
		t.Fatalf("refresh redirected or retained vanished version: %+v", tabs)
	}
	a.reconcileWorktreeTabs(nil)
	if !tabs.Active.Summary || len(tabs.OpenFiles) != 0 {
		t.Fatal("disappeared file did not return to overview")
	}
}
