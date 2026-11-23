package lobedesk

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCommandPaletteReferenceCatalog(t *testing.T) {
	data, err := os.ReadFile("testdata/sublime-merge-2125-palette.txt")
	if err != nil {
		t.Fatal(err)
	}
	reference := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(reference) != 75 {
		t.Fatal("expected 75 default commands from Build 2125")
	}
	a := &app{locale: localeEnglish}
	byCaption := map[string]appShortcut{}
	byReference := map[string]appShortcut{}
	for _, action := range a.commandPaletteEntries() {
		caption := a.t(action.Label)
		if _, exists := byCaption[caption]; exists || action.Run == nil {
			t.Fatalf("duplicate or unimplemented command %q", caption)
		}
		byCaption[caption] = action
		if reference := referenceCommandNames[action.ID]; reference != "" {
			byReference[reference] = action
		}
	}
	for _, caption := range append(reference, "About") {
		action, ok := byCaption[caption]
		if !ok {
			action, ok = byReference[caption]
		}
		if !ok {
			t.Errorf("missing reference command %q", caption)
			continue
		}
		a.locale = localeChinese
		if zh := a.t(action.Label); zh == action.Label || zh == "" || zh == caption {
			t.Errorf("%s has no Chinese label", caption)
		}
		a.commandPalette.query = caption
		if !slices.ContainsFunc(a.commandPaletteEntries(), func(s appShortcut) bool { return s.ID == action.ID }) {
			t.Errorf("cannot find %q by its reference name in Chinese", caption)
		}
		a.commandPalette.query = strings.ReplaceAll(caption, "…", "...")
		if !slices.ContainsFunc(a.commandPaletteEntries(), func(s appShortcut) bool { return s.ID == action.ID }) {
			t.Errorf("cannot find %q by its ASCII spelling", caption)
		}
	}
}

func TestCommandPaletteEmphasizesMatchedLetters(t *testing.T) {
	for _, sample := range []struct{ query, title, bold string }{
		{"crbr", "Create Branch…", "CrBr"}, {"分支", "创建分支…", "分支"}, {"ré", "Créer", "ré"},
	} {
		var all, bold strings.Builder
		for _, span := range commandHighlightSpans(sample.query, sample.title) {
			all.WriteString(span.Text)
			if span.Weight == 700 {
				bold.WriteString(span.Text)
			}
		}
		if all.String() != sample.title || bold.String() != sample.bold {
			t.Fatalf("%q: %q emphasized in %q", sample.query, bold.String(), all.String())
		}
	}
}

func TestCommandPaletteExactIDBeatsLongerPrefix(t *testing.T) {
	a := &app{locale: localeEnglish}
	for _, query := range []string{"stash.save", "Stash", "STASH...", "保存改动"} {
		a.commandPalette.query = query
		rows := a.commandPaletteEntries()
		if len(rows) == 0 || rows[0].ID != "stash.save" {
			t.Fatalf("%s resolved to %v", query, rows)
		}
	}
}

func TestCommandPaletteParameterBackRestoresTypedRemote(t *testing.T) {
	a, _ := overviewFixture(t)
	a.locale = localeEnglish
	a.openCommandPalette()
	a.runPaletteCommand("remote.add", a.commandPalette.generation)
	tt := paletteTester(a, false, 640, 480)
	tt.Frame()
	tt.Type("new-origin")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.commandPalette.step == nil || a.commandPalette.step.placeholder != a.t("remoteAddress") {
		t.Fatalf("expected address step, got %+v", a.commandPalette)
	}
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if a.commandPalette.query != "new-origin" || a.commandPalette.step.placeholder != a.t("remoteName") {
		t.Fatalf("Backspace lost the previous parameter: %+v", a.commandPalette)
	}
}

func TestCommandPaletteLargePickerWrapAndAccept(t *testing.T) {
	a := &app{locale: localeEnglish}
	rows := make([]paletteChoice, 1000)
	for i := range rows {
		rows[i] = paletteChoice{id: fmt.Sprint(i), label: fmt.Sprintf("branch-%04d", i)}
	}
	chosen := ""
	a.choosePalette("Branch", "Branch name", rows, func(row paletteChoice) { chosen = row.id })
	tt := paletteTester(a, false, 600, 480)
	tt.Frame()
	tt.Key(0, ui.KeyUp)
	tt.Frame()
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleOption, "branch-0999"); !ok || len(tt.AccessAll()) >= 1000 {
		t.Fatal("last option missing or the entire branch list was rendered")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if chosen != "999" {
		t.Fatalf("selected %q", chosen)
	}
}

func TestCommandPaletteBackRestoresClickedRow(t *testing.T) {
	a := &app{locale: localeEnglish}
	rows := []paletteChoice{{id: "first", label: "First"}, {id: "second", label: "Second"}}
	a.choosePalette("Command", "Choose", rows, func(row paletteChoice) {
		a.inputPalette(row.label, "Name", "", func(string) {})
	})
	tt := paletteTester(a, false, 600, 480)
	item, ok := tt.AccessByRole(ui.RoleOption, "Second")
	if !ok {
		t.Fatal("missing second option")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	tt.Frame()
	tt.Key(0, ui.KeyBackspace)
	tt.Frame()
	if a.commandPalette.step == nil || a.commandPalette.step.title != "Command" || a.commandPalette.cursor != 1 {
		t.Fatal("Back should highlight the clicked option")
	}
}
