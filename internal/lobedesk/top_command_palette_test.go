package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestTopCommandPaletteMatchOrdering(t *testing.T) {
	for _, tc := range []struct {
		query, name string
		matches     bool
	}{
		{"CRBR", "Create Branch", true},
		{"create br", "Create Branch", true},
		{"建分", "创建并切换分支", true},
		{"brcr", "Create Branch", false},
		{"zzzz", "Create Branch", false},
		{"  ", "Create Branch", true},
	} {
		if got := commandMatchScore(tc.query, tc.name) >= 0; got != tc.matches {
			t.Errorf("%q / %q: matched=%v", tc.query, tc.name, got)
		}
	}
	if commandMatchScore("branch", "Branch actions") <= commandMatchScore("branch", "Browse a new checkout") {
		t.Fatal("contiguous command text should rank above scattered letters")
	}
	if commandMatchScore("crbr", "Create", "Branch") >= 0 {
		t.Fatal("abbreviation crossed unrelated alias boundaries")
	}
}

func TestTopCommandPaletteFuzzySearch(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		a := createBranchApp(t)
		a.locale = language
		tt := paletteStashTester(a, "body", 900, 640)
		tt.Key(ui.Cmd, ui.KeyP)
		tt.Frame()
		tt.Type("crbr")
		tt.Frame()
		entries := a.commandPaletteEntries()
		if len(entries) == 0 || entries[0].ID != "branch.create" {
			t.Fatalf("%s: crbr should rank Create Branch first: %+v; query=%q label=%q score=%d", language, entries, a.commandPalette.query, messages[localeEnglish]["createBranchCommand"], commandMatchScore("crbr", messages[localeEnglish]["createBranchCommand"]))
		}
		tt.Key(0, ui.KeyEnter)
		tt.Frame()
		tt.Frame()
		if !a.prompt.open || a.commandPalette.open || a.busy {
			t.Fatal("fuzzy result did not enter branch-name phase")
		}
	}
}

func TestTopCommandPaletteToolbarAndPosition(t *testing.T) {
	a := createBranchApp(t)
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		a.registerShortcuts(c)
		ui.Column(c).Fill().Children(func() { a.titleBar(c); a.commitMessageFields(c); a.commandPickerDialogs(c) })
	}, 900, 640)
	assertCommandControlBounds(t, tt, a, 900)
	branch, _ := tt.AccessByRole(ui.RoleButton, a.t("branches"))
	tt.ClickAt(branch.Bounds.X+branch.Bounds.W/2, branch.Bounds.Y+branch.Bounds.H/2)
	tt.Frame()
	checkout, ok := tt.AccessByRole(ui.RoleDialog, a.t("quickCheckout"))
	if !ok || !tt.Focused(a.t("filterBranches")) {
		t.Fatal("branch did not open the focused Checkout picker")
	}
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	tt.Frame()
	palette, ok := tt.AccessByRole(ui.RoleDialog, a.t("commandPalette"))
	if !ok || !tt.Focused(a.t("commandPaletteSearch")) || checkout.Bounds.Y != palette.Bounds.Y {
		t.Fatal("empty Checkout did not return to all commands in place")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	tt.Frame()
	if !tt.Focused(a.t("branches")) {
		t.Fatal("Escape did not restore focus to the branch opener")
	}
	b, ok := tt.AccessByRole(ui.RoleButton, a.t("commandPalette"))
	if !ok {
		t.Fatal("toolbar command entry missing")
	}
	tt.ClickAt(b.Bounds.X+b.Bounds.W/2, b.Bounds.Y+b.Bounds.H/2)
	tt.Frame()
	before, ok := tt.AccessByRole(ui.RoleDialog, a.t("commandPalette"))
	if !ok || before.Bounds.Y > 90 || !tt.Focused(a.t("commandPaletteSearch")) {
		t.Fatalf("toolbar did not open focused palette near top: %+v", before)
	}
	tt.Type("branch.create")
	tt.Frame()
	after, _ := tt.AccessByRole(ui.RoleDialog, a.t("commandPalette"))
	if after.Bounds.Y != before.Bounds.Y {
		t.Fatalf("filtering moved input: %v -> %v", before.Bounds, after.Bounds)
	}
}

func TestTopCommandPaletteParameterBackspace(t *testing.T) {
	for _, command := range []string{"branch.create", "branch.quickCheckout", "history.chooseRef", "repo.quickSwitch"} {
		t.Run(command, func(t *testing.T) {
			a := createBranchApp(t)
			tt := paletteStashTester(a, "body", 900, 640)
			body, _ := tt.AccessByRole(ui.RoleTextField, a.t("commitDescription"))
			tt.ClickAt(body.Bounds.X+10, body.Bounds.Y+10)
			tt.Key(ui.Cmd, ui.KeyP)
			tt.Frame()
			tt.Type(command)
			tt.Frame()
			tt.Key(0, ui.KeyEnter)
			tt.Frame()
			tt.Frame()
			if a.commandPalette.open {
				t.Fatal("did not enter parameters")
			}
			retained := a.prompt.action
			tt.Type("x")
			tt.Frame()
			tt.Key(0, ui.KeyBackspace)
			tt.Frame()
			if a.commandPalette.open {
				t.Fatal("nonempty Backspace returned instead of editing")
			}
			tt.Key(0, ui.KeyBackspace)
			tt.Frame()
			tt.Frame()
			if !a.commandPalette.open || a.commandPalette.query != "" || !tt.Focused(a.t("commandPaletteSearch")) {
				t.Fatalf("empty parameter Backspace did not return to all commands: palette=%+v prompt=%+v checkout=%+v refs=%q switch=%+v access=%+v", a.commandPalette, a.prompt, a.quickCheckout, a.historyRefQuery, a.quickRepositorySwitch, tt.AccessAll())
			}
			if retained != nil {
				retained("cancelled-return-must-not-create")
				if a.busy {
					t.Fatal("return left an executable prompt callback")
				}
			}
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			tt.Frame()
			tt.Key(0, ui.KeyEnd)
			tt.Type("X")
			tt.Frame()
			if a.commitText != "title\n\nbodyX" || a.busy {
				t.Fatalf("return lost draft/focus: %q", a.commitText)
			}
		})
	}
}
