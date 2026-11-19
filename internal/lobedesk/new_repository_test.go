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
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestNewRepositoryEntrancesCancelWithoutChangingDraft(t *testing.T) {
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
				a.newRepositoryDialog(c)
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
		entry, ok := tt.AccessByRole(role, a.t("newRepo"))
		if !ok {
			t.Fatal("missing New repository entrance")
		}
		tt.ClickAt(entry.Bounds.X+entry.Bounds.W/2, entry.Bounds.Y+entry.Bounds.H/2)
		if !a.newRepoForm.open || !a.hasPendingDialog(toastMain) {
			t.Fatal("new repository form did not protect input")
		}
		tt.Key(0, ui.KeyEscape)
		if a.newRepoForm.open || a.commitText != "retained draft" || a.busy || a.loading {
			t.Fatal("dismissal changed checkout context")
		}
	}
}

func TestNewRepositoryFormCreatesUnbornMainAndOpensIt(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty state", true: "existing checkout"}[existing], func(t *testing.T) {
			a := &app{locale: localeEnglish, page: "history"}
			old := ""
			if existing {
				a, _, old = worktreeApp(t)
			} else {
				t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
				t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
			}
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			done := make(chan func(), 128)
			a.runOnMain = func(fn func()) { done <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			a.commitText = "old draft"
			dest := filepath.Join(t.TempDir(), "新建 repo")
			a.showNewRepository()
			a.newRepoForm.path = dest
			tt := dialogTester(a, a.newRepositoryDialog)
			tt.Key(0, ui.KeyEnter)
			if !a.busy || (existing && a.repo.Root != old) || a.commitText != "old draft" {
				t.Fatal("creation did not start from Enter or switched early")
			}
			runContinuation(t, done)
			for a.loading {
				runContinuation(t, done)
			}
			canonical, err := filepath.EvalSymlinks(dest)
			if err != nil {
				t.Fatal(err)
			}
			if a.repo == nil || a.repo.Root != canonical || a.status.Head != "" || len(a.commits) != 0 || a.loadErr != "" || a.historyErr != "" || a.newRepoForm.open || a.commitText != "" || len(a.recentRepositories) != 1 {
				t.Fatal("new unborn repository not accepted correctly")
			}
			head, err := exec.Command("git", "-C", dest, "symbolic-ref", "HEAD").Output()
			if err != nil || string(head) != "refs/heads/main\n" {
				t.Fatalf("wrong default branch: %s %v", head, err)
			}
		})
	}
}

func TestNewRepositoryRejectsOccupiedDestinationAndRetainsInput(t *testing.T) {
	for _, kind := range []string{"empty directory", "file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			a.commitText = "retained draft"
			parent := t.TempDir()
			dest := filepath.Join(parent, "existing")
			switch kind {
			case "empty directory":
				if err := os.Mkdir(dest, 0700); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.WriteFile(dest, []byte("preserve"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(filepath.Join(parent, "missing"), dest); err != nil {
					t.Skip(err)
				}
			}
			a.showNewRepository()
			a.newRepoForm.path = dest
			a.submitNewRepository()
			if a.busy || a.loading || !a.newRepoForm.open || a.newRepoForm.path != dest || a.newRepoForm.err == "" || a.repo.Root != root || a.commitText != "retained draft" {
				t.Fatal("occupied destination changed old checkout or lost correction context")
			}
			if _, err := os.Lstat(dest); err != nil {
				t.Fatal("occupied destination disappeared")
			}
			if kind == "file" {
				data, err := os.ReadFile(dest)
				if err != nil || string(data) != "preserve" {
					t.Fatal("existing file overwritten")
				}
			}
		})
	}
}

func TestNewRepositoryGitFailureCanBeCorrectedAndRetried(t *testing.T) {
	a, _, root := worktreeApp(t)
	a.commitText = "retained draft"
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	done := make(chan func(), 128)
	a.runOnMain = func(fn func()) { done <- fn }
	t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
	failedPath := filepath.Join(t.TempDir(), "failed-init")
	a.showNewRepository()
	a.newRepoForm.path = failedPath
	// This owned, temporary Git environment fails inside git init, after the
	// destination preflight. The subtest restores it before the UI retry.
	t.Run("Git failure", func(t *testing.T) {
		t.Setenv("GIT_DEFAULT_HASH", "invalid-format-for-fixture")
		a.submitNewRepository()
		if !a.busy {
			t.Fatal("test never reached Git init")
		}
		runContinuation(t, done)
		if a.busy || a.loading || !a.newRepoForm.open || a.newRepoForm.path != failedPath || a.newRepoForm.err == "" || a.repo.Root != root || a.commitText != "retained draft" {
			t.Fatal("Git failure lost original context or correction form")
		}
	})
	newPath := filepath.Join(t.TempDir(), "retry")
	a.newRepoForm.path = newPath
	a.submitNewRepository()
	runContinuation(t, done)
	for a.loading {
		runContinuation(t, done)
	}
	canonical, err := filepath.EvalSymlinks(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if a.newRepoForm.open || a.repo.Root != canonical || a.commitText != "" || a.loadErr != "" {
		t.Fatal("corrected new destination could not be created")
	}
}

func TestNewRepositoryEntryAndValidationGuards(t *testing.T) {
	for _, guard := range []string{"busy", "loading", "chooser", "prompt", "clone"} {
		t.Run(guard, func(t *testing.T) {
			a := &app{locale: localeEnglish}
			switch guard {
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "prompt":
				a.prompt.open = true
			case "clone":
				a.cloneForm.open = true
			}
			a.showNewRepository()
			if a.newRepoForm.open {
				t.Fatal("creation opened behind existing operation/dialog")
			}
		})
	}
	for _, path := range []string{"", "   ", "relative/path"} {
		a := &app{locale: localeEnglish}
		a.showNewRepository()
		a.newRepoForm.path = path
		a.submitNewRepository()
		if a.busy || a.loading || !a.newRepoForm.open || a.newRepoForm.err == "" {
			t.Fatal("invalid destination did not retain correction form")
		}
	}
}

func TestNewRepositoryOpenFailureExplainsCreatedPath(t *testing.T) {
	a, done, root := worktreeApp(t)
	a.locale = localeEnglish
	a.commitText = "retained draft"
	path := filepath.Join(t.TempDir(), "created-but-unavailable")
	a.newRepoOpeningRoot = path
	a.openRepository(path, true)
	for a.loading {
		runContinuation(t, done)
	}
	if a.repo.Root != root || a.commitText != "retained draft" || !strings.Contains(a.loadErr, path) || !strings.Contains(a.loadErr, "Created at") || len(a.recentRepositories) != 0 {
		t.Fatal("open failure lost created path or old checkout")
	}
}

func TestNewRepositoryCancelledSuccessRetainsOldCheckout(t *testing.T) {
	a, done, root := worktreeApp(t)
	a.commitText = "retained draft"
	dest := filepath.Join(t.TempDir(), "created")
	a.showNewRepository()
	a.newRepoForm.path = dest
	a.submitNewRepository()
	// Hold Git's successful completion, cancel through the actual dialog button,
	// then accept the callback to prove it cannot unexpectedly replace the root.
	var completion func()
	select {
	case completion = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("init did not finish")
	}
	tt := dialogTester(a, a.newRepositoryDialog)
	tt.Key(0, ui.KeyEscape)
	if !a.newRepoForm.open || !a.busy {
		t.Fatal("busy Escape hid creation")
	}
	primary, ok := tt.AccessByRole(ui.RoleButton, a.t("busyCreateRepo"))
	if !ok || !primary.Disabled {
		t.Fatal("busy creation allowed another submit")
	}
	a.submitNewRepository()
	if _, err := os.Stat(filepath.Join(dest, ".git")); err != nil {
		t.Fatal("held callback did not follow successful creation")
	}
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
	if !ok {
		t.Fatal("missing cancellation")
	}
	tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	completion()
	if a.repo.Root != root || a.commitText != "retained draft" || a.loading || a.busy || !strings.Contains(a.newRepoForm.err, dest) {
		t.Fatal("cancelled success changed checkout or hid created directory")
	}
}
