package lobedesk

import (
	"testing"
)

// The column decides how many capsules a row keeps. At the smallest window
// three capsules plus the graph leave the subject almost nothing, so the rest
// fold into +N.
func TestRefSlotsFollowTheColumnWidth(t *testing.T) {
	for _, test := range []struct {
		width float32
		slots int
	}{
		{0, 3},
		{280, 1},
		{309, 1},
		{339, 1},
		{360, 2},
		{479, 2},
		{508, 3},
		{900, 3},
	} {
		if got := refSlotsForWidth(test.width); got != test.slots {
			t.Errorf("a %v column shows %d capsules; want %d", test.width, got, test.slots)
		}
	}
}

// A narrow row is still recognized so the row can use its compact title gap.
// The capsule itself has no fixed point cap; its budget is what limits it.
func TestANarrowRowIsRecognized(t *testing.T) {
	if !rowRefsNarrow(309) {
		t.Error("a 309-point row is not treated as narrow")
	}
	if rowRefsNarrow(508) {
		t.Error("a 508-point row is treated as narrow")
	}
}
