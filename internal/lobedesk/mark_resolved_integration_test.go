package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Exercise the Lobe menu dispatcher, not the separate conflict-editor save API.
func TestMarkResolvedMenusStageOnlyTheirConflict(t *testing.T) {
	for _, header := range []bool{false, true} {
		name := "file row"
		if header {
			name = "diff header"
		}
		t.Run(name, func(t *testing.T) {
			a, done, root := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			t.Cleanup(cancel)
			git := func(args ...string) string {
				t.Helper()
				out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v\n%s", args, err, out)
				}
				return string(out)
			}
			write := func(path, text string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			read := func(path string) string {
				t.Helper()
				b, err := os.ReadFile(filepath.Join(root, path))
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			write("second.txt", "base\n")
			git("add", "--", "second.txt")
			git("commit", "-m", "two file base")
			git("checkout", "-b", "other")
			write("base.txt", "other\n")
			write("second.txt", "other\n")
			git("commit", "-am", "other side")
			git("checkout", "main")
			write("base.txt", "main\n")
			write("second.txt", "main\n")
			git("commit", "-am", "main side")
			if out, err := exec.Command("git", "-C", root, "merge", "--no-edit", "other").CombinedOutput(); err == nil {
				t.Fatalf("fixture merge did not conflict: %s", out)
			}
			var err error
			a.status, err = a.repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(a.status.Entries) != 2 || !a.status.Entries[0].Conflict || !a.status.Entries[1].Conflict {
				t.Fatal("fixture requires two unmerged files")
			}
			write("base.txt", "accepted resolution\n")
			head, refs, mergeHead := git("rev-parse", "HEAD"), git("for-each-ref", "--format=%(refname) %(objectname)"), git("rev-parse", "MERGE_HEAD")
			otherIndex, otherBytes := git("ls-files", "--stage", "--", "second.txt"), read("second.txt")
			a.selectedPath = "second.txt" // The row menu must follow its own target.
			if header {
				a.selectedPath = "base.txt"
			}
			target := a.status.Entries[0]
			for _, entry := range a.status.Entries {
				if entry.Path == "base.txt" {
					target = entry
				}
			}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				trigger := ui.ButtonBase(c).Label("conflict target").Width(220).Height(30)
				if header {
					a.diffHeaderMenu(c, trigger)
				} else {
					a.fileMenu(c, trigger, target, false)
				}
			}, 500, 500)
			row, ok := tt.AccessByRole(ui.RoleButton, "conflict target")
			if !ok {
				t.Fatal("missing menu trigger")
			}
			tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			action, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("markResolved"))
			if !ok {
				t.Fatal("missing Mark resolved menu action")
			}
			tt.ClickAt(action.Bounds.X+action.Bounds.W/2, action.Bounds.Y+action.Bounds.H/2)
			if !a.busy {
				t.Fatal("menu action did not start its write")
			}
			// Receiving completion waits for the actual Git write. Leave it queued:
			// applying it would also launch an unrelated repository refresh.
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("Mark resolved did not complete")
			}
			if git("ls-files", "--unmerged", "--", "base.txt") != "" || git("show", ":base.txt") != "accepted resolution\n" {
				t.Fatal("target Index was not resolved to current working bytes")
			}
			if git("ls-files", "--stage", "--", "second.txt") != otherIndex || read("second.txt") != otherBytes || read("base.txt") != "accepted resolution\n" {
				t.Fatal("Mark resolved modified unrelated Index or working bytes")
			}
			if git("rev-parse", "HEAD") != head || git("for-each-ref", "--format=%(refname) %(objectname)") != refs || git("rev-parse", "MERGE_HEAD") != mergeHead {
				t.Fatal("Mark resolved completed or changed the merge")
			}
			if err := a.repo.Continue(ctx, "merge"); err == nil {
				t.Fatal("Continue accepted the remaining unmerged file")
			}
			if git("rev-parse", "HEAD") != head || git("ls-files", "--stage", "--", "second.txt") != otherIndex {
				t.Fatal("refused Continue changed Git state")
			}
		})
	}
}
