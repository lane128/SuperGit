package lobedesk

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func readyCommitIdentity(root string) commitIdentityState {
	id := gitrepo.Identity{Name: "Fixture Author", Email: "author@example.invalid"}
	return commitIdentityState{Root: root, Loaded: true, Author: id, Committer: id, HeadAuthor: id}
}

func TestCommitIdentityFollowsEffectiveRepositoryAuthor(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: "first"}, loadSeq: 2}
	state := readyCommitIdentity("first")
	state.Author = gitrepo.Identity{Name: "Effective Author", Email: "effective@example.invalid"}
	a.applyCommitIdentity(2, "first", state)
	tt := commitFormTester(a)
	if !tt.HasText("Author: Effective Author <effective@example.invalid>") {
		t.Fatal("effective author is not shown on commit form")
	}
	// A same-named repository in a different directory cannot inherit identity.
	a.repo = &gitrepo.Repository{Root: "second"}
	a.loadSeq++
	a.applyCommitIdentity(2, "first", state)
	if a.commitIdentityReady() {
		t.Fatal("old repository identity is ready for a different root")
	}
	current := readyCommitIdentity("second")
	current.Author = gitrepo.Identity{Name: "Second Author", Email: "second@example.invalid"}
	a.applyCommitIdentity(3, "second", current)
	if a.commitIdentity.Author != current.Author {
		t.Fatal("current repository identity was lost")
	}
	// A previous refresh in the same root is stale too.
	a.applyCommitIdentity(2, "second", state)
	if a.commitIdentity.Author != current.Author {
		t.Fatal("old generation replaced effective identity")
	}
}

func TestAmendDisplaysPreservedAuthor(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: "fixture"}, amend: true, headMessage: "original"}
	a.commitIdentity = readyCommitIdentity("fixture")
	a.commitIdentity.HeadAuthor = gitrepo.Identity{Name: "Original Author", Email: "original@example.invalid"}
	a.commitIdentity.Committer = gitrepo.Identity{Name: "New Committer", Email: "committer@example.invalid"}
	a.commits = []gitrepo.Commit{{Author: "Unrelated History Author"}}
	tt := commitFormTester(a)
	if !tt.HasText("Author: Original Author <original@example.invalid>") {
		t.Fatal("amend does not show preserved HEAD author")
	}
	if !tt.HasText("Committer: New Committer <committer@example.invalid>") {
		t.Fatal("amend hides a different current committer")
	}
	if tt.HasText("Unrelated History Author") {
		t.Fatal("form took author from browsed history")
	}
}

func TestCommitIdentityErrorsBlockWritesAndRemainLocal(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: "fixture"}, commitText: "title", status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "a", Index: 'M'}}}}
	if a.canCommit() {
		t.Fatal("pending identity permits commit")
	}
	a.commitIdentity = readyCommitIdentity("fixture")
	if !a.canCommit() {
		t.Fatal("valid identity blocks staged commit")
	}
	a.commitIdentity.AuthorError = "author configuration missing"
	if a.canCommit() {
		t.Fatal("missing author permits commit")
	}
	tt := commitFormTester(a)
	if !tt.HasText(a.t("commitIdentityFailed")) || !tt.HasText(a.t("commitIdentityConfigHint")) {
		t.Fatal("missing author has no local configuration hint")
	}
	a.amend = true
	if !a.canCommit() {
		t.Fatal("amend should use original author when current author is missing")
	}
	a.commitIdentity.CommitterError = "committer configuration missing"
	if a.canCommit() {
		t.Fatal("amend permits missing committer")
	}
	// An unreadable repository reports a read failure, never a fabricated name.
	state := readCommitIdentity(context.Background(), &gitrepo.Repository{Root: t.TempDir() + "/missing"}, false)
	if state.AuthorError == "" || state.Author != (gitrepo.Identity{}) {
		t.Fatal("read failure fabricated author identity")
	}
}

func TestDisabledCommitKeepsButtonShape(t *testing.T) {
	a := &app{locale: localeEnglish}
	tt := commitFormTester(a)
	button, ok := tt.AccessByRole(ui.RoleButton, a.t("commit"))
	if !ok || !button.Disabled {
		t.Fatal("missing disabled Commit button")
	}
	img := tt.Image()
	x, y := int(button.Bounds.X+6), int(button.Bounds.Y+button.Bounds.H/2)
	if img.RGBAAt(x, y) == img.RGBAAt(int(button.Bounds.X-2), y) {
		t.Fatal("disabled Commit has no visible button container")
	}
}
