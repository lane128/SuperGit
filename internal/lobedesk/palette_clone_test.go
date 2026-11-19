package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func paletteCloneTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() {
			a.commitMessageFields(c)
			a.cloneRepositoryDialog(c)
			a.commandPickerDialogs(c)
		})
	}, 900, 640)
}

func openCloneViaPalette(t *testing.T, a *app, tt *ui.Tester, mod ui.Modifiers, activation string) {
	t.Helper()
	tt.Key(mod, ui.KeyP)
	tt.Frame()
	tt.Type("repo.clone")
	tt.Frame()
	row, ok := tt.AccessByRole(ui.RoleOption, a.t("cloneRepoTitle"))
	if !ok || row.Disabled {
		t.Fatal("Clone palette command missing or disabled")
	}
	if activation == "click" {
		tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
	} else {
		tt.Key(0, ui.KeyEnter)
	}
	tt.Frame()
	tt.Frame()
	if a.commandPalette.open || !a.cloneForm.open || a.busy || a.cloneForm.err != "" {
		t.Fatalf("Clone handoff failed or submitted: %+v", a.cloneForm)
	}
	dialogs := 0
	for _, n := range tt.AccessAll() {
		if n.Role == ui.RoleDialog {
			dialogs++
		}
	}
	if dialogs != 1 {
		t.Fatalf("expected one dialog, got %d", dialogs)
	}
}

func TestPaletteCloneHandoffCancel(t *testing.T) {
	for _, noRepo := range []bool{false, true} {
		for _, mod := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
			for _, activation := range []string{"enter", "click"} {
				for _, cancel := range []string{"escape", "backdrop", "button"} {
					name := shortcutModName(mod) + "/" + activation + "/" + cancel
					if noRepo {
						name = "empty/" + name
					} else {
						name = "repository/" + name
					}
					t.Run(name, func(t *testing.T) {
						a, _, _ := quickCheckoutFixture(t)
						if noRepo {
							a.repo = nil
						}
						tt := paletteCloneTester(a)
						body, ok := tt.AccessByRole(ui.RoleTextField, a.t("commitDescription"))
						if !ok {
							t.Fatal("Body missing")
						}
						tt.ClickAt(body.Bounds.X+10, body.Bounds.Y+10)
						tt.Key(ui.Cmd, ui.KeyA)
						openCloneViaPalette(t, a, tt, mod, activation)
						tt.Type("local-source")
						tt.Frame()
						if a.cloneForm.address != "local-source" {
							t.Fatalf("source did not get focus: %q", a.cloneForm.address)
						}
						repo, page := a.repo, a.page
						tt.Key(ui.Cmd, ui.KeyEnter)
						tt.Frame()
						if a.repo != repo || a.page != page || a.busy || a.commitText != "draft\n\nbody" || !a.amend {
							t.Fatal("commit shortcut escaped Clone modal")
						}
						for _, alias := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
							tt.Key(alias, ui.KeyP)
							tt.Frame()
						}
						if a.commandPalette.open || a.cloneForm.address != "local-source" {
							t.Fatal("repeated palette reset Clone")
						}
						switch cancel {
						case "escape":
							tt.Key(0, ui.KeyEscape)
						case "backdrop":
							tt.ClickAt(2, 2)
						case "button":
							b, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
							if !ok {
								t.Fatal("Cancel missing")
							}
							tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
						}
						tt.Frame()
						tt.Frame()
						tt.Type("X")
						tt.Frame()
						if a.cloneForm.open || a.commandPalette.open || a.commitText != "draft\n\nX" || !a.amend {
							t.Fatalf("cancel lost original selection/draft: %q", a.commitText)
						}
					})
				}
			}
		}
	}
}
