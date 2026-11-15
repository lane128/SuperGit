package appicon

import (
	"bytes"
	"errors"
	"image/png"
	"testing"
)

func TestApplyOnlyTouchesTheSystemOnAppearanceChange(t *testing.T) {
	var calls int
	var got [][]byte
	s := New(func(png []byte) error {
		calls++
		got = append(got, png)
		return nil
	})

	for range 3 {
		if err := s.Apply(false); err != nil {
			t.Fatalf("Apply(false): %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("same appearance called the setter %d times; want 1", calls)
	}
	if !bytes.Equal(got[0], Light) {
		t.Errorf("light appearance handed the setter the wrong artwork")
	}

	if err := s.Apply(true); err != nil {
		t.Fatalf("Apply(true): %v", err)
	}
	if calls != 2 {
		t.Fatalf("switching to dark called the setter %d times; want 2", calls)
	}
	if !bytes.Equal(got[1], Dark) {
		t.Errorf("dark appearance handed the setter the wrong artwork")
	}

	if err := s.Apply(false); err != nil {
		t.Fatalf("Apply(false): %v", err)
	}
	if calls != 3 {
		t.Fatalf("switching back called the setter %d times; want 3", calls)
	}
	if !bytes.Equal(got[2], Light) {
		t.Errorf("switching back handed the setter the wrong artwork")
	}
}

func TestApplyRecordsFailuresSoTheSameAppearanceIsNotRetried(t *testing.T) {
	boom := errors.New("dock rejected the icon")
	var calls int
	s := New(func([]byte) error {
		calls++
		return boom
	})

	if err := s.Apply(true); !errors.Is(err, boom) {
		t.Fatalf("Apply(true) = %v; want %v", err, boom)
	}
	if calls != 1 {
		t.Fatalf("first Apply called the setter %d times; want 1", calls)
	}

	if err := s.Apply(true); err != nil {
		t.Fatalf("repeated Apply(true) = %v; want nil (no retry)", err)
	}
	if calls != 1 {
		t.Fatalf("repeated Apply(true) called the setter %d times; want 1", calls)
	}

	if err := s.Apply(false); !errors.Is(err, boom) {
		t.Fatalf("Apply(false) = %v; want %v", err, boom)
	}
	if calls != 2 {
		t.Fatalf("appearance change called the setter %d times; want 2", calls)
	}
}

func TestApplyIsNilSafe(t *testing.T) {
	var nilSwitcher *Switcher
	if err := nilSwitcher.Apply(true); err != nil {
		t.Fatalf("nil Switcher Apply = %v; want nil", err)
	}
	if err := New(nil).Apply(false); err != nil {
		t.Fatalf("Switcher with nil setter Apply = %v; want nil", err)
	}
}

// macOS draws a Dock icon exactly as given, so the embedded PNGs have to be
// the finished icon: artwork inside the system's rounded square, transparent
// outside it. A full-bleed drawing here shows up as a hard-cornered square
// that sits outside every other app's icon.
func TestIconsCarryTheRoundedMask(t *testing.T) {
	if bytes.Equal(Light, Dark) {
		t.Fatal("light and dark icons are identical; a theme switch would show no change")
	}
	for _, icon := range []struct {
		name string
		data []byte
	}{
		{"light", Light},
		{"dark", Dark},
	} {
		img, err := png.Decode(bytes.NewReader(icon.data))
		if err != nil {
			t.Fatalf("%s icon: %v", icon.name, err)
		}
		bounds := img.Bounds()
		if bounds.Dx() != 1024 || bounds.Dy() != 1024 {
			t.Errorf("%s icon is %dx%d; want 1024x1024", icon.name, bounds.Dx(), bounds.Dy())
		}
		opaque := func(x, y int) uint8 {
			_, _, _, a := img.At(x, y).RGBA()
			return uint8(a >> 8)
		}
		if a := opaque(bounds.Dx()/2, bounds.Dy()/2); a != 255 {
			t.Errorf("%s icon center is %d%% opaque; want the artwork", icon.name, a*100/255)
		}
		for _, corner := range [][2]int{{0, 0}, {bounds.Dx() - 1, 0}, {0, bounds.Dy() - 1}, {bounds.Dx() - 1, bounds.Dy() - 1}} {
			if a := opaque(corner[0], corner[1]); a != 0 {
				t.Errorf("%s icon corner %v is %d%% opaque; want it cut by the mask", icon.name, corner, a*100/255)
			}
		}
	}
}
