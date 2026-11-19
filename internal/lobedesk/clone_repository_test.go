package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Removing the empty-state/menu entrance, accepting an occupied destination,
// or opening a clone before Git succeeds must break these user-flow tests.
func TestCloneEntrancesAndDismissalPreserveDraft(t *testing.T) {
	for _, existing := range []bool{false, true} {
		a := &app{locale: localeEnglish}
		if existing {
			a, _, _ = worktreeApp(t)
			a.locale = localeEnglish
		}
		a.commitText = "retained draft"
		tt := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			ui.Column(c).Fill().Children(func() {
				if existing {
					a.sidebarHeader(c)
				} else {
					a.emptyPage(c)
				}
				a.cloneRepositoryDialog(c)
			})
		}, 980, 640)
		if existing {
			title, _ := tt.AccessByRole(ui.RoleButton, a.repoName())
			tt.ClickAt(title.Bounds.X+title.Bounds.W/2, title.Bounds.Y+title.Bounds.H/2)
		}
		role := ui.RoleButton
		if existing {
			role = ui.RoleMenuItem
		}
		entry, ok := tt.AccessByRole(role, a.t("cloneRepo"))
		if !ok {
			t.Fatal("missing Clone entrance")
		}
		tt.ClickAt(entry.Bounds.X+entry.Bounds.W/2, entry.Bounds.Y+entry.Bounds.H/2)
		if !a.cloneForm.open || !a.hasPendingDialog(toastMain) {
			t.Fatal("Clone did not open protected form")
		}
		tt.Key(0, ui.KeyEscape)
		if a.cloneForm.open || a.commitText != "retained draft" || a.busy || a.loading {
			t.Fatal("dismiss changed current checkout context")
		}
	}
}

func TestCloneChooseParentFillsEditableDestination(t *testing.T) {
	parent := t.TempDir()
	a := &app{locale: localeEnglish, commitText: "existing draft"}
	calls := 0
	a.openDirectoryDialog = func(opts mygo.OpenDialogOptions) ([]string, error) {
		calls++
		if !opts.Directory || !a.repositoryChooserOpen {
			t.Fatal("clone needs a guarded native directory chooser")
		}
		if calls == 2 {
			return nil, nil
		}
		return []string{parent}, nil
	}
	a.showCloneRepository()
	a.cloneForm.address = "file:///tmp/Team%20Notes.git"
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.cloneRepositoryDialog(c)
	}, 980, 640)
	clickRepositoryTabsControl(t, view, ui.RoleButton, "Choose parent folder…")
	if want := filepath.Join(parent, "Team Notes"); a.cloneForm.path != want || a.busy || !a.cloneForm.open {
		t.Fatalf("choosing a folder should fill the clone destination without starting Git, got %+v", a.cloneForm)
	}
	a.cloneForm.path = filepath.Join(parent, "My checkout")
	view.Frame()
	clickRepositoryTabsControl(t, view, ui.RoleButton, "Choose parent folder…")
	if calls != 2 || a.cloneForm.path != filepath.Join(parent, "My checkout") || a.repositoryChooserOpen || a.commitText != "existing draft" {
		t.Fatal("canceling the picker changed the clone destination or draft")
	}
}

func TestCloneFormCreatesAndOpensOnlySuccessfulDestination(t *testing.T) {
	a, _, old := worktreeApp(t)
	_, _, source := worktreeApp(t)
	a.commitText = "old draft"
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	done := make(chan func(), 128)
	a.runOnMain = func(fn func()) { done <- fn }
	t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
	destination := filepath.Join(t.TempDir(), "new checkout")
	a.showCloneRepository()
	a.cloneForm.address = source
	a.cloneForm.path = destination
	a.submitCloneRepository()
	if !a.busy || a.repo.Root != old || a.commitText != "old draft" {
		t.Fatal("Clone switched before successful completion")
	}
	runContinuation(t, done)
	for a.loading {
		runContinuation(t, done)
	}
	canonical, err := filepath.EvalSymlinks(destination)
	if err != nil {
		t.Fatal(err)
	}
	if a.busy || a.cloneForm.open || a.repo.Root != canonical || a.commitText != "" || a.loadErr != "" {
		t.Fatalf("successful clone not opened: root=%s err=%s", a.repo.Root, a.loadErr)
	}
	out, err := exec.Command("git", "-C", destination, "show", "HEAD:base.txt").Output()
	if err != nil || string(out) != "base\n" {
		t.Fatalf("wrong cloned tree: %q %v", out, err)
	}
	remote, err := exec.Command("git", "-C", destination, "remote", "get-url", "origin").Output()
	if err != nil || string(remote) != source+"\n" {
		t.Fatalf("wrong origin: %q %v", remote, err)
	}
}

func TestCloneFailureRetainsFormAndCurrentRepository(t *testing.T) {
	for _, failure := range []string{"occupied", "missing source", "cancelled"} {
		t.Run(failure, func(t *testing.T) {
			a, done, root := worktreeApp(t)
			a.commitText = "retained draft"
			dest := filepath.Join(t.TempDir(), "target")
			source := filepath.Join(t.TempDir(), "missing")
			if failure == "occupied" {
				dest = t.TempDir()
				if err := os.WriteFile(filepath.Join(dest, "keep"), []byte("untouched"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if failure == "cancelled" {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				a.ctx = ctx
			}
			a.showCloneRepository()
			a.cloneForm.address = source
			a.cloneForm.path = dest
			a.submitCloneRepository()
			if a.busy {
				runContinuation(t, done)
			}
			if !a.cloneForm.open || a.cloneForm.address != source || a.cloneForm.path != dest || a.cloneForm.err == "" || a.repo.Root != root || a.commitText != "retained draft" || a.loading || a.busy {
				t.Fatal("failed clone lost form or replaced current checkout")
			}
			if failure == "occupied" {
				data, err := os.ReadFile(filepath.Join(dest, "keep"))
				if err != nil || string(data) != "untouched" {
					t.Fatal("occupied destination was modified")
				}
			}
		})
	}
}

func TestCloneCancellationWinsOverQueuedSuccess(t *testing.T) {
	a, done, root := worktreeApp(t)
	a.commitText = "retained draft"
	dest := filepath.Join(t.TempDir(), "cloned")
	a.showCloneRepository()
	a.cloneForm.address = root
	a.cloneForm.path = dest
	a.submitCloneRepository()
	var completion func()
	select {
	case completion = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("clone did not finish")
	}
	// Git has succeeded, but the UI hasn't accepted its callback. Cancellation
	// must prevent the queued success from switching away from the old draft.
	tt := dialogTester(a, a.cloneRepositoryDialog)
	tt.Key(0, ui.KeyEscape)
	if !a.cloneForm.open || !a.busy {
		t.Fatal("busy Escape hid running clone")
	}
	primary, ok := tt.AccessByRole(ui.RoleButton, a.t("busyClone"))
	if !ok || !primary.Disabled {
		t.Fatal("running Clone can be submitted again")
	}
	a.submitCloneRepository() // Reentry must leave the original job alone.
	for i := 0; i < 8; i++ {
		button, found := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
		if found && button.Focused {
			break
		}
		tt.Key(0, ui.KeyTab)
	}
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
	if !ok || !button.Focused {
		t.Fatal("busy Cancel is not keyboard reachable")
	}
	tt.Key(0, ui.KeyEnter)
	completion()
	if a.repo.Root != root || a.commitText != "retained draft" || a.loading || a.busy || !a.cloneForm.open || !strings.Contains(a.cloneForm.err, dest) {
		t.Fatal("cancelled success switched repository or hid retained destination")
	}
	if _, err := os.Stat(filepath.Join(dest, ".git")); err != nil {
		t.Fatal("test did not reach a successful on-disk clone")
	}
}

func TestCloneFromEmptyStateAcceptsUnbornHead(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	source := filepath.Join(t.TempDir(), "empty.git")
	if out, err := exec.Command("git", "init", "--bare", "-b", "main", source).CombinedOutput(); err != nil {
		t.Fatalf("fixture: %s %v", out, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan func(), 128)
	a := &app{ctx: ctx, locale: localeEnglish, page: "history", runOnMain: func(fn func()) { done <- fn }}
	t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
	dest := filepath.Join(t.TempDir(), "empty checkout")
	a.showCloneRepository()
	a.cloneForm.address = source
	a.cloneForm.path = dest
	a.submitCloneRepository()
	runContinuation(t, done)
	for a.loading {
		runContinuation(t, done)
	}
	canonical, err := filepath.EvalSymlinks(dest)
	if err != nil {
		t.Fatal(err)
	}
	if a.repo == nil || a.repo.Root != canonical || a.status.Head != "" || len(a.commits) != 0 || a.loadErr != "" || a.cloneForm.open || len(a.recentRepositories) != 1 {
		t.Fatal("empty clone was not accepted as an unborn repository")
	}
}

func TestClonedRepositoryOpenFailureExplainsDestination(t *testing.T) {
	a, done, root := worktreeApp(t)
	a.locale = localeEnglish
	a.commitText = "retained draft"
	missing := filepath.Join(t.TempDir(), "cloned-but-unavailable")
	a.cloneOpeningRoot = missing
	a.openRepository(missing, true)
	for a.loading {
		runContinuation(t, done)
	}
	if a.repo.Root != root || a.commitText != "retained draft" || !strings.Contains(a.loadErr, missing) || !strings.Contains(a.loadErr, "Cloned to") || len(a.recentRepositories) != 0 {
		t.Fatal("post-clone open failure lost old checkout or recovery path")
	}
}

func TestCloneEntryAndSubmitGuards(t *testing.T) {
	for _, guard := range []string{"busy", "loading", "chooser", "modal"} {
		t.Run(guard, func(t *testing.T) {
			a := &app{locale: localeEnglish}
			switch guard {
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "modal":
				a.prompt.open = true
			}
			a.showCloneRepository()
			if a.cloneForm.open {
				t.Fatal("Clone opened behind guard")
			}
		})
	}
	a := &app{locale: localeEnglish}
	a.showCloneRepository()
	a.cloneForm.address = "local-source"
	a.cloneForm.path = "relative-target"
	a.submitCloneRepository()
	if a.busy || !a.cloneForm.open || a.cloneForm.err == "" {
		t.Fatal("relative destination must require correction")
	}
}
