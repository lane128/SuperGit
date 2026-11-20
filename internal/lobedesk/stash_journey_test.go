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

// Complete current-workbench journeys over owned disposable repositories.
// Read the saved patch before restoring; never substitute a synthetic Diff.
func TestStashMergeRestoreJourney(t *testing.T)        { runStashMergeRestoreJourney(t, false) }
func TestStashPopConflictRecoveryJourney(t *testing.T) { runStashMergeRestoreJourney(t, true) }

func runStashMergeRestoreJourney(t *testing.T, conflict bool) {
	a, _, root := worktreeApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	a.ctx = ctx
	a.locale, a.page, a.worktreeSelected = localeEnglish, "history", true
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
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(path string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	write("notes.txt", "original notes\n")
	write("control.txt", "preserved bytes\n")
	splitCommitGit(t, root, "add", "--", "notes.txt", "control.txt")
	splitCommitGit(t, root, "commit", "-m", "Shared files")
	splitCommitGit(t, root, "checkout", "-b", "incoming")
	write("base.txt", "incoming tracked change\n")
	splitCommitGit(t, root, "commit", "-am", "Incoming branch change")
	incoming := strings.TrimSpace(splitCommitGit(t, root, "rev-parse", "HEAD"))
	splitCommitGit(t, root, "checkout", "main")
	write("local.txt", "local committed change\n")
	splitCommitGit(t, root, "add", "--", "local.txt")
	splitCommitGit(t, root, "commit", "-m", "Local branch change")
	local := strings.TrimSpace(splitCommitGit(t, root, "rev-parse", "HEAD"))
	write("notes.txt", "staged notes\n")
	splitCommitGit(t, root, "add", "--", "notes.txt")
	write("notes.txt", "staged notes\nunstaged notes\n")
	write("scratch.txt", "untracked bytes\n")
	if conflict {
		write("base.txt", "saved pending base change\n")
	}
	beforeIndex := splitCommitGit(t, root, "ls-files", "--stage")
	a.commitText = "Continue unfinished work\n\nPreserve this draft."
	a.reload()
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
			a.promptDialog(c)
			a.confirmDialog(c)
		})
	}, 1280, 900)
	wait := func() {
		t.Helper()
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		for {
			tt.Frame()
			pending := a.busy || a.loading || a.diffLoading || a.commitDiffLoading || a.commitFilesLoading
			for _, entry := range a.overviewEntries {
				pending = pending || entry.Loading
			}
			if !pending {
				break
			}
			select {
			case fn := <-callbacks:
				fn()
			case <-timer.C:
				t.Fatalf("stash journey did not settle: page=%s busy=%v loading=%v diff=%v commitFiles=%v commitDiff=%v selectedStash=%q seq=%d err=%q overview=%+v", a.page, a.busy, a.loading, a.diffLoading, a.commitFilesLoading, a.commitDiffLoading, a.selectedStash, a.commitSeq, a.commitDiffErr, a.overviewEntries)
			}
		}
		for range 3 {
			tt.Frame()
		}
	}
	control := func(role ui.Role, label string) ui.AccessInfo {
		t.Helper()
		node, ok := tt.AccessByRole(role, label)
		if !ok || node.Offscreen || node.Disabled || node.Bounds.W <= 0 || node.Bounds.H <= 0 || node.Bounds.X+node.Bounds.W > 1281 || node.Bounds.Y+node.Bounds.H > 901 {
			t.Fatalf("unusable %q: %+v; page=%s", label, node, a.page)
		}
		return node
	}
	click := func(role ui.Role, label string) {
		t.Helper()
		node := control(role, label)
		tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
	}
	clickWorktree := func() {
		t.Helper()
		label := a.t("noChangesShort")
		if n := len(a.status.Entries); n > 0 {
			label = a.changesLabel(n)
		}
		click(ui.RoleOption, label)
	}
	shot := func(name string) {
		t.Helper()
		dir := os.Getenv("SUPERGIT_STASH_JOURNEY_SHOTS")
		if conflict && dir != "" {
			dir = filepath.Join(dir, "pop-conflict")
		}
		if dir == "" {
			return
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, tt.Image())
		cerr := f.Close()
		if err != nil || cerr != nil {
			t.Fatalf("image: %v %v", err, cerr)
		}
	}
	wait()
	shot("01-mixed-dirty-worktree")
	// The stash action sits in the title bar; the working tree's row is
	// already the subject, so there is no page to switch to first.
	openSave := func() {
		t.Helper()
		click(ui.RoleMenuButton, a.t("saveStashAction"))
		click(ui.RoleMenuItem, a.t("saveStashUntracked"))
		if !a.prompt.open {
			t.Fatal("stash prompt missing")
		}
	}
	openSave()
	tt.Type("Pending changes before merge")
	click(ui.RoleButton, a.t("cancel"))
	if splitCommitGit(t, root, "ls-files", "--stage") != beforeIndex || read("notes.txt") != "staged notes\nunstaged notes\n" || read("scratch.txt") != "untracked bytes\n" {
		t.Fatal("cancel changed work")
	}
	openSave()
	tt.Type("Pending changes before merge")
	shot("02-save-stash-prompt")
	click(ui.RoleButton, a.t("save"))
	wait()
	if got := splitCommitGit(t, root, "status", "--porcelain=v1"); got != "" {
		t.Fatalf("stash left dirty worktree: %q", got)
	}
	if len(a.stashes) != 1 {
		t.Fatalf("stash count=%d", len(a.stashes))
	}
	stash := a.stashes[0]
	if got := splitCommitGit(t, root, "show", stash.SHA+"^2:notes.txt"); got != "staged notes\n" {
		t.Fatalf("stash Index=%q", got)
	}
	if got := splitCommitGit(t, root, "show", stash.SHA+":notes.txt"); got != "staged notes\nunstaged notes\n" {
		t.Fatalf("stash working=%q", got)
	}
	if got := splitCommitGit(t, root, "show", stash.SHA+"^3:scratch.txt"); got != "untracked bytes\n" {
		t.Fatalf("stash untracked=%q", got)
	}
	node := control(ui.RoleOption, "Incoming branch change")
	tt.RightClickAt(node.Bounds.X+4, node.Bounds.Y+node.Bounds.H/2)
	label := ""
	for _, commit := range a.historySource() {
		if commit.SHA == incoming {
			for _, item := range a.commitMenuItems(commit) {
				if item.Key == "merge" {
					label = item.Label
				}
			}
		}
	}
	if label == "" {
		t.Fatal("incoming merge missing")
	}
	click(ui.RoleMenuItem, label)
	shot("03-merge-clean-worktree")
	click(ui.RoleButton, a.t("mergeIntoCurrent"))
	wait()
	merged := strings.TrimSpace(splitCommitGit(t, root, "rev-parse", "HEAD"))
	if got := strings.TrimSpace(splitCommitGit(t, root, "show", "-s", "--format=%P", "HEAD")); got != local+" "+incoming {
		t.Fatalf("merge parents=%q", got)
	}
	click(ui.RoleOption, stashTitle(stash))
	wait()
	if a.commitDiffErr != "" || a.commitDiff.Text == "" {
		t.Fatalf("saved patch unreadable: %q", a.commitDiffErr)
	}
	for _, text := range []string{"+staged notes", "+unstaged notes", "+untracked bytes"} {
		if _, ok := tt.Find(text); !ok {
			t.Fatalf("saved patch hides %q", text)
		}
	}
	tracked, added := 1, 2 // notes.txt
	if conflict {
		tracked++ // This fixture also changes base.txt to provoke the conflict.
		added++
	}
	if len(a.commitFiles) != tracked+1 || a.stashTrackedCount != tracked || a.stashAdded != added {
		t.Fatalf("the summary must count tracked changes separately from the untracked snapshot: files=%+v tracked=%d added=%d", a.commitFiles, a.stashTrackedCount, a.stashAdded)
	}
	fileHeaders := 0
	for _, row := range a.commitDiff.Lines {
		if strings.HasPrefix(row.Text, "diff --git ") {
			fileHeaders++
		}
		if strings.HasPrefix(row.Text, "diff --git ") || row.Text == "--- /dev/null" || row.Text == "+++ b/scratch.txt" {
			if row.Kind != "meta" || row.Old != 0 || row.New != 0 || row.Hunk != -1 {
				t.Fatalf("file header displayed as changed source: %+v", row)
			}
		}
	}
	if fileHeaders < 2 || a.commitDiff.CanStageHunks {
		t.Fatal("multi-file Stash preview lost boundaries or enabled partial writes")
	}
	shot("04-review-saved-patch-before-restore")
	if conflict {
		click(ui.RoleMenuButton, a.t("stashActions"))
		click(ui.RoleMenuItem, a.t("popStash"))
	} else {
		click(ui.RoleMenuButton, a.t("stashActions"))
		click(ui.RoleMenuItem, a.t("applyStash"))
	}
	wait()
	if conflict {
		if len(a.stashes) != 1 || a.stashes[0].SHA != stash.SHA {
			t.Fatal("conflicting Pop lost recovery stash")
		}
		if a.operation.Kind != "" {
			t.Fatalf("stash conflict wrongly represented as %q operation", a.operation.Kind)
		}
		if !a.operationConflicts() {
			t.Fatalf("expected stash conflict: status=%+v notices=%+v", a.status, a.notices[toastMain])
		}
		if strings.TrimSpace(splitCommitGit(t, root, "rev-parse", "HEAD")) != merged {
			t.Fatal("Pop changed HEAD")
		}
		notesIndex := splitCommitGit(t, root, "show", ":notes.txt")
		t.Logf("post-conflict notes Index=%q; original staged version remains in stash parent", notesIndex)
		if read("notes.txt") != "staged notes\nunstaged notes\n" || read("scratch.txt") != "untracked bytes\n" {
			t.Fatal("Pop lost independent dirty files")
		}
		clickWorktree()
		wait()
		commit, ok := tt.AccessByRole(ui.RoleButton, a.t("commit"))
		if !ok || !commit.Disabled {
			t.Fatal("stash conflict allows commit")
		}
		shot("05-pop-conflict-retains-stash")
		click(ui.RoleButton, a.t("conflictEditorOpen"))
		for s := a.currentConflictEditor(); s != nil && s.Loading; s = a.currentConflictEditor() {
			select {
			case fn := <-callbacks:
				fn()
			case <-time.After(10 * time.Second):
				t.Fatal("conflict read missing")
			}
		}
		tt.Frame()
		click(ui.RoleTextField, a.t("conflictEditorResult"))
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Type("combined incoming and saved work\n")
		tt.Advance(150 * time.Millisecond)
		shot("06-edit-pop-conflict")
		click(ui.RoleButton, a.t("conflictEditorSave"))
		wait()
		if splitCommitGit(t, root, "show", ":notes.txt") != notesIndex || read("notes.txt") != "staged notes\nunstaged notes\n" {
			t.Fatal("conflict editor changed independent restored Index or work")
		}
		if got := splitCommitGit(t, root, "show", ":base.txt"); got != "combined incoming and saved work\n" {
			t.Fatalf("resolved Index=%q", got)
		}
		// Git's failed --index restoration staged notes' whole working copy.
		// Review that version and explicitly restore the intended boundary.
		var stagedTitle string
		for _, group := range a.workingFileGroups() {
			if group.Staged {
				stagedTitle = group.Title
			}
		}
		found := false
		for _, row := range tt.AccessAll() {
			if row.Role == ui.RoleOption && row.Name == "notes.txt" && row.Description == stagedTitle && !row.Offscreen {
				tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
				found = true
				break
			}
		}
		if !found {
			t.Fatal("missing recovered Index review row")
		}
		wait()
		if !a.selectedStaged || a.selectedPath != "notes.txt" {
			t.Fatal("review did not select recovered Index")
		}
		var lineIndex = -1
		for i, line := range a.diff.Lines {
			if line.Kind == "add" && line.Text == "+unstaged notes" {
				lineIndex = i
				break
			}
		}
		if lineIndex < 0 {
			t.Fatalf("recovered Index hides formerly unstaged line: %s", a.diff.Text)
		}
		click(ui.RoleButton, a.lineGutterLabel(a.diff.Lines[lineIndex]))
		gutter := control(ui.RoleButton, a.lineGutterLabel(a.diff.Lines[lineIndex]))
		tt.RightClickAt(gutter.Bounds.X+gutter.Bounds.W/2, gutter.Bounds.Y+gutter.Bounds.H/2)
		click(ui.RoleMenuItem, a.t("unstageLine"))
		wait()
		notesIndex = "staged notes\n"
		if got := splitCommitGit(t, root, "show", ":notes.txt"); got != notesIndex {
			t.Fatalf("corrected recovery Index=%q", got)
		}
		if read("notes.txt") != "staged notes\nunstaged notes\n" {
			t.Fatal("Index correction discarded working bytes")
		}
		shot("07-review-corrected-recovery-index")
		click(ui.RoleTextField, a.t("commitMessage"))
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Type("Resolve recovered work")
		tt.Key(0, ui.KeyEnter)
		tt.Key(0, ui.KeyEnter)
		tt.Type("Keep saved and incoming intent.")
		tt.Advance(150 * time.Millisecond)
		shot("07-ready-to-commit-recovery")
		click(ui.RoleButton, a.t("commit"))
		wait()
		if got := strings.TrimSpace(splitCommitGit(t, root, "show", "-s", "--format=%P", "HEAD")); got != merged {
			t.Fatalf("stash recovery parents=%q", got)
		}
		if splitCommitGit(t, root, "show", "HEAD:base.txt") != "combined incoming and saved work\n" || splitCommitGit(t, root, "show", "HEAD:notes.txt") != notesIndex || read("notes.txt") != "staged notes\nunstaged notes\n" || read("scratch.txt") != "untracked bytes\n" || read("control.txt") != "preserved bytes\n" {
			t.Fatal("recovery commit lost independent content")
		}
		if len(a.stashes) != 1 || a.stashes[0].SHA != stash.SHA {
			t.Fatal("resolution/commit consumed recovery stash")
		}
		if got := splitCommitGit(t, root, "show", stash.SHA+"^2:notes.txt"); got != "staged notes\n" {
			t.Fatal("recovery copy lost original staged content")
		}
		if got := splitCommitGit(t, root, "status", "--porcelain=v1"); got != " M notes.txt\n?? scratch.txt\n" {
			t.Fatalf("recovery remainder=%q", got)
		}
		if got := splitCommitGit(t, root, "show", "-s", "--format=%B", "HEAD"); got != "Resolve recovered work\n\nKeep saved and incoming intent.\n\n" {
			t.Fatalf("recovery message=%q", got)
		}
		shot("08-recovery-commit-completed")
		return
	}
	if got := splitCommitGit(t, root, "show", ":notes.txt"); got != "staged notes\n" {
		t.Fatalf("restored Index=%q", got)
	}
	if read("notes.txt") != "staged notes\nunstaged notes\n" || read("scratch.txt") != "untracked bytes\n" || read("control.txt") != "preserved bytes\n" || read("base.txt") != "incoming tracked change\n" || read("local.txt") != "local committed change\n" {
		t.Fatal("restore lost pending, merged, or unrelated bytes")
	}
	if len(a.stashes) != 1 || a.stashes[0].SHA != stash.SHA {
		t.Fatal("Apply deleted or changed stash")
	}
	if strings.TrimSpace(splitCommitGit(t, root, "rev-parse", "HEAD")) != merged || a.status.Branch != "main" {
		t.Fatal("restore changed branch identity")
	}
	if a.commitText != "Continue unfinished work\n\nPreserve this draft." {
		t.Fatalf("draft=%q", a.commitText)
	}
	if got := splitCommitGit(t, root, "status", "--porcelain=v1"); got != "MM notes.txt\n?? scratch.txt\n" {
		t.Fatalf("restored status=%q", got)
	}
	// A repeated Apply must fail without consuming the recovery copy or
	// replacing the restored dirty work. Inspect actual operation feedback.
	restoredIndex, restoredRefs := splitCommitGit(t, root, "ls-files", "--stage"), splitCommitGit(t, root, "show-ref")
	click(ui.RoleMenuButton, a.t("stashActions"))
	click(ui.RoleMenuItem, a.t("applyStash"))
	wait()
	if splitCommitGit(t, root, "ls-files", "--stage") != restoredIndex || splitCommitGit(t, root, "show-ref") != restoredRefs || read("notes.txt") != "staged notes\nunstaged notes\n" || read("scratch.txt") != "untracked bytes\n" {
		t.Fatal("refused repeated Apply changed work or refs")
	}
	if len(a.stashes) != 1 || a.stashes[0].SHA != stash.SHA {
		t.Fatal("refused Apply lost recovery stash")
	}
	failure := false
	for _, notice := range a.notices[toastMain] {
		if notice.kind == "error" && strings.Contains(notice.description, "git stash:") {
			failure = true
		}
	}
	if !failure {
		t.Fatalf("repeated Apply did not preserve actual Git failure: %+v", a.notices[toastMain])
	}
	clickWorktree()
	wait()
	shot("05-restored-mixed-worktree")
}
