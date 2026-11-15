package lobedesk

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func dialogTester(a *app, dialog func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(900).Height(650).Children(func() {
			dialog(c)
		})
	}, 900, 650)
}

// A dialog's field takes the focus when it appears, but not every frame: a
// field that asks each frame takes Tab back before the reader can leave it,
// so Cancel and the dialog's action were only reachable with the mouse.
func TestPromptDialogLetsTabLeaveTheField(t *testing.T) {
	a := &app{locale: localeChinese}
	a.prompt = promptState{
		open:        true,
		title:       "提交信息",
		placeholder: "写点什么",
		actionName:  "确定",
	}
	tt := dialogTester(a, a.promptDialog)

	field, ok := tt.AccessByRole(ui.RoleTextField, "")
	if !ok {
		t.Fatalf("the prompt shows no text field: %+v", tt.AccessAll())
	}
	if !field.Focused {
		t.Fatalf("the prompt's field does not take the focus when it opens: %+v", field)
	}

	tt.Key(0, ui.KeyTab)
	if field, ok := tt.AccessByRole(ui.RoleTextField, ""); ok && field.Focused {
		t.Fatal("Tab cannot leave the prompt's field")
	}
	cancel, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
	if !ok {
		t.Fatalf("the prompt has no %q button", a.t("cancel"))
	}
	if !cancel.Focused {
		t.Fatalf("after Tab the focus is not on %q: %+v", a.t("cancel"), cancel)
	}
	tt.Key(0, ui.KeyEnter)
	if a.prompt.open {
		t.Fatal("Enter on Cancel did not close the prompt")
	}
}

func TestWorktreeFormLetsTabLeaveTheFirstField(t *testing.T) {
	a := &app{locale: localeChinese}
	a.worktreeForm = worktreeFormState{open: true}
	tt := dialogTester(a, a.worktreeFormDialog)

	// The form opens on the branch, the one required field the reader has to
	// fill; the path and start point come prefilled.
	fields := 0
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleTextField && info.Focused {
			fields++
			if info.Placeholder != a.t("worktreeBranch") {
				t.Errorf("the form opens focused on %q; want the branch field", info.Placeholder)
			}
		}
	}
	if fields != 1 {
		t.Fatalf("%d worktree fields have the focus when the form opens; want one", fields)
	}

	// Tab walks the rest of the form in reading order: the start field, then
	// Cancel. Create is disabled while the form is empty, so Tab passes over
	// it.
	for _, want := range []struct {
		why  string
		test func(info ui.AccessInfo) bool
	}{
		{"start field", func(info ui.AccessInfo) bool { return info.Placeholder == a.t("worktreeStart") }},
		{"cancel button", func(info ui.AccessInfo) bool { return info.Name == a.t("cancel") }},
	} {
		tt.Key(0, ui.KeyTab)
		reached := false
		for _, info := range tt.AccessAll() {
			if info.Focused && want.test(info) {
				reached = true
			}
		}
		if !reached {
			t.Fatalf("Tab did not reach %s: %+v", want.why, tt.AccessAll())
		}
	}
	tt.Key(0, ui.KeyEnter)
	if a.worktreeForm.open {
		t.Fatal("Enter on Cancel did not close the worktree form")
	}
}

// Moving the dialog to AutoFocus keeps the field's own behavior: Enter still
// submits the value the reader typed.
func TestPromptDialogStillSubmitsWithEnter(t *testing.T) {
	a := &app{locale: localeChinese}
	submitted := ""
	a.prompt = promptState{
		open:        true,
		title:       "提交信息",
		placeholder: "写点什么",
		value:       "fix the parser",
		actionName:  "确定",
		action:      func(value string) { submitted = value },
	}
	tt := dialogTester(a, a.promptDialog)
	tt.Key(0, ui.KeyEnter)
	if submitted != "fix the parser" {
		t.Fatalf("Enter submitted %q; want the typed value", submitted)
	}
	if a.prompt.open {
		t.Fatal("Enter left the prompt open")
	}
}

// The reset dialog's first focusable is the Soft reset button; opening it
// from the commit menu and pressing Enter then resets the repository before
// the reader picked a mode. Cancel takes the focus instead.
func TestResetDialogStartsWithCancelFocused(t *testing.T) {
	a := &app{locale: localeChinese}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	a.reset = resetState{open: true, sha: "1234567890", subject: "the commit"}
	tt := dialogTester(a, a.resetDialog)

	cancel, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
	if !ok {
		t.Fatalf("the reset dialog has no %q button", a.t("cancel"))
	}
	if !cancel.Focused {
		t.Fatalf("the reset dialog opens focused on %q; want Cancel", a.t("resetSoft"))
	}
	tt.Key(0, ui.KeyEnter)
	if a.busy {
		t.Fatal("Enter on the opening dialog started a reset")
	}
	if a.reset.open {
		t.Fatal("Enter on Cancel left the reset dialog open")
	}
}

// A prefilled prompt shows the old value selected, as a browser's rename
// field does, so typing replaces it instead of appending to it. AutoFocus
// alone only puts the caret at the end.
func TestPrefilledPromptReplacesTheValueWhenTyping(t *testing.T) {
	a := &app{locale: localeChinese}
	a.askPrompt("重命名分支", "新名称", "main", "重命名", func(string) {})
	tt := dialogTester(a, a.promptDialog)
	tt.Type("dev")
	if a.prompt.value != "dev" {
		t.Errorf("typing into a prefilled prompt left %q; want dev", a.prompt.value)
	}
	// A selection kept for every frame would make each keystroke replace the
	// one before; the next character must come after the first word.
	tt.Type("x")
	if a.prompt.value != "devx" {
		t.Errorf("the second keystroke left %q; want devx, with the selection used once", a.prompt.value)
	}
}

// Enter in one of the worktree fields creates the worktree when the form is
// complete, as Enter does in the single-field prompts; the form used to
// ignore it and wait for the button.
func TestWorktreeFormSubmitsWithEnter(t *testing.T) {
	a := &app{locale: localeChinese}
	a.worktreeForm = worktreeFormState{
		open: true, path: "/tmp/wt", branch: "feat", start: "main",
	}
	tt := dialogTester(a, a.worktreeFormDialog)
	tt.Key(0, ui.KeyEnter)
	if a.worktreeForm.open {
		t.Fatal("Enter in a complete worktree form left it open")
	}
}

// Hard reset is the one mode that discards uncommitted work, so it opens the
// danger confirmation instead of resetting from the mode dialog. The other
// modes run directly.
func TestHardResetAsksAgainBeforeItRuns(t *testing.T) {
	a := &app{locale: localeChinese}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	a.reset = resetState{open: true, sha: "1234567890", subject: "the commit"}
	tt := dialogTester(a, a.resetDialog)

	hard, ok := tt.Find(a.t("resetHard"))
	if !ok {
		t.Fatalf("the reset dialog has no %q button", a.t("resetHard"))
	}
	tt.ClickAt(hard.X+hard.W/2, hard.Y+hard.H/2)
	if a.reset.open {
		t.Fatal("choosing hard reset left the mode dialog open")
	}
	if a.busy {
		t.Fatal("choosing hard reset started the reset before the confirmation")
	}
	if !a.confirm.open {
		t.Fatal("hard reset did not ask for the danger confirmation")
	}
	if !a.confirm.danger || a.confirm.actionName != a.t("resetHardAction") {
		t.Errorf("the confirmation reads danger=%v action=%q; want the hard-reset danger action",
			a.confirm.danger, a.confirm.actionName)
	}
}

// Soft and mixed only move references and the index; they run from the mode
// dialog without a second question.
func TestSoftResetRunsFromTheModeDialog(t *testing.T) {
	a := &app{locale: localeChinese}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	a.reset = resetState{open: true, sha: "1234567890", subject: "the commit"}
	tt := dialogTester(a, a.resetDialog)

	soft, ok := tt.Find(a.t("resetSoft"))
	if !ok {
		t.Fatalf("the reset dialog has no %q button", a.t("resetSoft"))
	}
	tt.ClickAt(soft.X+soft.W/2, soft.Y+soft.H/2)
	if a.confirm.open {
		t.Fatal("soft reset asked a second confirmation")
	}
	if !a.busy {
		t.Fatal("soft reset did not start")
	}
}

// Typing as soon as the form opens goes into the branch name, not into the
// prefilled destination: the old focus appended it to the path.
func TestWorktreeFormOpensOnTheEmptyBranchField(t *testing.T) {
	a := &app{locale: localeChinese}
	a.repo = &gitrepo.Repository{Root: "/repos/project"}
	a.worktreeForm = worktreeFormState{
		open: true, path: "/repos/project-worktree", start: "main",
	}
	tt := dialogTester(a, a.worktreeFormDialog)
	tt.Type("feat")
	if a.worktreeForm.branch != "feat" {
		t.Errorf("typing filled branch %q; want feat", a.worktreeForm.branch)
	}
	if a.worktreeForm.path != "/repos/project-worktree" {
		t.Errorf("typing changed the destination to %q", a.worktreeForm.path)
	}
}

// A reset cannot run while another operation or a read holds the repository;
// the modes say so instead of closing the dialog for a reset that never ran.
func TestResetModesAreDisabledWhileTheRepositoryIsBusy(t *testing.T) {
	a := &app{locale: localeChinese, busy: true}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.ctx = context.Background()
	a.reset = resetState{open: true, sha: "1234567890", subject: "the commit"}
	tt := dialogTester(a, a.resetDialog)
	for _, label := range []string{a.t("resetSoft"), a.t("resetMixed"), a.t("resetHard")} {
		button, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Fatalf("the reset dialog has no %q button", label)
		}
		if !button.Disabled {
			t.Errorf("%q is enabled while the repository is busy", label)
		}
	}
	soft, _ := tt.AccessByRole(ui.RoleButton, a.t("resetSoft"))
	tt.ClickAt(soft.Bounds.X+soft.Bounds.W/2, soft.Bounds.Y+soft.Bounds.H/2)
	if !a.reset.open || a.reset.sha != "1234567890" {
		t.Error("clicking a disabled mode closed the reset dialog")
	}
}

// When the operation cannot start, the dialog keeps the mode and the commit;
// clearing it first made the reset look accepted.
func TestResetKeepsItsDialogWhenNothingStarts(t *testing.T) {
	a := &app{locale: localeChinese, busy: true}
	a.reset = resetState{open: true, sha: "abc", subject: "the commit"}
	if a.performReset("soft", "abc") {
		t.Fatal("a busy app started a reset")
	}
	if !a.reset.open || a.reset.sha != "abc" {
		t.Error("the reset dialog closed without starting the reset")
	}
}
