package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// A page draws its first frame before the row has a width. The stash list must
// not take the whole page for that frame and leave the detail pane out: both
// sides keep their opening share, as the workspace does.
func TestStashPageSharesItsFirstFrame(t *testing.T) {
	a := &app{locale: localeChinese}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(900).Height(600).Children(func() {
			a.stashPage(c)
		})
	}, 900, 600)
	if a.stashSplit.rowWidth != stashOpeningRoom {
		t.Errorf("the first frame assumed a %v-wide row; want the %v opening width", a.stashSplit.rowWidth, stashOpeningRoom)
	}
	if _, ok := tt.Find(a.t("stashDetailHint")); !ok {
		t.Error("the first frame has no detail pane")
	}
	if _, ok := tt.Find(a.t("noStashes")); !ok {
		t.Error("the first frame has no empty state in the list")
	}
}
