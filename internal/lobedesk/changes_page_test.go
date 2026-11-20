package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

// A repository opens on its newest commit; a working-tree selection left
// behind by the previous repository does not survive the switch.
func TestRepositorySwitchSelectsTheNewestCommit(t *testing.T) {
	a, done, _ := worktreeApp(t)
	_, _, nextRoot := worktreeApp(t)
	a.selectWorktree()
	a.openRepository(nextRoot, true)
	runContinuation(t, done)
	if a.repo == nil || a.repo.Root != nextRoot || a.loadErr != "" {
		t.Fatalf("repository switch failed: %s", a.loadErr)
	}
	if a.worktreeSelected || a.selectedSHA == "" {
		t.Fatalf("the new repository opened on the working tree rather than its newest commit: worktree=%v sha=%s", a.worktreeSelected, a.selectedSHA)
	}
}

func changesLayoutTester(a *app) *ui.Tester {
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() {
			a.titleBar(c)
			row := ui.Row(c).Grow(1).Basis(0).MinHeight(0)
			a.sidebarSplit.rowWidth = row.Bounds().W
			row.Children(func() {
				a.sidebar(c)
				a.splitHandle(c, &a.sidebarSplit, shareForWidth(a.sidebarSplit.rowWidth-splitterWidth, sidebarWidth), sidebarPaneMin, a.workbenchContentMin(), a.t("resizeSidebarPane"))
				a.contentCard(c)
			})
		})
	}, 980, 640)
	for range 3 {
		tt.Frame()
	}
	return tt
}

func TestChangesCommitEditorFitsAtMinimumWindow(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
			t.Run(fmt.Sprintf("%s/%v", language, scale), func(t *testing.T) {
				setUIFontScale(scale)
				a := &app{page: "history", locale: language, themeMode: "light", palette: themePaletteSuperGit, worktreeSelected: true,
					repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Branch: "main", Entries: []gitrepo.Entry{{Path: "same.go", Index: 'M', Worktree: 'M'}}},
					selectedPath: "same.go", selectedStaged: true, commitText: "retained title\n\nretained body", headMessage: "previous commit",
					diff: gitrepo.Diff{Text: "+index version", Lines: []gitrepo.DiffLine{{Kind: "add", Text: "+index version"}}},
				}
				a.detailTabs().Active = detailTabKey{Summary: true}
				tt := changesLayoutTester(a)
				summary, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitSummary"))
				if !ok {
					t.Fatal("the working tree's Summary tab hides the commit editor")
				}
				commit, ok := tt.AccessByRole(ui.RoleButton, a.t("commit"))
				if !ok {
					t.Fatal("missing fixed commit action")
				}
				if summary.Bounds.X < sidebarPaneMin || summary.Bounds.X+summary.Bounds.W > 980 || commit.Bounds.Y+commit.Bounds.H > 640 || commit.Bounds.H <= 0 {
					t.Fatalf("commit editor/action clipped at the minimum window: editor=%+v action=%+v", summary.Bounds, commit.Bounds)
				}
				if a.commitText != "retained title\n\nretained body" {
					t.Fatal("layout changed draft")
				}
				// The file tab reaches the patch in the same window, and no
				// page tab splits Changes off from History.
				a.detailTabs().Open(detailFileKey{Path: "same.go", Staged: true})
				tt.Frame()
				tt.Frame()
				if _, ok := tt.Find("+index version"); !ok {
					t.Fatal("working file patch was hidden")
				}
				for _, label := range []string{a.t("changes"), a.t("history"), a.t("stash")} {
					if tab, ok := tt.AccessByRole(ui.RoleTab, label); ok {
						t.Fatalf("the window grew a %q page tab: %+v", label, tab.Bounds)
					}
				}
			})
		}
	}
}

func TestChangesKeyboardTargetsVisibleFilesAndCommit(t *testing.T) {
	a := &app{page: "history", worktreeSelected: true, repo: &gitrepo.Repository{Root: t.TempDir()}, selectedPath: "same.go", selectedStaged: true,
		status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "same.go", Index: 'M', Worktree: 'M'}}}}
	a.detailTabs().Active = detailTabKey{Summary: true}
	if !a.worktreeFilesShown() || !a.detailNavigationAllowed() || !a.canToggleSelectedStage() || !a.canCommitShortcut() {
		t.Fatal("the working tree's visible controls do not receive file/diff/commit shortcuts")
	}
	// The commit box is docked under the files column, so a file tab cannot
	// cover it; the diff shortcuts reach the file's patch instead.
	a.detailTabs().Open(detailFileKey{Path: "same.go", Staged: true})
	if !a.worktreeDiffShown() || !a.canCommitShortcut() {
		t.Fatal("opening a file tab dropped the patch or the commit shortcut")
	}
	a.detailTabs().Active = detailTabKey{Summary: true}
	a.confirm.open = true
	if a.canCommitShortcut() || a.detailNavigationAllowed() {
		t.Fatal("dialog failed to guard Changes shortcuts")
	}
	a.confirm.open = false
	a.historyDetailsHidden = true
	if a.canCommitShortcut() {
		t.Fatal("a hidden details pane still accepts the commit shortcut")
	}
}

func TestChangesNavigationRestoresHistoricalSelectionAndVersionedTabs(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.commitText = "draft\n\nbody"
	a.amend = true
	page, err := a.repo.HistoryPage(a.ctx, nil, 0, 200)
	if err != nil || len(page.Commits) == 0 {
		t.Fatalf("fixture history: %v", err)
	}
	a.commits = page.Commits
	sha := page.Commits[0].SHA
	a.worktreeSelected = false
	a.selectedSHA = sha
	a.commitFilesLoaded = true
	a.detailTabs().Open(detailFileKey{Path: "same.go"})
	a.commitDiffScroll = ui.ScrollState{Y: 87}
	a.commitFilesScroll = ui.ScrollState{Y: 45}
	a.openPage("changes")
	if !a.worktreeSelected || a.page != "history" {
		t.Fatal("Changes did not select working-tree context")
	}
	waitOverview(t, a, callbacks)
	a.detailTabs().Open(detailFileKey{Path: "same.go"})
	a.detailTabs().Open(detailFileKey{Path: "same.go", Staged: true})
	a.selectedPath, a.selectedStaged = "same.go", true
	a.diffScroll = ui.ScrollState{Y: 123}
	a.commitFilesScroll = ui.ScrollState{Y: 67}
	// Choosing the commit's row is what leaves the working tree now: the
	// History list is the only page, so the row replaces the old page tab.
	a.selectCommit(sha)
	if a.worktreeSelected || a.selectedSHA != sha || a.detailTabs().Active.File.Path != "same.go" || a.commitDiffScroll.Y != 87 || a.commitFilesScroll.Y != 45 {
		t.Fatalf("History selection/scroll lost: sha=%s tab=%+v diff=%v files=%v", a.selectedSHA, a.detailTabs().Active, a.commitDiffScroll.Y, a.commitFilesScroll.Y)
	}
	a.openPage("changes")
	waitOverview(t, a, callbacks)
	if !a.worktreeSelected || len(a.detailTabs().OpenFiles) != 2 || !a.detailTabs().Active.File.Staged || a.diffScroll.Y != 123 || a.commitFilesScroll.Y != 67 || a.commitText != "draft\n\nbody" || !a.amend {
		t.Fatalf("Changes versioned context/draft lost: tabs=%+v diff=%v files=%v", a.detailTabs(), a.diffScroll.Y, a.commitFilesScroll.Y)
	}
}
