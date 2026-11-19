package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func createTagPaletteEntry(t *testing.T, a *app) appShortcut {
	t.Helper()
	a.commandPalette.query = ""
	for _, entry := range a.commandPaletteEntries() {
		if entry.ID == "tag.create" {
			return entry
		}
	}
	t.Fatal("missing selected commit lightweight tag command")
	return appShortcut{}
}

func createTagPaletteApp(t *testing.T) *app {
	a := browsingApp(t)
	a.commitText = "title\n\nbody"
	a.amend = true
	return a
}

func TestPaletteCreateTagHandoffCancel(t *testing.T) {
	for _, opener := range []string{"summary", "body", "history"} {
		for _, mod := range []ui.Modifiers{ui.Cmd, ui.Cmd | ui.Shift} {
			for _, activate := range []string{"enter", "click"} {
				for _, dismiss := range []string{"escape", "backdrop", "button"} {
					t.Run(opener+"/"+shortcutModName(mod)+"/"+activate+"/"+dismiss, func(t *testing.T) {
						a := createTagPaletteApp(t)
						entry := createTagPaletteEntry(t, a)
						tt := paletteStashTester(a, opener, 900, 640)
						if opener == "history" {
							tt.Key(ui.Cmd, ui.KeyF)
							tt.Type("original")
							tt.Frame()
						} else {
							label := a.t("commitSummary")
							if opener == "body" {
								label = a.t("commitDescription")
							}
							f, ok := tt.AccessByRole(ui.RoleTextField, label)
							if !ok {
								t.Fatal("missing original input")
							}
							tt.ClickAt(f.Bounds.X+10, f.Bounds.Y+10)
						}
						query, selected := a.historyQuery, a.selectedSHA
						tt.Key(ui.Cmd, ui.KeyA)
						tt.Key(mod, ui.KeyP)
						tt.Frame()
						tt.Type(entry.ID)
						tt.Frame()
						row, ok := tt.AccessByRole(ui.RoleOption, a.t(entry.Label))
						if !ok || row.Disabled {
							t.Fatal("missing available tag action")
						}
						if activate == "enter" {
							tt.Key(0, ui.KeyEnter)
						} else {
							tt.ClickAt(row.Bounds.X+row.Bounds.W/2, row.Bounds.Y+row.Bounds.H/2)
						}
						tt.Frame()
						tt.Frame()
						if a.commandPalette.open || !a.prompt.open || a.busy || !tt.HasText(a.tf("createTagStart", shortSHA(selected))) {
							t.Fatal("missing captured tag target handoff")
						}
						dialogs := 0
						for _, n := range tt.AccessAll() {
							if n.Role == ui.RoleDialog {
								dialogs++
							}
						}
						if dialogs != 1 {
							t.Fatal("tag handoff built multiple overlays")
						}
						tt.Type("   ")
						tt.Frame()
						tt.Key(0, ui.KeyEnter)
						tt.Frame()
						if !a.prompt.open || a.busy {
							t.Fatal("whitespace accepted")
						}
						tt.Key(ui.Cmd, ui.KeyA)
						tt.Type("v1.0")
						tt.Frame()
						switch dismiss {
						case "escape":
							tt.Key(0, ui.KeyEscape)
						case "backdrop":
							tt.ClickAt(2, 2)
						case "button":
							b, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel"))
							if !ok {
								t.Fatal("missing Cancel")
							}
							tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
						}
						tt.Frame()
						tt.Frame()
						if a.prompt.open || a.busy || a.commitText != "title\n\nbody" || !a.amend || a.historyQuery != query || a.selectedSHA != selected {
							t.Fatal("cancel changed context")
						}
						tt.Type("X")
						tt.Frame()
						switch opener {
						case "summary":
							if a.commitText != "X\n\nbody" {
								t.Fatal("lost Summary selection")
							}
						case "body":
							if a.commitText != "title\n\nX" {
								t.Fatal("lost Body selection")
							}
						case "history":
							if a.historyQuery != "X" {
								t.Fatal("lost history selection")
							}
						}
					})
				}
			}
		}
	}
}
