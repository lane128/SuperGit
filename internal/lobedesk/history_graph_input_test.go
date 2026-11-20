package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestHistoryGraphPansFromCommitRowsAtMinimumWidth(t *testing.T) {
	a := &app{page: "history", locale: localeEnglish, showGraph: true, commits: manyLaneHistory(14)}
	first := a.commits[0]
	first.Subject = "First commit in the graph"
	a.commits[0] = first
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		w, _ := c.Size()
		a.historySplit.rowWidth = w - 180
		ui.Row(c).Fill().Children(func() {
			ui.Box(c).Width(180).Shrink(0)
			ui.Column(c).Width(a.commitListWidth()).AlignSelf(ui.Stretch).Children(func() { a.commitList(c) })
		})
	}, 3008, 640)
	settle := func() {
		for range 5 {
			tt.Frame()
		}
	}
	settle()
	if a.historyGraphScroll.MaxX != 0 {
		t.Fatal("wide view should fit all lanes")
	}
	tt.SetSize(980, 640)
	settle()
	if a.historyGraphScroll.MaxX <= 0 {
		t.Fatal("narrow view must have overflow")
	}
	const scrollbar = "Scroll commit graph"
	bar, ok := tt.AccessByRole(ui.RoleScroll, scrollbar)
	if !ok || !bar.Focusable || bar.Offscreen {
		t.Fatalf("overflow graph lacks an accessible scroll control: %+v", bar)
	}
	for i := 0; i < 6 && !bar.Focused; i++ {
		tt.Key(0, ui.KeyTab)
		tt.Frame()
		bar, _ = tt.AccessByRole(ui.RoleScroll, scrollbar)
	}
	if !bar.Focused {
		t.Fatal("could not tab to the graph scroll control")
	}
	tt.Key(0, ui.KeyRight)
	settle()
	if a.historyGraphScroll.X <= 0 {
		t.Fatal("Right arrow did not reveal hidden lanes")
	}
	tt.Key(0, ui.KeyLeft)
	settle()
	if a.historyGraphScroll.X != 0 {
		t.Fatal("Left arrow did not return the graph to its start")
	}
	lane := func(subject string) (float32, float32) {
		t.Helper()
		row, ok := tt.AccessByRole(ui.RoleOption, subject)
		if !ok || row.Offscreen {
			t.Fatalf("missing visible row %q", subject)
		}
		return row.Bounds.X + 28, row.Bounds.Y + row.Bounds.H/2
	}
	x, y := lane(first.Subject)
	tt.ClickAt(x, y)
	settle()
	if a.selectedSHA != first.SHA {
		t.Fatal("clicking the graph lane should select the commit")
	}
	x, y = lane(first.Subject)
	beforeY := a.historyScroll.Y
	tt.Scroll(x, y, 75, 5)
	settle()
	pan := a.historyGraphScroll.X
	if pan <= 0 || a.historyScroll.Y != beforeY || a.selectedSHA != first.SHA {
		t.Fatalf("horizontal gesture over a commit graph: pan=%g, list Y=%g, selected=%q", pan, a.historyScroll.Y, a.selectedSHA)
	}
	tt.Scroll(x, y, 5, 75)
	settle()
	if a.historyScroll.Y <= beforeY || a.historyGraphScroll.X != pan {
		t.Fatalf("vertical gesture over a commit graph: pan=%g, list Y=%g", a.historyGraphScroll.X, a.historyScroll.Y)
	}
	tt.Scroll(x, y, 0, -75)
	settle()
	x, y = lane(a.t("noChangesShort"))
	tt.Scroll(x, y, -75, 0)
	settle()
	if a.historyGraphScroll.X != 0 {
		t.Fatalf("working tree graph row should pan back from the shared offset, got %g", a.historyGraphScroll.X)
	}
	tt.SetSize(3008, 640)
	settle()
	if a.historyGraphScroll.MaxX != 0 || a.selectedSHA != first.SHA {
		t.Fatal("widening should fit the graph and preserve the selected commit")
	}
}
