package lobedesk

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func browsingApp(t *testing.T) *app {
	a := headShortcutApp(t)
	a.worktreeSelected = false
	a.selectedSHA = "a"
	a.commits = []gitrepo.Commit{{SHA: "a", Subject: "original", Parents: []string{"p"}}, {SHA: "b", Subject: "other child", Parents: []string{"p"}}, {SHA: "p", Subject: "parent"}}
	return a
}

func TestHistoryNavigationFollowsActualForkRoute(t *testing.T) {
	a := browsingApp(t)
	a.historyQuery = "original"
	a.navigateToParent()
	if a.selectedSHA != "p" || a.historyQuery != "" {
		t.Fatal("parent jump failed")
	}
	a.navigateToChild()
	a.chooseChildNavigation("b", a.childNavigation.generation)
	if a.selectedSHA != "b" {
		t.Fatal("child choice failed")
	}
	for _, want := range []struct {
		direction  int
		sha, query string
	}{{-1, "p", ""}, {-1, "a", "original"}, {1, "p", ""}, {1, "b", ""}} {
		a.navigateHistory(want.direction)
		if a.selectedSHA != want.sha || a.historyQuery != want.query || !a.revealSelection {
			t.Fatalf("route: got %s/%q want %s/%q", a.selectedSHA, a.historyQuery, want.sha, want.query)
		}
	}
	a.navigateHistory(-1)
	a.navigateToChild()
	a.chooseChildNavigation("a", a.childNavigation.generation)
	if a.canNavigateHistory(1) {
		t.Fatal("new path retained old forward branch")
	}
	a.selectCommit("b")
	if a.canNavigateHistory(-1) {
		t.Fatal("manual choice retained old route")
	}
}

func TestHistoryNavigationFailureAndScope(t *testing.T) {
	for _, change := range []string{"root", "epoch", "reload", "file", "worktree", "hidden"} {
		t.Run(change, func(t *testing.T) {
			a := browsingApp(t)
			a.navigateToParent()
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "reload":
				a.resetHistoryTraversal()
			case "file":
				a.fileFilter = "file.go"
			case "worktree":
				a.selectWorktree()
			case "hidden":
				a.historyQuery = "not visible"
			}
			before := a.selectedSHA
			a.navigateHistory(-1)
			if a.selectedSHA != before || a.canNavigateHistory(-1) {
				t.Fatal("invalid scope replayed")
			}
		})
	}
	a := browsingApp(t)
	a.commits[0].Parents = []string{"missing"}
	a.historyPaused = true
	a.navigateToParent()
	if a.canNavigateHistory(-1) || a.selectedSHA != "a" {
		t.Fatal("failed jump created route")
	}
}

func TestHistoryNavigationPendingQueryCancels(t *testing.T) {
	a := browsingApp(t)
	a.commits = []gitrepo.Commit{a.commits[0]}
	a.historyMore, a.historyLoadingMore = true, true
	a.navigateToParent()
	if a.pendingJumpSHA != "p" {
		t.Fatal("fixture did not start paging")
	}
	a.historyQuery = "original"
	a.commits = append(a.commits, gitrepo.Commit{SHA: "p", Subject: "parent"})
	a.resumePendingCommitJump()
	if a.selectedSHA != "a" || a.historyQuery != "original" || a.pendingJumpSHA != "" || a.canNavigateHistory(-1) {
		t.Fatal("late jump consumed edited query")
	}
}

func TestHistoryNavigationPendingEditThenClear(t *testing.T) {
	a := browsingApp(t)
	a.commits = a.commits[:1]
	a.historyMore, a.historyLoadingMore = true, true
	a.navigateToParent()
	tt := pageTester(a, func(c *ui.Context) { a.historyPage(c) })
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("original")
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if a.historyQuery != "" {
		t.Fatal("fixture did not clear real search")
	}
	a.commits = append(a.commits, gitrepo.Commit{SHA: "p", Subject: "parent"})
	a.resumePendingCommitJump()
	if a.selectedSHA != "a" || a.canNavigateHistory(-1) {
		t.Fatal("edited-then-cleared search revived old jump")
	}
}

func TestHistoryNavigationPendingModal(t *testing.T) {
	a := browsingApp(t)
	a.commits = a.commits[:1]
	a.historyMore, a.historyLoadingMore = true, true
	a.navigateToParent()
	a.openCommandPalette()
	if !a.commandPalette.open {
		t.Fatal("fixture did not open modal during paging")
	}
	a.commits = append(a.commits, gitrepo.Commit{SHA: "p", Subject: "parent"})
	a.resumePendingCommitJump()
	if a.selectedSHA != "a" || a.pendingJumpSHA != "" {
		t.Fatal("page callback navigated behind modal")
	}
}

func TestHistoryNavigationActualShortcutAndEditors(t *testing.T) {
	a := browsingApp(t)
	a.navigateToParent()
	mod := ui.Alt
	if runtime.GOOS == "darwin" {
		mod = ui.Cmd
	}
	tt := childNavigationTester(a)
	tt.Key(mod, ui.KeyLeft)
	tt.Frame()
	if a.selectedSHA != "a" {
		t.Fatal("Back shortcut did not replay source")
	}
	tt.Key(mod, ui.KeyRight)
	tt.Frame()
	if a.selectedSHA != "p" {
		t.Fatal("Forward shortcut did not replay target")
	}
	for _, label := range []string{a.t("commitMessage"), a.t("commitMessage")} {
		editor, ok := tt.AccessByRole(ui.RoleTextField, label)
		if !ok {
			t.Fatalf("missing real editor %q", label)
		}
		tt.ClickAt(editor.Bounds.X+10, editor.Bounds.Y+10)
		tt.Key(mod, ui.KeyLeft)
		tt.Frame()
		if a.selectedSHA != "p" || a.commitText != "retained draft" {
			t.Fatal("navigation swallowed editor key")
		}
	}
}

func TestHistoryNavigationParentButtonAndFrameCache(t *testing.T) {
	a := browsingApp(t)
	a.historyQuery = "original"
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); a.commitInfoContent(c, a.commits[0], theme.TokensOf(c)) }, 600, 500)
	button, ok := tt.AccessByRole(ui.RoleButton, "p")
	if !ok {
		t.Fatal("missing actual parent button")
	}
	tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	tt.Frame()
	if a.selectedSHA != "p" {
		t.Fatal("button did not jump")
	}
	a.historyRowsCache = a.commits
	a.historyRowsSet = true
	a.historyRowsFrame = a.frameSeq
	a.navigateHistory(-1)
	rows := a.historyRows()
	if a.selectedSHA != "a" || len(rows) != 1 || rows[0].SHA != "a" {
		t.Fatal("Back reused pre-query frame rows")
	}
	a.historyQuery = ""
	a.historyRowsSet = false
	a.moveSelection(1)
	if a.selectedSHA != "b" || a.canNavigateHistory(-1) {
		t.Fatal("subsequent list movement used stale position")
	}
}

func TestHistoryNavigationOwnedPagingAndCancellation(t *testing.T) {
	for _, change := range []string{"success", "selection", "query", "pause", "modal"} {
		t.Run(change, func(t *testing.T) {
			a, callbacks, _ := automaticHistoryFixture(t, 420)
			source := a.commits[199]
			if len(source.Parents) != 1 {
				t.Fatal("missing fixture parent")
			}
			a.page = "history"
			a.worktreeSelected = false
			a.selectedSHA = source.SHA
			a.commitText = "draft\n\nbody"
			a.amend = true
			fingerprint := func() string {
				b, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
				if err != nil {
					t.Fatal(err)
				}
				return pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") + pullFixtureGit(t, a.repo.Root, "symbolic-ref", "HEAD") + pullFixtureGit(t, a.repo.Root, "for-each-ref") + pullFixtureGit(t, a.repo.Root, "ls-files", "--stage") + pullFixtureGit(t, a.repo.Root, "status", "--porcelain=v1") + string(b)
			}
			before := fingerprint()
			a.navigateToParent()
			if a.selectedSHA != source.SHA || a.canNavigateHistory(-1) || a.pendingJumpSHA != source.Parents[0] {
				t.Fatal("pending jump advanced before page completion")
			}
			pageCallback := historyCompletion(t, callbacks)
			switch change {
			case "selection":
				a.selectCommit(a.commits[0].SHA)
			case "query":
				a.historyQuery = source.Subject
			case "pause":
				a.pauseHistoryTraversal()
			case "modal":
				a.openCommandPalette()
			}
			expected := a.selectedSHA
			pageCallback()
			for a.historyLoadingMore || a.commitFilesLoading || a.commitDiffLoading {
				historyCompletion(t, callbacks)()
			}
			if change == "success" {
				if a.selectedSHA != source.Parents[0] {
					t.Fatal("page did not complete exact parent")
				}
				a.navigateHistory(-1)
				if a.selectedSHA != source.SHA {
					t.Fatal("page success did not record source")
				}
				a.navigateHistory(1)
				if a.selectedSHA != source.Parents[0] {
					t.Fatal("page success did not record target")
				}
			} else if a.selectedSHA != expected || a.pendingJumpSHA != "" {
				t.Fatal("cancelled late page changed selection")
			}
			for a.commitFilesLoading || a.commitDiffLoading {
				historyCompletion(t, callbacks)()
			}
			if fingerprint() != before || a.commitText != "draft\n\nbody" || !a.amend {
				t.Fatal("read-only route changed Git or draft")
			}
		})
	}
}

func TestHistoryNavigationRetainedCallbacksRespectGuards(t *testing.T) {
	for _, reason := range []string{"busy", "loading", "reload", "chooser", "ref", "prompt", "palette"} {
		t.Run(reason, func(t *testing.T) {
			a := browsingApp(t)
			a.navigateToParent()
			var run func(*app)
			for _, s := range shortcutTable() {
				if s.ID == "history.back" {
					run = s.Run
				}
			}
			if run == nil {
				t.Fatal("missing shared Back action")
			}
			switch reason {
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "reload":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "ref":
				a.historyRefOpen = true
			case "prompt":
				a.prompt.open = true
				a.prompt.root = a.repo.Root
			case "palette":
				a.openCommandPalette()
			}
			a.dispatchWindowShortcut(run)
			if a.selectedSHA != "p" {
				t.Fatal("retained callback bypassed navigation guard")
			}
		})
	}
}

func TestHistoryNavigationActualSearchRetainsEditingKey(t *testing.T) {
	a := browsingApp(t)
	a.navigateToParent()
	tt := pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.historyPage(c) })
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("parent")
	tt.Frame()
	mod := ui.Alt
	if runtime.GOOS == "darwin" {
		mod = ui.Cmd
	}
	tt.Key(mod, ui.KeyLeft)
	tt.Frame()
	if a.selectedSHA != "p" || a.historyQuery != "parent" {
		t.Fatal("Back swallowed history search editing key")
	}
}

func TestHistoryNavigationOwnedMergeRoute(t *testing.T) {
	a, callbacks, parent := quickCheckoutFixture(t)
	first := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD")
	tree := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD^{tree}")
	second := pullFixtureGit(t, a.repo.Root, "commit-tree", tree, "-p", parent, "-m", "second child")
	merge := pullFixtureGit(t, a.repo.Root, "commit-tree", tree, "-p", first, "-p", second, "-m", "merge")
	pullFixtureGit(t, a.repo.Root, "update-ref", "refs/heads/route-merge", merge)
	page, err := a.repo.ReadHistory(a.ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	a.commits = page.Commits
	a.page = "history"
	a.selectedSHA = merge
	a.worktreeSelected = false
	fingerprint := func() string {
		b, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
		if err != nil {
			t.Fatal(err)
		}
		return pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") + pullFixtureGit(t, a.repo.Root, "symbolic-ref", "HEAD") + pullFixtureGit(t, a.repo.Root, "for-each-ref") + pullFixtureGit(t, a.repo.Root, "ls-files", "--stage") + pullFixtureGit(t, a.repo.Root, "status", "--porcelain=v1") + string(b)
	}
	before := fingerprint()
	a.navigateToSecondParent()
	if a.selectedSHA != second {
		t.Fatal("second parent jump failed")
	}
	a.navigateToParent()
	a.navigateToChild()
	a.chooseChildNavigation(first, a.childNavigation.generation)
	if a.selectedSHA != first {
		t.Fatal("explicit fork choice failed")
	}
	for _, want := range []string{parent, second, merge} {
		a.navigateHistory(-1)
		if a.selectedSHA != want {
			t.Fatal("Back lost real merge route")
		}
	}
	for _, want := range []string{second, parent, first} {
		a.navigateHistory(1)
		if a.selectedSHA != want {
			t.Fatal("Forward lost real merge route")
		}
	}
	for a.commitFilesLoading || a.commitDiffLoading {
		historyCompletion(t, callbacks)()
	}
	if fingerprint() != before || a.commitText != "draft\n\nbody" || !a.amend {
		t.Fatal("merge browsing changed Git or draft")
	}
}

func TestHistoryNavigationSettingsAtMinimumWidth(t *testing.T) {
	isolateSettings(t)
	previous := uiFontScaleValue()
	setUIFontScale(1.15)
	t.Cleanup(func() { setUIFontScale(previous) })
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			t.Run(fmt.Sprintf("%s-%s", language, mode), func(t *testing.T) {
				a := &app{locale: language, themeMode: mode, settingsSection: "shortcuts"}
				tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.settingsPage(c) }, settingsWindowMinW, settingsWindowMinH)
				bounds, ok := tt.Find(a.t("historyBack"))
				if !ok {
					t.Fatal("Back action missing from Settings")
				}
				a.settingsScroll.Y += bounds.Y - 180
				for range 4 {
					tt.Frame()
				}
				for _, key := range []string{"historyBack", "historyForward"} {
					r, ok := tt.Find(a.t(key))
					if !ok || r.Y < 0 || r.Y+r.H > settingsWindowMinH || r.X+r.W > settingsWindowMinW {
						t.Fatalf("navigation label outside Settings: %+v", r)
					}
				}
				if dir := os.Getenv("SUPERGIT_NAVIGATION_CAPTURE_DIR"); dir != "" {
					f, err := os.Create(filepath.Join(dir, fmt.Sprintf("settings-%s-%s.png", language, mode)))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					closeErr := f.Close()
					if err != nil {
						t.Fatal(err)
					}
					if closeErr != nil {
						t.Fatal(closeErr)
					}
				}
			})
		}
	}
}

func TestHistoryNavigationSameHeadUpdatesQuery(t *testing.T) {
	a := browsingApp(t)
	a.historyQuery = "original"
	a.navigateToParent()
	a.historyQuery = "parent"
	a.status.Head = "p"
	a.navigateToHead()
	a.navigateHistory(-1)
	if a.selectedSHA != "a" || a.historyQuery != "original" {
		t.Fatal("same HEAD duplicated anchor")
	}
	a.navigateHistory(1)
	if a.selectedSHA != "p" || a.historyQuery != "" || a.canNavigateHistory(1) {
		t.Fatal("same HEAD restored stale query")
	}
}
