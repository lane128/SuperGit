package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func unstageAllBinding(t *testing.T) appShortcut {
	t.Helper()
	for _, s := range shortcutTable() {
		if s.ID == "workspace.unstageAll" {
			return s
		}
	}
	t.Fatal("missing Unstage All shortcut")
	return appShortcut{}
}

func TestUnstageAllShortcutTable(t *testing.T) {
	s := unstageAllBinding(t)
	want := "Ctrl+Shift+R"
	if runtime.GOOS == "darwin" {
		want = "⌘⇧R"
	}
	if s.Mod != ui.Cmd|ui.Shift || s.Key != ui.KeyR || s.Keys != want || s.Label != "unstageAll" || s.Group != shortcutGroupWorkspace || s.Field || s.Available == nil {
		t.Fatalf("incorrect binding: %+v", s)
	}
}

func TestUnstageAllShortcutRealIndexAndRefresh(t *testing.T) {
	for _, unborn := range []bool{false, true} {
		t.Run(map[bool]string{false: "committed", true: "unborn"}[unborn], func(t *testing.T) {
			root := t.TempDir()
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
				cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_OPTIONAL_LOCKS=0")
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v %s", args, err, out)
				}
				return string(out)
			}
			write := func(name string, data []byte) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			git("init", "-b", "main")
			git("config", "user.name", "Shortcut Fixture")
			git("config", "user.email", "shortcut@example.invalid")
			if !unborn {
				for _, name := range []string{"partial.txt", "deleted.txt", "old name.txt"} {
					write(name, []byte("baseline\n"))
				}
				git("add", "--all")
				git("commit", "--no-gpg-sign", "-m", "base")
				git("mv", "--", "old name.txt", "新 name.txt")
				git("rm", "--", "deleted.txt")
				write("partial.txt", []byte("Index version\n"))
			}
			write("added.txt", []byte("added\n"))
			write("binary.bin", []byte{0, 255, 10})
			git("add", "--all")
			// Later edits must survive even when the unborn Index differs from disk.
			write("added.txt", []byte("added, then edited again\n"))
			write("binary.bin", []byte{0, 255, 10, 42})
			if !unborn {
				write("partial.txt", []byte("working version\n"))
			}
			write("untracked.txt", []byte("keep untracked\n"))
			refs := git("for-each-ref", "--format=%(refname) %(objectname)")
			files := map[string]string{}
			for _, name := range []string{"partial.txt", "新 name.txt", "added.txt", "binary.bin", "untracked.txt"} {
				data, err := os.ReadFile(filepath.Join(root, name))
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				files[name] = string(data)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			repo := &gitrepo.Repository{Root: root}
			status, err := repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			callbacks := make(chan func(), 128)
			a := &app{ctx: ctx, repo: repo, status: status, page: "history", worktreeSelected: true, locale: localeEnglish, commitText: "retain subject\n\nretain body", runOnMain: func(fn func()) { callbacks <- fn }}
			// A folded staged group and a diff selection must not narrow an all-Index action.
			a.filesCollapsed = map[string]bool{"staged": true}
			a.lineSel = map[int]bool{1: true}
			head := status.Head
			if unborn {
				a.page = "changes"
			}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.registerShortcuts(c)
				field.Input(c, &a.commitText, field.InputProps{}).Label("Draft")
			}, 400, 200)
			if unborn {
				editor, ok := tt.AccessByRole(ui.RoleTextField, "Draft")
				if !ok {
					t.Fatal("missing draft input")
				}
				tt.ClickAt(editor.Bounds.X+editor.Bounds.W/2, editor.Bounds.Y+editor.Bounds.H/2)
			}
			tt.Key(ui.Shift, ui.KeyR)
			if a.busy {
				t.Fatal("wrong modifier started Unstage All")
			}
			tt.Key(ui.Cmd|ui.Shift, ui.KeyR)
			if !a.busy {
				t.Fatal("shortcut did not start Unstage All")
			}
			deadline := time.After(15 * time.Second)
			for a.busy || a.loading {
				select {
				case fn := <-callbacks:
					fn()
				case <-deadline:
					t.Fatal("operation/refresh did not complete")
				}
			}
			if a.status.Head != head || a.commitText != "retain subject\n\nretain body" || git("for-each-ref", "--format=%(refname) %(objectname)") != refs {
				t.Fatal("shortcut changed HEAD, refs or draft")
			}
			for _, e := range a.status.Entries {
				if e.Staged() {
					t.Fatalf("refresh retained staged entry: %+v", e)
				}
			}
			if unborn {
				if got := git("ls-files", "--stage", "-z"); got != "" {
					t.Fatalf("unborn Index not empty: %q", got)
				}
			} else {
				git("diff", "--cached", "--exit-code", "HEAD", "--")
			}
			for name, want := range files {
				data, err := os.ReadFile(filepath.Join(root, name))
				if err != nil || string(data) != want {
					t.Fatalf("working file changed: %s", name)
				}
			}
			for _, name := range []string{"deleted.txt", "old name.txt"} {
				if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
					t.Fatalf("unstage restored working path %s", name)
				}
			}
		})
	}
}

func TestUnstageAllShortcutGuards(t *testing.T) {
	testBatchShortcutGuards(t, unstageAllBinding(t))
}

func testBatchShortcutGuards(t *testing.T, s appShortcut) {
	t.Helper()
	cases := []struct {
		name  string
		setup func(*app)
	}{
		{"nil repo", func(a *app) { a.repo = nil }}, {"empty Index", func(a *app) { a.status.Entries = nil }},
		{"busy", func(a *app) { a.busy = true }}, {"loading", func(a *app) { a.loading = true }},
		{"commit selected", func(a *app) { a.worktreeSelected = false }}, {"other page", func(a *app) { a.page = "stash" }},
		{"hidden detail", func(a *app) { a.historyDetailsHidden = true }}, {"chooser", func(a *app) { a.repositoryChooserOpen = true }},
		{"ref picker", func(a *app) { a.historyRefOpen = true }}, {"confirmation", func(a *app) { a.confirm.open = true; a.confirm.owner = toastMain }},
		{"prompt", func(a *app) { a.prompt.open = true }}, {"reset", func(a *app) { a.reset.open = true }},
		{"checkout", func(a *app) { a.checkoutBranch.open = true }}, {"worktree", func(a *app) { a.worktreeForm.open = true }},
		{"clone", func(a *app) { a.cloneForm.open = true }}, {"new repository", func(a *app) { a.newRepoForm.open = true }},
		{"blame", func(a *app) { a.blameOpen = true }}, {"conflict", func(a *app) { a.status.Entries[0].Conflict = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{ctx: canceledContext(), runOnMain: func(func()) {}, repo: &gitrepo.Repository{Root: t.TempDir()}, page: "history", worktreeSelected: true, status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "a.txt", Index: 'M', Worktree: 'M'}}}}
			if !s.Available(a) {
				t.Fatal("valid working-tree context unavailable")
			}
			tc.setup(a)
			beforeBusy, beforeSeq := a.busy, a.loadSeq
			if s.Available(a) {
				t.Fatal("guarded shortcut remains available")
			}
			tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
			tt.Key(s.Mod, s.Key)
			// A closure retained before the state change must recheck at execution too.
			s.Run(a)
			if a.busy != beforeBusy || a.loadSeq != beforeSeq {
				t.Fatal("guarded shortcut started operation")
			}
		})
	}
}
