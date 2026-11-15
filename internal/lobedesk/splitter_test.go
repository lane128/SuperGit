package lobedesk

import (
	"math"
	"testing"
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
		{"too narrow for both", 0.6, 500, 0.5},
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
