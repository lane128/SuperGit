package lobedesk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestHunkEnterLeavesTextInputAlone(t *testing.T) {
	a := &app{ctx: canceledContext(), runOnMain: func(func()) {}, locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, worktreeSelected: true, selectedPath: "a.txt", status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "a.txt", Index: '.', Worktree: 'M'}}}, diff: gitrepo.Diff{CanStageHunks: true, Hunks: []gitrepo.Hunk{{Patch: "original patch"}}}}
	a.detailTabs().Open(detailFileKey{Path: "a.txt"})
	value := "retained commit draft"
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() {
			a.diffHunkLine(c, gitrepo.DiffLine{Kind: "hunk", Text: "@@ -1 +1 @@", Hunk: 0}, 500)
			field.Input(c, &value, field.InputProps{}).Label("commit editor").Focus()
		})
	}, 500, 180)
	tt.Key(0, ui.KeyEnter)
	if a.busy || value != "retained commit draft" {
		t.Fatal("text input Enter triggered a hunk action or changed draft")
	}
}

func TestHunkEnterRespectsCurrentTargetAndGuards(t *testing.T) {
	cases := []struct {
		name   string
		change func(*app)
	}{
		{"busy", func(a *app) { a.busy = true }},
		{"loading", func(a *app) { a.loading = true }},
		{"diff loading", func(a *app) { a.diffLoading = true }},
		{"diff error", func(a *app) { a.diffErr = "read failed" }},
		{"history", func(a *app) { a.worktreeSelected = false }},
		{"hidden details", func(a *app) { a.page = "history"; a.historyDetailsHidden = true }},
		{"summary", func(a *app) { a.detailTabs().Active = detailTabKey{Summary: true} }},
		{"ref picker", func(a *app) { a.historyRefOpen = true }},
		{"chooser", func(a *app) { a.repositoryChooserOpen = true }},
		{"prompt", func(a *app) { a.prompt.open = true }},
		{"confirmation", func(a *app) { a.confirm.open = true }},
		{"blame", func(a *app) { a.blameOpen = true }},
		{"worktree form", func(a *app) { a.worktreeForm.open = true }},
		{"reset", func(a *app) { a.reset.open = true }},
		{"line selection", func(a *app) { a.lineSel = map[int]bool{1: true} }},
		{"conflict", func(a *app) { a.status.Entries[0].Conflict = true }},
		{"missing file", func(a *app) { a.status.Entries = nil }},
		{"stale file", func(a *app) { a.selectedPath = "another.txt" }},
		{"stale version", func(a *app) { a.selectedStaged = true }},
		{"stale read", func(a *app) { a.diffSeq++ }},
		{"stale open", func(a *app) { a.loadSeq++ }},
		{"stale patch", func(a *app) { a.diff.Hunks[0].Patch = "replacement patch" }},
		{"missing hunk", func(a *app) { a.diff.Hunks = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{ctx: canceledContext(), runOnMain: func(func()) {}, locale: localeEnglish, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}, worktreeSelected: true, selectedPath: "a.txt", status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "a.txt", Index: '.', Worktree: 'M'}}}, diff: gitrepo.Diff{CanStageHunks: true, Hunks: []gitrepo.Hunk{{Patch: "original patch"}}}}
			a.detailTabs().Open(detailFileKey{Path: "a.txt"})
			line := gitrepo.DiffLine{Kind: "hunk", Text: "@@ -1 +1 @@", Hunk: 0}
			globalActivations := 0
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				c.OnWindowShortcut(0, ui.KeyEnter, func() { globalActivations++ })
				ui.Column(c).Fill().Children(func() { a.diffHunkLine(c, line, 500) })
			}, 500, 120)
			for i := 0; i < 8; i++ {
				row, _ := tt.AccessByRole(ui.RoleButton, line.Text)
				if row.Focused {
					break
				}
				tt.Key(0, ui.KeyTab)
			}
			row, _ := tt.AccessByRole(ui.RoleButton, line.Text)
			if !row.Focused {
				t.Fatal("hunk title not focused")
			}
			tc.change(a)
			busy, seq := a.busy, a.loadSeq
			tt.Key(0, ui.KeyEnter)
			// A replaced row loses focus, so the runtime can offer Enter to the
			// window. Its production guard must then forbid an external file open.
			if globalActivations != 0 && a.canOpenSelectedFile() {
				t.Fatal("Enter on a replaced hunk would open an external application")
			}
			if a.busy != busy || a.loadSeq != seq {
				t.Fatal("guarded or replaced hunk started an operation")
			}
			if tc.name == "line selection" && !a.lineSel[1] {
				t.Fatal("blocked hunk action cleared line selection")
			}
		})
	}
}

// A focused hunk must change only its own Index fragment. Replacing both
// hunks, reversing the version, or letting Enter fall through breaks this test.
func TestHunkEnterAppliesOnlyFocusedReplacement(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(fmt.Sprint("staged=", staged), func(t *testing.T) {
			a, done, root := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			t.Cleanup(cancel)
			git := func(args ...string) string {
				t.Helper()
				out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v %s", args, err, out)
				}
				return string(out)
			}
			lines := make([]string, 30)
			for i := range lines {
				lines[i] = fmt.Sprintf("base line %02d", i+1)
			}
			text := func(values []string) string { return strings.Join(values, "\n") + "\n" }
			original := text(lines)
			path := filepath.Join(root, "hunks.txt")
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			git("add", "--", "hunks.txt")
			git("commit", "-m", "two separated hunks")
			head := git("rev-parse", "HEAD")
			refs := git("show-ref")
			protected := git("ls-files", "--stage", "--", "base.txt")
			changed := append([]string(nil), lines...)
			changed[3] = "first replacement"
			changed[24] = "second replacement"
			working := text(changed)
			if err := os.WriteFile(path, []byte(working), 0600); err != nil {
				t.Fatal(err)
			}
			if staged {
				git("add", "--", "hunks.txt")
			}
			var err error
			a.status, err = a.repo.Status(ctx)
			if err != nil {
				t.Fatal(err)
			}
			a.diff, err = a.repo.Diff(ctx, "hunks.txt", staged)
			if err != nil {
				t.Fatal(err)
			}
			if len(a.diff.Hunks) != 2 {
				t.Fatalf("expected two hunks: %+v", a.diff.Hunks)
			}
			a.page, a.locale, a.worktreeSelected = "history", localeEnglish, true
			a.selectedPath, a.selectedStaged = "hunks.txt", staged
			a.detailTabs().Open(detailFileKey{Path: "hunks.txt", Staged: staged})
			var line gitrepo.DiffLine
			for _, candidate := range a.diff.Lines {
				if candidate.Kind == "hunk" && candidate.Hunk == 1 {
					line = candidate
					break
				}
			}
			if line.Text == "" {
				t.Fatal("second hunk header missing")
			}
			globalActivations := 0
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				// A real window shortcut makes event consumption observable without
				// launching the system's external file opener on the pre-feature RED run.
				c.OnWindowShortcut(0, ui.KeyEnter, func() { globalActivations++ })
				ui.Column(c).Fill().Children(func() { a.diffHunkLine(c, line, 500) })
			}, 500, 150)
			for i := 0; i < 8; i++ {
				row, _ := tt.AccessByRole(ui.RoleButton, line.Text)
				if row.Focused {
					break
				}
				tt.Key(0, ui.KeyTab)
			}
			row, _ := tt.AccessByRole(ui.RoleButton, line.Text)
			if !row.Focused {
				t.Fatal("hunk title is not keyboard reachable")
			}
			tt.Key(0, ui.KeyEnter)
			if !a.busy || globalActivations != 0 {
				t.Fatalf("Enter did not exclusively start hunk action: busy=%v global=%d", a.busy, globalActivations)
			}
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("hunk write did not complete")
			}
			expected := append([]string(nil), lines...)
			if staged {
				expected[3] = "first replacement"
			} else {
				expected[24] = "second replacement"
			}
			if got := git("show", ":hunks.txt"); got != text(expected) {
				t.Fatalf("wrong Index fragment: %q", got)
			}
			if got, err := os.ReadFile(path); err != nil || string(got) != working {
				t.Fatalf("working file changed: %q %v", got, err)
			}
			if git("rev-parse", "HEAD") != head || git("show-ref") != refs || git("ls-files", "--stage", "--", "base.txt") != protected {
				t.Fatal("hunk action changed HEAD, refs or unrelated Index")
			}
		})
	}
}
