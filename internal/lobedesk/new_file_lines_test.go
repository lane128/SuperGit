package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Preserve exact bytes: tree contents and commit messages include final newlines.
func splitCommitGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestNewFileLineMenuStagesOnlySelectedContent(t *testing.T) {
	testNewFileLineMenu(t, false)
}

func TestNewFileLineMenuUnstagesOnlySelectedContent(t *testing.T) {
	testNewFileLineMenu(t, true)
}

func testNewFileLineMenu(t *testing.T, staged bool) {
	t.Helper()
	a, callbacks := overviewFixture(t)
	a.page, a.locale = "changes", localeEnglish
	path := "new.txt"
	if err := os.WriteFile(filepath.Join(a.repo.Root, path), []byte("first\nsecond\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if staged {
		if err := a.repo.Stage(a.ctx, path); err != nil {
			t.Fatal(err)
		}
	}
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.selectedPath, a.selectedStaged = path, staged
	a.diff, err = a.repo.Diff(a.ctx, path, staged)
	if err != nil {
		t.Fatal(err)
	}
	a.detailTabs().Open(detailFileKey{Path: path, Staged: staged})
	var index int = -1
	for i, line := range a.diff.Lines {
		if line.Text == "+first" {
			index = i
		}
	}
	if index < 0 {
		t.Fatal("new-file patch missing")
	}
	line := a.diff.Lines[index]
	tt := pageTester(a, func(c *ui.Context) { a.diffLine(c, line, 700, index) })
	handle, ok := tt.AccessByRole(ui.RoleButton, a.lineGutterLabel(line))
	if !ok {
		t.Fatal("new-file selection handle missing")
	}
	tt.ClickAt(handle.Bounds.X+handle.Bounds.W/2, handle.Bounds.Y+handle.Bounds.H/2)
	tt.Key(0, ui.KeyContextMenu)
	label := "stageLine"
	want := "first\n"
	if staged {
		label, want = "unstageLine", "second\n"
	}
	item, ok := tt.AccessByRole(ui.RoleMenuItem, a.t(label))
	if !ok {
		t.Fatal("new text file offers no Stage line menu")
	}
	if _, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("discardLine")); ok {
		t.Fatal("new-file staging also enabled destructive line discard")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	if !a.busy {
		t.Fatal("Stage line did not start")
	}
	runContinuation(t, callbacks)
	if got := splitCommitGit(t, a.repo.Root, "show", ":new.txt"); got != want {
		t.Fatalf("Index = %q", got)
	}
	if got := splitCommitGit(t, a.repo.Root, "show", ":same.go"); got != "index version\n" {
		t.Fatalf("unrelated Index = %q", got)
	}
	if body, err := os.ReadFile(filepath.Join(a.repo.Root, path)); err != nil || string(body) != "first\nsecond\n" {
		t.Fatalf("working file = %q, %v", body, err)
	}
	for a.loading {
		runContinuation(t, callbacks)
	}
}
