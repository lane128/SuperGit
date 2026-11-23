package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestSettingsChoicesStayOnOneLineWhenTheyFit(t *testing.T) {
	before := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(before) })
	for _, scale := range []float32{1, 1.15} {
		setUIFontScale(scale)
		for _, current := range []string{"system", "light", "dark"} {
			a := &app{locale: localeEnglish, themeMode: "dark"}
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				ui.Column(c).Width(606).Children(func() {
					a.choiceRow(c, "Theme", []choice.ToggleItem{{Value: "system", Label: "System"}, {Value: "light", Label: "Light"}, {Value: "dark", Label: "Dark"}}, current, func(string) {})
				})
			}, 606, 160)
			first, ok := tt.AccessByRole(ui.RoleToggleButton, "System")
			if !ok {
				t.Fatal("System choice missing")
			}
			for _, label := range []string{"Light", "Dark"} {
				node, ok := tt.AccessByRole(ui.RoleToggleButton, label)
				if !ok || node.Bounds.Y != first.Bounds.Y {
					t.Errorf("scale %g selection %s: %s wrapped despite sufficient space: %+v", scale, current, label, node.Bounds)
				}
			}
		}
	}
}

func TestSettingsChoiceUsesSharedSelectionAndKeepsOneValue(t *testing.T) {
	current, changes := "light", 0
	a := &app{locale: localeEnglish}
	var cardGround ui.Color
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		cardGround = theme.TokensOf(c).BgElevated
		ui.Column(c).Width(460).Background(cardGround).Children(func() {
			a.choiceRow(c, "Theme", []choice.ToggleItem{{Value: "light", Label: "Light"}, {Value: "dark", Label: "Dark"}}, current, func(value string) { current = value; changes++ })
		})
	}, 460, 160)
	darkEnd, ok := tt.AccessByRole(ui.RoleToggleButton, "Dark")
	if !ok {
		t.Fatal("Dark choice is missing")
	}
	// The segmented track ends with its options, leaving the rest of the card clear.
	trackTail := tt.Image().RGBAAt(int(darkEnd.Bounds.X+darkEnd.Bounds.W+12), int(darkEnd.Bounds.Y+darkEnd.Bounds.H/2))
	if got := ui.RGB(trackTail.R, trackTail.G, trackTail.B); got != cardGround {
		t.Errorf("segmented background extends past its options: %v", got)
	}
	light, ok := tt.AccessByRole(ui.RoleToggleButton, "Light")
	if !ok || !light.Checked {
		t.Fatal("current setting has no checked toggle")
	}
	if light.Bounds.H != scaledFont(29) {
		t.Errorf("setting toggle height=%v want %v", light.Bounds.H, scaledFont(29))
	}
	p := tt.Image().RGBAAt(int(light.Bounds.X+3), int(light.Bounds.Y+light.Bounds.H/2))
	if got := ui.RGB(p.R, p.G, p.B); got != ui.Hex("#EDF4FF") {
		t.Errorf("selected setting surface=%v want shared blue", got)
	}
	tt.ClickAt(light.Bounds.X+light.Bounds.W/2, light.Bounds.Y+light.Bounds.H/2)
	for range 3 {
		tt.Frame()
	}
	if current != "light" || changes != 0 {
		t.Fatal("clicking current setting cleared or saved it")
	}
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	for range 3 {
		tt.Frame()
	}
	dark, ok := tt.AccessByRole(ui.RoleToggleButton, "Dark")
	light, _ = tt.AccessByRole(ui.RoleToggleButton, "Light")
	if !ok || !dark.Checked || light.Checked || current != "dark" || changes != 1 {
		t.Fatalf("keyboard choice did not select exactly once: value=%s changes=%d", current, changes)
	}
}
