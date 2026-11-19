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
	"supergit/internal/gitrepo"
)

func stageTrackedBinding(t *testing.T) appShortcut {
	t.Helper()
	for _, s := range shortcutTable() {
		if s.ID == "workspace.stageModified" {
			return s
		}
	}
	t.Fatal("missing Stage tracked changes shortcut")
	return appShortcut{}
}

func TestStageTrackedShortcutTableAndGuards(t *testing.T) {
	s := stageTrackedBinding(t)
	want := "Ctrl+Shift+A"
	if runtime.GOOS == "darwin" {
		want = "⌘⇧A"
	}
	if s.Mod != ui.Cmd|ui.Shift || s.Key != ui.KeyA || s.Keys != want || s.Label != "stageTracked" || s.Group != shortcutGroupWorkspace {
		t.Fatalf("wrong binding: %+v", s)
	}
	testBatchShortcutGuards(t, s)
	for _, entries := range [][]gitrepo.Entry{nil, {{Path: "new.txt", Index: '?', Worktree: '?'}}, {{Path: "added.txt", Index: 'A', Worktree: '.'}}} {
		a := &app{repo: &gitrepo.Repository{Root: t.TempDir()}, page: "history", worktreeSelected: true, status: gitrepo.Status{Entries: entries}}
		if s.Available(a) {
			t.Fatalf("stage tracked available with no tracked worktree changes: %+v", entries)
		}
		tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
		tt.Key(ui.Cmd|ui.Shift, ui.KeyA)
		if a.busy {
			t.Fatal("empty tracked scope started write")
		}
	}
}

func TestStageTrackedShortcutMatchesGitUpdateOnly(t *testing.T) {
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
			write := func(name string, b []byte) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, name), b, 0600); err != nil {
					t.Fatal(err)
				}
			}
			git("init", "-b", "main")
			git("config", "user.name", "Tracked Fixture")
			git("config", "user.email", "tracked@example.invalid")
			for _, name := range []string{"partial.txt", "deleted.txt", "moved.txt", "rename.txt"} {
				write(name, []byte("base\n"))
			}
			write("binary.bin", []byte{0, 1, 255})
			git("add", "--all")
			if !unborn {
				git("commit", "--no-gpg-sign", "-m", "base")
			}
			write("partial.txt", []byte("partial Index\n"))
			write("added.txt", []byte("added Index\n"))
			git("add", "--", "partial.txt", "added.txt")
			git("mv", "--", "rename.txt", "renamed.txt")
			write("renamed.txt", []byte("renamed latest\n"))
			write("partial.txt", []byte("partial latest\n"))
			write("added.txt", []byte("added latest\n"))
			write("binary.bin", []byte{0, 2, 255})
			if err := os.Remove(filepath.Join(root, "deleted.txt")); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(filepath.Join(root, "moved.txt"), filepath.Join(root, "untracked move.txt")); err != nil {
				t.Fatal(err)
			}
			write("sentinel.txt", []byte("never stage\n"))
			write("intent.txt", []byte("intent latest\n"))
			git("add", "-N", "--", "intent.txt")
			// Oracle runs in a separate Index file so the tested Index/status are untouched.
			oracle := filepath.Join(t.TempDir(), "oracle-index")
			indexBytes, err := os.ReadFile(filepath.Join(root, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(oracle, indexBytes, 0600); err != nil {
				t.Fatal(err)
			}
			oracleGit := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
				cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_INDEX_FILE="+oracle)
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("oracle %v: %v %s", args, err, out)
				}
				return string(out)
			}
			oracleGit("add", "-u", "--")
			wantTree := oracleGit("write-tree")
			refs := git("for-each-ref", "--format=%(refname) %(objectname)")
			files := map[string]string{}
			for _, name := range []string{"partial.txt", "added.txt", "binary.bin", "renamed.txt", "untracked move.txt", "sentinel.txt", "intent.txt"} {
				b, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				files[name] = string(b)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			repo := &gitrepo.Repository{Root: root}
			status, err := repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			callbacks := make(chan func(), 128)
			a := &app{ctx: ctx, repo: repo, status: status, page: "history", worktreeSelected: true, locale: localeEnglish, commitText: "draft\n\nbody", runOnMain: func(fn func()) { callbacks <- fn }, filesCollapsed: map[string]bool{filesSectionUnstaged: true, filesSectionStaged: true}, lineSel: map[int]bool{1: true}}
			if unborn {
				a.page = "changes"
			}
			head := status.Head
			tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
			tt.Key(ui.Cmd|ui.Shift, ui.KeyA)
			if !a.busy {
				t.Fatal("stage tracked shortcut did not start")
			}
			deadline := time.After(15 * time.Second)
			for a.busy || a.loading {
				select {
				case fn := <-callbacks:
					fn()
				case <-deadline:
					t.Fatal("stage tracked refresh timeout")
				}
			}
			if got := git("write-tree"); got != wantTree {
				t.Fatalf("Index differs from git add -u: %s != %s", got, wantTree)
			}
			if a.status.Head != head || a.commitText != "draft\n\nbody" || git("for-each-ref", "--format=%(refname) %(objectname)") != refs {
				t.Fatal("changed HEAD/refs/draft")
			}
			for name, want := range files {
				b, err := os.ReadFile(filepath.Join(root, name))
				if err != nil || string(b) != want {
					t.Fatalf("working bytes changed %s", name)
				}
			}
			if got := git("ls-files", "--", "untracked move.txt", "sentinel.txt"); got != "" {
				t.Fatalf("staged untracked paths: %q", got)
			}
			for _, entry := range a.status.Entries {
				if entry.Unstaged() && !isUntracked(entry) {
					t.Fatalf("refresh retained tracked work: %+v", entry)
				}
			}
		})
	}
}
