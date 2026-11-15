package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// Stashes created without a message share the same default text, so the row
// tells them apart by reference and date. The destructive confirmation used to
// drop both, leaving two identically worded dialogs for different entries; it
// now names the reference and the short SHA as well.
func TestStashDropConfirmationIdentifiesTheStash(t *testing.T) {
	a := &app{locale: localeChinese}
	stash := gitrepo.Stash{SHA: "abcdef1234567890", Ref: "stash@{1}", Message: "WIP on main"}
	a.confirmDropStash(stash)
	if !a.confirm.open {
		t.Fatal("dropping a stash opens no confirmation")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.confirmDialog(c)
	}, 640, 420)
	if _, ok := tt.Find(a.confirm.message); !ok {
		t.Fatal("the confirmation does not draw its message")
	}
	for _, want := range []string{stash.Ref, shortSHA(stash.SHA)} {
		if !strings.Contains(a.confirm.message, want) {
			t.Errorf("the confirmation reads %q; want it to name %q", a.confirm.message, want)
		}
	}
}
