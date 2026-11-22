package lobedesk

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func quickCheckoutTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() { a.commitMessageField(c); a.quickCheckoutDialog(c) })
	}, 600, 500)
}

func TestQuickCheckoutNoAvailableNotClipped(t *testing.T) {
	branches := []gitrepo.Branch{
		{Name: "codex/sublime-merge", SHA: "a"},
		{Name: "codex/unified-branch-command", SHA: "b"},
		{Name: "main", SHA: "m", Current: true},
	}
	for i := 0; i < 20; i++ {
		branches = append(branches, gitrepo.Branch{Name: fmt.Sprintf("worktree-agent-%02d", i), SHA: fmt.Sprintf("s%d", i)})
	}
	a := &app{
		repo:      &gitrepo.Repository{Root: t.TempDir()},
		locale:    localeEnglish,
		themeMode: "light",
		status:    gitrepo.Status{Branch: "main", Entries: []gitrepo.Entry{{Path: "x.go", Conflict: true}}},
		branches:  branches,
	}
	a.openQuickCheckout()
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Background(theme.TokensOf(c).BgLayout)
		a.quickCheckoutDialog(c)
	}, 900, 640)
	tt.Type("code")
	tt.Frame()
	if dir := os.Getenv("SUPERGIT_QUICK_CHECKOUT_CAPTURE_DIR"); dir != "" {
		f, err := os.Create(filepath.Join(dir, "no-available.png"))
		if err != nil {
			t.Fatal(err)
		}
		_ = png.Encode(f, tt.Image())
		_ = f.Close()
	}
	dialog, ok := tt.AccessByRole(ui.RoleDialog, a.t("quickCheckout"))
	if !ok {
		t.Fatal("dialog missing")
	}
	msg, ok := tt.AccessByRole(ui.RoleText, a.t("quickCheckoutNoAvailable"))
	if !ok {
		t.Fatal("no-available message missing")
	}
	t.Logf("dialog=%+v msg=%+v", dialog.Bounds, msg.Bounds)
	if msg.Bounds.Y+msg.Bounds.H > dialog.Bounds.Y+dialog.Bounds.H {
		t.Fatalf("no-available text clipped by dialog: msg=%+v dialog=%+v", msg.Bounds, dialog.Bounds)
	}
}

func TestQuickCheckoutRenderedBounds(t *testing.T) {
	for _, size := range [][2]int{{360, 360}, {900, 640}} {
		for _, language := range []locale{localeEnglish, localeChinese} {
			for _, mode := range []string{"light", "dark"} {
				name := fmt.Sprintf("%dx%d-%s-%s", size[0], size[1], language, mode)
				t.Run(name, func(t *testing.T) {
					a := &app{repo: &gitrepo.Repository{Root: t.TempDir()}, locale: language, themeMode: mode, status: gitrepo.Status{Branch: "main"},
						branches:  []gitrepo.Branch{{Name: "main", SHA: "a", Current: true}, {Name: "feature/long-branch-name-kept-as-the-accessible-label", SHA: "b"}, {Name: "occupied", SHA: "c"}},
						worktrees: []gitrepo.Worktree{{Branch: "occupied", Path: "/other-owned-worktree"}}}
					a.openQuickCheckout()
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Fill().Background(theme.TokensOf(c).BgLayout)
						a.quickCheckoutDialog(c)
					}, size[0], size[1])
					for _, node := range tt.AccessAll() {
						if node.Role != ui.RoleTextField && node.Role != ui.RoleOption && node.Role != ui.RoleDialog {
							continue
						}
						vertical := node.Role != ui.RoleOption && (node.Bounds.Y < 16 || node.Bounds.Y+node.Bounds.H > float32(size[1]-16))
						if node.Bounds.X < 16 || node.Bounds.X+node.Bounds.W > float32(size[0]-16) || vertical {
							t.Errorf("control outside window gutters: %+v", node)
						}
					}
					if dir := os.Getenv("SUPERGIT_QUICK_CHECKOUT_CAPTURE_DIR"); dir != "" {
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
				})
			}
		}
	}
}

func TestQuickCheckoutCancelActualSummaryAndHistorySearch(t *testing.T) {
	for _, source := range []string{"summary", "search"} {
		for _, cancel := range []string{"escape", "backdrop"} {
			t.Run(source+"/"+cancel, func(t *testing.T) {
				a, _ := overviewFixture(t)
				a.locale, a.page = localeEnglish, "changes"
				a.commitText = "draft\n\nbody"
				if source == "search" {
					a.page = "history"
					a.historyQuery = "needle"
				}
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					a.registerShortcuts(c)
					ui.Column(c).Fill().Children(func() {
						if source == "search" {
							a.historyPage(c)
						} else {
							a.commitMessageField(c)
						}
						a.quickCheckoutDialog(c)
					})
				}, 900, 640)
				if source == "search" {
					tt.Key(ui.Cmd, ui.KeyF)
				} else {
					node, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
					if !ok {
						t.Fatal("actual Summary missing")
					}
					tt.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
					tt.Key(ui.Cmd, ui.KeyA)
					tt.Key(0, ui.KeyLeft)
				}
				tt.Key(ui.Cmd, ui.KeyB)
				tt.Frame()
				tt.Type("no-matching-branch")
				tt.Frame()
				if !tt.HasText(a.t("searchNoBranches")) {
					t.Fatal("empty results lack feedback")
				}
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				if !a.quickCheckout.open || a.busy {
					t.Fatal("empty Enter executed or closed")
				}
				if cancel == "escape" {
					tt.Key(0, ui.KeyEscape)
				} else {
					tt.ClickAt(2, 2)
				}
				tt.Frame()
				tt.Frame()
				tt.Key(0, ui.KeyEnd)
				tt.Type("X")
				if a.quickCheckout.open {
					t.Fatal("cancel did not close")
				}
				if source == "search" {
					if a.historyQuery != "needleX" || a.commitText != "draft\n\nbody" {
						t.Fatal("search/draft changed")
					}
				} else if a.commitText != "draftX\n\nbody" {
					t.Fatal("Summary focus lost")
				}
			})
		}
	}
}

func TestQuickCheckoutOpensFromBodyAndCancelsOnce(t *testing.T) {
	a, _ := overviewFixture(t)
	a.locale, a.page = localeEnglish, "changes"
	a.commitText, a.amend = "draft\n\nbody", true
	a.status.Entries = []gitrepo.Entry{{Path: "same.go", Conflict: true}}
	a.branches = []gitrepo.Branch{{Name: "main", SHA: "main-tip", Current: true}, {Name: "topic", SHA: "different-tip"}}
	tt := quickCheckoutTester(a)
	body, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
	if !ok {
		t.Fatal("actual Body missing")
	}
	tt.ClickAt(body.Bounds.X+body.Bounds.W/2, body.Bounds.Y+body.Bounds.H/2)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyRight)
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Frame()
	if !a.quickCheckout.open || !tt.HasText("topic") || !tt.HasText(a.t("quickCheckoutConflict")) {
		t.Fatal("conflicted repository could not inspect all local branches/reasons")
	}
	tt.SetFocused(true)
	tt.Key(0, ui.KeyTab)
	tt.Advance(2 * time.Second)
	tt.Key(ui.Cmd, ui.Key4)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.busy || a.page != "changes" || !a.quickCheckout.open {
		t.Fatal("disabled picker leaked background keys or executed")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Frame()
	if a.quickCheckout.open || !tt.Focused(a.t("commitMessage")) {
		t.Fatal("one Escape did not restore Body")
	}
	tt.Key(0, ui.KeyEnd)
	tt.Type("X")
	if a.commitText != "draft\n\nbodyX" || !a.amend {
		t.Fatal("draft/amend focus lost")
	}
}

func TestQuickCheckoutDirtyTreeAllowsMouseClick(t *testing.T) {
	t.Run("non-conflicting carries local changes", func(t *testing.T) {
		a, callbacks, target := quickCheckoutFixture(t)
		notesPath := filepath.Join(a.repo.Root, "notes.txt")
		if err := os.WriteFile(notesPath, []byte("keep local note\n"), 0600); err != nil {
			t.Fatal(err)
		}
		var err error
		a.status, err = a.repo.Status(a.ctx)
		if err != nil {
			t.Fatal(err)
		}
		tt := quickCheckoutTester(a)
		tt.Key(ui.Cmd, ui.KeyB)
		tt.Frame()
		row, ok := tt.AccessByRole(ui.RoleOption, "topic · "+a.t("localBranches"))
		if !ok {
			t.Fatal("topic option missing")
		}
		if row.Disabled {
			t.Fatal("dirty non-conflicting branch should be clickable")
		}
		tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
		tt.Frame()
		if a.quickCheckout.open {
			t.Fatal("clicking topic did not close quick checkout")
		}
		waitQuickCheckout(t, a, callbacks)
		if pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") != target || a.status.Branch != "topic" {
			t.Fatalf("dirty mouse checkout did not switch to topic: branch=%q", a.status.Branch)
		}
		data, err := os.ReadFile(notesPath)
		if err != nil || string(data) != "keep local note\n" {
			t.Fatalf("dirty mouse checkout lost local file: %q, %v", data, err)
		}
	})

	t.Run("conflicting change rejected with localized guidance and git detail", func(t *testing.T) {
		a, callbacks, _ := quickCheckoutFixture(t)
		samePath := filepath.Join(a.repo.Root, "same.go")
		if err := os.WriteFile(samePath, []byte("conflicting dirty edit\n"), 0600); err != nil {
			t.Fatal(err)
		}
		var err error
		a.status, err = a.repo.Status(a.ctx)
		if err != nil {
			t.Fatal(err)
		}
		beforeHead := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD")
		tt := quickCheckoutTester(a)
		tt.Key(ui.Cmd, ui.KeyB)
		tt.Frame()
		row, ok := tt.AccessByRole(ui.RoleOption, "topic · "+a.t("localBranches"))
		if !ok || row.Disabled {
			t.Fatal("topic option should be clickable so git switch decides")
		}
		tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
		tt.Frame()
		waitQuickCheckout(t, a, callbacks)
		if pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") != beforeHead {
			t.Fatal("conflicting checkout moved HEAD")
		}
		data, err := os.ReadFile(samePath)
		if err != nil || string(data) != "conflicting dirty edit\n" {
			t.Fatalf("conflicting checkout lost bytes: %q, %v", data, err)
		}
		if len(a.notices[toastMain]) == 0 {
			t.Fatal("conflicting checkout missing error notice")
		}
		desc := a.notices[toastMain][0].description
		if !strings.Contains(desc, a.t("dirtyWorkingTreeGuidance")) || !strings.Contains(desc, "same.go") {
			t.Fatalf("expected localized guidance and conflicting file path, got %q", desc)
		}
	})
}

func TestQuickCheckoutAvailabilityAndStaleChoices(t *testing.T) {
	for _, change := range []string{"current", "occupied", "operation", "busy", "loading", "history", "chooser", "ref", "dialog", "root", "epoch", "generation", "tip", "closed", "conflict", "status error", "branches error", "worktrees error"} {
		t.Run(change, func(t *testing.T) {
			a, _ := overviewFixture(t)
			a.status.Entries = nil
			a.branches = []gitrepo.Branch{{Name: "main", SHA: "main-tip", Current: true}, {Name: "topic", SHA: "topic-tip"}}
			a.openQuickCheckout()
			generation := a.quickCheckout.generation
			name, sha := "topic", "topic-tip"
			switch change {
			case "closed":
				a.quickCheckout.open = false
			case "conflict":
				a.status.Entries = []gitrepo.Entry{{Path: "owned", Conflict: true}}
			case "status error":
				a.loadErr = "owned read failure"
			case "branches error":
				a.branchesErr = "owned read failure"
			case "worktrees error":
				a.worktreesErr = "owned read failure"
			case "current":
				name, sha = "main", "main-tip"
			case "occupied":
				a.worktrees = []gitrepo.Worktree{{Path: "/other-owned-worktree", Branch: "topic"}}
			case "operation":
				a.operation.Kind = "merge"
			case "busy":
				a.busy = true
			case "loading":
				a.loading = true
			case "history":
				a.historyReloading = true
			case "chooser":
				a.repositoryChooserOpen = true
			case "ref":
				a.historyRefOpen = true
			case "dialog":
				a.cloneForm.open = true
			case "root":
				a.repo = nil
			case "epoch":
				a.loadSeq++
			case "generation":
				a.quickCheckout.open = false
				a.openQuickCheckout()
			case "tip":
				a.branches[1].SHA = "moved-tip"
			}
			started := a.progressSince
			a.runQuickCheckout(name, sha, generation)
			if a.progressSince != started {
				t.Fatal("invalid/stale choice started checkout")
			}
		})
	}
}

func quickCheckoutFixture(t *testing.T) (*app, chan func(), string) {
	t.Helper()
	a, callbacks := overviewFixture(t)
	pullFixtureGit(t, a.repo.Root, "add", "-A")
	pullFixtureGit(t, a.repo.Root, "commit", "--no-gpg-sign", "-m", "Clean main fixture")
	pullFixtureGit(t, a.repo.Root, "branch", "topic", "HEAD~1")
	target := pullFixtureGit(t, a.repo.Root, "rev-parse", "topic")
	var err error
	a.status, err = a.repo.Status(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.branches, err = a.repo.Branches(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.worktrees, err = a.repo.Worktrees(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.locale, a.page = localeEnglish, "changes"
	a.commitText, a.amend = "draft\n\nbody", true
	return a, callbacks, target
}

func waitQuickCheckout(t *testing.T, a *app, callbacks chan func()) {
	t.Helper()
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for a.busy || a.loading {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("checkout/reload did not finish")
		}
	}
}

func TestQuickCheckoutExactLocalBranchAndBackendRejection(t *testing.T) {
	for _, scenario := range []string{"success", "external dirty", "external tip", "worktree occupied"} {
		t.Run(scenario, func(t *testing.T) {
			a, callbacks, target := quickCheckoutFixture(t)
			tt := quickCheckoutTester(a)
			tt.Key(ui.Cmd, ui.KeyB)
			tt.Frame()
			tt.Type("topic")
			tt.Frame()
			if a.quickCheckout.cursor != 0 {
				t.Fatal("nonselected-commit branch missing from complete local catalog")
			}
			beforeHead := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD")
			beforeIndex := pullFixtureGit(t, a.repo.Root, "ls-files", "--stage", "-z")
			beforeBytes, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "external dirty":
				beforeBytes = []byte("external dirty bytes\n")
				if err := os.WriteFile(filepath.Join(a.repo.Root, "same.go"), beforeBytes, 0600); err != nil {
					t.Fatal(err)
				}
			case "external tip":
				pullFixtureGit(t, a.repo.Root, "branch", "-f", "topic", "main")
			case "worktree occupied":
				pullFixtureGit(t, a.repo.Root, "worktree", "add", filepath.Join(t.TempDir(), "occupied"), "topic")
			}
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if a.quickCheckout.open {
				t.Fatal("valid displayed choice did not close before backend execution")
			}
			waitQuickCheckout(t, a, callbacks)
			if scenario == "success" {
				if pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") != target || a.status.Branch != "topic" {
					t.Fatal("did not check out exact displayed branch/commit")
				}
			} else {
				bytes, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
				if err != nil || string(bytes) != string(beforeBytes) || pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") != beforeHead || pullFixtureGit(t, a.repo.Root, "ls-files", "--stage", "-z") != beforeIndex {
					t.Fatal("rejected checkout changed local state")
				}
				if len(a.notices[toastMain]) == 0 {
					t.Fatal("backend rejection missing feedback")
				}
			}
			if a.commitText != "draft\n\nbody" || !a.amend {
				t.Fatal("checkout changed draft/amend")
			}
		})
	}
}
