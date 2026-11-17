package lobedesk

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSuperGitPaletteRegistrationAndPersistence(t *testing.T) {
	var found bool
	for _, th := range appThemes {
		if th.key != "supergit" {
			continue
		}
		found = true
		for _, language := range []locale{localeChinese, localeEnglish} {
			a := &app{locale: language}
			if got := a.t(th.label); got != "SuperGit" {
				t.Errorf("%s palette label = %q", language, got)
			}
		}
	}
	if !found {
		t.Fatal("SuperGit palette is not registered")
	}
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := writeSettings(path, persistedSettings{Palette: "supergit", Theme: "dark"}); err != nil {
		t.Fatal(err)
	}
	a := &app{}
	a.applySettings(readSettings(path))
	if a.palette != "supergit" || a.themeMode != "dark" {
		t.Fatalf("palette did not survive restart: %q %q", a.palette, a.themeMode)
	}
}

func TestSuperGitSelectionAndStatusFamiliesStayConsistent(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tk := themeConfig("supergit").Tokens(dark)
		for _, pair := range [][2]ui.Color{{tk.Blue, tk.Primary}, {tk.Green, tk.Success}, {tk.Red, tk.Error}, {tk.Yellow, tk.Warning}} {
			if pair[0] != pair[1] {
				t.Errorf("dark=%v semantic family base %v differs from ink %v", dark, pair[0], pair[1])
			}
		}
		selected := selectionBg(tk, tk.BgContainer)
		if selected != tk.Blue.Alpha(0.14).Over(tk.BgContainer) {
			t.Error("selection does not use semantic blue")
		}
		a := &app{palette: "supergit"}
		lanes := a.lanePalette(tk)
		want := catppuccinLatte.laneColors()
		if dark {
			want = catppuccinMocha.laneColors()
		}
		if len(lanes) != len(want) {
			t.Fatalf("dark=%v history palette has %d lanes; want %d", dark, len(lanes), len(want))
		}
		for i := range lanes {
			if lanes[i] != want[i] {
				t.Errorf("dark=%v lane %d changed with semantic inks", dark, i)
			}
		}
	}
}

func TestSuperGitSemanticColors(t *testing.T) {
	for _, dark := range []bool{false, true} {
		tk := themeConfig("supergit").Tokens(dark)
		colors := []string{"#F3F5F8", "#FFFFFF", "#F9FAFC", "#DCE3ED", "#1B2535", "#4D5B70", "#5B6A80", "#2456BD", "#E8F0FF", "#16603A", "#EAF6EE", "#9A3346", "#FAECEF", "#7D521A", "#FFF2DC"}
		if dark {
			colors = []string{"#13161B", "#191C21", "#1B1E24", "#303641", "#ECF0F7", "#A8B4C7", "#93A1B8", "#8CB4FF", "#213653", "#8AD5AA", "#1A3428", "#F2A1AE", "#39252D", "#EEC783", "#3C3321"}
		}
		got := []ui.Color{tk.BgLayout, tk.BgContainer, tk.BgContainerSecondary, tk.Border, tk.Text, tk.TextSecondary, tk.TextTertiary, tk.Primary, tk.PrimaryBg, tk.Success, tk.SuccessBg, tk.Error, tk.ErrorBg, tk.Warning, tk.WarningBg}
		for i, want := range colors {
			if got[i] != ui.Hex(want) {
				t.Errorf("dark=%v semantic color %d = %v; want %s", dark, i, got[i], want)
			}
		}
		for _, pair := range [][2]ui.Color{{tk.Text, tk.BgContainer}, {tk.TextSecondary, tk.BgContainerSecondary}, {tk.TextTertiary, tk.BgContainer}, {tk.Primary, tk.PrimaryBg}, {tk.Success, tk.SuccessBg}, {tk.Error, tk.ErrorBg}, {tk.Warning, tk.WarningBg}} {
			if ratio := contrastRatio(pair[0], pair[1]); ratio < 4.5 {
				t.Errorf("dark=%v text contrast %.2f < 4.5", dark, ratio)
			}
		}
	}
}
