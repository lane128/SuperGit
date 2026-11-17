package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
)

func TestAppearanceHelpShowsCompleteTextAtMinimumWidth(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	setUIFontScale(1.15)
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			a := &app{locale: language, themeMode: "light"}
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				ui.Column(c).Fill().Children(func() { a.settingsPage(c) })
			}, settingsWindowMinW, settingsWindowMinH)
			text := a.t("appearanceHelp")
			shown, ok := tt.Find(text)
			if !ok {
				t.Fatal("appearance explanation missing")
			}
			// Measure the complete localized explanation at the actual content
			// width. Finding its accessibility label alone misses ellipsis.
			reference := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				ui.Column(c).Width(shown.W).Children(func() {
					display.Text(c, text, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11)}).Width(shown.W)
				})
			}, int(shown.W)+1, 600)
			full, ok := reference.Find(text)
			if !ok || shown.H+1 < full.H {
				t.Fatalf("appearance help truncated: shown=%+v complete=%+v", shown, full)
			}
		})
	}
}
