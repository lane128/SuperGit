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

func TestPaletteRepositorySwitchHandoff(t *testing.T) {
	for _, opener := range []string{"message", "history"} {
		for _, mod := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
			for _, activate := range []string{"enter", "click"} {
				for _, cancel := range []string{"escape", "backdrop"} {
					t.Run(opener+"/"+shortcutModName(mod)+"/"+activate+"/"+cancel, func(t *testing.T) {
						a := browsingApp(t)
						a.commitText = "title\n\nbody"
						a.amend = true
						a.recentRepositories = []string{a.repo.Root, "/fixtures/other"}
						var tt *ui.Tester
						if opener == "history" {
							tt = pageTester(a, func(c *ui.Context) { a.registerShortcuts(c); a.historyPage(c); a.commandPickerDialogs(c) })
							tt.Key(ui.Cmd, ui.KeyF)
							tt.Type("parent")
							tt.Frame()
						} else {
							tt = paletteCheckoutTester(a)
							label := a.t("commitMessage")
							field, ok := tt.AccessByRole(ui.RoleTextField, label)
							if !ok {
								t.Fatal("missing opener")
							}
							tt.ClickAt(field.Bounds.X+10, field.Bounds.Y+10)
						}
						tt.Key(ui.Cmd, ui.KeyA)
						tt.Frame()
						tt.Key(mod, ui.KeyP)
						tt.Frame()
						tt.Type("repo.quickSwitch")
						tt.Frame()
						row, ok := tt.AccessByRole(ui.RoleOption, a.t("quickRepositorySwitchCommand"))
						if !ok || row.Disabled {
							t.Fatal("missing enabled repository-switch command")
						}
						if activate == "enter" {
							tt.Key(0, ui.KeyEnter)
						} else {
							tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
						}
						tt.Frame()
						tt.Frame()
						if a.commandPalette.open || !a.quickRepositorySwitch.open || a.loading {
							t.Fatal("handoff did not open only switcher")
						}
						dialogs := 0
						for _, node := range tt.AccessAll() {
							if node.Role == ui.RoleDialog {
								dialogs++
							}
						}
						if dialogs != 1 {
							t.Fatalf("constructed %d dialogs", dialogs)
						}
						tt.Type("other")
						tt.Frame()
						if a.quickRepositorySwitch.query != "other" || a.commitText != "title\n\nbody" {
							t.Fatal("search did not own input")
						}
						page := a.page
						tt.Key(ui.Cmd, ui.Key4)
						tt.Frame()
						if a.page != page {
							t.Fatal("background shortcut leaked")
						}
						if cancel == "escape" {
							tt.Key(0, ui.KeyEscape)
						} else {
							tt.ClickAt(2, 2)
						}
						tt.Frame()
						tt.Frame()
						tt.Type("X")
						tt.Frame()
						if a.commandPalette.open || a.quickRepositorySwitch.open || a.loading || !a.amend {
							t.Fatal("cancel failed or changed state")
						}
						switch opener {
						case "message":
							if a.commitText != "X" {
								t.Fatalf("lost original commit message selection: %q", a.commitText)
							}
						case "history":
							if a.historyQuery != "X" || a.commitText != "title\n\nbody" {
								t.Fatal("lost original history selection")
							}
						}
						tt.Key(mod, ui.KeyP)
						tt.Frame()
						if !a.commandPalette.open || a.commandPalette.query != "" {
							t.Fatal("palette reopen retained query")
						}
						tt.Key(0, ui.KeyEscape)
						tt.Frame()
						tt.Frame()
						tt.Key(ui.Cmd|ui.Shift, ui.KeyO)
						tt.Frame()
						if !a.quickRepositorySwitch.open || a.quickRepositorySwitch.query != "" {
							t.Fatal("direct switch reopen retained query")
						}
					})
				}
			}
		}
	}
}

func TestPaletteRepositorySwitchCatalogAndEmpty(t *testing.T) {
	a := headShortcutApp(t)
	entries := a.commandPaletteEntries()
	count := 0
	for _, entry := range entries {
		if entry.ID == "repo.quickSwitch" {
			count++
		}
		if entry.ID == "repo.quickSwitchAlias" {
			t.Fatal("alias duplicated")
		}
	}
	if count != 1 {
		t.Fatalf("catalog has %d repository switch commands; want one", count)
	}
	a.openCommandPalette()
	a.runPaletteCommand("repo.quickSwitch", a.commandPalette.generation)
	if !a.quickRepositorySwitch.open || a.loading {
		t.Fatal("no-alternatives should open empty picker")
	}
	tt := paletteCheckoutTester(a)
	if !tt.HasText(a.t("quickRepositorySwitchEmpty")) {
		t.Fatal("empty picker lacked actual empty state")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.quickRepositorySwitch.open || a.commandPalette.open {
		t.Fatal("empty picker did not close in one Escape")
	}
	a = &app{locale: localeEnglish}
	a.openCommandPalette()
	a.commandPalette.query = "repo.quickSwitch"
	tt = paletteTester(a, false, 360, 360)
	row, ok := tt.AccessByRole(ui.RoleOption, a.t("quickRepositorySwitchCommand"))
	if !ok || !row.Disabled || a.commandPaletteUnavailable("repo.quickSwitch") != a.t("commandPaletteNeedRepo") {
		t.Fatal("no-repository command not disabled in actual UI")
	}
}

func TestPaletteRepositorySwitchTabLeavesSearch(t *testing.T) {
	a := headShortcutApp(t)
	a.recentRepositories = []string{a.repo.Root, "/fixtures/other"}
	typing := false
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() { a.commitMessageField(c); a.commandPickerDialogs(c) })
		typing = c.FocusTakesText()
	}, 600, 500)
	tt.Key(ui.Cmd, ui.KeyP)
	tt.Frame()
	tt.Type("repo.quickSwitch")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if !typing {
		t.Fatal("search was not the logical text input")
	}
	tt.Key(0, ui.KeyTab)
	tt.Frame()
	tt.Frame()
	if typing {
		t.Fatal("Tab did not leave search or a later frame stole focus")
	}
	if !tt.Focused("/fixtures/other") {
		t.Fatal("Tab did not focus repository option")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.commandPalette.open || a.quickRepositorySwitch.open {
		t.Fatal("single Escape after Tab did not close session")
	}
}

func TestPaletteRepositorySwitchOwnedConfirmation(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			a, _, root := worktreeApp(t)
			_, _, target := worktreeApp(t)
			if failure {
				target = filepath.Join(t.TempDir(), "missing repository")
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
				return pullFixtureGit(t, path, "rev-parse", "HEAD") + pullFixtureGit(t, path, "for-each-ref") + pullFixtureGit(t, path, "ls-files", "--stage") + pullFixtureGit(t, path, "status", "--porcelain=v1")
			}
			before := fingerprint(root)
			targetBefore := ""
			if !failure {
				targetBefore = fingerprint(target)
			}
			tt := paletteCheckoutTester(a)
			tt.Key(ui.Cmd, ui.KeyP)
			tt.Frame()
			tt.Type("repo.quickSwitch")
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Frame()
			tt.Type(filepath.Base(target))
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if !a.loading || a.quickRepositorySwitch.open || a.commandPalette.open {
				t.Fatal("confirmation did not leave modal and start open")
			}
			for a.loading {
				runContinuation(t, done)
			}
			if failure {
				if a.repo.Root != root || a.commitText != "draft\n\nbody" || !a.amend || !reflect.DeepEqual(a.recentRepositories, recent) || !strings.Contains(a.loadErr, target) {
					t.Fatal("failure changed original context or omitted target")
				}
			} else if a.repo.Root != target || a.commitText != "" || a.amend || a.recentRepositories[0] != target {
				t.Fatal("successful handoff switch violated existing reset rules")
			}
			if fingerprint(root) != before || !failure && fingerprint(target) != targetBefore {
				t.Fatal("handoff changed repository contents")
			}
		})
	}
}

func TestPaletteRepositorySwitchRenderedEntry(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			name := fmt.Sprintf("repository-command-360x360-%s-%s", language, mode)
			t.Run(name, func(t *testing.T) {
				a := headShortcutApp(t)
				a.locale, a.themeMode = language, mode
				a.openCommandPalette()
				a.commandPalette.query = "repo.quickSwitch"
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					ui.Column(c).Fill().Background(theme.TokensOf(c).BgLayout).Children(func() { a.commandPickerDialogs(c) })
				}, 360, 360)
				row, ok := tt.AccessByRole(ui.RoleOption, a.t("quickRepositorySwitchCommand"))
				if !ok || row.Disabled {
					t.Fatal("localized command unavailable")
				}
				if row.Bounds.X < 16 || row.Bounds.X+row.Bounds.W > 344 {
					t.Fatal("command outside gutters")
				}
				if dir := os.Getenv("SUPERGIT_REPOSITORY_COMMAND_CAPTURE_DIR"); dir != "" {
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

func TestPaletteRepositorySwitchRowsReflectGuards(t *testing.T) {
	for _, reason := range []string{"busy", "loading", "history", "chooser", "ref", "prompt"} {
		t.Run(reason, func(t *testing.T) {
			a := headShortcutApp(t)
			a.openCommandPalette()
			a.commandPalette.query = "repo.quickSwitch"
			tt := paletteCheckoutTester(a)
			switch reason {
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
			case "prompt":
				a.prompt.open = true
				a.prompt.root = a.repo.Root
			}
			tt.Frame()
			row, ok := tt.AccessByRole(ui.RoleOption, a.t("quickRepositorySwitchCommand"))
			if !ok || !row.Disabled {
				t.Fatal("row did not reflect guard", reason)
			}
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if a.quickRepositorySwitch.open {
				t.Fatal("disabled row executed")
			}
		})
	}
}

func TestPaletteRepositorySwitchRetainedCallback(t *testing.T) {
	for _, change := range []string{"root", "epoch", "reopen", "busy", "loading", "history", "chooser", "ref", "prompt"} {
		t.Run(change, func(t *testing.T) {
			a := headShortcutApp(t)
			a.openCommandPalette()
			generation := a.commandPalette.generation
			switch change {
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "reopen":
				a.commandPalette = commandPaletteState{}
				a.openCommandPalette()
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
			case "prompt":
				a.prompt.open = true
				a.prompt.root = a.repo.Root
			}
			a.runPaletteCommand("repo.quickSwitch", generation)
			if a.quickRepositorySwitch.open {
				t.Fatal("retained command bypassed guard")
			}
			if change == "reopen" && !a.commandPalette.open {
				t.Fatal("old callback closed new palette")
			}
		})
	}
}
