package lobedesk

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
)

func discardShortcutFixture(t *testing.T) (*app, chan func(), *ui.Tester) {
	t.Helper()
	a, callbacks, _ := workspaceSelectionFixture(t)
	tt := pageTester(a, func(c *ui.Context) {
		a.registerShortcuts(c)
		ui.Column(c).FillWidth().Children(func() {
			for _, group := range a.workingFileGroups() {
				for _, entry := range group.Entries {
					a.fileRow(c.Key(group.Key+entry.Path), entry, group.Staged).Description(group.Key)
				}
			}
			field.Input(c, &a.commitText, field.InputProps{}).Label("discard-input")
		})
		a.confirmDialog(c)
	})
	return a, callbacks, tt
}

func waitShortcutDiscard(t *testing.T, a *app, callbacks chan func()) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for !a.confirm.open {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("Backspace did not open discard confirmation")
		}
	}
}

func assertDiscardBytes(t *testing.T, a *app, path, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(a.repo.Root, path))
	if err != nil || string(data) != want {
		t.Fatalf("%s = %q, %v; want %q", path, data, err, want)
	}
}

func TestFileDiscardShortcutSelectedScopeAndCancel(t *testing.T) {
	a, callbacks, tt := discardShortcutFixture(t)
	head := selectionGit(t, a.repo.Root, "rev-parse", "HEAD")
	index := selectionGit(t, a.repo.Root, "ls-files", "--stage")
	a.commitText = "keep draft"
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, ui.Cmd)
	tt.Key(0, ui.KeyBackspace)
	waitShortcutDiscard(t, a, callbacks)
	if a.confirm.message != a.tf("discardSelectedMessage", 2) {
		t.Fatal("collection confirmation names only one target")
	}
	tt.Frame()
	if err := tt.Click(a.t("cancel")); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"b.go", "c.go", "same.go", "untracked.go"} {
		assertDiscardBytes(t, a, p, "working version\n")
	}
	if selectionGit(t, a.repo.Root, "rev-parse", "HEAD") != head || selectionGit(t, a.repo.Root, "ls-files", "--stage") != index || a.commitText != "keep draft" {
		t.Fatal("cancel changed Git or draft")
	}
}

func TestFileDiscardShortcutLatestPreparationWins(t *testing.T) {
	a, callbacks, tt := discardShortcutFixture(t)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	next := func() func() {
		t.Helper()
		select {
		case fn := <-callbacks:
			return fn
		case <-time.After(5 * time.Second):
			t.Fatal("preparation callback missing")
			return nil
		}
	}
	for a.diffLoading {
		next()()
	}
	tt.Key(0, ui.KeyBackspace)
	first := next()
	if err := os.WriteFile(filepath.Join(a.repo.Root, "b.go"), []byte("second prepared version\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyBackspace)
	second := next()
	second()
	first()
	if !a.confirm.open {
		t.Fatal("latest preparation did not open confirmation")
	}
	tt.Frame()
	if err := tt.Click(a.confirm.actionName); err != nil {
		t.Fatal(err)
	}
	waitSelectionOperation(t, a, callbacks)
	assertDiscardBytes(t, a, "b.go", "original\n")
	assertDiscardBytes(t, a, "c.go", "working version\n")
}

func TestFileDiscardShortcutKeyboardCancel(t *testing.T) {
	for _, test := range []struct {
		name string
		key  ui.Key
	}{{"Enter", ui.KeyEnter}, {"Escape", ui.KeyEscape}} {
		t.Run(test.name, func(t *testing.T) {
			a, callbacks, tt := discardShortcutFixture(t)
			clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
			tt.Key(0, ui.KeyBackspace)
			waitShortcutDiscard(t, a, callbacks)
			tt.Frame()
			tt.Frame()
			cancel, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
			if !ok || !cancel.Focused {
				t.Fatal("discard confirmation must initially focus Cancel")
			}
			tt.Key(0, test.key)
			if a.confirm.open || a.busy {
				t.Fatal("keyboard cancellation failed")
			}
			assertDiscardBytes(t, a, "b.go", "working version\n")
			row := selectionRow(t, tt, "b.go", filesSectionUnstaged)
			if !row.Focused {
				t.Fatal("cancellation did not restore source row focus")
			}
		})
	}
}

func TestFileDiscardShortcutExcludedScopes(t *testing.T) {
	for _, scope := range []string{"input", "shift", "staged", "untracked", "historical", "chooser", "line selection", "conflict", "submodule"} {
		t.Run(scope, func(t *testing.T) {
			a, _, tt := discardShortcutFixture(t)
			path, group := "b.go", filesSectionUnstaged
			switch scope {
			case "staged":
				path, group = "same.go", filesSectionStaged
			case "untracked":
				path, group = "untracked.go", filesSectionUntracked
			case "conflict", "submodule":
				for i := range a.status.Entries {
					if a.status.Entries[i].Path == path {
						if scope == "conflict" {
							a.status.Entries[i].Conflict = true
						} else {
							a.status.Entries[i].Submodule = "S.MU"
						}
					}
				}
				tt.Frame()
			}
			clickSelectionRow(t, tt, path, group, 0)
			mods := ui.Modifiers(0)
			switch scope {
			case "input":
				a.commitText = "draftX"
				tt.Frame()
				if err := tt.Click("discard-input"); err != nil {
					t.Fatal(err)
				}
				tt.Key(ui.Cmd, ui.KeyA)
				tt.Key(0, ui.KeyEnd)
			case "shift":
				mods = ui.Shift
			case "historical":
				a.worktreeSelected = false
			case "chooser":
				a.repositoryChooserOpen = true
			case "line selection":
				a.lineSel = map[int]bool{0: true}
			}
			tt.Key(mods, ui.KeyBackspace)
			if a.discardPrepareSeq != 0 || a.confirm.open || a.busy {
				t.Fatal("excluded surface started discard")
			}
			if scope == "input" && a.commitText != "draft" {
				t.Fatalf("Backspace failed text editing: %q", a.commitText)
			}
		})
	}
}

func TestFileDiscardShortcutFrozenTargetsAndStaleContent(t *testing.T) {
	for _, change := range []string{"selection before preparation", "selection after confirmation", "refresh", "external bytes", "external index"} {
		t.Run(change, func(t *testing.T) {
			a, callbacks, tt := discardShortcutFixture(t)
			clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
			clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, ui.Cmd)
			tt.Key(0, ui.KeyBackspace)
			if change == "selection before preparation" {
				a.selectWorkspaceFile(fileRef{"same.go", false}, 0, false)
			}
			waitShortcutDiscard(t, a, callbacks)
			if change == "selection after confirmation" {
				a.selectWorkspaceFile(fileRef{"same.go", false}, 0, false)
			}
			if change == "refresh" {
				a.loadSeq++
			}
			if change == "external bytes" {
				if err := os.WriteFile(filepath.Join(a.repo.Root, "b.go"), []byte("external edit\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if change == "external index" {
				selectionGit(t, a.repo.Root, "add", "--", "b.go")
			}
			tt.Frame()
			if err := tt.Click(a.confirm.actionName); err != nil {
				t.Fatal(err)
			}
			waitSelectionOperation(t, a, callbacks)
			if change == "selection before preparation" || change == "selection after confirmation" {
				assertDiscardBytes(t, a, "b.go", "original\n")
				assertDiscardBytes(t, a, "c.go", "original\n")
			} else {
				want := "working version\n"
				if change == "external bytes" {
					want = "external edit\n"
				}
				assertDiscardBytes(t, a, "b.go", want)
				assertDiscardBytes(t, a, "c.go", "working version\n")
			}
			assertDiscardBytes(t, a, "same.go", "working version\n")
		})
	}
}

func TestFileDiscardShortcutFocusOutsideCollection(t *testing.T) {
	a, callbacks, tt := discardShortcutFixture(t)
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, ui.Cmd)
	target := detailFileKey{Path: "same.go"}
	a.detailFileFocus = &target
	a.detailFileFocusContext = a.currentDetailContext()
	tt.Frame()
	if row := selectionRow(t, tt, "same.go", filesSectionUnstaged); !row.Focused {
		t.Fatal("outside row not focused")
	}
	tt.Key(0, ui.KeyBackspace)
	waitShortcutDiscard(t, a, callbacks)
	if a.confirm.message != a.tf("discardMessage", "same.go") {
		t.Fatal("outside focus did not capture singleton")
	}
	tt.Frame()
	if err := tt.Click(a.confirm.actionName); err != nil {
		t.Fatal(err)
	}
	waitSelectionOperation(t, a, callbacks)
	assertDiscardBytes(t, a, "same.go", "index version\n")
	assertDiscardBytes(t, a, "b.go", "working version\n")
	assertDiscardBytes(t, a, "c.go", "working version\n")
}

func TestFileDiscardShortcutAcceptsOnlySelectedFiles(t *testing.T) {
	a, callbacks, tt := discardShortcutFixture(t)
	head := selectionGit(t, a.repo.Root, "rev-parse", "HEAD")
	index := selectionGit(t, a.repo.Root, "ls-files", "--stage")
	clickSelectionRow(t, tt, "b.go", filesSectionUnstaged, 0)
	clickSelectionRow(t, tt, "c.go", filesSectionUnstaged, ui.Cmd)
	tt.Key(0, ui.KeyBackspace)
	waitShortcutDiscard(t, a, callbacks)
	tt.Frame()
	if err := tt.Click(a.confirm.actionName); err != nil {
		t.Fatal(err)
	}
	waitSelectionOperation(t, a, callbacks)
	for _, p := range []string{"b.go", "c.go"} {
		assertDiscardBytes(t, a, p, "original\n")
	}
	for _, p := range []string{"same.go", "untracked.go"} {
		assertDiscardBytes(t, a, p, "working version\n")
	}
	if selectionGit(t, a.repo.Root, "rev-parse", "HEAD") != head || selectionGit(t, a.repo.Root, "ls-files", "--stage") != index {
		t.Fatal("selected discard changed HEAD or Index")
	}
}
