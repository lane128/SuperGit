package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"supergit/internal/gitrepo"
)

func selectionGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

func workspaceSelectionFixture(t *testing.T) (*app, chan func(), *ui.Tester) {
	t.Helper()
	a, callbacks := overviewFixture(t)
	root := a.repo.Root
	for _, path := range []string{"b.go", "c.go"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte("original\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	selectionGit(t, root, "add", "--", "b.go", "c.go")
	selectionGit(t, root, "commit", "--no-gpg-sign", "-m", "controls", "--only", "--", "b.go", "c.go")
	for _, path := range []string{"b.go", "c.go", "untracked.go"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte("working version\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var err error
	a.status, err = a.repo.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a.page, a.locale = "history", localeEnglish
	input := "draft text"
	tt := pageTester(a, func(c *ui.Context) {
		a.registerShortcuts(c)
		ui.Column(c).FillWidth().Children(func() {
			for _, group := range a.workingFileGroups() {
				for _, entry := range group.Entries {
					a.fileRow(c.Key(group.Key+entry.Path), entry, group.Staged).Description(group.Key)
				}
			}
			field.Input(c, &input, field.InputProps{}).Label("selection-input")
		})
	})
	return a, callbacks, tt
}

func waitSelectionOperation(t *testing.T, a *app, callbacks chan func()) {
	t.Helper()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for a.busy || a.loading {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("selected file operation did not finish")
		}
	}
}

func selectionRow(t *testing.T, tt *ui.Tester, path, group string) ui.AccessInfo {
	t.Helper()
	for _, node := range tt.AccessAll() {
		if node.Role == ui.RoleOption && node.Name == path && node.Description == group {
			return node
		}
	}
	t.Fatalf("missing %s row in %s", path, group)
	return ui.AccessInfo{}
}

func clickSelectionRow(t *testing.T, tt *ui.Tester, path, group string, mods ui.Modifiers) {
	t.Helper()
	row := selectionRow(t, tt, path, group)
	tt.ClickAtWith(mods, row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
}

func TestWorkspaceRemovingNonprimaryPreservesPreview(t *testing.T) {
	a, _, tt := workspaceSelectionFixture(t)
	for i, path := range []string{"b.go", "c.go", "same.go"} {
		mods := ui.Cmd
		if i == 0 {
			mods = 0
		}
		clickSelectionRow(t, tt, path, filesSectionUnstaged, mods)
	}
	clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, ui.Cmd)
	if a.selectedPath != "same.go" {
		t.Fatalf("removing nonprimary changed preview to %q", a.selectedPath)
	}
	clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
	if a.selectedPath != "b.go" {
		t.Fatalf("removing primary did not choose remaining file: %q", a.selectedPath)
	}
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, ui.Cmd)
	if a.selectedPath != "" || a.diff.Text != "" {
		t.Fatal("empty selection retained preview")
	}
}

func TestWorkspaceRefreshPreservesSummaryAndSelectionIntersection(t *testing.T) {
	a, _, tt := workspaceSelectionFixture(t)
	clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
	a.activateDetailTab(detailTabKey{Summary: true})
	selectionGit(t, a.repo.Root, "restore", "--worktree", "--", "same.go")
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.reconcileSelection()
	if a.detailTabShown() != detailTabSummary {
		t.Fatal("refresh activated file tab from Summary")
	}
	if a.selectedPath != "c.go" || len(a.workspaceSelection.Files) != 1 || a.workspaceSelection.Anchor.path != "c.go" {
		t.Fatalf("wrong retained selection/anchor: %q %#v", a.selectedPath, a.workspaceSelection)
	}
	a.loadSeq++
	selectionGit(t, a.repo.Root, "restore", "--worktree", "--", "c.go")
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.reconcileSelection()
	if a.selectedPath != "" || len(a.workspaceSelection.Files) != 0 {
		t.Fatal("removed identities followed another group or unrelated file")
	}
}

func TestWorkspaceShiftSelectionUnstagesOnlySelectedVersions(t *testing.T) {
	a, callbacks, tt := workspaceSelectionFixture(t)
	if err := os.WriteFile(filepath.Join(a.repo.Root, "z.go"), []byte("original\n"), 0600); err != nil {
		t.Fatal(err)
	}
	selectionGit(t, a.repo.Root, "add", "--", "z.go")
	selectionGit(t, a.repo.Root, "commit", "--no-gpg-sign", "-m", "unselected staged control", "--only", "--", "z.go")
	for _, path := range []string{"b.go", "c.go", "z.go"} {
		if err := os.WriteFile(filepath.Join(a.repo.Root, path), []byte("index version\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	selectionGit(t, a.repo.Root, "add", "--", "b.go", "c.go", "z.go")
	for _, path := range []string{"b.go", "c.go", "z.go"} {
		if err := os.WriteFile(filepath.Join(a.repo.Root, path), []byte("working version\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	clickSelectionRow(t, tt, "b.go", filesSectionStaged, 0)
	clickSelectionRow(t, tt, "same.go", filesSectionStaged, ui.Shift)
	row := selectionRow(t, tt, "b.go", filesSectionStaged)
	tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if err := tt.Click(a.tf("unstageSelectedFiles", 3)); err != nil {
		t.Fatal(err)
	}
	if !a.busy {
		t.Fatal("selected unstage menu did not start operation")
	}
	waitSelectionOperation(t, a, callbacks)
	if got := selectionGit(t, a.repo.Root, "show", ":z.go"); got != "index version\n" {
		t.Fatalf("unselected staged file changed: %q", got)
	}
	if data, err := os.ReadFile(filepath.Join(a.repo.Root, "z.go")); err != nil || string(data) != "working version\n" {
		t.Fatalf("unselected worktree changed: %q %v", data, err)
	}
	for _, path := range []string{"b.go", "same.go", "c.go"} {
		if got := selectionGit(t, a.repo.Root, "show", ":"+path); got != "original\n" {
			t.Fatalf("%s index = %q", path, got)
		}
		data, err := os.ReadFile(filepath.Join(a.repo.Root, path))
		if err != nil || string(data) != "working version\n" {
			t.Fatalf("worktree %s changed: %q %v", path, data, err)
		}
	}
}

func TestWorkspaceSelectAllKeepsGroupAndTextInputScopes(t *testing.T) {
	a, callbacks, tt := workspaceSelectionFixture(t)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	tt.Key(ui.Cmd, ui.KeyA)
	if len(a.workspaceSelection.Files) != 3 || a.workspaceSelection.Group != filesSectionUnstaged {
		t.Fatalf("wrong select-all scope: %#v", a.workspaceSelection)
	}
	if err := tt.Click("selection-input"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("replacement")
	if node, ok := tt.Access("selection-input"); !ok || node.Value != "replacement" {
		t.Fatalf("text select-all intercepted: %#v", node)
	}
	if len(a.workspaceSelection.Files) != 3 {
		t.Fatal("input changed file selection")
	}
	clickSelectionRow(t, tt, "untracked.go", filesSectionUntracked, ui.Cmd)
	if len(a.workspaceSelection.Files) != 1 || a.workspaceSelection.Group != filesSectionUntracked {
		t.Fatal("modifier mixed file groups")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyEnter)
	if !a.busy {
		t.Fatal("Enter did not stage selected untracked group")
	}
	waitSelectionOperation(t, a, callbacks)
	if got := selectionGit(t, a.repo.Root, "show", ":same.go"); got != "index version\n" {
		t.Fatalf("other version changed: %q", got)
	}
	if got := selectionGit(t, a.repo.Root, "show", ":b.go"); got != "original\n" {
		t.Fatalf("other group staged: %q", got)
	}
	if got := selectionGit(t, a.repo.Root, "show", ":untracked.go"); got != "working version\n" {
		t.Fatalf("untracked not staged: %q", got)
	}
}

func TestWorkspaceSelectedStageRejectsStaleOrProtectedTargets(t *testing.T) {
	for _, kind := range []string{"generation", "root", "busy", "dialog", "conflict", "removed"} {
		t.Run(kind, func(t *testing.T) {
			a, _, tt := workspaceSelectionFixture(t)
			clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
			clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
			var entry gitrepo.Entry
			for _, e := range a.status.Entries {
				if e.Path == "same.go" {
					entry = e
				}
			}
			targets := a.workspaceStageTargets(entry, false)
			oldRoot := a.repo.Root
			switch kind {
			case "generation":
				a.loadSeq++
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "busy":
				a.busy = true
			case "dialog":
				a.confirm.open = true
			case "conflict":
				for i := range a.status.Entries {
					if a.status.Entries[i].Path == "same.go" {
						a.status.Entries[i].Conflict = true
					}
				}
			case "removed":
				for i, e := range a.status.Entries {
					if e.Path == "same.go" {
						a.status.Entries = append(a.status.Entries[:i], a.status.Entries[i+1:]...)
						break
					}
				}
			}
			if a.workspaceStageReady(targets) {
				t.Fatal("stale/protected target set accepted")
			}
			a.runWorkspaceStage(targets)
			if kind != "busy" && a.busy {
				t.Fatal("rejected target started operation")
			}
			if got := selectionGit(t, oldRoot, "show", ":same.go"); got != "index version\n" {
				t.Fatalf("rejected action changed index: %q", got)
			}
			if got := selectionGit(t, oldRoot, "show", ":b.go"); got != "original\n" {
				t.Fatalf("rejected action partially staged: %q", got)
			}
		})
	}
}

func TestWorkspaceMultiselectMenuExplainsSingleFileActions(t *testing.T) {
	a, _, tt := workspaceSelectionFixture(t)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
	row := selectionRow(t, tt, "b.go", filesSectionUnstaged)
	tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	for _, label := range []string{a.tf("stageSelectedFiles", 2), a.t("discardThisFile"), a.t("stashThisFile")} {
		if _, ok := tt.AccessByRole(ui.RoleMenuItem, label); !ok {
			t.Fatalf("menu missing explicit scope %q", label)
		}
	}
	if len(a.workspaceSelection.Files) != 2 || a.selectedPath != "b.go" {
		t.Fatal("right-click changed selected membership or failed to preview row")
	}
}

func TestWorkspaceOpenMenuRejectsChangedRepositoryOrGeneration(t *testing.T) {
	for _, kind := range []string{"generation", "root"} {
		t.Run(kind, func(t *testing.T) {
			a, _, tt := workspaceSelectionFixture(t)
			clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
			clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
			row := selectionRow(t, tt, "b.go", filesSectionUnstaged)
			tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			oldRoot := a.repo.Root
			if kind == "generation" {
				a.loadSeq++
			} else {
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			}
			tt.Frame()
			item, ok := tt.AccessByRole(ui.RoleMenuItem, a.tf("stageSelectedFiles", 2))
			if !ok || !item.Disabled {
				t.Fatalf("stale menu not disabled: %#v", item)
			}
			tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
			if a.busy {
				t.Fatal("stale menu started operation")
			}
			if got := selectionGit(t, oldRoot, "show", ":same.go"); got != "index version\n" {
				t.Fatalf("stale menu changed index: %q", got)
			}
			if got := selectionGit(t, oldRoot, "show", ":b.go"); got != "original\n" {
				t.Fatalf("stale menu changed control: %q", got)
			}
		})
	}
}

func TestWorkspaceMultiselectRowIconStagesOnlyItsOwnFile(t *testing.T) {
	a, callbacks, tt := workspaceSelectionFixture(t)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
	row := selectionRow(t, tt, "b.go", filesSectionUnstaged)
	clicked := false
	for _, node := range tt.AccessAll() {
		if node.Name == a.t("stageFile") && node.Role == ui.RoleButton && node.Bounds.Y >= row.Bounds.Y && node.Bounds.Y+node.Bounds.H <= row.Bounds.Y+row.Bounds.H {
			tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
			clicked = true
			break
		}
	}
	if !clicked || !a.busy {
		t.Fatal("single-row stage icon did not run")
	}
	waitSelectionOperation(t, a, callbacks)
	if got := selectionGit(t, a.repo.Root, "show", ":b.go"); got != "working version\n" {
		t.Fatalf("clicked row not staged: %q", got)
	}
	if got := selectionGit(t, a.repo.Root, "show", ":same.go"); got != "index version\n" {
		t.Fatalf("row icon staged another selected file: %q", got)
	}
}

func TestWorkspaceSelectedBatchRevalidatesActualStatusBeforeWriting(t *testing.T) {
	a, callbacks, tt := workspaceSelectionFixture(t)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
	var entry gitrepo.Entry
	for _, e := range a.status.Entries {
		if e.Path == "same.go" {
			entry = e
		}
	}
	targets := a.workspaceStageTargets(entry, false)
	selectionGit(t, a.repo.Root, "restore", "--worktree", "--", "b.go")
	a.runWorkspaceStage(targets)
	if !a.busy {
		t.Fatal("fixture did not reach worker revalidation")
	}
	waitSelectionOperation(t, a, callbacks)
	if got := selectionGit(t, a.repo.Root, "show", ":same.go"); got != "index version\n" {
		t.Fatalf("worker partially staged stale target set: %q", got)
	}
	if got := selectionGit(t, a.repo.Root, "show", ":b.go"); got != "original\n" {
		t.Fatalf("worker changed stale control: %q", got)
	}
}

func TestWorkspaceSelectedSetRejectsProtectedMember(t *testing.T) {
	for _, kind := range []string{"conflict", "dirty-submodule"} {
		t.Run(kind, func(t *testing.T) {
			a, _, tt := workspaceSelectionFixture(t)
			// Deliberate status fixtures: test set-level protection, independently
			// of entry mismatch and the backend's own status revalidation.
			var protected gitrepo.Entry
			for i := range a.status.Entries {
				if a.status.Entries[i].Path == "same.go" {
					if kind == "conflict" {
						a.status.Entries[i].Conflict = true
					} else {
						a.status.Entries[i].Submodule = "S.M."
					}
					protected = a.status.Entries[i]
				}
			}
			tt.Frame()
			clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
			clickSelectionRow(t, tt, "same.go", filesSectionUnstaged, ui.Cmd)
			targets := a.workspaceStageTargets(protected, false)
			if len(targets) != 2 || targets[1].Entry != protected {
				t.Fatalf("protected member silently excluded: %#v", targets)
			}
			if a.workspaceStageReady(targets) {
				t.Fatal("current protected member accepted")
			}
			row := selectionRow(t, tt, "b.go", filesSectionUnstaged)
			tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
			item, ok := tt.AccessByRole(ui.RoleMenuItem, a.tf("stageSelectedFiles", 2))
			if !ok || !item.Disabled {
				t.Fatalf("protected set menu enabled: %#v", item)
			}
			tt.Key(0, ui.KeyEscape)
			tt.Key(0, ui.KeyEnter)
			if a.busy {
				t.Fatal("Enter partially staged protected set")
			}
			if got := selectionGit(t, a.repo.Root, "show", ":same.go"); got != "index version\n" {
				t.Fatalf("protected index changed: %q", got)
			}
			if got := selectionGit(t, a.repo.Root, "show", ":b.go"); got != "original\n" {
				t.Fatalf("normal member partially staged: %q", got)
			}
		})
	}
}

func TestWorkspaceRefreshKeepsFilePreviewInsideSurvivingSelection(t *testing.T) {
	a, _, tt := workspaceSelectionFixture(t)
	// Open tabs in an order different from selection order. Closing c must
	// not let the unrelated b tab take ownership of the selected patch.
	for _, path := range []string{"c.go", "b.go", "same.go"} {
		clickSelectionRow(t, tt, path, filesSectionUnstaged, 0)
	}
	clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, ui.Cmd)
	selectionGit(t, a.repo.Root, "restore", "--worktree", "--", "c.go")
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.reconcileSelection()
	a.reconcileWorktreeTabs(a.overviewKeys())
	if a.selectedPath != "same.go" || a.detailTabs().Active.File.Path != "same.go" {
		t.Fatalf("refresh preview escaped selection: %q active %#v", a.selectedPath, a.detailTabs().Active)
	}
}

func TestWorkspaceModifierSelectionStagesOnlySelectedFiles(t *testing.T) {
	a, callbacks, tt := workspaceSelectionFixture(t)
	if err := tt.Click("same.go"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ClickWith(ui.Cmd, "b.go"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEnter)
	if !a.busy {
		t.Fatal("Enter did not start selected stage operation")
	}
	waitSelectionOperation(t, a, callbacks)
	for _, path := range []string{"same.go", "b.go"} {
		if got := selectionGit(t, a.repo.Root, "show", ":"+path); got != "working version\n" {
			t.Fatalf("selected %s index = %q", path, got)
		}
	}
	if got := selectionGit(t, a.repo.Root, "show", ":c.go"); got != "original\n" {
		t.Fatalf("unselected tracked index = %q", got)
	}
	if got := selectionGit(t, a.repo.Root, "ls-files", "--", "untracked.go"); got != "" {
		t.Fatalf("unselected untracked added: %q", got)
	}
	for _, path := range []string{"same.go", "b.go", "c.go", "untracked.go"} {
		data, err := os.ReadFile(filepath.Join(a.repo.Root, path))
		if err != nil || string(data) != "working version\n" {
			t.Fatalf("worktree %s changed: %q %v", path, data, err)
		}
	}
}
