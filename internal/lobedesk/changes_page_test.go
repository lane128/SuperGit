package lobedesk

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestChangesRepositorySwitchKeepsWorkingTreeContext(t *testing.T) {
	for _, page := range []string{"changes", "history"} {
		t.Run(page, func(t *testing.T) {
			a, done, _ := worktreeApp(t)
			_, _, nextRoot := worktreeApp(t)
			a.page = page
			a.openRepository(nextRoot, true)
			runContinuation(t, done)
			if a.repo == nil || a.repo.Root != nextRoot || a.loadErr != "" {
				t.Fatalf("repository switch failed: %s", a.loadErr)
			}
			if page == "changes" && (!a.worktreeSelected || a.selectedSHA != "") {
				t.Fatalf("Changes opened historical context: worktree=%v sha=%s", a.worktreeSelected, a.selectedSHA)
			}
			if page == "history" && (a.worktreeSelected || a.selectedSHA == "") {
				t.Fatal("History no longer selects the initial commit")
			}
		})
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

func TestChangesKeepsCommitBesideFileDiffAtMinimumWindow(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
			t.Run(fmt.Sprintf("%s/%v", language, scale), func(t *testing.T) {
				setUIFontScale(scale)
				a := &app{page: "changes", locale: language, themeMode: "light", palette: themePaletteSuperGit, worktreeSelected: true,
					repo: &gitrepo.Repository{Root: t.TempDir()}, status: gitrepo.Status{Branch: "main", Entries: []gitrepo.Entry{{Path: "same.go", Index: 'M', Worktree: 'M'}}},
					selectedPath: "same.go", selectedStaged: true, commitText: "retained title\n\nretained body", headMessage: "previous commit",
					historyDetailsHidden: true, detailsChangesHidden: true,
					diff: gitrepo.Diff{Text: "+index version", Lines: []gitrepo.DiffLine{{Kind: "add", Text: "+index version"}}},
				}
				a.detailTabs().Open(detailFileKey{Path: "same.go", Staged: true})
				tt := changesLayoutTester(a)
				summary, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitSummary"))
				if !ok {
					t.Fatal("opening a file hides the commit editor")
				}
				commit, ok := tt.AccessByRole(ui.RoleButton, a.t("commit"))
				if !ok {
					t.Fatal("missing fixed commit action")
				}
				patch, ok := tt.Find("+index version")
				if !ok {
					t.Fatal("working file patch was hidden by History preferences")
				}
				if summary.Bounds.X < sidebarPaneMin || summary.Bounds.X+summary.Bounds.W > patch.X || commit.Bounds.Y+commit.Bounds.H > 640 || commit.Bounds.H <= 0 {
					t.Fatalf("commit editor/action must stay in Files column beside patch: editor=%+v action=%+v patch=%+v", summary.Bounds, commit.Bounds, patch)
				}
				if a.commitText != "retained title\n\nretained body" {
					t.Fatal("layout changed draft")
				}
				for _, label := range []string{a.t("changes"), a.t("history"), a.t("stash")} {
					tab, ok := tt.AccessByRole(ui.RoleTab, label)
					if !ok || tab.Bounds.X < 0 || tab.Bounds.X+tab.Bounds.W > 980 {
						t.Fatalf("missing or clipped page tab %q", label)
					}
				}
			})
		}
	}
}

func TestChangesKeyboardTargetsVisibleFilesAndCommit(t *testing.T) {
	a := &app{page: "changes", worktreeSelected: true, repo: &gitrepo.Repository{Root: t.TempDir()}, selectedPath: "same.go", selectedStaged: true,
		status: gitrepo.Status{Entries: []gitrepo.Entry{{Path: "same.go", Index: 'M', Worktree: 'M'}}}, historyDetailsHidden: true}
	a.detailTabs().Open(detailFileKey{Path: "same.go", Staged: true})
	if !a.worktreeFilesShown() || !a.detailNavigationAllowed() || !a.canToggleSelectedStage() || !a.canCommitShortcut() || !a.worktreeDiffShown() {
		t.Fatal("visible Changes controls do not receive file/diff/commit shortcuts independently of History visibility")
	}
	a.confirm.open = true
	if a.canCommitShortcut() || a.detailNavigationAllowed() {
		t.Fatal("dialog failed to guard Changes shortcuts")
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
	if !a.worktreeSelected || a.page != "changes" {
		t.Fatal("Changes did not select working-tree context")
	}
	waitOverview(t, a, callbacks)
	a.detailTabs().Open(detailFileKey{Path: "same.go"})
	a.detailTabs().Open(detailFileKey{Path: "same.go", Staged: true})
	a.selectedPath, a.selectedStaged = "same.go", true
	a.diffScroll = ui.ScrollState{Y: 123}
	a.commitFilesScroll = ui.ScrollState{Y: 67}
	a.openPage("history")
	if a.worktreeSelected || a.selectedSHA != sha || a.detailTabs().Active.File.Path != "same.go" || a.commitDiffScroll.Y != 87 || a.commitFilesScroll.Y != 45 {
		t.Fatalf("History selection/scroll lost: sha=%s tab=%+v diff=%v files=%v", a.selectedSHA, a.detailTabs().Active, a.commitDiffScroll.Y, a.commitFilesScroll.Y)
	}
	a.openPage("changes")
	waitOverview(t, a, callbacks)
	if !a.worktreeSelected || len(a.detailTabs().OpenFiles) != 2 || !a.detailTabs().Active.File.Staged || a.diffScroll.Y != 123 || a.commitFilesScroll.Y != 67 || a.commitText != "draft\n\nbody" || !a.amend {
		t.Fatalf("Changes versioned context/draft lost: tabs=%+v diff=%v files=%v", a.detailTabs(), a.diffScroll.Y, a.commitFilesScroll.Y)
	}
}
