package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func blameRowsApp() *app {
	return &app{
		locale:    localeChinese,
		blamePath: "main.go",
		blame: []gitrepo.BlameLine{
			{SHA: "abcdef1234567890", Author: "Ada", Date: "2026-01-01T10:00:00Z", Line: 3, Text: "fmt.Println()"},
			{SHA: "0000000000000000", Author: "", Line: 4, Text: "}"},
		},
	}
}

func blameRowsTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(400).Children(func() {
			a.blameRows(c)
		})
	}, 700, 400)
}

// A blame row was a plain row: nothing to focus, so the menu key and Shift+F10
// could not reach the actions it offers. It is a button now whose name says
// which commit and line it is, without making a screen reader read the source.
func TestBlameRowIsANamedButton(t *testing.T) {
	a := blameRowsApp()
	tt := blameRowsTester(a)

	committed := a.blameRowName(a.blame[0])
	if !strings.Contains(committed, shortSHA(a.blame[0].SHA)) || !strings.Contains(committed, "3") {
		t.Fatalf("the row name %q wants the commit and the line number", committed)
	}
	info, ok := tt.AccessByRole(ui.RoleButton, committed)
	if !ok {
		t.Fatalf("no button named %q is laid out", committed)
	}
	if info.Name != committed {
		t.Errorf("the row reads %q; want %q", info.Name, committed)
	}
	// The label replaces the row's text for assistive technology; the author
	// and the source line it draws have to reach the reader another way.
	if !strings.Contains(info.Description, "Ada") || !strings.Contains(info.Description, "fmt.Println()") {
		t.Errorf("the row describes itself as %q; want the author and the source line", info.Description)
	}
	if date := formatDateTime(a.blame[0].Date); !strings.Contains(info.Description, date) {
		t.Errorf("the row describes itself as %q; want the blamed date %q", info.Description, date)
	}
	uncommitted := a.blameRowName(a.blame[1])
	if !strings.Contains(uncommitted, a.t("uncommitted")) {
		t.Errorf("the uncommitted row reads %q; want it named as uncommitted", uncommitted)
	}
	info, ok = tt.AccessByRole(ui.RoleButton, uncommitted)
	if !ok {
		t.Errorf("the uncommitted row has no button named %q", uncommitted)
	}
	// An uncommitted line has no author or date; the description must not
	// open with the separator those empty parts used to leave behind.
	if strings.HasPrefix(info.Description, " ·") {
		t.Errorf("the uncommitted row describes itself as %q; want the source line alone", info.Description)
	}
}

// The menu key and Shift+F10 open the row's menu once a click has given it the
// focus. The uncommitted row is the one clicked: its click has no other action,
// so opening the menu is the only thing the test observes.
func TestBlameRowMenuOpensFromTheKeyboard(t *testing.T) {
	a := blameRowsApp()
	tt := blameRowsTester(a)
	row, ok := tt.Find(a.t("uncommitted"))
	if !ok {
		t.Fatal("the uncommitted row is not laid out")
	}
	item := a.t("fileHistory")
	for _, test := range []struct {
		name string
		send func()
	}{
		{"the menu key", func() { tt.Key(0, ui.KeyContextMenu) }},
		{"Shift+F10", func() { tt.Key(ui.Shift, ui.KeyF10) }},
	} {
		if _, ok := tt.Find(item); ok {
			t.Fatalf("%s: the menu is open before the key", test.name)
		}
		tt.ClickAt(row.X+row.W/2, row.Y+row.H/2)
		test.send()
		if _, ok := tt.Find(item); !ok {
			t.Errorf("%s shows no %q item", test.name, item)
		}
		tt.Key(0, ui.KeyEscape)
	}
}
