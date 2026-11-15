package lobedesk

import (
	"math"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

// The horizontal scroll width of a patch is estimated from its longest line.
// The estimate must follow the reader's text size: a fixed advance that fits
// the standard preset leaves the large preset's longest lines clipped at the
// row's right edge, where scrolling cannot reach them.
func TestDiffWidthFollowsTheFontScale(t *testing.T) {
	diff := gitrepo.Diff{Lines: []gitrepo.DiffLine{
		{Kind: "context", Text: strings.Repeat("x", 500)},
	}}
	defer setUIFontScale(uiFontScaleValue())

	setUIFontScale(fontScaleStandard)
	standard := diffWidth(diff)
	setUIFontScale(fontScaleLarge)
	large := diffWidth(diff)

	want := (standard - 110) * fontScaleLarge
	if got := large - 110; math.Abs(float64(got-want)) > 1 {
		t.Errorf("large preset leaves %.1f for the code; want %.1f (1.15× the standard %.1f)", got, want, standard-110)
	}
	// The estimate has to cover the glyphs the large preset actually draws.
	if least := 500*scaledFont(11.5)*0.60 + 110; large < least {
		t.Errorf("large preset width %.1f is narrower than the text it has to hold (%.1f)", large, least)
	}
}
