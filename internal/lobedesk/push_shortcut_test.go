package lobedesk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func pushShortcut(t *testing.T) appShortcut {
	t.Helper()
	for _, s := range shortcutTable() {
		if s.ID == "repo.pushF8" {
			return s
		}
	}
	t.Fatal("missing F8 ordinary Push shortcut")
	return appShortcut{}
}

func pushShortcutApp(t *testing.T) *app {
	a := headShortcutApp(t)
	a.status.Branch = "main"
	a.remotes = []gitrepo.Remote{{Name: "origin", URL: "/owned/remote.git"}}
	return a
}

func TestPushShortcutGuards(t *testing.T) {
	s := pushShortcut(t)
	if s.Mod != 0 || s.Key != ui.KeyF8 || s.Keys != "F8" || s.Label != "pushButton" {
		t.Fatal("incorrect F8 binding")
	}
	for _, reason := range []string{"nil", "branch", "head", "remote", "busy", "loading", "reload", "chooser", "ref", "confirm", "prompt", "reset", "checkout", "worktree", "blame", "clone", "new", "palette", "quick-checkout", "child", "switch", "details"} {
		t.Run(reason, func(t *testing.T) {
			a := pushShortcutApp(t)
			switch reason {
			case "nil":
				a.repo = nil
			case "branch":
				a.status.Branch = ""
			case "head":
				a.status.Head = ""
			case "remote":
				a.remotes = nil
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "reload":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "ref":
				a.historyRefOpen = true
			case "confirm":
				a.confirm.open = true
				a.confirm.title = "retained"
			case "prompt":
				a.prompt.open = true
				a.prompt.root = a.repo.Root
			case "reset":
				a.reset.open = true
				a.reset.root = a.repo.Root
				a.reset.head, a.reset.branch = a.status.Head, a.status.Branch
			case "checkout":
				a.checkoutBranch.open = true
				a.checkoutBranch.root = a.repo.Root
			case "worktree":
				a.worktreeForm.open = true
				a.worktreeForm.root = a.repo.Root
			case "blame":
				a.blameOpen = true
			case "clone":
				a.cloneForm.open = true
			case "new":
				a.newRepoForm.open = true
			case "details":
				a.noticeDetails[toastMain] = noticeDetailsState{open: true, notice: &toastNotice{root: a.repo.Root, description: "Pending error"}}
			case "palette":
				a.openCommandPalette()
			case "quick-checkout":
				a.quickCheckout.open = true
				a.quickCheckout.root = a.repo.Root
				a.quickCheckout.epoch = a.loadSeq
			case "child":
				a.worktreeSelected = false
				a.childNavigation.open = true
				a.childNavigation.root = a.repo.Root
				a.childNavigation.epoch = a.loadSeq
				a.childNavigation.source = a.selectedSHA
			case "switch":
				a.openQuickRepositorySwitch()
			}
			if s.Available == nil || s.Available(a) {
				t.Fatal("guarded Push offered", reason)
			}
			before := a.confirm.title
			tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
			tt.Key(0, ui.KeyF8)
			s.Run(a)
			if a.confirm.title != before || !a.progressSince.IsZero() || a.busyKey != "" {
				t.Fatal("guarded key or retained callback opened/started Push")
			}
		})
	}
}

func TestPushShortcutActualInputsCancel(t *testing.T) {
	pushShortcut(t)
	for _, opener := range []string{"summary", "body", "search"} {
		for _, cancel := range []string{"escape", "backdrop", "button"} {
			t.Run(opener+"/"+cancel, func(t *testing.T) {
				a := pushShortcutApp(t)
				a.commitText = "title\n\nbody"
				a.amend = true
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					a.registerShortcuts(c)
					ui.Column(c).Fill().Children(func() {
						if opener == "search" {
							a.historyPage(c)
						} else {
							a.commitMessageFields(c)
						}
						a.confirmDialog(c)
					})
				}, 900, 640)
				if opener == "search" {
					tt.Key(ui.Cmd, ui.KeyF)
					tt.Type("needle")
					tt.Frame()
				} else {
					label := a.t("commitSummary")
					if opener == "body" {
						label = a.t("commitDescription")
					}
					field, ok := tt.AccessByRole(ui.RoleTextField, label)
					if !ok {
						t.Fatal("missing opener")
					}
					tt.ClickAt(field.Bounds.X+10, field.Bounds.Y+10)
				}
				tt.Key(ui.Cmd, ui.KeyA)
				tt.Frame()
				for _, mod := range []ui.Modifiers{ui.Shift, ui.Alt, ui.Cmd} {
					tt.Key(mod, ui.KeyF8)
				}
				if a.confirm.open {
					t.Fatal("modified F8 opened Push")
				}
				tt.Key(0, ui.KeyF8)
				tt.Frame()
				if !a.confirm.open || a.busy || !strings.Contains(a.confirm.message, "origin") || !strings.Contains(a.confirm.message, "main") || !strings.Contains(a.confirm.message, shortSHA(a.status.Head)) {
					t.Fatal("F8 did not identify ordinary Push source and target")
				}
				message := a.confirm.message
				tt.Key(0, ui.KeyF8)
				pushShortcut(t).Run(a)
				tt.Frame()
				dialogs := 0
				for _, node := range tt.AccessAll() {
					if node.Role == ui.RoleDialog {
						dialogs++
					}
				}
				if dialogs != 1 || a.confirm.message != message || a.busy {
					t.Fatal("repeat F8 stacked/restarted Push")
				}
				switch cancel {
				case "escape":
					tt.Key(0, ui.KeyEscape)
				case "backdrop":
					tt.ClickAt(2, 2)
				case "button":
					button, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
					if !ok {
						t.Fatal("missing cancel")
					}
					tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
				}
				tt.Frame()
				tt.Frame()
				tt.Type("X")
				tt.Frame()
				if a.confirm.open || a.busy || !a.amend {
					t.Fatal("cancel changed modal/write/amend state")
				}
				switch opener {
				case "summary":
					if a.commitText != "X\n\nbody" {
						t.Fatal("lost summary selection", a.commitText)
					}
				case "body":
					if a.commitText != "title\n\nX" {
						t.Fatal("lost Body selection", a.commitText)
					}
				case "search":
					if a.historyQuery != "X" || a.commitText != "title\n\nbody" {
						t.Fatal("lost search selection/draft")
					}
				}
			})
		}
	}
}

func TestPushShortcutOwnedConfirmation(t *testing.T) {
	pushShortcut(t)
	for _, tracking := range []bool{false, true} {
		t.Run(fmt.Sprint(tracking), func(t *testing.T) {
			a, _, root := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			done := make(chan func(), 64)
			a.runOnMain = func(fn func()) { done <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			bare := filepath.Join(t.TempDir(), "owned remote.git")
			pullFixtureGit(t, root, "init", "--bare", bare)
			pullFixtureGit(t, root, "remote", "add", "origin", bare)
			destination := "main"
			if tracking {
				destination = "published"
				pullFixtureGit(t, root, "push", "-u", "origin", "HEAD:refs/heads/"+destination)
				pullFixtureGit(t, root, "commit", "--allow-empty", "-m", "local advance")
			}
			if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("staged\n"), 0600); err != nil {
				t.Fatal(err)
			}
			pullFixtureGit(t, root, "add", "--", "base.txt")
			if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("unstaged\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("untracked\n"), 0600); err != nil {
				t.Fatal(err)
			}
			var err error
			a.status, err = a.repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.remotes, err = a.repo.Remotes(ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.commitText = "draft\n\nbody"
			want := a.status.Head
			readBytes := func(path string) string {
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			configBefore := readBytes(filepath.Join(root, ".git", "config"))
			fingerprint := func() string {
				b, err := os.ReadFile(filepath.Join(root, "base.txt"))
				if err != nil {
					t.Fatal(err)
				}
				u, err := os.ReadFile(filepath.Join(root, "untracked.txt"))
				if err != nil {
					t.Fatal(err)
				}
				return pullFixtureGit(t, root, "rev-parse", "HEAD") + pullFixtureGit(t, root, "symbolic-ref", "HEAD") + pullFixtureGit(t, root, "ls-files", "--stage") + pullFixtureGit(t, root, "status", "--porcelain=v1") + string(b) + string(u) + readBytes(filepath.Join(root, ".git", "index"))
			}
			before := fingerprint()
			remoteBefore := pullFixtureGit(t, bare, "for-each-ref")
			localRefs := pullFixtureGit(t, root, "for-each-ref")
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.registerShortcuts(c)
				ui.Column(c).Fill().Children(func() { a.commitMessageFields(c); a.confirmDialog(c) })
			}, 900, 640)
			tt.Key(0, ui.KeyF8)
			tt.Frame()
			tt.Key(0, ui.KeyF8)
			tt.Frame()
			if a.busy || !a.confirm.open || pullFixtureGit(t, bare, "for-each-ref") != remoteBefore || pullFixtureGit(t, root, "for-each-ref") != localRefs || fingerprint() != before || readBytes(filepath.Join(root, ".git", "config")) != configBefore {
				t.Fatal("invocation wrote Git before confirmation")
			}
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			if fingerprint() != before || pullFixtureGit(t, bare, "for-each-ref") != remoteBefore || pullFixtureGit(t, root, "for-each-ref") != localRefs || readBytes(filepath.Join(root, ".git", "config")) != configBefore {
				t.Fatal("cancel wrote Git")
			}
			tt.Key(0, ui.KeyF8)
			tt.Frame()
			button, ok := tt.AccessByRole(ui.RoleButton, a.t("pushButton"))
			if !ok {
				t.Fatal("missing actual Push confirmation")
			}
			tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
			tt.Frame()
			if !a.busy {
				t.Fatal("confirmation did not start Push")
			}
			for a.busy || a.loading {
				runContinuation(t, done)
			}
			if pullFixtureGit(t, bare, "rev-parse", "refs/heads/"+destination) != want || fingerprint() != before || a.commitText != "draft\n\nbody" {
				t.Fatal("accepted push changed local content/draft or published wrong OID")
			}
			if pullFixtureGit(t, root, "rev-parse", "--abbrev-ref", "main@{upstream}") != "origin/"+destination {
				t.Fatal("upstream policy changed")
			}
		})
	}
}
