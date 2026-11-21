package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestSettingsChoiceUsesSharedSelectionAndKeepsOneValue(t *testing.T) {
	current, changes := "light", 0
	a := &app{locale: localeEnglish}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(460).Children(func() {
			a.choiceRow(c, "Theme", []choice.ToggleItem{{Value: "light", Label: "Light"}, {Value: "dark", Label: "Dark"}}, current, func(value string) { current = value; changes++ })
		})
	}, 460, 160)
	light, ok := tt.AccessByRole(ui.RoleToggleButton, "Light")
	if !ok || !light.Checked {
		t.Fatal("current setting has no checked toggle")
	}
	if light.Bounds.H != scaledFont(29) {
		t.Errorf("setting toggle height=%v want %v", light.Bounds.H, scaledFont(29))
	}
	p := tt.Image().RGBAAt(int(light.Bounds.X+3), int(light.Bounds.Y+light.Bounds.H/2))
	if got := ui.RGB(p.R, p.G, p.B); got != ui.Hex("#D6E1F7") {
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
