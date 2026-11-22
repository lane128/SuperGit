package lobedesk

import (
	"image"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// An accessibility box can be centered even when its contents are drawn high.
// Measure the actual glyphs and decorative icon in the settled 1x frame.
func noticeInkBounds(t *testing.T, frame *image.RGBA, area image.Rectangle, ink ui.Color) image.Rectangle {
	t.Helper()
	var bounds image.Rectangle
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			c := frame.RGBAAt(x, y)
			// Allow partial coverage along antialiased edges.
			if c.A == 255 && math.Abs(float64(c.R)-float64(ink.R)) <= 40 &&
				math.Abs(float64(c.G)-float64(ink.G)) <= 40 && math.Abs(float64(c.B)-float64(ink.B)) <= 40 {
				bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if bounds.Empty() {
		t.Fatalf("no ink matching %+v in %v", ink, area)
	}
	return bounds
}

type noticeAlignmentSample struct {
	text, icon image.Rectangle
	close      ui.Rect
}

func renderNoticeAlignmentSample(t *testing.T, a *app, message string) noticeAlignmentSample {
	t.Helper()
	a.queue("success", message, "")
	var iconInk, textInk ui.Color
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		tk := theme.TokensOf(c)
		iconInk, textInk = tk.Success, tk.Text.Over(tk.BgElevated)
		a.toastHost(c, toastMain)
		a.flushNotice(c, toastMain)
	}, 480, 220)
	tt.Frame()
	tt.Advance(time.Second)
	text, found := tt.Find(message)
	close, buttonFound := tt.AccessByRole(ui.RoleButton, a.t("close"))
	card, cardFound := tt.AccessByRole(ui.RoleStatus, "")
	if !found || !buttonFound || !cardFound {
		t.Fatalf("toast contents missing: title=%t close=%t status=%t", found, buttonFound, cardFound)
	}
	if text.X+text.W > close.Bounds.X || text.Y < card.Bounds.Y || text.Y+text.H > card.Bounds.Y+card.Bounds.H {
		t.Fatalf("toast text overlaps close button or extends outside card: text=%+v close=%+v card=%+v", text, close.Bounds, card.Bounds)
	}
	textArea := image.Rect(int(text.X), int(text.Y), int(text.X+text.W), int(text.Y+text.H))
	iconArea := image.Rect(int(card.Bounds.X), int(card.Bounds.Y), int(text.X), int(card.Bounds.Y+card.Bounds.H))
	return noticeAlignmentSample{
		text:  noticeInkBounds(t, tt.Image(), textArea, textInk),
		icon:  noticeInkBounds(t, tt.Image(), iconArea, iconInk),
		close: close.Bounds,
	}
}

func TestNoticeSingleLineAlignmentAndWrapping(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			a := &app{locale: language, themeMode: "light"}
			message := a.t("copied")
			short := renderNoticeAlignmentSample(t, a, message)
			t.Run("single", func(t *testing.T) {
				textY := float64(short.text.Min.Y+short.text.Max.Y) / 2
				iconY := float64(short.icon.Min.Y+short.icon.Max.Y) / 2
				closeY := float64(short.close.Y + short.close.H/2)
				// Glyph ascenders and descenders allow a little more variation
				// than the symmetric circle icon and button hit target.
				if math.Abs(iconY-closeY) > 1 || max(textY, iconY, closeY)-min(textY, iconY, closeY) > 2 {
					t.Errorf("single-line toast centers differ: text=%.1f icon=%.1f close=%.1f DIP", textY, iconY, closeY)
				}
			})
			t.Run("wrapped", func(t *testing.T) {
				word := message
				if language == localeEnglish {
					word += " "
				}
				long := renderNoticeAlignmentSample(t, a, strings.TrimSpace(strings.Repeat(word, 24)))
				if long.text.Dy() <= 2*short.text.Dy() {
					t.Fatalf("long notice did not paint multiple lines: short ink=%v long ink=%v", short.text, long.text)
				}
			})
		})
	}
}
