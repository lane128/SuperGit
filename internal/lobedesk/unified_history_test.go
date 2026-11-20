package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

// Sublime Merge has one main window: the log, whose first row is the working
// tree and whose stash entries are rows in the same list. The Changes and
// Stash pages were separate destinations with their own tabs; the tabs are
// gone, and both live inside History now.
func TestTheWorkbenchHasNoChangesHistoryStashTabs(t *testing.T) {
	a := &app{page: "history", locale: localeEnglish, themeMode: "light", palette: themePaletteSuperGit,
		repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Branch: "main"}}
	tt := changesLayoutTester(a)
	for _, label := range []string{a.t("changes"), a.t("history"), a.t("stash")} {
		if tab, ok := tt.AccessByRole(ui.RoleTab, label); ok {
			t.Fatalf("the workbench still shows a %q page tab at %+v", label, tab.Bounds)
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleTab, a.t("branches")); ok {
		t.Fatal("the workbench grew a Branches page tab")
	}
}

// Opening Changes is selecting the working tree's row: the history list's
// first row is the working tree, and its details are the changes.
func TestOpeningChangesSelectsTheWorkingTreeRow(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.selectCommit("0123456789abcdef0123456789abcdef01234567")
	a.openPage("changes")
	if a.page != "history" {
		t.Fatalf("Changes switched to page %q; the working tree is a row in History", a.page)
	}
	if !a.worktreeSelected || a.selectedSHA != "" {
		t.Fatalf("Changes left the historical selection in place: worktree=%t sha=%q", a.worktreeSelected, a.selectedSHA)
	}
	waitOverview(t, a, callbacks)
}

// With the working tree selected, History's Summary tab draws the commit
// editor, which is what the Changes page used to be.
func TestHistoryWorkingTreeShowsTheCommitEditor(t *testing.T) {
	a := &app{page: "history", locale: localeEnglish, themeMode: "light", palette: themePaletteSuperGit, worktreeSelected: true,
		repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Branch: "main", Entries: []gitrepo.Entry{{Path: "same.go", Index: 'M', Worktree: 'M'}}},
		selectedPath: "same.go", commitText: "retained title\n\nretained body",
		diff: gitrepo.Diff{Text: "+index version", Lines: []gitrepo.DiffLine{{Kind: "add", Text: "+index version"}}},
	}
	tt := changesLayoutTester(a)
	if _, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitSummary")); !ok {
		t.Fatal("the working tree's History selection hides the commit editor")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("commit")); !ok {
		t.Fatal("the working tree's History selection hides the commit action")
	}
	if a.commitText != "retained title\n\nretained body" {
		t.Fatal("the layout changed the draft")
	}
}

// Stash entries are rows in the same list, above the commits, the way Sublime
// Merge lists them in the log.
func TestHistoryListsStashesAboveTheCommits(t *testing.T) {
	a := &app{page: "history", locale: localeEnglish, themeMode: "light", palette: themePaletteSuperGit,
		repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Branch: "main"}}
	a.stashes = []gitrepo.Stash{
		{Ref: "stash@{0}", SHA: "1111111111111111111111111111111111111111", Message: "newest work in progress"},
		{Ref: "stash@{1}", SHA: "2222222222222222222222222222222222222222", Message: "older work in progress"},
	}
	tt := changesLayoutTester(a)
	newest, ok := tt.AccessByRole(ui.RoleOption, "newest work in progress")
	if !ok {
		t.Fatal("History does not list the newest stash")
	}
	older, ok := tt.AccessByRole(ui.RoleOption, "older work in progress")
	if !ok {
		t.Fatal("History does not list the older stash")
	}
	if older.Bounds.Y <= newest.Bounds.Y {
		t.Fatalf("stashes are listed out of order: %v then %v", newest.Bounds, older.Bounds)
	}
}

// Choosing a stash row makes the stash, not a commit or the working tree, the
// details' subject, and the page stays History.
func TestSelectingAStashMakesItTheHistorySubject(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.commits = []gitrepo.Commit{{SHA: "0123456789abcdef0123456789abcdef01234567", Subject: "fixture"}}
	a.selectCommit(a.commits[0].SHA)
	a.selectStash("2222222222222222222222222222222222222222")
	if a.page != "history" {
		t.Fatalf("selecting a stash switched to page %q", a.page)
	}
	if a.worktreeSelected || a.selectedSHA != "" {
		t.Fatalf("selecting a stash kept the commit/worktree subject: worktree=%t sha=%q", a.worktreeSelected, a.selectedSHA)
	}
	if a.selectedStash != "2222222222222222222222222222222222222222" {
		t.Fatalf("selected stash = %q", a.selectedStash)
	}
	_ = callbacks
}
