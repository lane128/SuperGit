package lobedesk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func commitUndoUIFixture(t *testing.T) (*app, chan func(), *ui.Tester) {
	t.Helper()
	a, callbacks, tt := commitPatchFixture(t)
	root := a.repo.Root
	if err := os.WriteFile(filepath.Join(root, "same.go"), []byte("source change\n"), 0600); err != nil {
		t.Fatal(err)
	}
	selectionGit(t, root, "add", "--all")
	selectionGit(t, root, "commit", "--no-gpg-sign", "-m", "source")
	a.selectedSHA = strings.TrimSpace(selectionGit(t, root, "rev-parse", "HEAD"))
	parent := strings.TrimSpace(selectionGit(t, root, "rev-parse", "HEAD^"))
	a.commits = []gitrepo.Commit{{SHA: a.selectedSHA, Parents: []string{parent}}}
	var err error
	a.status, err = a.repo.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a.commitFiles, err = a.repo.CommitFiles(context.Background(), a.selectedSHA)
	if err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	return a, callbacks, tt
}
func openUndoUIMenu(t *testing.T, a *app, tt *ui.Tester) ui.AccessInfo {
	t.Helper()
	row, ok := tt.AccessByRole(ui.RoleOption, "same.go")
	if !ok {
		t.Fatal("historical row missing")
	}
	tt.RightClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, "Undo This Commit’s Changes to This File…")
	if !ok {
		t.Fatal("historical menu lacks single-file undo")
	}
	return item
}
func requestUndoUI(t *testing.T, a *app, callbacks chan func(), tt *ui.Tester) {
	t.Helper()
	item := openUndoUIMenu(t, a, tt)
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	waitSelectionOperation(t, a, callbacks)
	tt.Frame()
	if !a.confirm.open {
		t.Fatal("undo confirmation did not open")
	}
}
func TestCommitFileUndoUIEnterDefaultsToCancel(t *testing.T) {
	a, callbacks, tt := commitUndoUIFixture(t)
	requestUndoUI(t, a, callbacks, tt)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.confirm.open || a.busy {
		t.Fatal("Enter did not activate default Cancel")
	}
	got, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
	if err != nil || string(got) != "source change\n" || selectionGit(t, a.repo.Root, "status", "--porcelain=v1", "-z") != "" {
		t.Fatal("confirmation cancellation changed source")
	}
}
func TestCommitFileUndoUIConfirmPreservesDraftAndIndex(t *testing.T) {
	a, callbacks, tt := commitUndoUIFixture(t)
	root := a.repo.Root
	head, index := selectionGit(t, root, "rev-parse", "HEAD"), selectionGit(t, root, "ls-files", "--stage", "-z")
	requestUndoUI(t, a, callbacks, tt)
	if !strings.Contains(a.confirm.message, strings.TrimSpace(head)) || !strings.Contains(a.confirm.message, "same.go") || !strings.Contains(a.confirm.message, "main") {
		t.Fatal("confirmation lacks full target scope")
	}
	clickPatchScopeButton(t, tt, "Undo File Changes")
	waitSelectionOperation(t, a, callbacks)
	got, err := os.ReadFile(filepath.Join(root, "same.go"))
	if err != nil || string(got) != "original\n" {
		t.Fatalf("undo did not restore selected source change: %q %v", got, err)
	}
	if selectionGit(t, root, "rev-parse", "HEAD") != head || selectionGit(t, root, "ls-files", "--stage", "-z") != index || a.commitFields.Summary != "keep draft title" || a.commitFields.Body != "keep draft body" {
		t.Fatal("undo changed history/index/draft")
	}
	found := false
	for _, notice := range a.notices[toastMain] {
		found = found || notice.kind == "success"
	}
	if !found {
		t.Fatal("missing undo success feedback")
	}
}
