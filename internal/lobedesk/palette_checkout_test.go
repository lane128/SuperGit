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

func paletteCheckoutTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() { a.commitMessageField(c); a.commandPickerDialogs(c) })
	}, 900, 640)
}

func TestPaletteCheckoutActualBodyHandoff(t *testing.T) {
	for _, mod := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
		for _, activate := range []string{"enter", "click"} {
			t.Run(shortcutModName(mod)+"/"+activate, func(t *testing.T) {
				a, _, _ := quickCheckoutFixture(t)
				tt := paletteCheckoutTester(a)
				body, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
				if !ok {
					t.Fatal("Body missing")
				}
				tt.ClickAt(body.Bounds.X+10, body.Bounds.Y+10)
				tt.Key(mod, ui.KeyP)
				tt.Frame()
				tt.Type("branch.quickCheckout")
				tt.Frame()
				row, ok := tt.AccessByRole(ui.RoleOption, a.t(quickCheckoutShortcut().Label))
				if !ok || row.Disabled {
					t.Fatal("checkout command missing or disabled by palette itself")
				}
				if quickCheckoutShortcut().Available(a) {
					t.Fatal("shortcut guard must still block background dispatch while palette owns modal")
				}
				if activate == "click" {
					tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
				} else {
					tt.Key(0, ui.KeyEnter)
				}
				tt.Frame()
				tt.Frame()
				if a.commandPalette.open || !a.quickCheckout.open || a.busy {
					t.Fatal("handoff did not exclusively open checkout")
				}
				dialogs := 0
				for _, node := range tt.AccessAll() {
					if node.Role == ui.RoleDialog {
						dialogs++
					}
				}
				if dialogs != 1 {
					t.Fatalf("handoff constructed %d dialogs", dialogs)
				}
				tt.Type("topic")
				tt.Frame()
				if a.quickCheckout.query != "topic" {
					t.Fatal("checkout search did not receive actual typing")
				}
				page := a.page
				tt.Key(ui.Cmd, ui.Key4)
				tt.Frame()
				if a.page != page {
					t.Fatal("background shortcut leaked")
				}
				tt.Key(0, ui.KeyEscape)
				tt.Frame()
				tt.Frame()
				tt.Key(0, ui.KeyEnd)
				tt.Type("X")
				tt.Frame()
				if a.quickCheckout.open || a.commandPalette.open || a.commitText != "draftX\n\nbody" || !a.amend {
					t.Fatalf("cancel lost original input/draft: %q", a.commitText)
				}
			})
		}
	}
}

func TestPaletteCheckoutBackdropRestoresBody(t *testing.T) {
	a, _, _ := quickCheckoutFixture(t)
	tt := paletteCheckoutTester(a)
	body, _ := tt.AccessByRole(ui.RoleTextField, a.t("commitMessage"))
	tt.ClickAt(body.Bounds.X+10, body.Bounds.Y+10)
	tt.Key(ui.Cmd, ui.KeyP)
	tt.Frame()
	tt.Type("branch.quickCheckout")
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	tt.ClickAt(2, 2)
	tt.Frame()
	tt.Frame()
	tt.Key(0, ui.KeyEnd)
	tt.Type("X")
	tt.Frame()
	if a.quickCheckout.open || a.commandPalette.open || a.commitText != "draftX\n\nbody" || !a.amend {
		t.Fatal("backdrop cancel lost original input or draft")
	}
}

func TestPaletteCheckoutRenderedEntry(t *testing.T) {
	for _, size := range [][2]int{{360, 360}, {900, 640}} {
		for _, language := range []locale{localeEnglish, localeChinese} {
			for _, mode := range []string{"light", "dark"} {
				name := fmt.Sprintf("checkout-%dx%d-%s-%s", size[0], size[1], language, mode)
				t.Run(name, func(t *testing.T) {
					a := &app{repo: &gitrepo.Repository{Root: t.TempDir()}, locale: language, themeMode: mode}
					a.openCommandPalette()
					a.commandPalette.query = "branch.quickCheckout"
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Fill().Background(theme.TokensOf(c).BgLayout).Children(func() { a.commandPickerDialogs(c) })
					}, size[0], size[1])
					row, ok := tt.AccessByRole(ui.RoleOption, a.t("quickCheckoutCommand"))
					if !ok || row.Disabled {
						t.Fatal("localized checkout command missing or unavailable")
					}
					for _, node := range tt.AccessAll() {
						if node.Role != ui.RoleDialog && node.Role != ui.RoleTextField && node.Role != ui.RoleOption {
							continue
						}
						if node.Bounds.X < 16 || node.Bounds.Y < 16 || node.Bounds.X+node.Bounds.W > float32(size[0]-16) || node.Bounds.Y+node.Bounds.H > float32(size[1]-16) {
							t.Errorf("checkout command outside gutters: %+v", node)
						}
					}
					if dir := os.Getenv("SUPERGIT_PALETTE_CAPTURE_DIR"); dir != "" {
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

func TestPaletteCheckoutStaleCommand(t *testing.T) {
	for _, change := range []string{"busy", "loading", "history", "chooser", "ref", "dialog", "root", "epoch", "reopen"} {
		t.Run(change, func(t *testing.T) {
			a, _, _ := quickCheckoutFixture(t)
			a.openCommandPalette()
			generation := a.commandPalette.generation
			switch change {
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
				a.prompt.open = true
				a.prompt.root = a.repo.Root
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "reopen":
				a.commandPalette = commandPaletteState{}
				a.openCommandPalette()
			}
			a.runPaletteCommand("branch.quickCheckout", generation)
			if a.quickCheckout.open {
				t.Fatal("stale/ineligible command opened checkout")
			}
			if change == "reopen" && !a.commandPalette.open {
				t.Fatal("old callback closed new palette generation")
			}
		})
	}
}

func TestPaletteCheckoutOwnedAction(t *testing.T) {
	for _, scenario := range []string{"success", "dirty", "tip", "occupied"} {
		t.Run(scenario, func(t *testing.T) {
			a, callbacks, target := quickCheckoutFixture(t)
			tt := paletteCheckoutTester(a)
			tt.Key(ui.Cmd, ui.KeyP)
			tt.Frame()
			tt.Type("branch.quickCheckout")
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Frame()
			tt.Type("topic")
			tt.Frame()
			if !a.quickCheckout.open {
				t.Fatal("checkout handoff failed")
			}
			switch scenario {
			case "dirty":
				if err := os.WriteFile(filepath.Join(a.repo.Root, "same.go"), []byte("external bytes\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "tip":
				pullFixtureGit(t, a.repo.Root, "branch", "-f", "topic", "HEAD")
			case "occupied":
				pullFixtureGit(t, a.repo.Root, "worktree", "add", filepath.Join(t.TempDir(), "owned-worktree"), "topic")
			}
			beforeHead := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD")
			beforeIndex := pullFixtureGit(t, a.repo.Root, "ls-files", "--stage")
			beforeBytes, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
			if err != nil {
				t.Fatal(err)
			}
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			waitQuickCheckout(t, a, callbacks)
			if scenario == "success" {
				if got := pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD"); got != target {
					t.Fatal("wrong displayed target")
				}
				if got := pullFixtureGit(t, a.repo.Root, "branch", "--show-current"); got != "topic" {
					t.Fatal("wrong branch")
				}
			} else {
				afterBytes, err := os.ReadFile(filepath.Join(a.repo.Root, "same.go"))
				if err != nil {
					t.Fatal(err)
				}
				if pullFixtureGit(t, a.repo.Root, "rev-parse", "HEAD") != beforeHead || pullFixtureGit(t, a.repo.Root, "ls-files", "--stage") != beforeIndex || string(afterBytes) != string(beforeBytes) {
					t.Fatal("rejection changed repository baseline")
				}
				if len(a.notices[toastMain]) == 0 {
					t.Fatal("rejection lacks feedback")
				}
			}
			if a.commitText != "draft\n\nbody" || !a.amend {
				t.Fatal("checkout changed draft")
			}
		})
	}
}
