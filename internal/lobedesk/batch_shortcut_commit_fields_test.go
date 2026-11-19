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

func TestBatchShortcutsFromActualCommitFieldsPreserveDraftAndFocus(t *testing.T) {
	for _, action := range []string{"stage", "unstage"} {
		for _, label := range []string{"commitSummary", "commitDescription"} {
			t.Run(action+"/"+label, func(t *testing.T) {
				a, callbacks := overviewFixture(t)
				a.page = "history"
				a.locale = localeEnglish
				draft := "summary preserved\n\nfirst body line\nsecond body line\n\n"
				a.commitText = draft
				readGit := func(args ...string) string {
					t.Helper()
					out, err := exec.Command("git", append([]string{"-C", a.repo.Root}, args...)...).CombinedOutput()
					if err != nil {
						t.Fatalf("git %v: %v %s", args, err, out)
					}
					return string(out)
				}
				head, refs := readGit("rev-parse", "HEAD"), readGit("for-each-ref", "--format=%(refname) %(objectname)")
				key, want := ui.KeyA, "working version\n"
				if action == "unstage" {
					key, want = ui.KeyR, "original\n"
				}
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					a.registerShortcuts(c)
					ui.Column(c).Width(580).Children(func() { a.commitMessageFields(c) })
				}, 600, 400)
				editor, ok := tt.AccessByRole(ui.RoleTextField, a.t(label))
				if !ok {
					t.Fatal("missing actual commit editor")
				}
				tt.ClickAt(editor.Bounds.X+editor.Bounds.W/2, editor.Bounds.Y+editor.Bounds.H/2)
				if !tt.Focused(a.t(label)) {
					t.Fatal("commit field did not receive focus")
				}
				tt.Key(ui.Cmd, ui.KeyA)
				tt.Key(ui.Cmd, ui.KeyC)
				wantCopy := "summary preserved"
				if label == "commitDescription" {
					wantCopy = "first body line\nsecond body line\n\n"
				}
				if tt.Clipboard() != wantCopy || a.busy || a.commitText != draft {
					t.Fatal("plain Select All/Copy changed batch state or editor behavior")
				}
				tt.Key(ui.Cmd|ui.Shift, key)
				if !a.busy {
					t.Fatal("commit field swallowed batch shortcut")
				}
				deadline := time.After(15 * time.Second)
				for a.busy || a.loading {
					select {
					case fn := <-callbacks:
						fn()
					case <-deadline:
						t.Fatal("batch/refresh did not complete")
					}
				}
				tt.Frame()
				if a.commitText != draft || a.commitFields.Source != draft || !tt.Focused(a.t(label)) {
					t.Fatal("batch shortcut changed draft projection or editor focus")
				}
				if readGit("show", ":same.go") != want || readGit("rev-parse", "HEAD") != head || readGit("for-each-ref", "--format=%(refname) %(objectname)") != refs {
					t.Fatal("wrong Index, HEAD or refs after batch shortcut")
				}
				bytes, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
				if err != nil || string(bytes) != "working version\n" {
					t.Fatal("batch shortcut changed worktree bytes")
				}
			})
		}
	}
}
