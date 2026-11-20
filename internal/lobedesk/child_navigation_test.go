package lobedesk

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func childNavigationApp(t *testing.T) *app {
	a := headShortcutApp(t)
	a.worktreeSelected = false
	a.selectedSHA = "source"
	a.commits = []gitrepo.Commit{{SHA: "child-one", Subject: "first child", Parents: []string{"source"}}, {SHA: "child-two", Subject: "merge child", Parents: []string{"other", "source"}}, {SHA: "other"}, {SHA: "source", Subject: "source marker"}}
	return a
}
func childNavigationTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() { a.commitMessageField(c); a.childNavigationDialog(c) })
	}, 600, 500)
}

func TestChildNavigationUsesFullPrefixAndSecondParentEdges(t *testing.T) {
	a := childNavigationApp(t)
	a.historyQuery = "source marker"
	children := a.childNavigationCandidates("source")
	if len(children) != 2 || children[0].SHA != "child-one" || children[1].SHA != "child-two" {
		t.Fatalf("wrong direct children: %+v", children)
	}
	a.commits = append(a.commits[:1], a.commits[3:]...)
	a.historyPaused = true
	a.historyMore = true
	a.historyPageErr = "tail read failed"
	tt := childNavigationTester(a)
	tt.Key(ui.Alt, ui.KeyUp)
	tt.Frame()
	if a.selectedSHA != "child-one" || !a.revealSelection || a.historyQuery != "" || a.commitText != "retained draft" {
		t.Fatal("in-scope known child failed to jump through search/paused tail")
	}
}

func TestChildNavigationPickerAndStaleChoices(t *testing.T) {
	for _, change := range []string{"choose", "cancel", "root", "epoch", "reload", "selection", "reopen", "edge"} {
		t.Run(change, func(t *testing.T) {
			a := childNavigationApp(t)
			tt := childNavigationTester(a)
			tt.Key(ui.Alt, ui.KeyUp)
			tt.Frame()
			if !a.childNavigation.open || a.selectedSHA != "source" {
				t.Fatal("multiple children were not offered explicitly")
			}
			generation := a.childNavigation.generation
			page := a.page
			tt.Key(ui.Cmd, ui.Key4)
			tt.Frame()
			for _, binding := range shortcutTable() {
				if binding.ID == "page.worktrees" {
					a.dispatchWindowShortcut(binding.Run)
				}
			}
			if a.page != page {
				t.Fatal("background page shortcut leaked through child picker")
			}
			switch change {
			case "choose":
				tt.Type("no such child")
				tt.Frame()
				if !tt.HasText(a.t("searchNoMatch")) {
					t.Fatal("missing commit search empty-state message")
				}
				tt.Key(ui.Cmd, ui.KeyA)
				tt.Type("merge child")
				tt.Frame()
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				if a.selectedSHA != "child-two" || a.childNavigation.open {
					t.Fatal("did not select exact filtered merge child")
				}
				return
			case "cancel":
				tt.Key(0, ui.KeyEscape)
				tt.Frame()
				if a.childNavigation.open || a.selectedSHA != "source" {
					t.Fatal("cancel navigated")
				}
				return
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "reload":
				a.historyReload++
			case "selection":
				a.selectedSHA = "other"
			case "reopen":
				a.childNavigation = childNavigationState{}
				a.navigateToChild()
			case "edge":
				a.commits[0].Parents = []string{"other"}
			}
			before := a.selectedSHA
			a.chooseChildNavigation("child-one", generation)
			if a.selectedSHA != before {
				t.Fatal("stale choice changed selection")
			}
			if change == "reopen" && !a.childNavigation.open {
				t.Fatal("old generation closed new picker")
			}
		})
	}
}

func TestChildNavigationGuardsAndInput(t *testing.T) {
	for _, reason := range []string{"workspace", "page", "busy", "loading", "reload", "chooser", "ref", "dialog", "hidden", "file", "missing-prefix"} {
		t.Run(reason, func(t *testing.T) {
			a := childNavigationApp(t)
			switch reason {
			case "workspace":
				a.worktreeSelected = true
			case "page":
				a.page = "stash"
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
			case "dialog":
				a.prompt.open = true
				a.prompt.root = a.repo.Root
			case "hidden":
				a.historyQuery = "no match"
			case "file":
				a.fileFilter = "file.go"
				a.fileCommits = []gitrepo.Commit{a.commits[3]}
			case "missing-prefix":
				a.commits = nil
			}
			a.navigateToChild()
			if a.selectedSHA != "source" || a.childNavigation.open {
				t.Fatal("protected context navigated")
			}
		})
	}
	a := childNavigationApp(t)
	tt := childNavigationTester(a)
	body, _ := tt.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
	tt.ClickAt(body.Bounds.X+10, body.Bounds.Y+10)
	tt.Key(ui.Alt, ui.KeyUp)
	tt.Frame()
	if a.childNavigation.open || a.selectedSHA != "source" {
		t.Fatal("Alt+Up escaped an actual editor")
	}
}

func TestChildNavigationCancelRestoresInput(t *testing.T) {
	for _, backdrop := range []bool{false, true} {
		t.Run(fmt.Sprint(backdrop), func(t *testing.T) {
			a := childNavigationApp(t)
			tt := childNavigationTester(a)
			body, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
			if !ok {
				t.Fatal("missing description editor")
			}
			tt.ClickAt(body.Bounds.X+10, body.Bounds.Y+10)
			// Open directly to exercise restoration to an existing input opener;
			// the shortcut intentionally belongs to the editor while it has focus.
			a.navigateToChild()
			tt.Frame()
			tt.Frame()
			if backdrop {
				tt.ClickAt(2, 2)
			} else {
				tt.Key(0, ui.KeyEscape)
			}
			tt.Frame()
			tt.Frame()
			tt.Key(0, ui.KeyEnd)
			tt.Type("X")
			tt.Frame()
			if a.childNavigation.open || a.selectedSHA != "source" || a.commitText != "retained draftX" {
				t.Fatalf("cancel lost input opener or state: draft=%q", a.commitText)
			}
		})
	}
}

func TestChildNavigationRenderedPicker(t *testing.T) {
	for _, size := range [][2]int{{360, 360}, {900, 640}} {
		for _, language := range []locale{localeEnglish, localeChinese} {
			for _, mode := range []string{"light", "dark"} {
				name := fmt.Sprintf("child-%dx%d-%s-%s", size[0], size[1], language, mode)
				t.Run(name, func(t *testing.T) {
					a := childNavigationApp(t)
					a.locale, a.themeMode = language, mode
					a.navigateToChild()
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Fill().Background(theme.TokensOf(c).BgLayout).Children(func() { a.childNavigationDialog(c) })
					}, size[0], size[1])
					if _, ok := tt.AccessByRole(ui.RoleOption, "child-two merge child"); !ok {
						t.Fatal("missing merge child option")
					}
					for _, node := range tt.AccessAll() {
						if node.Role != ui.RoleDialog && node.Role != ui.RoleTextField && node.Role != ui.RoleOption {
							continue
						}
						if node.Bounds.X < 16 || node.Bounds.Y < 16 || node.Bounds.X+node.Bounds.W > float32(size[0]-16) || node.Bounds.Y+node.Bounds.H > float32(size[1]-16) {
							t.Errorf("picker outside gutters: %+v", node)
						}
					}
					if dir := os.Getenv("SUPERGIT_CHILD_CAPTURE_DIR"); dir != "" {
						f, err := os.Create(filepath.Join(dir, name+".png"))
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
					tt.Key(0, ui.KeyDown)
					tt.Frame()
					tt.Key(0, ui.KeyEnter)
					tt.Frame()
					if a.selectedSHA != "child-two" || a.childNavigation.open {
						t.Fatal("arrow/Enter did not choose second child")
					}
				})
			}
		}
	}
}

func TestChildNavigationScopedFeedback(t *testing.T) {
	for _, fileHistory := range []bool{false, true} {
		t.Run(fmt.Sprint(fileHistory), func(t *testing.T) {
			a := childNavigationApp(t)
			a.selectedSHA = "child-one"
			message := "childNavigationNone"
			if fileHistory {
				a.fileFilter = "file.go"
				a.fileCommits = []gitrepo.Commit{a.commits[0]}
				message = "childNavigationIncomplete"
			}
			a.navigateToChild()
			if a.selectedSHA != "child-one" || a.childNavigation.open || a.commitText != "retained draft" {
				t.Fatal("feedback changed selection or draft")
			}
			if len(a.notices[toastMain]) == 0 {
				t.Fatal("missing scoped feedback")
			}
			if got := a.notices[toastMain][0].description; got != a.t(message) {
				t.Fatalf("wrong scope/recovery feedback: %s", got)
			}
		})
	}
}

func TestChildNavigationPinnedMultiTipPages(t *testing.T) {
	a, _, _ := quickCheckoutFixture(t)
	source := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD")
	tree := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD^{tree}")
	first := pullFixtureGit(t, a.repo.Root, "commit-tree", tree, "-p", source, "-m", "first child")
	second := pullFixtureGit(t, a.repo.Root, "commit-tree", tree, "-p", source, "-m", "second child")
	pullFixtureGit(t, a.repo.Root, "update-ref", "refs/heads/first-child", first)
	pullFixtureGit(t, a.repo.Root, "update-ref", "refs/heads/second-child", second)
	page, err := a.repo.HistoryPage(a.ctx, nil, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	tips := page.Tips
	// A new ref after traversal starts must not expand the pinned range.
	late := pullFixtureGit(t, a.repo.Root, "commit-tree", tree, "-p", source, "-m", "late child")
	pullFixtureGit(t, a.repo.Root, "update-ref", "refs/heads/late-child", late)
	a.commits = nil
	found := false
	for !found {
		a.commits = append(a.commits, page.Commits...)
		for _, commit := range page.Commits {
			if commit.SHA == source {
				found = true
			}
		}
		if found {
			break
		}
		if !page.HasMore {
			t.Fatal("source absent from pinned traversal")
		}
		page, err = a.repo.HistoryPage(a.ctx, tips, len(a.commits), 1)
		if err != nil {
			t.Fatal(err)
		}
	}
	if !page.HasMore {
		t.Fatal("fixture did not leave an unread older page")
	}
	a.page, a.selectedSHA, a.worktreeSelected = "history", source, false
	a.historyMore, a.historyPaused = true, true
	children := a.childNavigationCandidates(source)
	if len(children) != 2 || children[0].SHA == children[1].SHA {
		t.Fatalf("incomplete children across pages: %+v", children)
	}
	for _, child := range children {
		if child.SHA != first && child.SHA != second {
			t.Fatalf("off-range child offered: %s", child.SHA)
		}
	}
	a.navigateToChild()
	if !a.childNavigation.open {
		t.Fatal("paused multi-tip prefix did not offer explicit choice")
	}
}

func TestChildNavigationOwnedGitIsReadOnly(t *testing.T) {
	a, callbacks, _ := quickCheckoutFixture(t)
	page, err := a.repo.ReadHistory(a.ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	a.commits = page.Commits
	a.page = "history"
	a.worktreeSelected = false
	a.selectedSHA = a.commits[len(a.commits)-1].SHA
	snapshot := func() string {
		b, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
		if err != nil {
			t.Fatal(err)
		}
		return pullFixtureGit(t, a.repo.Root, "show-ref") + pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") + pullFixtureGit(t, a.repo.Root, "ls-files", "--stage") + string(b)
	}
	before := snapshot()
	source := a.selectedSHA
	a.navigateToChild()
	if a.childNavigation.open {
		a.chooseChildNavigation(a.childNavigationCandidates(source)[0].SHA, a.childNavigation.generation)
	}
	for a.commitFilesLoading || a.commitDiffLoading {
		historyCompletion(t, callbacks)()
	}
	if a.selectedSHA == source || snapshot() != before || a.commitText != "draft\n\nbody" || !a.amend {
		t.Fatal("navigation failed or mutated Git/draft")
	}
}
