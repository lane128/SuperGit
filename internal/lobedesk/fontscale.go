package lobedesk

import (
	"math"
	"sync/atomic"
)

// The interface text scale. MyGo lets a theme set one base font size, but this
// app gives almost every label an explicit size, so a theme-level change would
// only move the few labels that fall back to it. Scaling at each call site
// instead keeps every size in proportion: the reader picks a preset, and every
// text and icon the app draws follows, from commit rows to diffs to the
// settings page itself.
//
// The value is read while drawing, which runs on the UI thread; it is stored in
// an atomic so a settings test or a future background reload can write it
// without racing the frame.
const (
	fontScaleSmall    float32 = 0.9
	fontScaleStandard float32 = 1
	fontScaleLarge    float32 = 1.15
)

var uiFontScale atomic.Uint32

func init() {
	uiFontScale.Store(math.Float32bits(fontScaleStandard))
}

// setUIFontScale applies a preset. A value outside the offered range falls back
// to standard rather than letting a hand-edited settings file shrink the app
// into unreadability.
func setUIFontScale(scale float32) {
	if scale < fontScaleSmall || scale > fontScaleLarge {
		scale = fontScaleStandard
	}
	uiFontScale.Store(math.Float32bits(scale))
}

// scaledFont is the size a text or icon asks for, in the reader's scale.
func scaledFont(size float32) float32 {
	return size * math.Float32frombits(uiFontScale.Load())
}

// uiFontScaleValue is the current preset, for the settings file and page.
func uiFontScaleValue() float32 {
	return math.Float32frombits(uiFontScale.Load())
}

// fontScaleKey names a preset for the settings page's segmented control.
func fontScaleKey(scale float32) string {
	switch scale {
	case fontScaleSmall:
		return "small"
	case fontScaleLarge:
		return "large"
	}
	return "standard"
}

// fontScaleOf resolves a preset key; an unknown key is standard.
func fontScaleOf(key string) float32 {
	switch key {
	case "small":
		return fontScaleSmall
	case "large":
		return fontScaleLarge
	}
	return fontScaleStandard
}

// fontScaleSetting keeps the standard scale, the fresh-install value, out of
// the settings file.
func fontScaleSetting(scale float32) *float32 {
	if scale == fontScaleStandard {
		return nil
	}
	return &scale
}
