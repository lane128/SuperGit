package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
	"testing"
)

func TestHistoryStashBadgeReservesFullTimestamp(t *testing.T) {
	old := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(old) })
	for _, style := range []string{"segmented", "stripe"} {
		for _, scale := range []float32{1, 1.15} {
			setUIFontScale(scale)
			stash := gitrepo.Stash{SHA: "s", Ref: "stash@{12}", Message: "Saved work", Date: "2020-01-01T10:20:00Z", Author: "Someone"}
			a := &app{historyRefStyle: style, showGraph: true, hideAuthor: true, historyDetailsHidden: true}
			view := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				a.historySplit.rowWidth, _ = c.Size()
				ui.Column(c).FillWidth().PaddingX(historyListBorder + historyListPadding).Children(func() { a.historyEntryRow(c, stashCommit(stash), &stash, 0, 0, 1, &commitGraph{Width: 76}) })
			}, 260, 150)
			row, _ := view.AccessByRole(ui.RoleOption, stash.Message)
			for _, n := range view.AccessAll() {
				if n.Role == ui.RoleText && !n.Offscreen && n.Bounds.Y >= row.Bounds.Y && n.Bounds.Y+n.Bounds.H <= row.Bounds.Y+row.Bounds.H {
					if n.Bounds.X+n.Bounds.W > row.Bounds.X+row.Bounds.W+.5 {
						t.Errorf("%s scale=%v: %s overflows row %+v: %+v", style, scale, n.Name, row.Bounds, n.Bounds)
					}
				}
			}
			label, ok := view.AccessByRole(ui.RoleText, stash.Ref)
			if !ok || label.Offscreen || label.Bounds.W < 40 {
				t.Errorf("stash identity lost: %+v", label)
			}
		}
	}
}
