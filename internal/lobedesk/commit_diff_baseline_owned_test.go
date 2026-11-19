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
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestHistoricalDiffBaselineLateRead(t *testing.T) {
	a, _, root := worktreeApp(t)
	a.locale = localeEnglish
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("after-marker\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "base.txt"}, {"commit", "-m", "after"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("fixture: %v %s", err, out)
		}
	}
	commits, err := a.repo.History(context.Background(), 10)
	if err != nil || len(commits) != 2 {
		t.Fatalf("history: %v", err)
	}
	a.commits = commits
	callbacks := make(chan func(), 8)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	collect := func() []func() {
		t.Helper()
		var out []func()
		// Each unsigned owned commit produces its signature and full-detail completion.
		for len(out) < 2 {
			select {
			case fn := <-callbacks:
				out = append(out, fn)
			case <-time.After(10 * time.Second):
				t.Fatal("detail callbacks timed out")
			}
		}
		return out
	}
	a.selectCommit(commits[1].SHA)
	old := collect()
	a.selectCommit(commits[0].SHA)
	a.commitText = "draft\n\nbody"
	a.amend = true
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.historyDiffFile(c, theme.TokensOf(c)) })
	}, 300, 320)
	expected := fmt.Sprintf("First parent %s → Commit %s", shortSHA(commits[1].SHA), shortSHA(commits[0].SHA))
	if !tt.HasText(expected) || !tt.HasText(a.t("diffLoading")) {
		t.Fatal("new selection lacks baseline/loading feedback")
	}
	for _, fn := range collect() {
		fn()
	}
	for _, fn := range old {
		fn()
	}
	tt.Frame()
	if !tt.HasText(expected) || !strings.Contains(a.commitDiff.Text, "+after-marker") || a.commitDiffLoading || a.commitDiffErr != "" {
		t.Fatal("late old read changed new baseline or patch")
	}
	header, ok := tt.AccessByRole(ui.RoleButton, "base.txt")
	if !ok || header.Bounds.Y < 0 || header.Bounds.Y+header.Bounds.H > 320 {
		t.Fatal("narrow pane cannot display baseline and patch together")
	}
	if a.selectedSHA != commits[0].SHA || a.commitText != "draft\n\nbody" || !a.amend {
		t.Fatal("late read changed target or draft")
	}
}

func TestHistoricalDiffBaselineOwnedContent(t *testing.T) {
	a, _, root := worktreeApp(t)
	a.locale = localeEnglish
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	initial := git("rev-parse", "HEAD")
	write("first.txt", "first-parent-only\n")
	git("add", "first.txt")
	git("commit", "-m", "first-parent")
	first := git("rev-parse", "HEAD")
	git("switch", "-c", "topic", initial)
	write("second.txt", "second-parent-only\n")
	git("add", "second.txt")
	git("commit", "-m", "second-parent")
	second := git("rev-parse", "HEAD")
	git("switch", "main")
	git("merge", "--no-ff", "topic", "-m", "merge")
	merge := git("rev-parse", "HEAD")
	git("commit", "--allow-empty", "-m", "empty")
	empty := git("rev-parse", "HEAD")
	write("base.txt", "staged\n")
	git("add", "base.txt")
	write("base.txt", "working\n")
	write("untracked.txt", "keep\n")
	snapshot := func() string {
		t.Helper()
		index, err := os.ReadFile(filepath.Join(root, ".git", "index"))
		if err != nil {
			t.Fatal(err)
		}
		return git("rev-parse", "HEAD") + git("show-ref") + git("status", "--porcelain=v1", "--untracked-files=all") + string(index) + git("diff", "--binary") + git("diff", "--cached", "--binary")
	}
	before := snapshot()
	commits, err := a.repo.History(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	a.commits = commits
	a.commitText = "draft\n\nbody"
	a.amend = true
	for _, tc := range []struct{ name, sha, parent, added, absent string }{
		{"root", initial, "", "+base", ""},
		{"ordinary", first, initial, "+first-parent-only", "second-parent-only"},
		{"merge", merge, first, "+second-parent-only", "first-parent-only"},
		{"empty", empty, merge, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files, err := a.repo.ShowAll(context.Background(), tc.sha)
			if err != nil {
				t.Fatal(err)
			}
			a.selectedSHA = tc.sha
			a.commitDiff, a.commitSections = buildCommitDiff(files)
			if tc.added != "" && !strings.Contains(a.commitDiff.Text, tc.added) {
				t.Fatalf("missing addition %q", tc.added)
			}
			if tc.absent != "" && strings.Contains(a.commitDiff.Text, tc.absent) {
				t.Fatalf("diff used wrong parent: %q", tc.absent)
			}
			expected := "Root commit; no parent"
			if tc.parent != "" {
				expected = fmt.Sprintf("First parent %s → Commit %s", shortSHA(tc.parent), shortSHA(tc.sha))
			}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				ui.Column(c).Fill().Children(func() { a.historyDiffFile(c, theme.TokensOf(c)) })
			}, 700, 500)
			if !tt.HasText(expected) {
				t.Fatalf("missing content-matched baseline %q", expected)
			}
			if tc.name == "merge" {
				selected, ok := a.selectedCommit()
				if !ok || len(selected.Parents) != 2 || selected.Parents[0] != first || selected.Parents[1] != second || first == second {
					t.Fatal("merge fixture parent order incorrect")
				}
			}
			if tc.name == "empty" && (len(a.commitDiff.Lines) != 0 || !tt.HasText(a.t("diffEmpty"))) {
				t.Fatal("same-tree commit not empty")
			}
			if a.selectedSHA != tc.sha || a.commitText != "draft\n\nbody" || !a.amend {
				t.Fatal("render changed draft/target")
			}
		})
	}
	if after := snapshot(); after != before {
		t.Fatal("diff loading/render changed owned repository")
	}
	data, err := os.ReadFile(filepath.Join(root, "untracked.txt"))
	if err != nil || string(data) != "keep\n" {
		t.Fatal("untracked changed")
	}
}
