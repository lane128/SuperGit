package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func pullShortcut(t *testing.T) appShortcut {
	t.Helper()
	for _, s := range shortcutTable() {
		if s.ID == "repo.pullF7" {
			return s
		}
	}
	t.Fatal("missing F7 Pull shortcut")
	return appShortcut{}
}

func TestPullShortcutGuards(t *testing.T) {
	s := pullShortcut(t)
	if s.Mod != 0 || s.Key != ui.KeyF7 || s.Label != "pull" || s.Keys != "F7" {
		t.Fatal("incorrect Pull binding")
	}
	for _, reason := range []string{"nil", "busy", "loading", "history reload", "chooser", "reference", "confirm", "prompt", "reset", "checkout", "worktree", "blame", "details", "clone", "new", "no upstream", "detached", "gone upstream"} {
		t.Run(reason, func(t *testing.T) {
			a := headShortcutApp(t)
			a.status.Branch = "main"
			a.status.Upstream = "origin/main"
			switch reason {
			case "nil":
				a.repo = nil
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "history reload":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "reference":
				a.historyRefOpen = true
			case "confirm":
				a.confirm.open = true
			case "prompt":
				a.prompt.open = true
			case "reset":
				a.reset.open = true
			case "checkout":
				a.checkoutBranch.open = true
			case "worktree":
				a.worktreeForm.open = true
			case "blame":
				a.blameOpen = true
			case "details":
				a.noticeDetails[toastMain] = noticeDetailsState{open: true, notice: &toastNotice{root: a.repo.Root, description: "Pending error"}}
			case "clone":
				a.cloneForm.open = true
			case "new":
				a.newRepoForm.open = true
			case "no upstream":
				a.status.Upstream = ""
			case "detached":
				a.status.Branch = ""
			case "gone upstream":
				a.branches = []gitrepo.Branch{{Name: "main", Current: true, Upstream: "origin/main", UpstreamGone: true}}
			}
			if s.Available == nil || s.Available(a) {
				t.Fatal("guarded Pull offered")
			}
			tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
			tt.Key(0, ui.KeyF7)
			s.Run(a)
			if !a.progressSince.IsZero() || a.busyKey != "" {
				t.Fatal("guarded or retained action started Pull")
			}
		})
	}
}

func pullFixtureGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_ALLOW_PROTOCOL=file")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestPullShortcutFromActualBodyUsesFastForwardOnly(t *testing.T) {
	for _, scenario := range []string{"fast forward", "dirty", "divergent", "summary", "search"} {
		t.Run(scenario, func(t *testing.T) {
			a, callbacks := overviewFixture(t)
			source := a.repo.Root
			bare := filepath.Join(t.TempDir(), "remote.git")
			peer := filepath.Join(t.TempDir(), "peer")
			pullFixtureGit(t, source, "clone", "--bare", source, bare)
			pullFixtureGit(t, source, "clone", bare, peer)
			pullFixtureGit(t, peer, "config", "user.name", "Pull Fixture")
			pullFixtureGit(t, peer, "config", "user.email", "pull@example.invalid")
			if err := os.WriteFile(filepath.Join(source, "same.go"), []byte("remote new bytes\n"), 0600); err != nil {
				t.Fatal(err)
			}
			pullFixtureGit(t, source, "commit", "--no-gpg-sign", "-am", "remote update")
			pullFixtureGit(t, source, "push", bare, "main")
			want := pullFixtureGit(t, source, "rev-parse", "HEAD")
			if scenario == "dirty" {
				if err := os.WriteFile(filepath.Join(peer, "same.go"), []byte("local dirty bytes\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "divergent" {
				if err := os.WriteFile(filepath.Join(peer, "local.txt"), []byte("local commit bytes\n"), 0600); err != nil {
					t.Fatal(err)
				}
				pullFixtureGit(t, peer, "add", "local.txt")
				pullFixtureGit(t, peer, "commit", "--no-gpg-sign", "-m", "local divergence")
			}
			a.repo = &gitrepo.Repository{Root: peer}
			var err error
			a.status, err = a.repo.Status(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.branches, err = a.repo.Branches(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.locale, a.page = localeEnglish, "changes"
			draft := "retained pull draft\n\nretained body\n\n"
			a.commitText = draft
			beforeHead := pullFixtureGit(t, peer, "rev-parse", "HEAD")
			beforeIndex := pullFixtureGit(t, peer, "ls-files", "--stage", "-z")
			beforeBytes, err := os.ReadFile(filepath.Join(peer, "same.go"))
			if err != nil {
				t.Fatal(err)
			}
			s := pullShortcut(t)
			if scenario == "search" {
				a.page = "history"
			}
			label := a.t("commitDescription")
			if scenario == "summary" {
				label = a.t("commitSummary")
			}
			tt := pageTester(a, func(c *ui.Context) {
				a.registerShortcuts(c)
				if scenario == "search" {
					a.historyPage(c)
				} else {
					a.commitMessageFields(c)
				}
			})
			if scenario == "search" {
				tt.Key(ui.Cmd, ui.KeyF)
				tt.Type("needle")
			} else {
				body, ok := tt.AccessByRole(ui.RoleTextField, label)
				if !ok {
					t.Fatal("missing actual commit editor")
				}
				tt.ClickAt(body.Bounds.X+body.Bounds.W/2, body.Bounds.Y+body.Bounds.H/2)
			}
			for _, mods := range []ui.Modifiers{ui.Shift, ui.Alt, ui.Cmd} {
				tt.Key(mods, ui.KeyF7)
			}
			if a.busy {
				t.Fatal("modified F7 started Pull")
			}
			tt.Key(0, ui.KeyF7)
			if !a.busy || a.busyKey != string(busyPull) {
				t.Fatal("actual Body did not dispatch F7 Pull")
			}
			started := a.progressSince
			tt.Key(0, ui.KeyF7)
			s.Run(a)
			if a.progressSince != started {
				t.Fatal("repeated F7 restarted operation")
			}
			deadline := time.After(15 * time.Second)
			for a.busy || a.loading {
				select {
				case fn := <-callbacks:
					fn()
				case <-deadline:
					t.Fatal("Pull/reload did not finish")
				}
			}
			tt.Frame()
			if a.commitText != draft {
				t.Fatal("Pull lost draft")
			}
			if scenario == "search" {
				tt.Type("X")
				if a.historyQuery != "needleX" {
					t.Fatal("Pull lost search query or editor focus")
				}
			} else if !tt.Focused(label) {
				t.Fatal("Pull lost commit editor focus")
			}
			head := pullFixtureGit(t, peer, "rev-parse", "HEAD")
			afterBytes, err := os.ReadFile(filepath.Join(peer, "same.go"))
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "dirty" && scenario != "divergent" {
				if head != want || string(afterBytes) != "remote new bytes\n" || pullFixtureGit(t, peer, "status", "--porcelain") != "" {
					t.Fatal("Pull did not fast forward exact clean tree")
				}
			} else {
				if head != beforeHead || pullFixtureGit(t, peer, "ls-files", "--stage", "-z") != beforeIndex || string(afterBytes) != string(beforeBytes) {
					t.Fatal("rejected Pull changed local state")
				}
				if len(a.notices[toastMain]) == 0 {
					t.Fatal("rejected Pull lacked feedback")
				}
			}
		})
	}
}
