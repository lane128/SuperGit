package lobedesk

import "testing"

func TestDetailTrackBatchedPressKeepsOriginalGrabPoint(t *testing.T) {
	for _, test := range []struct {
		name                      string
		x, dx, left, length, want float32
	}{
		{"separate frames", 10, 0, 0, 100, 10},
		{"move before first frame", 30, 20, 0, 100, 10},
		{"cross thumb edge before first frame", 120, 110, 0, 100, 10},
		{"move left before first frame", 25, -35, 50, 100, 10},
		{"click outside thumb", 190, 0, 50, 100, 50},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := detailTrackAnchor(test.x, test.dx, test.left, test.length); got != test.want {
				t.Fatalf("grab offset=%g, want original down offset %g", got, test.want)
			}
		})
	}
}
