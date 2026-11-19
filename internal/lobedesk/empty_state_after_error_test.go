package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func pageTester(a *app, page func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(900).Height(650).Children(func() {
			page(c)
		})
	}, 900, 650)
}

// Standalone page fixtures have no outer workbench dividers.
func (a *app) historyPage(c *ui.Context) {
	width, _ := c.Size()
	a.historyPageAtWidth(c, width)
}

func (a *app) contentCard(c *ui.Context) {
	width, _ := c.Size()
	if a.sidebarSplit.rowWidth > 0 {
		width -= a.workbenchSidebarWidth() + splitterWidth
	}
	a.contentCardAtWidth(c, width)
}

// A failed read leaves the list empty, but that is not the same as "there are
// no branches": the page has to show the error alone, without an empty state
// inviting the reader to create what it says could not be read.
func TestBranchesPageShowsNoEmptyStateAfterAFailedRead(t *testing.T) {
	const err = "fatal: cannot read refs"
	a := &app{locale: localeChinese, branchesErr: err}
	tt := pageTester(a, a.branchesPage)
	if _, ok := tt.Find(err); !ok {
		t.Error("the page hides the read error")
	}
	if _, ok := tt.Find(a.t("noBranches")); ok {
		t.Error("the page offers to create a branch beside the error")
	}

	a = &app{locale: localeChinese, tagsErr: err}
	tt = pageTester(a, a.branchesPage)
	if _, ok := tt.Find(err); !ok {
		t.Error("the page hides the tag read error")
	}
	if _, ok := tt.Find(a.t("noTags")); ok {
		t.Error("the page offers an empty tag section beside the error")
	}
}

func TestStashPageShowsNoEmptyStateAfterAFailedRead(t *testing.T) {
	const err = "fatal: cannot read stashes"
	a := &app{locale: localeChinese, stashesErr: err}
	tt := pageTester(a, a.stashPage)
	if _, ok := tt.Find(err); !ok {
		t.Error("the page hides the read error")
	}
	if _, ok := tt.Find(a.t("noStashes")); ok {
		t.Error("the page offers an empty stash list beside the error")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retry")); !ok {
		t.Error("the failed stash read offers no retry")
	}
}

func TestWorktreesPageShowsNoEmptyStateAfterAFailedRead(t *testing.T) {
	const err = "fatal: cannot read worktrees"
	a := &app{locale: localeChinese, worktreesErr: err}
	tt := pageTester(a, a.worktreesPage)
	if _, ok := tt.Find(err); !ok {
		t.Error("the page hides the read error")
	}
	if _, ok := tt.Find(a.t("noWorktrees")); ok {
		t.Error("the page offers to create a worktree beside the worktree error")
	}
	// The submodule read is a separate command and succeeded, so an empty
	// submodule section still means the repository has none.
	if _, ok := tt.Find(a.t("noSubmodules")); !ok {
		t.Error("a failed worktree read hides the submodule empty state")
	}
	if _, ok := tt.Find(a.t("noSubmodulesHint")); !ok {
		t.Error("a failed worktree read hides the submodule hint")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retryWorktrees")); !ok {
		t.Error("the failed worktree read offers no retry")
	}
}

// Each group's read stands on its own: a submodule failure must not empty the
// worktree list, and the subtitle names only the group that failed.
func TestAWorkspaceReadFailureStaysInItsSection(t *testing.T) {
	const err = "fatal: cannot read submodules"
	a := &app{locale: localeChinese, submodulesErr: err, worktrees: []gitrepo.Worktree{{Path: "/tmp/wt", Branch: "wt"}}}
	tt := pageTester(a, a.worktreesPage)
	if _, ok := tt.Find(err); !ok {
		t.Error("the page hides the submodule read error")
	}
	if _, ok := tt.Find(a.t("noSubmodules")); ok {
		t.Error("the page offers to add a submodule beside the error")
	}
	if _, ok := tt.Find("/tmp/wt"); !ok {
		t.Error("a failed submodule read hides the worktree that did load")
	}
	if _, ok := tt.Find("1 个工作树 · 子模块读取失败"); !ok {
		t.Error("the subtitle does not name the failed group")
	}
}

// Both workspace reads can fail together; each section keeps its own named
// retry so the reader knows which list the button rereads.
func TestBothWorkspaceSectionsOfferTheirOwnRetry(t *testing.T) {
	a := &app{locale: localeChinese, worktreesErr: "fatal: cannot read worktrees", submodulesErr: "fatal: cannot read submodules"}
	tt := pageTester(a, a.worktreesPage)
	for _, name := range []string{a.t("retryWorktrees"), a.t("retrySubmodules")} {
		if _, ok := tt.AccessByRole(ui.RoleButton, name); !ok {
			t.Errorf("the failed reads do not offer %q", name)
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, a.t("retry")); ok {
		t.Error("the page still shows a plain Retry beside the named ones")
	}
}
