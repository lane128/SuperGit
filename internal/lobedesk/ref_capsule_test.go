package lobedesk

import "testing"

// A capsule's width cap has to grow with the interface text size. The label is
// elided to a fixed number of runes before it reaches the pill, so a cap that
// stays at its standard size cuts an already-elided name once more: at the
// large preset "origin/r…onate-links" became "origin/r…onate-lin…" on a row.
// Scaling the cap keeps exactly one ellipsis in the label.
func TestCapsuleWidthsFollowTheTextScale(t *testing.T) {
	defer setUIFontScale(fontScaleStandard)

	setUIFontScale(fontScaleStandard)
	if got := refCapsuleWidth(refCapsuleRowWidth); got != refCapsuleRowWidth {
		t.Fatalf("at standard scale the row cap is %v; want %v", got, refCapsuleRowWidth)
	}

	for _, test := range []struct {
		name string
		base float32
	}{
		{"row", refCapsuleRowWidth},
		{"details", refCapsuleDetailWidth},
	} {
		setUIFontScale(fontScaleLarge)
		if got, want := refCapsuleWidth(test.base), test.base*fontScaleLarge; got != want {
			t.Errorf("%s cap at the large preset is %v; want %v", test.name, got, want)
		}
		setUIFontScale(fontScaleSmall)
		if got, want := refCapsuleWidth(test.base), test.base*fontScaleSmall; got != want {
			t.Errorf("%s cap at the small preset is %v; want %v", test.name, got, want)
		}
	}
}
