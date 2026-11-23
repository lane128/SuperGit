package lobedesk

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestSettingsWindowOptionsCompactGeometry(t *testing.T) {
	a := &app{}
	options := a.settingsWindowOptions()
	for _, size := range []struct {
		name       string
		width      int
		height     int
		wantWidth  int
		wantHeight int
	}{
		{"default", options.Width, options.Height, 848, 680},
		{"minimum", options.MinWidth, options.MinHeight, 560, 480},
		{"maximum", options.MaxWidth, options.MaxHeight, 848, 860},
	} {
		t.Run(size.name, func(t *testing.T) {
			if size.width != size.wantWidth || size.height != size.wantHeight {
				t.Errorf("size = %dx%d; want %dx%d", size.width, size.height, size.wantWidth, size.wantHeight)
			}
		})
	}
	if !options.DisableMaximize || !options.DisableFullScreen {
		t.Errorf("settings must disable maximize and full screen: maximize=%v fullScreen=%v", options.DisableMaximize, options.DisableFullScreen)
	}
	if options.StateKey != "settings-compact-v1" {
		t.Errorf("StateKey = %q; want settings-compact-v1 to avoid restoring legacy geometry", options.StateKey)
	}
}

func TestSettingsWindowOptionsPreservePresentation(t *testing.T) {
	a := &app{}
	options := a.settingsWindowOptions()
	if options.Title != a.settingsWindowTitle() {
		t.Errorf("Title = %q; want %q", options.Title, a.settingsWindowTitle())
	}
	if options.TitleBarStyle != mygo.TitleBarHidden {
		t.Errorf("TitleBarStyle = %v; want hidden", options.TitleBarStyle)
	}
	if options.TrafficLightPosition == nil || *options.TrafficLightPosition != (mygo.Point{X: 12, Y: 12}) {
		t.Errorf("TrafficLightPosition = %v; want (12, 12)", options.TrafficLightPosition)
	}
	if options.BackgroundColor != "light-dark(#f7f8fa, #09090b)" {
		t.Errorf("BackgroundColor = %q", options.BackgroundColor)
	}
	if options.Content == nil {
		t.Fatal("settings content must be supplied")
	}
	if options.DisableResize {
		t.Error("settings must remain resizable within the allowed bounds")
	}
}
