// Package appicon holds the finished Dock artwork and the small state machine
// that keeps the Dock in step with the app's appearance.
package appicon

import _ "embed"

// The PNGs are the finished icon, not the raw drawing: macOS draws a Dock icon
// exactly as given, so a full-bleed square would sit outside the icon grid that
// every other app follows. tools/icongen scales the drawing into the 824px
// grid, masks it with the system's squircle and adds the shadow; "make icons"
// regenerates them from the source iconsets.
//
//go:embed icons/app-light.png
var Light []byte

//go:embed icons/app-dark.png
var Dark []byte

// Setter hands an icon to the system. It is a function rather than an interface
// so this package needs no knowledge of the UI toolkit, which keeps it testable
// with a plain fake.
type Setter func(png []byte) error

// Switcher remembers which appearance the Dock is showing, so a caller that
// runs every frame only touches the system when the appearance changes.
type Switcher struct {
	set     Setter
	applied bool
	dark    bool
}

// New returns a Switcher that hands icons to set. A nil setter makes Apply a
// no-op, which lets the callers stay free of nil checks.
func New(set Setter) *Switcher {
	return &Switcher{set: set}
}

// Apply makes the Dock match dark. It is safe on a nil receiver. A failure still
// records the attempted appearance: the artwork is decoration, and retrying
// every frame would not make it appear; the next appearance change tries again.
func (s *Switcher) Apply(dark bool) error {
	if s == nil || s.set == nil {
		return nil
	}
	if s.applied && s.dark == dark {
		return nil
	}
	png := Light
	if dark {
		png = Dark
	}
	err := s.set(png)
	s.applied = true
	s.dark = dark
	return err
}
