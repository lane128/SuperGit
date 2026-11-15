package lobedesk

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// sidebarTester renders the sidebar the way the window does: a full-height
// column, so the footer stays at the bottom while groups fold and unfold. A
// bare column shrinks to its content, which moves every row below a fold.
func sidebarTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Width(260).Height(620).Children(func() { a.sidebar(c) })
	}, 260, 620)
}

// The rail showed five icons with no names, and two of them (the workspace's
// and the branch icon) looked alike. The sidebar names every page, so the
// reader does not have to learn the glyphs.
func TestTheSidebarNamesEachPage(t *testing.T) {
	a := &app{locale: localeChinese, page: "history", repo: &gitrepo.Repository{Root: t.TempDir()}}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.sidebar(c)
	}, 240, 520)
	for _, page := range []string{"history", "branches", "worktrees", "stashes", "settings"} {
		name := a.t(page)
		if _, ok := tt.Find(name); !ok {
			t.Errorf("the sidebar does not name %q", name)
		}
	}
	if _, ok := tt.Find(a.t("openRepo")); !ok {
		t.Error("the sidebar does not offer to open a repository")
	}
}

// A named row is a page switch, not decoration: clicking it opens its page.
func TestASidebarRowOpensItsPage(t *testing.T) {
	a := &app{locale: localeChinese, page: "branches", repo: &gitrepo.Repository{Root: t.TempDir()}}
	tt := sidebarTester(a)
	row, ok := tt.AccessByRole(ui.RoleButton, a.t("history"))
	if !ok {
		t.Fatal("the history row is not laid out")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if a.page != "history" {
		t.Errorf("clicking the history row opened %q", a.page)
	}
}

// Without a repository there is nothing on the other pages to show: the
// history row reads unavailable, and a group heading opens nothing.
func TestWithoutARepositoryTheSidebarOpensNoPage(t *testing.T) {
	a := &app{locale: localeChinese, page: "worktrees"}
	tt := sidebarTester(a)
	history, ok := tt.AccessByRole(ui.RoleButton, a.t("history"))
	if !ok {
		t.Fatal("the history row is not in the accessibility tree")
	}
	if !history.Disabled {
		t.Error("the history row is enabled with no repository")
	}
	heading, ok := tt.AccessByRole(ui.RoleDisclosure, a.t("branches"))
	if !ok {
		t.Fatal("the branches heading is not in the accessibility tree")
	}
	tt.ClickAt(heading.Bounds.X+heading.Bounds.W/2, heading.Bounds.Y+heading.Bounds.H/2)
	if a.page != "worktrees" {
		t.Errorf("a group heading opened %q with no repository", a.page)
	}
	if a.sidebarGroupOpen("branches") {
		t.Error("a group heading did not fold its list with no repository")
	}
}

// canceledContext stands in for the app's context in tests: the reads it
// starts fail at once rather than reaching for a repository that is not there.
func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// A branch in the sidebar is a scope for the history list, as the branch tree
// in the Git clients this follows is: one click goes from the ref to its
// commits, without finding the branch again on its own page.
func TestABranchInTheSidebarScopesTheHistory(t *testing.T) {
	a := &app{
		locale: localeChinese, page: "history",
		repo:     &gitrepo.Repository{Root: t.TempDir()},
		branches: []gitrepo.Branch{{Name: "main", Current: true}, {Name: "feat"}},
		// Scoping the history starts a walk; a canceled context ends it at
		// once instead of running git against the empty folder.
		ctx: canceledContext(),
	}
	tt := sidebarTester(a)
	row, ok := tt.AccessByRole(ui.RoleButton, "feat")
	if !ok {
		t.Fatal("the feat branch is not in the sidebar")
	}
	tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	if got := a.historyRef; got != (historyRef{Kind: historyRefLocal, Name: "feat"}) {
		t.Errorf("clicking feat scoped history to %+v", got)
	}
	// The page switch itself is posted to the main loop once the row's menu
	// would have closed; the headless tester does not run one.
}

// The checked-out branch used to append a middle dot to its name; a clipped
// long name lost the marker first. The mark is its own glyph now, so the name
// stays whole and its tooltip still gives the full text.
func TestTheCurrentBranchMarkIsNotPartOfItsName(t *testing.T) {
	const long = "feature/a-branch-name-long-enough-to-clip-in-the-sidebar"
	a := &app{
		locale:   localeChinese,
		page:     "history",
		repo:     &gitrepo.Repository{Root: t.TempDir()},
		branches: []gitrepo.Branch{{Name: long, Current: true}, {Name: "feat"}},
	}
	tt := sidebarTester(a)
	if _, ok := tt.Find(long + " ·"); ok {
		t.Error("the current mark is still part of the branch name")
	}
	row, ok := tt.AccessByRole(ui.RoleButton, long)
	if !ok {
		t.Fatal("the current branch is not named by its full name")
	}
	if !strings.Contains(row.Description, long) {
		t.Errorf("the current branch tooltip says %q; want the full name", row.Description)
	}
	if !strings.Contains(row.Description, a.t("current")) {
		t.Errorf("the current branch is not announced as current: %q", row.Description)
	}
	short, ok := tt.AccessByRole(ui.RoleButton, "feat")
	if !ok {
		t.Fatal("the short branch is not in the sidebar")
	}
	if short.Description != "" {
		t.Errorf("a short branch name opens a tooltip that repeats it: %q", short.Description)
	}
}

// The whole heading is the fold control, not just its 20-wide chevron: the
// reader aims at the group, and a click on its text folds the list instead of
// switching pages. The heading is a disclosure, so assistive technology hears
// whether the list below shows.
func TestTheWholeGroupHeadingFoldsItsList(t *testing.T) {
	a := &app{
		locale: localeChinese, page: "history",
		repo:     &gitrepo.Repository{Root: t.TempDir()},
		branches: []gitrepo.Branch{{Name: "main"}, {Name: "feat"}},
	}
	tt := sidebarTester(a)
	heading, ok := tt.AccessByRole(ui.RoleDisclosure, a.t("branches"))
	if !ok {
		t.Fatal("the branches group has no disclosure heading")
	}
	if !heading.Expanded {
		t.Error("the open branches group does not say it is expanded")
	}
	tt.ClickAt(heading.Bounds.X+heading.Bounds.W/2, heading.Bounds.Y+heading.Bounds.H/2)
	tt.Frame()
	if _, ok := tt.Find("feat"); ok {
		t.Error("folding the group leaves its branches on screen")
	}
	if a.page != "history" {
		t.Errorf("folding a group switched to %q", a.page)
	}
	if folded, ok := tt.AccessByRole(ui.RoleDisclosure, a.t("branches")); !ok || folded.Expanded {
		t.Error("the folded group still says it is expanded")
	}
	tt.ClickAt(heading.Bounds.X+heading.Bounds.W/2, heading.Bounds.Y+heading.Bounds.H/2)
	tt.Frame()
	if _, ok := tt.Find("feat"); !ok {
		t.Error("unfolding the group does not bring its branches back")
	}

	// The title itself is inside the control: clicking it away from the
	// chevron folds too.
	worktrees, ok := tt.AccessByRole(ui.RoleDisclosure, a.t("worktrees"))
	if !ok {
		t.Fatal("the worktrees group has no disclosure heading")
	}
	tt.ClickAt(worktrees.Bounds.X+worktrees.Bounds.W-20, worktrees.Bounds.Y+worktrees.Bounds.H/2)
	if a.page != "history" {
		t.Errorf("clicking the worktrees heading opened %q", a.page)
	}
}

// Every worktree in the sidebar looked alike, including the one this window
// has open, and a short folder name offered no way to see the full path. The
// current one now carries the same mark as the current branch, and every
// worktree names its path.
func TestTheSidebarNamesTheCurrentWorktree(t *testing.T) {
	current := filepath.Join(t.TempDir(), "app")
	other := filepath.Join(t.TempDir(), "app-copy")
	a := &app{
		locale: localeChinese, page: "history",
		repo:        &gitrepo.Repository{Root: current},
		worktrees:   []gitrepo.Worktree{{Path: current}, {Path: other}},
		sidebarOpen: map[string]bool{"worktrees": true},
	}
	tt := sidebarTester(a)
	row, ok := tt.AccessByRole(ui.RoleButton, "app")
	if !ok {
		t.Fatal("the current worktree is not in the sidebar")
	}
	if !strings.Contains(row.Description, a.t("current")) || !strings.Contains(row.Description, current) {
		t.Errorf("the current worktree is described as %q; want the current mark and %s", row.Description, current)
	}
	sibling, ok := tt.AccessByRole(ui.RoleButton, "app-copy")
	if !ok {
		t.Fatal("the other worktree is not in the sidebar")
	}
	if sibling.Description != other {
		t.Errorf("the other worktree is described as %q; want its full path %s", sibling.Description, other)
	}
}
