package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

func TestMergeOperationTagNamesCommit(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		a := &app{locale: language, repo: &gitrepo.Repository{Root: t.TempDir()}, operation: gitrepo.Operation{Kind: "merge"}}
		tt := pageTester(a, a.titleBar)
		tag, ok := tt.AccessByRole(ui.RoleButton, a.tf("operation", "merge"))
		want := "Commit to finish the merge, or abort it in Changes."
		if language == localeChinese {
			want = "在「改动」页提交以完成合并，或中止合并。"
		}
		if !ok || tag.Description != want {
			t.Fatalf("merge guidance does not name its actual completion action: %+v", tag)
		}
	}
}

// A cancelled rebase leaves the repository mid-operation, and the page the
// reader was on used to say only "Cancelled". The title bar is on every page,
// so it names the operation that is still in progress.
func TestTheTitleBarNamesAnOperationLeftRunning(t *testing.T) {
	a := &app{locale: localeChinese, operation: gitrepo.Operation{Kind: "rebase"}}
	a.repo = &gitrepo.Repository{Root: t.TempDir()}
	a.status = gitrepo.Status{Branch: "main"}
	tt := pageTester(a, a.titleBar)
	label := a.tf("operation", "rebase")
	if _, ok := tt.Find(label); !ok {
		t.Fatalf("the title bar does not show %q", label)
	}
	// The tag says where the continue and abort actions live; the title bar
	// only reports the state.
	found := false
	for _, info := range tt.AccessAll() {
		if info.Name == label && info.Description == a.t("operationWhere") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("the operation tag does not say where its actions are")
	}
	tag, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok {
		t.Fatalf("the operation tag is not a button, so it cannot lead to its actions")
	}
	tt.ClickAt(tag.Bounds.X+tag.Bounds.W/2, tag.Bounds.Y+tag.Bounds.H/2)
	if a.page != "history" || !a.worktreeSelected {
		t.Errorf("clicking the operation tag opened %q, worktree=%t; want History with the working tree selected", a.page, a.worktreeSelected)
	}
}
