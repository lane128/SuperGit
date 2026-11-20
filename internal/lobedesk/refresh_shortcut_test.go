package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func refreshBinding(t *testing.T, id string) appShortcut {
	t.Helper()
	for _, s := range shortcutTable() {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("missing refresh shortcut %s", id)
	return appShortcut{}
}

func TestRefreshShortcutsReadExternalChangesWithoutWriting(t *testing.T) {
	for _, id := range []string{"repo.reload", "repo.reloadF5"} {
		t.Run(id, func(t *testing.T) {
			s := refreshBinding(t, id)
			if id == "repo.reloadF5" && (s.Mod != 0 || s.Key != ui.KeyF5 || s.Keys != "F5" || s.Label != "shortcutReload") {
				t.Fatalf("incorrect F5 binding: %+v", s)
			}
			a, callbacks := overviewFixture(t)
			a.locale = localeEnglish
			a.page = "stash"
			a.commitText = "preserve summary\n\nbody\n"
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", append([]string{"-C", a.repo.Root}, args...)...)
				cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v %s", args, err, out)
				}
				return string(out)
			}
			root, head, index, refs := a.repo.Root, git("rev-parse", "HEAD"), git("ls-files", "--stage", "-z"), git("for-each-ref", "--format=%(refname) %(objectname)")
			canonicalRoot, err := filepath.EvalSymlinks(root)
			if err != nil {
				t.Fatal(err)
			}
			name := "external space.txt"
			if err := os.WriteFile(filepath.Join(root, name), []byte("external saved content\n"), 0600); err != nil {
				t.Fatal(err)
			}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				a.registerShortcuts(c)
				ui.Column(c).Width(550).Children(func() { a.commitMessageField(c) })
			}, 600, 400)
			editor, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
			if !ok {
				t.Fatal("missing actual body editor")
			}
			tt.ClickAt(editor.Bounds.X+editor.Bounds.W/2, editor.Bounds.Y+editor.Bounds.H/2)
			seq := a.loadSeq
			if id == "repo.reloadF5" {
				tt.Key(ui.Shift, ui.KeyF5)
				tt.Key(ui.Ctrl, ui.KeyF5)
				if a.loadSeq != seq {
					t.Fatal("modified F5 triggered bare binding")
				}
			}
			tt.Key(s.Mod, s.Key)
			if !a.loading || a.loadSeq != seq+1 {
				t.Fatal("refresh did not start exactly one read")
			}
			tt.Key(s.Mod, s.Key)
			if a.loadSeq != seq+1 {
				t.Fatal("refresh reentered during loading")
			}
			deadline := time.After(15 * time.Second)
			for a.loading {
				select {
				case fn := <-callbacks:
					fn()
				case <-deadline:
					t.Fatal("refresh did not complete")
				}
			}
			found := false
			for _, e := range a.status.Entries {
				found = found || e.Path == name
			}
			if !found {
				t.Fatal("refresh omitted external change")
			}
			tt.Frame()
			if a.repo.Root != canonicalRoot || a.commitText != "preserve summary\n\nbody\n" || !tt.Focused(a.t("commitMessage")) {
				t.Fatalf("refresh state: root=%q want=%q draft=%q focused=%v", a.repo.Root, root, a.commitText, tt.Focused(a.t("commitMessage")))
			}
			if git("rev-parse", "HEAD") != head || git("ls-files", "--stage", "-z") != index || git("for-each-ref", "--format=%(refname) %(objectname)") != refs {
				t.Fatal("refresh changed Git state")
			}
			for path, want := range map[string]string{"same.go": "working version\n", name: "external saved content\n"} {
				b, err := os.ReadFile(filepath.Join(root, path))
				if err != nil || string(b) != want {
					t.Fatalf("refresh changed working bytes: %s", path)
				}
			}
		})
	}
}

func TestRefreshShortcutsRespectManualReadGuards(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*app)
	}{
		{"nil repo", func(a *app) { a.repo = nil }}, {"loading", func(a *app) { a.loading = true }}, {"busy", func(a *app) { a.busy = true }},
		{"chooser", func(a *app) { a.repositoryChooserOpen = true }}, {"confirmation", func(a *app) { a.confirm.open = true; a.confirm.owner = toastMain }},
		{"prompt", func(a *app) { a.prompt.open = true }}, {"reset", func(a *app) { a.reset.open = true }},
		{"checkout", func(a *app) { a.checkoutBranch.open = true }}, {"worktree", func(a *app) { a.worktreeForm.open = true }},
		{"clone", func(a *app) { a.cloneForm.open = true }}, {"new", func(a *app) { a.newRepoForm.open = true }}, {"blame", func(a *app) { a.blameOpen = true }},
	}
	for _, id := range []string{"repo.reload", "repo.reloadF5"} {
		for _, tc := range cases {
			t.Run(id+"/"+tc.name, func(t *testing.T) {
				a, _ := overviewFixture(t)
				s := refreshBinding(t, id)
				if s.Available == nil || !s.Available(a) {
					t.Fatal("valid refresh missing availability check")
				}
				tc.setup(a)
				seq, loading := a.loadSeq, a.loading
				if s.Available(a) {
					t.Fatal("guarded manual refresh available")
				}
				tt := ui.NewTester(func(c *ui.Context) { a.registerShortcuts(c); ui.Column(c).Fill() }, 400, 200)
				tt.Key(s.Mod, s.Key)
				s.Run(a)
				if a.loadSeq != seq || a.loading != loading {
					t.Fatal("guarded manual refresh started read")
				}
			})
		}
	}
}
