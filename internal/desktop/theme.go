package desktop

import "github.com/egoist/mygo/ui"

func appTheme(dark bool) *ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
	}
	t.FontSize = 13
	t.Spacing = 4
	t.Radius = 5
	if dark {
		t.Background = ui.Hex("#202c35")
		t.Surface = ui.Hex("#26343f")
		t.Border = ui.Hex("#3b4b57")
		t.Text = ui.Hex("#e5edf2")
		t.TextMuted = ui.Hex("#a4b5c1")
		t.Accent = ui.Hex("#267c78")
		t.AccentHover = ui.Hex("#318c87")
	} else {
		t.Background = ui.Hex("#fbfcfd")
		t.Surface = ui.Hex("#edf1f4")
		t.Border = ui.Hex("#d5dee5")
		t.Text = ui.Hex("#243744")
		t.TextMuted = ui.Hex("#5c7180")
		t.Accent = ui.Hex("#206d68")
		t.AccentHover = ui.Hex("#185c58")
	}
	return t
}
func graphColor(t *ui.Theme, lane int) ui.Color {
	if t.Dark {
		if lane%2 == 0 {
			return ui.Hex("#3987e5")
		}
		return ui.Hex("#d95926")
	}
	if lane%2 == 0 {
		return ui.Hex("#2a78d6")
	}
	return ui.Hex("#eb6834")
}
