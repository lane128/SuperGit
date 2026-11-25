package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestHistoryVisualChoicesFitMinimumSettingsWidth(t *testing.T) {
	old := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(old) })
	setUIFontScale(fontScaleLarge)
	for _, language := range []locale{localeEnglish, localeChinese} {
		a := &app{locale: language}
		view := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			ui.Column(c).FillWidth().Children(func() { a.historyColumnsCard(c); a.graphCard(c) })
		}, 352, 1000)
		for _, n := range view.AccessAll() {
			if n.Role == ui.RoleToggleButton && (n.Bounds.X < 0 || n.Bounds.X+n.Bounds.W > 352) {
				t.Errorf("%s option %s exceeds narrow settings: %+v", language, n.Name, n.Bounds)
			}
		}
	}
}
