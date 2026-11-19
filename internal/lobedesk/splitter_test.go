package lobedesk

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// A divider may not squeeze either pane below its minimum, and a window too
// narrow to hold both minimums still needs a defined answer.
func TestClampSplitKeepsBothPanesUsable(t *testing.T) {
	room := float32(1000 - splitterWidth)
	cases := []struct {
		name  string
		share float32
		row   float32
		want  float32
	}{
		{"inside the range", 0.6, 1000, 0.6},
		{"too small for the left pane", 0.1, 1000, 240 / room},
		{"too large for the right pane", 0.95, 1000, 1 - 300/room},
		// Neither pane fits: each keeps its minimum's share of the shortfall.
		{"too narrow for both", 0.6, 500, 240.0 / (240 + 300)},
		{"no width yet", 0.3, 0, 0.3},
	}
	for _, tc := range cases {
		got := clampSplit(tc.share, tc.row, 240, 300)
		if math.Abs(float64(got-tc.want)) > 0.0001 {
			t.Fatalf("%s: clamp(%v, %v) = %v, want %v", tc.name, tc.share, tc.row, got, tc.want)
		}
	}
}

// A saved share is what the window opens with; the page's default stands in
// only when no share was ever saved.
func TestSplitFallsBackToTheDefault(t *testing.T) {
	a := &app{}
	if got := a.historyShare(); got != historySplitDefault {
		t.Fatalf("unset share = %v, want %v", got, historySplitDefault)
	}
	a.historySplit.share = 0.45
	if got := a.historyShare(); got != 0.45 {
		t.Fatalf("stored share = %v, want 0.45", got)
	}
}

// A page whose opening width is a number of points turns it into a share of
// the room it has; without a width yet the page declares no share.
func TestShareForWidth(t *testing.T) {
	if got := shareForWidth(1000, 300); math.Abs(float64(got-0.3)) > 0.0001 {
		t.Fatalf("shareForWidth(1000, 300) = %v, want 0.3", got)
	}
	if got := shareForWidth(0, 300); got != 0 {
		t.Fatalf("shareForWidth(0, 300) = %v, want 0", got)
	}
}

func TestClampedHistoryDividerDoesNotReplaceWidePreference(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []float32
	}{
		{"push past the limit", []float32{50}},
		{"drag away and return to the limit", []float32{-25, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{page: "history"}
			a.historySplit.share = .5
			var listWidth float32
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				width, _ := c.Size()
				// The default sidebar and its divider occupy 191 points.
				a.historySplit.rowWidth = width - 191
				listWidth = a.historyShare() * (a.historySplit.rowWidth - splitterWidth)
				ui.Row(c).FillWidth().FillHeight().Children(func() {
					ui.Box(c).Width(191).Shrink(0)
					ui.Box(c).Width(listWidth).Shrink(0)
					a.splitHandle(c, &a.historySplit, historySplitDefault, historySplitMinList, a.historyDetailMinWidth(), "Resize history")
				})
			}, 1320, 640)
			tt.Frame()
			wide := listWidth
			tt.SetSize(980, 640)
			tt.Frame()
			const y = 300
			narrow := listWidth
			x := 191 + narrow + splitterWidth/2
			tt.Press(x, y)
			for _, dx := range tc.steps {
				tt.Move(x+dx, y)
				if want := narrow + min(dx, 0); math.Abs(float64(listWidth-want)) > .5 {
					t.Fatalf("during drag: got %.1f points, want %.1f", listWidth, want)
				}
			}
			// Finish through the release handler's state transition without
			// allowing this test to write to the user's preference file.
			if a.historySplit.finishDrag(historySplitDefault, historySplitMinList, a.historyDetailMinWidth()) || a.historySplit.share != .5 {
				t.Fatalf("drag replaced the preferred share: %.3f", a.historySplit.share)
			}
			tt.Release(x+tc.steps[len(tc.steps)-1], y)
			tt.SetSize(1320, 640)
			tt.Frame()
			if math.Abs(float64(listWidth-wide)) > .5 {
				t.Fatalf("widening restored %.1f points, want %.1f", listWidth, wide)
			}
		})
	}
}

func TestDividerShowsResizeCursorOnlyWhenItCanMove(t *testing.T) {
	const left, right float32 = 180, 300
	s := splitDrag{share: .8}
	var boundary float32
	view := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		s.rowWidth, _ = c.Size()
		boundary = s.fraction(.5, left, right) * (s.rowWidth - splitterWidth)
		ui.Row(c).Fill().Children(func() {
			ui.Box(c).Width(boundary).Shrink(0)
			(&app{}).splitHandle(c, &s, .5, left, right, "Resize Files")
		})
	}, 487, 200)
	view.Move(boundary+splitterWidth/2, 100)
	if got := view.Cursor(); got != ui.CursorDefault {
		t.Errorf("fixed divider displayed resize cursor %v", got)
	}
	view.SetSize(527, 200)
	view.Move(boundary+splitterWidth/2, 100)
	if got := view.Cursor(); got != ui.CursorResizeEW {
		t.Errorf("divider can move inward at its right limit, got cursor %v", got)
	}
}

func TestFilesDividerStaysAtLimitUntilPointerReturns(t *testing.T) {
	const rightLimit = 693 // 1000 point row minus the 7 point divider and 300 point Diff minimum.
	for _, tc := range []struct {
		page      string
		leftLimit float32
		render    func(*app, *ui.Context)
	}{
		{"history", 180, func(a *app, c *ui.Context) {
			width, _ := c.Size()
			a.historyDetailsPane(c, 1, width)
		}},
	} {
		t.Run(tc.page, func(t *testing.T) {
			a := &app{page: tc.page, themeMode: "light"}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				tc.render(a, c)
			}, 1000, 600)
			tt.Frame()
			const y = 300
			tt.Press(a.historyFilesWidth()+splitterWidth/2, y)
			// Stay in the same gesture. The existing restart test covers saving on release.
			for _, step := range []struct {
				pointer, want float32
			}{
				{850, rightLimit},
				{750, rightLimit},
				{650, 646.5},
				{50, tc.leftLimit},
				{100, tc.leftLimit},
				{350, 346.5},
			} {
				tt.Move(step.pointer, y)
				if got := a.historyFilesWidth(); math.Abs(float64(got-step.want)) > 0.5 {
					t.Fatalf("pointer at %.1f: Files is %.1f points wide, want %.1f", step.pointer, got, step.want)
				}
			}
		})
	}
}
