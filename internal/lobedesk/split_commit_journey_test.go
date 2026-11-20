package lobedesk

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// This exercises current workbench controls and real Git. It is headless UI
// evidence; native window focus and platform rendering are separate acceptance.
func TestMixedChangesSplitIntoTwoCommits(t *testing.T) {
	a, _, root := worktreeApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	a.page, a.locale, a.worktreeSelected = "history", localeEnglish, true
	callbacks := make(chan func(), 128)
	a.runOnMain = func(fn func()) { callbacks <- fn }
	t.Cleanup(func() {
		cancel()
		a.resetHistoryTraversal()
		a.resetFileHistory()
		if a.overviewCancel != nil {
			a.overviewCancel()
		}
	})
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("base.txt", "base\nfirst tracked\nsecond tracked\n")
	write("new.txt", "first new\nsecond new\n")
	write("control.txt", "leave unselected\n")
	initial := splitCommitGit(t, root, "rev-parse", "HEAD")
	splitCommitGit(t, root, "tag", "preserved-reference")
	refsBefore := splitCommitGit(t, root, "show-ref")
	var err error
	a.status, err = a.repo.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.commitIdentity = readCommitIdentity(ctx, a.repo, true)
	a.detailTabs().Active = detailTabKey{Summary: true}
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() {
			a.titleBar(c)
			row := ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch)
			a.sidebarSplit.rowWidth = row.Bounds().W
			row.Children(func() {
				a.sidebar(c)
				a.splitHandle(c, &a.sidebarSplit, shareForWidth(a.sidebarSplit.rowWidth-splitterWidth, sidebarWidth), sidebarPaneMin, a.workbenchContentMin(), a.t("resizeSidebarPane"))
				a.contentCard(c)
			})
		})
	}, 1280, 900)
	for range 3 {
		tt.Frame()
	}
	wait := func() {
		t.Helper()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		for a.busy || a.loading || a.diffLoading || a.commitFilesLoading || a.commitDiffLoading {
			select {
			case fn := <-callbacks:
				fn()
			case <-timer.C:
				t.Fatal("workbench operation did not deliver its completion")
			}
		}
		tt.Frame()
	}
	click := func(role ui.Role, name string) {
		t.Helper()
		node, ok := tt.AccessByRole(role, name)
		if !ok {
			var visible []string
			for _, info := range tt.AccessAll() {
				if info.Role == ui.RoleMenuItem || info.Focused {
					visible = append(visible, info.Name)
				}
			}
			t.Fatalf("missing control %q (%v); focused/menu=%v selection=%v target=%s", name, role, visible, a.lineSel, a.selectedPath)
		}
		if node.Offscreen || node.Bounds.W <= 0 || node.Bounds.H <= 0 {
			t.Fatalf("control %q is not visible: %+v", name, node)
		}
		tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
	}
	stageLine := func(path, text string) {
		t.Helper()
		click(ui.RoleOption, path)
		wait()
		var index = -1
		for i, line := range a.diff.Lines {
			if line.Text == text {
				index = i
				break
			}
		}
		if index < 0 {
			t.Fatalf("missing line %q: selected=%q staged=%v tab=%+v loading=%v err=%q diff=%s", text, a.selectedPath, a.selectedStaged, a.detailTabs().Active, a.diffLoading, a.diffErr, a.diff.Text)
		}
		click(ui.RoleButton, a.lineGutterLabel(a.diff.Lines[index]))
		handle, _ := tt.AccessByRole(ui.RoleButton, a.lineGutterLabel(a.diff.Lines[index]))
		tt.RightClickAt(handle.Bounds.X+handle.Bounds.W/2, handle.Bounds.Y+handle.Bounds.H/2)
		click(ui.RoleMenuItem, a.t("stageLine"))
		wait()
	}
	snapshot := func(name string) {
		t.Helper()
		dir := os.Getenv("SUPERGIT_SPLIT_SHOTS")
		if dir == "" {
			return
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(file, tt.Image())
		closeErr := file.Close()
		if encodeErr != nil || closeErr != nil {
			t.Fatalf("save frame: %v %v", encodeErr, closeErr)
		}
	}
	reviewIndex := func(path, want string) {
		t.Helper()
		var title string
		for _, group := range a.workingFileGroups() {
			if group.Staged {
				title = group.Title
			}
		}
		found := false
		for _, node := range tt.AccessAll() {
			if node.Role == ui.RoleOption && node.Name == path && node.Description == title && !node.Offscreen {
				tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("visible staged row missing: %s", path)
		}
		wait()
		if !a.selectedStaged || a.selectedPath != path {
			t.Fatal("review opened the wrong file version")
		}
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("Index review did not render %q", want)
		}
	}
	commit := func(summary, body string) {
		t.Helper()
		click(ui.RoleTextField, a.t("commitSummary"))
		tt.Type(summary)
		click(ui.RoleTextField, a.t("commitDescription"))
		tt.Type(body)
		if a.commitText != summary+"\n\n"+body {
			t.Fatalf("draft = %q", a.commitText)
		}
		if !a.canCommit() {
			t.Fatalf("commit blocked: %s", a.commitDisabledReason())
		}
		// The headless clock is frozen between inputs. Let the 100 ms button
		// color transition settle before evaluating its visual ready state.
		tt.Advance(150 * time.Millisecond)
		snapshot(strings.ReplaceAll(strings.ToLower(summary), " ", "-") + "-ready")
		click(ui.RoleButton, a.t("commit"))
		wait()
		if a.commitText != "" {
			t.Fatalf("successful commit retained draft: %q", a.commitText)
		}
		if got := splitCommitGit(t, root, "log", "-1", "--format=%B"); got != summary+"\n\n"+body+"\n\n" {
			t.Fatalf("commit message = %q", got)
		}
	}
	stageLine("new.txt", "+first new")
	stageLine("base.txt", "+first tracked")
	if got := splitCommitGit(t, root, "show", ":new.txt"); got != "first new\n" {
		t.Fatalf("first new Index = %q", got)
	}
	if got := splitCommitGit(t, root, "show", ":base.txt"); got != "base\nfirst tracked\n" {
		t.Fatalf("first tracked Index = %q", got)
	}
	reviewIndex("new.txt", "+first new")
	if _, ok := tt.Find("+second new"); ok {
		t.Fatal("Index preview includes unstaged content")
	}
	snapshot("first-index-review")
	commit("First logical change", "Include only the first part.\nKeep the second part for later.")
	first := splitCommitGit(t, root, "rev-parse", "HEAD")
	if first == initial || splitCommitGit(t, root, "rev-parse", "HEAD^") != initial {
		t.Fatal("first commit parent changed")
	}
	if got := splitCommitGit(t, root, "show", "HEAD:new.txt"); got != "first new\n" {
		t.Fatalf("first tree new file = %q", got)
	}
	if got := splitCommitGit(t, root, "show", "HEAD:base.txt"); got != "base\nfirst tracked\n" {
		t.Fatalf("first tree tracked = %q", got)
	}
	stageLine("new.txt", "+second new")
	stageLine("base.txt", "+second tracked")
	reviewIndex("new.txt", "+second new")
	commit("Second logical change", "Complete the remaining work.")
	if got := splitCommitGit(t, root, "ls-tree", "-r", "--name-only", "HEAD"); got != "base.txt\nnew.txt\n" {
		t.Fatalf("unexpected committed paths: %q", got)
	}
	final := splitCommitGit(t, root, "rev-parse", "HEAD")
	wantRefs := strings.Replace(refsBefore, strings.TrimSpace(initial)+" refs/heads/main", strings.TrimSpace(final)+" refs/heads/main", 1)
	if got := splitCommitGit(t, root, "show-ref"); got != wantRefs {
		t.Fatalf("unexpected ref changes: %q", got)
	}
	if body, err := os.ReadFile(filepath.Join(root, "control.txt")); err != nil || string(body) != "leave unselected\n" {
		t.Fatalf("unselected file changed: %q %v", body, err)
	}
	if splitCommitGit(t, root, "rev-parse", "HEAD^") != first {
		t.Fatal("second commit parent changed")
	}
	for path, want := range map[string]string{"new.txt": "first new\nsecond new\n", "base.txt": "base\nfirst tracked\nsecond tracked\n"} {
		if got := splitCommitGit(t, root, "show", "HEAD:"+path); got != want {
			t.Fatalf("final %s = %q", path, got)
		}
		bytes, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || string(bytes) != want {
			t.Fatalf("working %s = %q, %v", path, bytes, err)
		}
	}
	if got := splitCommitGit(t, root, "status", "--porcelain=v1"); strings.TrimSpace(got) != "?? control.txt" {
		t.Fatalf("remaining status = %q", got)
	}
	if got := splitCommitGit(t, root, "branch", "--show-current"); got != "main\n" {
		t.Fatalf("branch changed: %q", got)
	}
	// There is no History tab any more: choosing the list's first commit is
	// what leaves the working tree's row.
	if source := a.historySource(); len(source) > 0 {
		a.selectCommit(source[0].SHA)
	}
	wait()
	if a.worktreeSelected || a.selectedSHA == "" {
		t.Fatal("the commit list did not open a committed result")
	}
	snapshot("completed-history")
	// Read the result in the history model used by the workbench, as well as Git.
	if len(a.historySource()) < 2 || a.historySource()[0].Subject != "Second logical change" || a.historySource()[1].Subject != "First logical change" {
		t.Fatalf("history did not show both completed changes: %+v", a.historySource())
	}
}
