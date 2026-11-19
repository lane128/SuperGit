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

// A narrow row also caps the capsule earlier and drops the separate "HEAD"
// word: the branch capsule wears the check instead, which is where a long
// name would otherwise push the subject out.
func TestANarrowRowCapsItsRefs(t *testing.T) {
	if !rowRefsNarrow(309) {
		t.Error("a 309-point row is not treated as narrow")
	}
	if rowRefsNarrow(508) {
		t.Error("a 508-point row is treated as narrow")
	}
	if got := rowCapsuleWidthForList(309); got > 80 {
		t.Errorf("a 309-point row caps a capsule at %v; want about 76", got)
	}
	if got, want := rowCapsuleWidthForList(1200), float32(refCapsuleRowWidth); got != want {
		t.Errorf("a wide row caps a capsule at %v; want %v", got, want)
	}
}
