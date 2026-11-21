package lobedesk

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestUnicodeDiffRowDoesNotEllipsizeTheTail(t *testing.T) {
	defer setUIFontScale(uiFontScaleValue())
	for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
		setUIFontScale(scale)
		text := "working long patch line 3999" + strings.Repeat(" 中文宽行ＡＢＣ", 16) + " wide-tail-3999"
		line := gitrepo.DiffLine{Kind: "context", Text: text}
		width := (&app{}).diffWidth(gitrepo.Diff{Lines: []gitrepo.DiffLine{line}})
		var measured float32
		a := &app{locale: localeEnglish}
		tt := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			measured, _ = c.MeasureText(0, ui.Span{Text: text, Font: a.patchFontFamily(), Size: scaledFont(diffCodeSize)})
			a.diffLineFor(c, line, width, true, 0)
		}, int(math.Ceil(float64(width))), 100)
		bounds, ok := tt.Find(text)
		if !ok {
			t.Fatal("code text missing")
		}
		if bounds.W < measured {
			t.Errorf("scale %.2f code gets %.1f DIP but needs %.1f; tail is ellipsized", scale, bounds.W, measured)
		}
	}
}

func TestUnicodeDiffWidthCoversTheShapedCodeText(t *testing.T) {
	defer setUIFontScale(uiFontScaleValue())
	defer setCodeFontKey(codeFontKeyValue())
	for _, opt := range codeFontOptions {
		setCodeFontKey(opt.Key)
		for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
			setUIFontScale(scale)
			for _, text := range []string{
				strings.Repeat("中文差异", 100) + " tail-marker",
				strings.Repeat("ＡＢＣ１２３", 70) + " tail-marker",
				strings.Repeat("mixed中文\t", 70) + " tail-marker",
			} {
				var measured float32
				ui.NewTester(func(c *ui.Context) {
					measured, _ = c.MeasureText(0, ui.Span{Text: strings.ReplaceAll(text, "\t", "    "), Font: codeFontFamily(), Size: scaledFont(diffCodeSize)})
				}, 560, 100)
				got := (&app{}).diffWidth(gitrepo.Diff{Lines: []gitrepo.DiffLine{{Kind: "context", Text: text}}})
				if got < measured+diffCodeReserve {
					t.Errorf("%s scale %.2f scroll extent %.1f cuts shaped Unicode code width %.1f plus gutters", opt.Key, scale, got, measured)
				}
				// At the right scroll limit, excess blank extent must not push
				// the actual tail entirely past the minimum details viewport.
				if got-measured >= historyDiffPaneMin {
					t.Errorf("%s scale %.2f excess scroll extent %.1f hides the Unicode tail at the right limit", opt.Key, scale, got-measured)
				}
			}
		}
	}
}

func TestExplicitCodeFontResizesThePatchIndependently(t *testing.T) {
	defer setUIFontScale(uiFontScaleValue())
	a := &app{locale: localeEnglish, codeFontSize: 19}
	text := strings.Repeat("mixed 中文ＡＢＣ ", 25) + "tail"
	line := gitrepo.DiffLine{Kind: "context", Text: text}
	diff := gitrepo.Diff{Lines: []gitrepo.DiffLine{line}}
	setUIFontScale(fontScaleStandard)
	width := a.diffWidth(diff)
	setUIFontScale(fontScaleLarge)
	if a.diffWidth(diff) != width || a.patchFontSize() != 19 || a.patchRowHeight() <= diffRowH {
		t.Fatal("the fixed code size should control the patch geometry without changing with the interface font")
	}
	view := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		a.diffLineFor(c, line, width, true, 0)
	}, int(math.Ceil(float64(width))), 100)
	if bounds, ok := view.Find(text); !ok || bounds.H < 19 {
		t.Fatalf("the large patch text is missing or vertically clipped: %+v", bounds)
	}
	line.Text = strings.Repeat("x", 2000) + " tail"
	diff.Lines[0] = line
	if width := a.diffWidth(diff); width < float32(len(line.Text))*19*0.60+diffCodeReserve {
		t.Fatalf("horizontal scroll cannot reach the code tail at 19 pt; width=%g", width)
	}
}

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
	standard := (&app{}).diffWidth(diff)
	setUIFontScale(fontScaleLarge)
	large := (&app{}).diffWidth(diff)

	want := (standard - diffCodeReserve) * fontScaleLarge
	if got := large - diffCodeReserve; math.Abs(float64(got-want)) > 1 {
		t.Errorf("large preset leaves %.1f for the code; want %.1f (1.15× the standard %.1f)", got, want, standard-diffCodeReserve)
	}
	// The estimate has to cover the glyphs the large preset actually draws.
	if least := 500*scaledFont(11.5)*0.60 + diffCodeReserve; large < least {
		t.Errorf("large preset width %.1f is narrower than the text it has to hold (%.1f)", large, least)
	}
}
