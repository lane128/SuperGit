package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// A reader that hears only "selected" does not know where in the list it is.
// Each row carries its place and the size of its list, and the History list
// puts the rows inside one list role so the two numbers are read together:
// the working tree is the first row, then the stashes, then the commits.
func TestStashListReportsPositions(t *testing.T) {
	a := &app{
		locale: localeChinese,
		stashes: []gitrepo.Stash{
			{SHA: "aaaa", Ref: "stash@{0}", Message: "first"},
			{SHA: "bbbb", Ref: "stash@{1}", Message: "second"},
			{SHA: "cccc", Ref: "stash@{2}", Message: "third"},
		},
	}
	a.selectedStash = "bbbb"
	tt := pageTester(a, a.historyPage)
	if _, ok := tt.AccessByRole(ui.RoleList, ""); !ok {
		t.Fatal("the stash rows are not inside a list")
	}
	second, ok := tt.AccessByRole(ui.RoleOption, "second")
	if !ok {
		t.Fatal("the second stash is not an option")
	}
	// The working tree's row is the list's first row; the stashes follow it.
	if second.PosInSet != 3 || second.SetSize != 4 {
		t.Errorf("the second stash reads place %d of %d; want 3 of 4", second.PosInSet, second.SetSize)
	}
	if !second.Selected {
		t.Errorf("the second stash reads %+v; want it selected", second)
	}
}

func TestRowsCarryTheirPosition(t *testing.T) {
	a := &app{locale: localeChinese, selectedSHA: "bbbb"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(300).Children(func() {
			a.commitRow(c, gitrepo.Commit{SHA: "bbbb", Subject: "second"}, 6, 0, 19, &commitGraph{})
		})
	}, 700, 300)
	commit, ok := tt.AccessByRole(ui.RoleOption, "second")
	if !ok {
		t.Fatal("the commit row is not an option")
	}
	if commit.PosInSet != 7 || commit.SetSize != 19 {
		t.Errorf("the commit reads place %d of %d; want 7 of 19", commit.PosInSet, commit.SetSize)
	}

	tt = ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(500).Height(200).Children(func() {
			a.stashRow(c, gitrepo.Stash{SHA: "aaaa", Ref: "stash@{0}", Message: "only"}, 4, 9)
		})
	}, 500, 200)
	stash, ok := tt.AccessByRole(ui.RoleOption, "only")
	if !ok {
		t.Fatal("the stash row is not an option")
	}
	if stash.PosInSet != 4 || stash.SetSize != 9 {
		t.Errorf("the stash reads place %d of %d; want 4 of 9", stash.PosInSet, stash.SetSize)
	}
}

func TestBranchPageRowsReportPositions(t *testing.T) {
	a := &app{
		locale: localeChinese,
		page:   "branches",
		branches: []gitrepo.Branch{
			{Name: "main", SHA: "aaaa", Current: true},
			{Name: "feat", SHA: "bbbb"},
			{Name: "topic", SHA: "cccc"},
		},
		tags: []gitrepo.Tag{{Name: "v1.0", Commit: gitrepo.Commit{SHA: "dddd"}}},
	}
	a.branchSel = "local:main"
	tt := pageTester(a, a.branchesPage)
	lists := 0
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleList {
			lists++
		}
	}
	if lists < 2 {
		t.Fatalf("the branch page shows %d lists; want at least the local and tag lists", lists)
	}
	feat, ok := tt.AccessByRole(ui.RoleOption, "feat")
	if !ok {
		t.Fatal("the feat row is not an option")
	}
	if feat.PosInSet != 2 || feat.SetSize != 3 {
		t.Errorf("feat reads place %d of %d; want 2 of 3", feat.PosInSet, feat.SetSize)
	}
	tag, ok := tt.AccessByRole(ui.RoleOption, "v1.0")
	if !ok {
		t.Fatal("the tag row is not an option")
	}
	if tag.PosInSet != 1 || tag.SetSize != 1 {
		t.Errorf("the tag reads place %d of %d; want 1 of 1", tag.PosInSet, tag.SetSize)
	}
}

func TestTagRowReportsTheSelectedTag(t *testing.T) {
	a := &app{
		locale: localeChinese,
		page:   "branches",
		branches: []gitrepo.Branch{
			{Name: "main", SHA: "aaaa", Current: true},
		},
		tags: []gitrepo.Tag{{Name: "v1.0", Commit: gitrepo.Commit{SHA: "dddd"}}},
	}
	a.branchSel = "tag:v1.0"
	tt := pageTester(a, a.branchesPage)
	tag, ok := tt.AccessByRole(ui.RoleOption, "v1.0")
	if !ok {
		t.Fatal("the tag row is not an option")
	}
	if !tag.Selected {
		t.Errorf("the chosen tag reads %+v; want it selected", tag)
	}
}

// The history page's list is a list too: a reader hears each commit's place
// among the loaded rows, not only the selected state.
func TestHistoryPageListReportsPositions(t *testing.T) {
	a := &app{
		locale:      localeChinese,
		page:        "history",
		selectedSHA: "bbbb",
		commits: []gitrepo.Commit{
			{SHA: "aaaa", Subject: "first"},
			{SHA: "bbbb", Subject: "second"},
		},
	}
	tt := pageTester(a, a.historyPage)
	if _, ok := tt.AccessByRole(ui.RoleList, ""); !ok {
		t.Fatal("the commit rows are not inside a list")
	}
	second, ok := tt.AccessByRole(ui.RoleOption, "second")
	if !ok {
		t.Fatal("the second commit is not an option")
	}
	// The working tree's row is the list's first row, so two commits are
	// places 2 and 3 of 3.
	if second.PosInSet != 3 || second.SetSize != 3 {
		t.Errorf("the second commit reads place %d of %d; want 3 of 3", second.PosInSet, second.SetSize)
	}
	if !second.Selected {
		t.Errorf("the chosen commit reads %+v; want it selected", second)
	}
	worktree, ok := tt.AccessByRole(ui.RoleOption, a.t("noChangesShort"))
	if !ok {
		t.Fatal("the working tree's row is not an option")
	}
	if worktree.PosInSet != 1 || worktree.SetSize != 3 {
		t.Errorf("the working tree reads place %d of %d; want 1 of 3", worktree.PosInSet, worktree.SetSize)
	}
}

// The worktree page holds two lists, and each row reports its place in its
// own list rather than across both.
func TestWorktreePageRowsReportPositions(t *testing.T) {
	a := &app{
		locale: localeChinese,
		page:   "worktrees",
		worktrees: []gitrepo.Worktree{
			{Path: "/repos/main", Branch: "main"},
			{Path: "/repos/feat", Branch: "feat"},
		},
		submodules: []gitrepo.Submodule{
			{Name: "lib", Path: "vendor/lib", Initialized: true},
			{Name: "docs", Path: "docs", Initialized: true},
		},
	}
	tt := pageTester(a, a.worktreesPage)
	lists := 0
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleList {
			lists++
		}
	}
	if lists < 2 {
		t.Fatalf("the worktree page shows %d lists; want the worktree and submodule lists", lists)
	}
	feat, ok := tt.AccessByRole(ui.RoleOption, "feat")
	if !ok {
		t.Fatal("the feat worktree is not an option")
	}
	if feat.PosInSet != 2 || feat.SetSize != 2 {
		t.Errorf("feat reads place %d of %d; want 2 of 2", feat.PosInSet, feat.SetSize)
	}
	docs, ok := tt.AccessByRole(ui.RoleOption, "docs")
	if !ok {
		t.Fatal("the docs submodule is not an option")
	}
	if docs.PosInSet != 2 || docs.SetSize != 2 {
		t.Errorf("docs reads place %d of %d; want 2 of 2", docs.PosInSet, docs.SetSize)
	}
}

// A search narrows the list, so the place and size a reader hears describe
// the rows still shown, not the whole history behind them.
func TestFilteredHistoryReportsTheVisiblePositions(t *testing.T) {
	a := &app{
		locale:       localeChinese,
		page:         "history",
		historyQuery: "second",
		selectedSHA:  "bbbb",
		commits: []gitrepo.Commit{
			{SHA: "aaaa", Subject: "first"},
			{SHA: "bbbb", Subject: "second"},
			{SHA: "cccc", Subject: "third"},
		},
	}
	tt := pageTester(a, a.historyPage)
	second, ok := tt.AccessByRole(ui.RoleOption, "second")
	if !ok {
		t.Fatal("the matching commit is not an option")
	}
	if second.PosInSet != 2 || second.SetSize != 2 {
		t.Errorf("the only match reads place %d of %d; want 2 of 2", second.PosInSet, second.SetSize)
	}
	if _, ok := tt.AccessByRole(ui.RoleOption, "first"); ok {
		t.Error("a filtered-out commit is still an option")
	}
}
