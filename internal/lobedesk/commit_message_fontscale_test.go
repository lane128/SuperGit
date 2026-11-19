package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestMessageTargetMenuFollowsReaderFontScale(t *testing.T) {
	oldScale := uiFontScaleValue()
	t.Cleanup(func() { setUIFontScale(oldScale) })
	raw := "https://example.invalid/" + strings.Repeat("readable/", 25) + "?q=exact#tail"
	for _, language := range []locale{localeEnglish, localeChinese} {
		for _, mode := range []string{"light", "dark"} {
			t.Run(string(language)+"/"+mode, func(t *testing.T) {
				var heights []float32
				for _, scale := range []float32{fontScaleStandard, fontScaleLarge} {
					setUIFontScale(scale)
					a := &app{locale: language, themeMode: mode}
					commit := gitrepo.Commit{Subject: raw}
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						if c.Theme().Dark != (mode == "dark") {
							t.Fatal("theme mismatch")
						}
						ui.Column(c).Fill().Children(func() { a.commitInfoContent(c, commit, theme.TokensOf(c)) })
					}, 480, 360)
					r, ok := tt.Find(raw)
					if !ok {
						t.Fatal("message absent")
					}
					tt.RightClickAt(r.X+5, r.Y+5)
					item, ok := tt.AccessByRole(ui.RoleMenuItem, raw)
					if !ok || item.Offscreen || item.Bounds.X < 0 || item.Bounds.X+item.Bounds.W > 480 || item.Bounds.Y < 0 || item.Bounds.Y+item.Bounds.H > 360 {
						t.Fatalf("scaled menu target outside viewport: %+v", item)
					}
					heights = append(heights, item.Bounds.H)
					tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
					if got := tt.OpenedURLs(); len(got) != 1 || got[0] != raw {
						t.Fatal("scaled menu changed exact target")
					}
				}
				if heights[1] <= heights[0] {
					t.Fatalf("reader large-font setting did not enlarge wrapped menu text: standard=%g large=%g", heights[0], heights[1])
				}
			})
		}
	}
}
