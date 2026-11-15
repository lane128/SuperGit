package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// An element shortcut on the focused row must take Enter before the button
// activates itself, so a button row can open its item instead of swallowing
// the key.
func TestButtonRowShortcutTakesEnterBeforeTheButton(t *testing.T) {
	rowClicked := false
	rowShortcut := false
	pageShortcut := false
	tt := ui.NewTester(func(c *ui.Context) {
		c.OnWindowShortcut(0, ui.KeyEnter, func() { pageShortcut = true })
		row := ui.ButtonBase(c).Row().Width(400).Height(40)
		row.OnShortcut(0, ui.KeyEnter, func() { rowShortcut = true })
		if row.Clicked() {
			rowClicked = true
		}
	}, 600, 200)
	tt.ClickAt(10, 10)
	rowClicked = false
	rowShortcut = false
	tt.Key(0, ui.KeyEnter)
	t.Logf("clicked=%v rowShortcut=%v pageShortcut=%v", rowClicked, rowShortcut, pageShortcut)
	if !rowShortcut || rowClicked || pageShortcut {
		t.Errorf("Enter after focusing a button row: clicked=%v shortcut=%v page=%v; want the row shortcut alone", rowClicked, rowShortcut, pageShortcut)
	}
}
