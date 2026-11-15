package desktop

import (
	"testing"

	"supergit/internal/appicon"
)

func TestSyncDockIconOnlyTouchesTheSystemOnAppearanceChange(t *testing.T) {
	var calls int
	a := &app{dockIcon: appicon.New(func([]byte) error {
		calls++
		return nil
	})}

	a.syncDockIcon(false)
	a.syncDockIcon(false)
	if calls != 1 {
		t.Fatalf("same appearance called the setter %d times; want 1", calls)
	}

	a.syncDockIcon(true)
	a.syncDockIcon(true)
	if calls != 2 {
		t.Fatalf("switching to dark called the setter %d times; want 2", calls)
	}

	a.syncDockIcon(false)
	if calls != 3 {
		t.Fatalf("switching back called the setter %d times; want 3", calls)
	}
}

func TestSyncDockIconWorksWithoutASwitcher(t *testing.T) {
	// A nil Switcher must not panic; older app values and tests may lack one.
	(&app{}).syncDockIcon(true)
}
