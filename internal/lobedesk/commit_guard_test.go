package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestCommitConflictBlocksEvenWithOtherStagedFiles(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: "fixture"}, commitIdentity: readyCommitIdentity("fixture"), commitText: "title", status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "ready", Index: 'M'}, {Path: "unresolved", Conflict: true}}}}
	if a.canCommit() {
		t.Fatal("an unrelated staged file bypassed unresolved conflict guard")
	}
	a.amend = true
	if a.canCommit() {
		t.Fatal("amend bypassed unresolved conflict guard")
	}
}

func TestDisabledCommitExplainsWhy(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: "fixture"}, commitIdentity: readyCommitIdentity("fixture"), status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "ready", Index: 'M'}}}}
	tt := commitFormTester(a)
	button, _ := tt.AccessByRole(ui.RoleButton, a.t("commit"))
	if button.Description == "" {
		t.Fatal("empty title has no disabled reason")
	}
	titleReason := button.Description
	a.commitText = "title"
	a.busy = true
	tt.Frame()
	button, _ = tt.AccessByRole(ui.RoleButton, a.t("commit"))
	if button.Description == "" || button.Description == titleReason {
		t.Fatal("busy commit has no specific disabled reason")
	}
	a.busy = false
	a.loading = true
	tt.Frame()
	button, _ = tt.AccessByRole(ui.RoleButton, a.t("commit"))
	if button.Description != a.t("loadingRepo") {
		t.Fatal("loading reason is missing")
	}
}
