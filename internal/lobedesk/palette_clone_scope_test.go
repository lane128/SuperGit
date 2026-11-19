package lobedesk

import (
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestPaletteCloneGuards(t *testing.T) {
	for _, change := range []string{"busy", "loading", "history", "chooser", "ref", "dialog", "clone", "root", "epoch", "reopen"} {
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
			case "clone":
				a.cloneForm = cloneFormState{open: true, address: "keep-source", path: "keep-path"}
			case "root":
				a.repo = &gitrepo.Repository{Root: t.TempDir()}
			case "epoch":
				a.loadSeq++
			case "reopen":
				a.commandPalette = commandPaletteState{}
				a.openCommandPalette()
			}
			a.runPaletteCommand("repo.clone", generation)
			if change == "clone" {
				if a.cloneForm.address != "keep-source" || a.cloneForm.path != "keep-path" {
					t.Fatal("existing form reset")
				}
			} else if a.cloneForm.open {
				t.Fatal("ineligible callback opened Clone")
			}
			if change == "reopen" && !a.commandPalette.open {
				t.Fatal("old callback closed new palette")
			}
		})
	}
	if s := clonePaletteCommand(); s.Key != 0 || s.Keys != "" {
		t.Fatal("Clone added shortcut")
	}
}

func TestPaletteCloneOwnedSubmit(t *testing.T) {
	for _, scenario := range []string{"success", "occupied", "queued-cancel"} {
		t.Run(scenario, func(t *testing.T) {
			a, done, old := worktreeApp(t)
			_, _, source := worktreeApp(t)
			a.commitText = "draft\n\nbody"
			a.amend = true
			tt := paletteCloneTester(a)
			openCloneViaPalette(t, a, tt, ui.Cmd, "enter")
			destination := filepath.Join(t.TempDir(), "checkout")
			if scenario == "occupied" {
				destination = t.TempDir()
				if err := os.WriteFile(filepath.Join(destination, "keep"), []byte("untouched"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			tt.Type(source)
			tt.Key(0, ui.KeyTab)
			tt.Type(destination)
			tt.Frame()
			if a.cloneForm.address != source || a.cloneForm.path != destination {
				t.Fatal("form typing failed")
			}
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			if scenario == "occupied" {
				data, err := os.ReadFile(filepath.Join(destination, "keep"))
				if err != nil || string(data) != "untouched" || a.busy || a.cloneForm.err == "" || a.repo.Root != old || a.commitText != "draft\n\nbody" {
					t.Fatal("occupied target changed state")
				}
				return
			}
			if !a.busy || a.repo.Root != old || a.commitText != "draft\n\nbody" {
				t.Fatal("Clone switched before completion")
			}
			if scenario == "queued-cancel" {
				var completion func()
				select {
				case completion = <-done:
				case <-time.After(10 * time.Second):
					t.Fatal("Clone timeout")
				}
				tt.Key(0, ui.KeyEscape)
				tt.Frame()
				tt.ClickAt(2, 2)
				tt.Frame()
				if !a.cloneForm.open || !a.busy {
					t.Fatal("running Clone hidden")
				}
				b, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
				if !ok {
					t.Fatal("Cancel missing")
				}
				tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
				tt.Frame()
				completion()
				if a.repo.Root != old || a.busy || a.loading || !a.cloneForm.open || a.cloneForm.err == "" || a.commitText != "draft\n\nbody" {
					t.Fatal("queued cancel switched or lost feedback")
				}
				return
			}
			runContinuation(t, done)
			for a.loading {
				runContinuation(t, done)
			}
			canonical, err := filepath.EvalSymlinks(destination)
			if err != nil || a.repo.Root != canonical || a.cloneForm.open || a.busy || a.loadErr != "" {
				t.Fatal("successful clone did not open")
			}
			out, err := exec.Command("git", "-C", destination, "show", "HEAD:base.txt").Output()
			if err != nil || string(out) != "base\n" {
				t.Fatal("wrong cloned tree")
			}
		})
	}
}

func TestPaletteCloneNarrowError(t *testing.T) {
	a := &app{locale: localeEnglish}
	a.openCommandPalette()
	a.runPaletteCommand("repo.clone", a.commandPalette.generation)
	a.cloneForm.err = "Clone failed. The destination remains on disk. Choose a different destination before retrying."
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.commandPickerDialogs(c)
	}, 360, 360)
	for _, n := range tt.AccessAll() {
		if n.Role != ui.RoleDialog && n.Role != ui.RoleTextField && n.Role != ui.RoleButton {
			continue
		}
		if n.Bounds.Y < 16 || n.Bounds.Y+n.Bounds.H > 344 {
			t.Errorf("error control outside gutters: %+v", n)
		}
	}
}

func TestPaletteCloneNarrow(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			t.Run(string(language)+"/"+mode, func(t *testing.T) {
				a := &app{locale: language, themeMode: mode}
				a.openCommandPalette()
				a.commandPalette.query = "repo.clone"
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					ui.Column(c).Fill().Background(theme.TokensOf(c).BgLayout).Children(func() { a.cloneRepositoryDialog(c); a.commandPickerDialogs(c) })
				}, 360, 360)
				tt.Key(0, ui.KeyEnter)
				tt.Frame()
				tt.Frame()
				tt.Advance(150 * time.Millisecond)
				for _, n := range tt.AccessAll() {
					if n.Role != ui.RoleDialog && n.Role != ui.RoleTextField && n.Role != ui.RoleButton {
						continue
					}
					if n.Bounds.X < 16 || n.Bounds.Y < 16 || n.Bounds.X+n.Bounds.W > 344 || n.Bounds.Y+n.Bounds.H > 344 {
						t.Errorf("control outside gutters: %+v", n)
					}
				}
				tt.Type("source")
				tt.Frame()
				if a.cloneForm.address != "source" {
					t.Fatal("source focus missing")
				}
				tt.Key(0, ui.KeyTab)
				tt.Type("target")
				tt.Frame()
				if a.cloneForm.path != "target" {
					t.Fatal("target not keyboard reachable")
				}
				button, ok := tt.AccessByRole(ui.RoleButton, a.t("cloneAction"))
				if !ok || button.Disabled {
					t.Fatal("ready Clone button unavailable")
				}
				tt.Advance(150 * time.Millisecond)
				if dir := os.Getenv("SUPERGIT_PALETTE_CAPTURE_DIR"); dir != "" {
					f, err := os.Create(filepath.Join(dir, fmt.Sprintf("clone-360x360-%s-%s.png", language, mode)))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					closeErr := f.Close()
					if err != nil || closeErr != nil {
						t.Fatalf("capture: %v %v", err, closeErr)
					}
				}
			})
		}
	}
}
