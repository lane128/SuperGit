package lobedesk

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func repositorySwitchTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() { a.commitMessageFields(c); a.commandPickerDialogs(c) })
	}, 600, 500)
}

func TestQuickRepositorySwitchCandidatesAndEmpty(t *testing.T) {
	a := headShortcutApp(t)
	a.recentRepositories = []string{a.repo.Root, "/fixtures/甲/same", "/fixtures/B/same"}
	a.openQuickRepositorySwitch()
	if got := a.quickRepositorySwitchCandidates(); !reflect.DeepEqual(got, []string{"/fixtures/甲/same", "/fixtures/B/same"}) {
		t.Fatalf("wrong session candidates: %v", got)
	}
	a.quickRepositorySwitch.query = "/B/"
	if got := a.quickRepositorySwitchCandidates(); !reflect.DeepEqual(got, []string{"/fixtures/B/same"}) {
		t.Fatalf("wrong full-path search: %v", got)
	}
	a.quickRepositorySwitch.query = "missing"
	tt := repositorySwitchTester(a)
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.loading || !a.quickRepositorySwitch.open || !tt.HasText(a.t("quickRepositorySwitchNoMatch")) {
		t.Fatal("empty result executed or lacked feedback")
	}
	a.quickRepositorySwitch.query = ""
	a.recentRepositories = []string{a.repo.Root}
	tt.Frame()
	if !tt.HasText(a.t("quickRepositorySwitchEmpty")) {
		t.Fatal("missing session-empty state")
	}
}

func TestQuickRepositorySwitchActualInputCancel(t *testing.T) {
	for _, backdrop := range []bool{false, true} {
		t.Run(itoa(intBool(backdrop)), func(t *testing.T) {
			a := headShortcutApp(t)
			a.commitText = "title\n\nbody"
			a.recentRepositories = []string{a.repo.Root, "/fixtures/other"}
			tt := repositorySwitchTester(a)
			body, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitDescription"))
			if !ok {
				t.Fatal("missing actual Body")
			}
			tt.ClickAt(body.Bounds.X+10, body.Bounds.Y+10)
			tt.Key(ui.Cmd|ui.Shift, ui.KeyO)
			tt.Frame()
			if !a.quickRepositorySwitch.open {
				t.Fatal("shortcut did not open from editor")
			}
			tt.Type("other")
			tt.Frame()
			if a.quickRepositorySwitch.query != "other" || a.commitText != "title\n\nbody" {
				t.Fatal("search did not own input")
			}
			before := a.page
			tt.Key(ui.Cmd, ui.Key4)
			tt.Frame()
			for _, s := range shortcutTable() {
				if s.ID == "page.worktrees" {
					a.dispatchWindowShortcut(s.Run)
				}
			}
			if a.page != before {
				t.Fatal("background callback leaked through switcher")
			}
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
			if a.quickRepositorySwitch.open || a.commitText != "title\n\nbodyX" || a.loading {
				t.Fatal("cancel lost original input/draft")
			}
		})
	}
}

func intBool(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestQuickRepositorySwitchOpenerSelection(t *testing.T) {
	for _, opener := range []string{"summary", "body", "history"} {
		t.Run(opener, func(t *testing.T) {
			a := browsingApp(t)
			a.commitText = "title\n\nbody"
			a.recentRepositories = []string{a.repo.Root, "/fixtures/other"}
			var tt *ui.Tester
			if opener == "history" {
				tt = pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.historyPage(c); a.quickRepositorySwitchDialog(c) })
				tt.Key(ui.Cmd, ui.KeyF)
				tt.Type("parent")
				tt.Frame()
			} else {
				tt = repositorySwitchTester(a)
				label := a.t("commitSummary")
				if opener == "body" {
					label = a.t("commitDescription")
				}
				field, ok := tt.AccessByRole(ui.RoleTextField, label)
				if !ok {
					t.Fatal("missing opener", label)
				}
				tt.ClickAt(field.Bounds.X+10, field.Bounds.Y+10)
			}
			tt.Key(ui.Cmd, ui.KeyA)
			tt.Frame()
			tt.Key(ui.Cmd|ui.Shift, ui.KeyO)
			tt.Frame()
			tt.Type("other")
			tt.Frame()
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			tt.Frame()
			tt.Type("X")
			tt.Frame()
			switch opener {
			case "summary":
				if a.commitText != "X\n\nbody" {
					t.Fatalf("summary selection lost: %q", a.commitText)
				}
			case "body":
				if a.commitText != "title\n\nX" {
					t.Fatalf("Body selection lost: %q", a.commitText)
				}
			case "history":
				if a.historyQuery != "X" || a.commitText != "title\n\nbody" {
					t.Fatalf("history selection lost: query=%q draft=%q", a.historyQuery, a.commitText)
				}
			}
		})
	}
}

func TestQuickRepositorySwitchStaleAndGuards(t *testing.T) {
	for _, change := range []string{"root", "epoch", "reopen", "candidate", "busy", "loading", "reload", "chooser", "ref", "prompt"} {
		t.Run(change, func(t *testing.T) {
			a := headShortcutApp(t)
			target := "/fixtures/other"
			a.recentRepositories = []string{a.repo.Root, target}
			a.openQuickRepositorySwitch()
			generation := a.quickRepositorySwitch.generation
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "reopen":
				a.quickRepositorySwitch = quickRepositorySwitchState{}
				a.openQuickRepositorySwitch()
			case "candidate":
				a.recentRepositories = []string{a.repo.Root}
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
			}
			before := a.loadSeq
			a.chooseQuickRepositorySwitch(target, generation)
			if a.loadSeq != before || a.commitText != "retained draft" {
				t.Fatal("stale/guarded selection opened a repository")
			}
			if change == "reopen" && !a.quickRepositorySwitch.open {
				t.Fatal("old callback closed reopened picker")
			}
		})
	}
}

func TestQuickRepositorySwitchRendered(t *testing.T) {
	for _, size := range [][2]int{{360, 360}, {900, 640}} {
		for _, language := range []locale{localeEnglish, localeChinese} {
			for _, mode := range []string{"light", "dark"} {
				for _, state := range []string{"paths", "empty", "no-match"} {
					name := fmt.Sprintf("repo-%dx%d-%s-%s-%s", size[0], size[1], language, mode, state)
					t.Run(name, func(t *testing.T) {
						a := headShortcutApp(t)
						a.locale, a.themeMode = language, mode
						first := "/fixtures/团队项目/long parent directory/another long directory/甲/same repository"
						second := "/fixtures/团队项目/long parent directory/another long directory/乙/same repository"
						a.recentRepositories = []string{a.repo.Root, first, second}
						a.openQuickRepositorySwitch()
						if state == "empty" {
							a.recentRepositories = []string{a.repo.Root}
						}
						if state == "no-match" {
							a.quickRepositorySwitch.query = "missing"
						}
						tt := ui.NewTester(func(c *ui.Context) {
							a.applyTheme(c)
							ui.Column(c).Fill().Background(theme.TokensOf(c).BgLayout).Children(func() { a.quickRepositorySwitchDialog(c) })
						}, size[0], size[1])
						for _, node := range tt.AccessAll() {
							if node.Role != ui.RoleDialog && node.Role != ui.RoleTextField && node.Role != ui.RoleOption {
								continue
							}
							if node.Bounds.X < 16 || node.Bounds.Y < 16 || node.Bounds.X+node.Bounds.W > float32(size[0]-16) || node.Bounds.Y+node.Bounds.H > float32(size[1]-16) {
								t.Errorf("outside gutters: %+v", node)
							}
						}
						if state == "paths" {
							for _, root := range []string{first, second} {
								if _, ok := tt.AccessByRole(ui.RoleOption, root); !ok {
									t.Fatal("missing full-path option", root)
								}
							}
							tt.Key(0, ui.KeyDown)
							tt.Frame()
							if a.quickRepositorySwitch.cursor != 1 || a.loading {
								t.Fatal("arrow did not move cursor without switching")
							}
						}
						if dir := os.Getenv("SUPERGIT_REPOSITORY_SWITCH_CAPTURE_DIR"); dir != "" {
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
}

func TestQuickRepositorySwitchShortcutAliasesAndReopen(t *testing.T) {
	for _, binding := range quickRepositorySwitchShortcuts() {
		t.Run(binding.ID, func(t *testing.T) {
			a := headShortcutApp(t)
			a.recentRepositories = []string{a.repo.Root, "/fixtures/other"}
			tt := repositorySwitchTester(a)
			tt.Key(binding.Mod, binding.Key)
			tt.Frame()
			if !a.quickRepositorySwitch.open {
				t.Fatal("binding did not open picker")
			}
			tt.Type("other")
			tt.Frame()
			tt.Key(0, ui.KeyTab)
			tt.Frame()
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			if a.quickRepositorySwitch.open || a.loading {
				t.Fatal("Escape after Tab did not cancel")
			}
			tt.Key(binding.Mod, binding.Key)
			tt.Frame()
			if !a.quickRepositorySwitch.open || a.quickRepositorySwitch.query != "" {
				t.Fatal("reopen did not reset query")
			}
		})
	}
}

func TestQuickRepositorySwitchOwnedFailureBoundaries(t *testing.T) {
	for _, failure := range []string{"removed-git", "status", "stale-failure"} {
		t.Run(failure, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			_, _, target := worktreeApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			done := make(chan func(), 64)
			a.runOnMain = func(fn func()) { done <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			a.openRepository(root, true)
			for a.loading {
				runContinuation(t, done)
			}
			a.rememberRepository(target)
			a.rememberRepository(root)
			a.commitText = "retained draft"
			a.amend = true
			recent := append([]string(nil), a.recentRepositories...)
			if failure == "status" {
				if err := os.WriteFile(filepath.Join(target, ".git", "index"), []byte("invalid index"), 0o644); err != nil {
					t.Fatal(err)
				}
				if _, err := gitrepo.Open(ctx, target); err != nil {
					t.Fatalf("expected Open to succeed before Status failure: %v", err)
				}
			} else {
				if err := os.Rename(filepath.Join(target, ".git"), filepath.Join(target, "fixture-git-backup")); err != nil {
					t.Fatal(err)
				}
			}
			a.openQuickRepositorySwitch()
			a.chooseQuickRepositorySwitch(target, a.quickRepositorySwitch.generation)
			if !a.loading {
				t.Fatal("switch did not start")
			}
			if failure == "stale-failure" {
				old := historyCompletion(t, done)
				a.openRepository(root, false)
				old()
				if !a.loading || a.loadErr != "" || len(a.notices[toastMain]) != 0 {
					t.Fatal("stale failure polluted current loading/error/toast")
				}
			}
			for a.loading {
				runContinuation(t, done)
			}
			if a.repo.Root != root || a.commitText != "retained draft" || !a.amend || !reflect.DeepEqual(a.recentRepositories, recent) {
				t.Fatal("failure changed current repository/draft/MRU")
			}
			if failure != "stale-failure" && !strings.Contains(a.loadErr, target) {
				t.Fatal("failure omitted target", a.loadErr)
			}
			if failure == "stale-failure" && a.loadErr != "" {
				t.Fatal("stale failure survived newer success", a.loadErr)
			}
		})
	}
}

func TestQuickRepositorySwitchOwnedSuccessAndFailure(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(itoa(intBool(failure)), func(t *testing.T) {
			a, _, root := worktreeApp(t)
			_, _, target := worktreeApp(t)
			for _, fixture := range []string{root, target} {
				if err := os.WriteFile(filepath.Join(fixture, "base.txt"), []byte("staged\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				pullFixtureGit(t, fixture, "add", "--", "base.txt")
				if err := os.WriteFile(filepath.Join(fixture, "base.txt"), []byte("unstaged\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(fixture, "untracked.txt"), []byte("untracked\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if failure {
				target = filepath.Join(t.TempDir(), "missing target")
			}
			ctx, cancel := context.WithCancel(context.Background())
			a.ctx = ctx
			done := make(chan func(), 64)
			a.runOnMain = func(fn func()) { done <- fn }
			t.Cleanup(func() { cancel(); a.resetHistoryTraversal(); a.resetFileHistory() })
			a.openRepository(root, true)
			for a.loading {
				runContinuation(t, done)
			}
			a.rememberRepository(target)
			a.rememberRepository(root)
			recent := append([]string(nil), a.recentRepositories...)
			a.commitText = "draft\n\nbody"
			a.amend = true
			fingerprint := func(path string) string {
				entries, err := os.ReadDir(path)
				if err != nil {
					t.Fatal(err)
				}
				value := pullFixtureGit(t, path, "rev-parse", "HEAD") + pullFixtureGit(t, path, "symbolic-ref", "HEAD") + pullFixtureGit(t, path, "for-each-ref") + pullFixtureGit(t, path, "ls-files", "--stage") + pullFixtureGit(t, path, "status", "--porcelain=v1")
				for _, entry := range entries {
					if !entry.IsDir() {
						b, err := os.ReadFile(filepath.Join(path, entry.Name()))
						if err != nil {
							t.Fatal(err)
						}
						value += entry.Name() + string(b)
					}
				}
				return value
			}
			before := fingerprint(root)
			otherBefore := ""
			if !failure {
				otherBefore = fingerprint(target)
			}
			tt := repositorySwitchTester(a)
			tt.Key(ui.Cmd|ui.Shift, ui.KeyO)
			tt.Frame()
			tt.Type(filepath.Base(target))
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if !a.loading || a.quickRepositorySwitch.open {
				t.Fatal("confirmation was blocked by its own modal")
			}
			for a.loading {
				runContinuation(t, done)
			}
			if failure {
				if a.repo.Root != root || a.commitText != "draft\n\nbody" || !a.amend || !reflect.DeepEqual(recent, a.recentRepositories) {
					t.Fatal("failed switch changed root/draft/MRU")
				}
				if !strings.Contains(a.loadErr, target) {
					t.Fatal("failure omitted attempted path")
				}
			} else {
				if a.repo.Root != target || a.commitText != "" || a.amend || a.recentRepositories[0] != target {
					t.Fatal("successful switch did not reset/promote exact target")
				}
				a.openQuickRepositorySwitch()
				a.chooseQuickRepositorySwitch(root, a.quickRepositorySwitch.generation)
				for a.loading {
					runContinuation(t, done)
				}
				if a.repo.Root != root || a.commitText != "" {
					t.Fatal("switch back unexpectedly restored old draft")
				}
			}
			if fingerprint(root) != before || !failure && fingerprint(target) != otherBefore {
				t.Fatal("switch changed Git or working bytes")
			}
		})
	}
}
