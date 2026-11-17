package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestWorkbenchButtonColorsFollowSpecification(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for role, colors := range map[buttonRole][]string{
			buttonNormal:  {"#E8EDF5", "#DCE5F2", "#D2DDEE"},
			buttonPrimary: {"#176840", "#1B7547", "#125736"},
			buttonDanger:  {"#A53A47", "#AF3B4A", "#8D2D38"},
		} {
			if dark {
				switch role {
				case buttonNormal:
					colors = []string{"#2C333F", "#384252", "#424F62"}
				case buttonPrimary:
					colors = []string{"#196C43", "#1B7448", "#145C39"}
				case buttonDanger:
					colors = []string{"#A44451", "#AC4655", "#923642"}
				}
			}
			for state, want := range colors {
				tk := themeConfig("supergit").Tokens(dark)
				bg, ink := workbenchButtonColors(tk, role, state == 1, state == 2, false, false)
				if bg != ui.Hex(want) {
					t.Errorf("dark=%v role=%v state=%v fill=%v; want %s", dark, role, state, bg, want)
				}
				if contrastRatio(ink, bg) < 4.5 {
					t.Errorf("dark=%v role=%v state=%v text contrast below 4.5", dark, role, state)
				}
			}
		}
	}
}

func TestWorkbenchButtonPreservesNativeActivationAndDisabledState(t *testing.T) {
	clicks := 0
	disabled := false
	tt := ui.NewTester(func(c *ui.Context) {
		themeConfig("supergit").Apply(c)
		if workbenchButton(c, "Save", buttonPrimary, disabled).Clicked() {
			clicks++
		}
	}, 300, 80)
	tt.Click("Save")
	if clicks != 1 {
		t.Fatalf("pointer activation count=%d", clicks)
	}
	tt.Key(0, ui.KeyEnter)
	if clicks != 2 {
		t.Fatalf("keyboard activation count=%d", clicks)
	}
	disabled = true
	tt.Frame()
	info, ok := tt.AccessByRole(ui.RoleButton, "Save")
	if !ok || !info.Disabled {
		t.Fatal("disabled native state missing")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Key(0, ui.KeyEnter)
	if clicks != 2 {
		t.Fatalf("disabled button activated: %d", clicks)
	}
}

func TestWorkbenchToolbarButtonsUseStandardSize(t *testing.T) {
	setUIFontScale(fontScaleStandard)
	defer setUIFontScale(fontScaleStandard)
	a := &app{locale: localeEnglish}
	tt := remoteActionsTester(a)
	for _, label := range []string{a.t("fetch"), a.t("pull"), a.t("pushButton")} {
		info, ok := tt.AccessByRole(ui.RoleButton, label)
		if !ok {
			t.Fatalf("missing %q", label)
		}
		if info.Bounds.W != 29 || info.Bounds.H != 29 {
			t.Errorf("%s bounds=%v; want 29 × 29", label, info.Bounds)
		}
		if !info.Disabled {
			t.Errorf("%s enabled without repository", label)
		}
	}
}
