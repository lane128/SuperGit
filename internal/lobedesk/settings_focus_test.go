package lobedesk

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// isolateSettings points the settings file at a throwaway HOME and refuses to
// run the test anywhere else. The theme tests save the choice they click, and
// the developer's real settings file is not theirs to replace.
func isolateSettings(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, home) {
		t.Fatalf("the test would write the real settings file at %s; HOME is not isolated", path)
	}
	return path
}

// The panel used to rebuild its whole page on a theme change so the segmented
// controls took the new colors, which dropped the focused control. The page
// now keeps one identity: the controls follow the new tokens themselves, and
// the keyboard stays where the change was made.
func TestThemeChangeKeepsTheFocusOnTheChosenTheme(t *testing.T) {
	path := isolateSettings(t)
	a := &app{locale: localeChinese, themeMode: "light"}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		a.settingsRoot(c)
	}, 900, 650)
	dark, ok := tt.Find(a.t("themeDark"))
	if !ok {
		t.Fatal("the dark theme choice is not laid out")
	}
	tt.ClickAt(dark.X+dark.W/2, dark.Y+dark.H/2)
	if a.themeMode != "dark" {
		t.Fatalf("the click chose theme %q", a.themeMode)
	}
	// The change needs a frame to take effect, and a remount would drop the
	// focus on the one after it.
	tt.Frame()
	tt.Frame()
	info, ok := tt.Access(a.t("themeDark"))
	if !ok {
		t.Fatal("the theme switch is not in the accessibility tree")
	}
	if !info.Focused {
		t.Errorf("the theme switch reads %+v; want focus kept on it", info)
	}
	if got := readSettings(path).Theme; got != "dark" {
		t.Errorf("the saved theme is %q; want the click written to the settings file", got)
	}
}

// The selected block follows the new theme instead of keeping the colors it
// was built under. The color transition settles within a moment, so the check
// looks at the page after it.
func TestTheSelectedBlockTakesTheNewThemeColors(t *testing.T) {
	path := isolateSettings(t)
	a := &app{locale: localeChinese, themeMode: "light"}
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		a.settingsPage(c)
	}, 900, 650)
	// "中文" is the chosen language: its block is BgElevated, the color that
	// tells it apart from the track.
	block := func() [3]uint8 {
		r, ok := tt.Find("中文")
		if !ok {
			t.Fatal("the language row is not laid out")
		}
		p := tt.Image().RGBAAt(int(r.X+3), int(r.Y+r.H/2))
		return [3]uint8{p.R, p.G, p.B}
	}
	before := block()
	choice, _ := tt.Find(a.t("themeDark"))
	tt.ClickAt(choice.X+choice.W/2, choice.Y+choice.H/2)
	for i := 0; i < 12; i++ {
		tt.Advance(20 * time.Millisecond)
		tt.Frame()
	}
	after := block()
	if before[0] < 200 {
		t.Fatalf("the light block reads %v; the test needs a light start", before)
	}
	if after[0] > 80 {
		t.Errorf("the block reads %v after the dark switch; want the dark BgElevated, not the light one", after)
	}
	if got := readSettings(path).Theme; got != "dark" {
		t.Errorf("the saved theme is %q; want the click written to the settings file", got)
	}
}

// The palette selector showed its dots and the scheme name but carried no
// name of its own, so a reader reaching it heard only the current value
// without knowing what it selected.
func TestThePaletteTriggerNamesItself(t *testing.T) {
	isolateSettings(t)
	a := &app{locale: localeChinese, themeMode: "light"}
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		a.settingsPage(c)
	}, 900, 650)
	trigger, ok := tt.AccessByRole(ui.RoleButton, a.t("palette"))
	if !ok {
		t.Fatal("the palette selector has no accessible name")
	}
	if trigger.Description != a.t(a.currentTheme().label) {
		t.Errorf("the palette selector describes %q; want the current scheme", trigger.Description)
	}
}

// The branch switch in the title bar showed the branch name and icon with no
// control name, so a reader heard a branch but not what the button did.
func TestTheBranchSelectorNamesItself(t *testing.T) {
	a := &app{
		locale: localeChinese,
		status: gitrepo.Status{Branch: "main"},
		branches: []gitrepo.Branch{
			{Name: "main", Current: true},
			{Name: "dev"},
		},
	}
	tt := pageTester(a, a.branchSelector)
	trigger, ok := tt.AccessByRole(ui.RoleButton, a.t("branches"))
	if !ok {
		t.Fatal("the branch selector has no accessible name")
	}
	if trigger.Description != "main" {
		t.Errorf("the branch selector describes %q; want the current branch", trigger.Description)
	}
}
