package lobedesk

import "testing"

// The size the window opens at is the default unless the automated run asked
// for another one. It must never trust the variable blindly: a typo or a size
// below the window's minimum falls back to the default rather than opening a
// window the reader could not produce by dragging.
func TestParseWindowSize(t *testing.T) {
	cases := []struct {
		raw  string
		w, h int
		ok   bool
	}{
		{"980x640", 980, 640, true},
		{"900x700", 900, 700, true},
		{"1024X768", 1024, 768, true},
		{" 1000 x 800 ", 1000, 800, true},
		{"", 0, 0, false},
		{"1000", 0, 0, false},
		{"1000x", 0, 0, false},
		{"axb", 0, 0, false},
		{"899x700", 0, 0, false},
		{"900x639", 0, 0, false},
		{"1320x840x900", 0, 0, false},
	}
	for _, c := range cases {
		w, h, ok := parseWindowSize(c.raw)
		if w != c.w || h != c.h || ok != c.ok {
			t.Errorf("parseWindowSize(%q) = %d,%d,%v; want %d,%d,%v", c.raw, w, h, ok, c.w, c.h, c.ok)
		}
	}
}

// An unset variable keeps the window at the size it has always opened at.
func TestWindowSizeDefaultsWhenUnset(t *testing.T) {
	t.Setenv(windowSizeEnv, "")
	if w, h := windowSize(); w != defaultWindowWidth || h != defaultWindowHeight {
		t.Fatalf("unset = %dx%d; want %dx%d", w, h, defaultWindowWidth, defaultWindowHeight)
	}
	t.Setenv(windowSizeEnv, "980x640")
	if w, h := windowSize(); w != 980 || h != 640 {
		t.Fatalf("set = %dx%d; want 980x640", w, h)
	}
	t.Setenv(windowSizeEnv, "not a size")
	if w, h := windowSize(); w != defaultWindowWidth || h != defaultWindowHeight {
		t.Fatalf("malformed = %dx%d; want the default", w, h)
	}
}
