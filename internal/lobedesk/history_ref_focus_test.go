package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// The ref popup normally keeps the keyboard in its search box so typing and
// the arrow keys keep working. When the read has failed, Retry is the only
// useful control: the box must not hold the focus every frame, or the error
// can only be retried with the mouse.
func TestRefErrorRetryIsReachableByTab(t *testing.T) {
	a := &app{
		locale: localeChinese, page: "history",
		historyRefOpen: true,
		branchesErr:    "fatal: cannot read refs",
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(400).Children(func() {
			a.historyRefMenu(c)
		})
	}, 700, 400)
	tt.Frame()
	tt.Key(0, ui.KeyTab)
	retry, ok := tt.AccessByRole(ui.RoleButton, a.t("retry"))
	if !ok {
		t.Fatal("the failed ref read offers no retry")
	}
	if !retry.Focused {
		t.Fatalf("Tab reached %+v; want the Retry button", tt.AccessAll())
	}
}
