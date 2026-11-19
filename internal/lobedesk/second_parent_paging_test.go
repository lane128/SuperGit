package lobedesk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// A real merge whose independent second-parent branch is outside the first page.
func TestSecondParentActualKeyPagesRealMerge(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		name := "success"
		if cancel {
			name = "selection cancels late page"
		}
		t.Run(name, func(t *testing.T) {
			a, callbacks, all := automaticHistoryFixture(t, 420)
			t.Cleanup(a.resetHistoryTraversal)
			first, base := all[0], all[350]
			tree := pullFixtureGit(t, a.repo.Root, "rev-parse", first+"^{tree}")
			t.Setenv("GIT_COMMITTER_DATE", "1500000000 +0000")
			second := pullFixtureGit(t, a.repo.Root, "commit-tree", tree, "-p", base, "-m", "Independent second parent")
			t.Setenv("GIT_COMMITTER_DATE", "2000000000 +0000")
			merge := pullFixtureGit(t, a.repo.Root, "commit-tree", tree, "-p", first, "-p", second, "-m", "Paged merge")
			pullFixtureGit(t, a.repo.Root, "update-ref", "refs/heads/second-parent-route", merge)
			// Every descendant must precede the target in the ordinary topo traversal,
			// independently of which of the merge's parent branches Git visits first.
			var input strings.Builder
			for i := range 240 {
				message := fmt.Sprintf("Second-parent descendant %03d", i)
				fmt.Fprintf(&input, "commit refs/heads/second-parent-descendants\ncommitter Fixture <fixture@example.invalid> %d +0000\ndata %d\n%s\n", 1700010000+i, len(message), message)
				if i == 0 {
					fmt.Fprintf(&input, "from %s\n", second)
				}
				input.WriteByte('\n')
			}
			input.WriteString("done\n")
			command := exec.Command("git", "-C", a.repo.Root, "fast-import", "--quiet")
			command.Stdin = strings.NewReader(input.String())
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("descendant fixture: %v %s", err, output)
			}
			page, err := a.repo.HistoryPage(a.ctx, nil, 0, 200)
			if err != nil {
				t.Fatal(err)
			}
			a.commits, a.historyTips, a.historyMore = page.Commits, page.Tips, page.HasMore
			found := false
			for _, commit := range a.commits {
				if commit.SHA == second {
					t.Fatal("second parent unexpectedly loaded in first page")
				}
				if commit.SHA == merge {
					found = true
					if len(commit.Parents) != 2 || commit.Parents[0] != first || commit.Parents[1] != second {
						t.Fatal("Git merge parent order differs")
					}
				}
			}
			if !found || !a.historyMore {
				t.Fatal("missing paged merge fixture")
			}
			a.page, a.worktreeSelected, a.selectedSHA = "history", false, merge
			a.commitText, a.amend = "retained draft\n\nbody", true
			fingerprint := func() string {
				b, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
				if err != nil {
					t.Fatal(err)
				}
				return pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") + pullFixtureGit(t, a.repo.Root, "symbolic-ref", "HEAD") + pullFixtureGit(t, a.repo.Root, "for-each-ref") + pullFixtureGit(t, a.repo.Root, "ls-files", "--stage") + pullFixtureGit(t, a.repo.Root, "status", "--porcelain=v1") + string(b)
			}
			before := fingerprint()
			tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.historyPage(c) })
			row, ok := tt.AccessByRole(ui.RoleOption, "Paged merge")
			if !ok {
				t.Fatal("real merge row missing")
			}
			tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			for a.commitFilesLoading || a.commitDiffLoading {
				historyCompletion(t, callbacks)()
			}
			tt.Key(ui.Alt|ui.Shift, ui.KeyDown)
			if a.selectedSHA != merge || a.pendingJumpSHA != second || a.canNavigateHistory(-1) {
				t.Fatal("actual second-parent key did not begin an uncommitted paging jump")
			}
			done := historyCompletion(t, callbacks)
			expected := second
			if cancel {
				expected = first
				a.selectCommit(first)
			}
			done()
			for a.historyLoadingMore || a.pendingJumpSHA != "" || a.commitFilesLoading || a.commitDiffLoading {
				historyCompletion(t, callbacks)()
			}
			tt.Frame()
			if a.selectedSHA != expected {
				t.Fatalf("selected %s want %s", a.selectedSHA, expected)
			}
			if cancel {
				if a.canNavigateHistory(-1) || a.historyNavigation.pending != nil {
					t.Fatal("manual selection retained old second-parent route")
				}
			} else {
				target, ok := tt.AccessByRole(ui.RoleOption, "Independent second parent")
				if !ok || target.Bounds.Y < a.historyViewport.Y || target.Bounds.Y+target.Bounds.H > a.historyViewport.Y+a.historyViewport.H {
					t.Fatal("paged second parent was not revealed")
				}
				a.navigateHistory(-1)
				if a.selectedSHA != merge {
					t.Fatal("Back did not return to invoking merge")
				}
				a.navigateHistory(1)
				if a.selectedSHA != second {
					t.Fatal("Forward did not return to exact second parent")
				}
			}
			for a.commitFilesLoading || a.commitDiffLoading {
				historyCompletion(t, callbacks)()
			}
			if fingerprint() != before || a.commitText != "retained draft\n\nbody" || !a.amend {
				t.Fatal("second-parent browsing changed Git or draft")
			}
		})
	}
}
