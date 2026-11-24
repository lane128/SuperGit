package lobedesk

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestCatppuccinTokensFollowTheFlavor(t *testing.T) {
	light := catppuccinLatte.tokens(*theme.Light())
	dark := catppuccinMocha.tokens(*theme.Dark())

	for _, c := range []struct {
		name string
		got  ui.Color
		want ui.Color
	}{
		{"light layout", light.BgLayout, ui.Hex("#dce0e8")},
		{"light text", light.Text, ui.Hex("#4c4f69")},
		{"light primary", light.Primary, ui.Hex("#1e66f5")},
		{"dark layout", dark.BgLayout, ui.Hex("#11111b")},
		{"dark text", dark.Text, ui.Hex("#cdd6f4")},
		{"dark primary", dark.Primary, ui.Hex("#89b4fa")},
		{"dark success", dark.Success, ui.Hex("#a6e3a1")},
		{"dark error", dark.Error, ui.Hex("#f38ba8")},
		{"frappe base", catppuccinFrappe.tokens(*theme.Dark()).BgContainer, ui.Hex("#303446")},
		{"frappe blue", catppuccinFrappe.tokens(*theme.Dark()).Primary, ui.Hex("#8caaee")},
		{"macchiato base", catppuccinMacchiato.tokens(*theme.Dark()).BgContainer, ui.Hex("#24273a")},
		{"macchiato text", catppuccinMacchiato.tokens(*theme.Dark()).Text, ui.Hex("#cad3f5")},
	} {
		if c.got != c.want {
			t.Errorf("%s = %v; want %v", c.name, c.got, c.want)
		}
	}
	if light.Dark || !dark.Dark {
		t.Fatalf("palette tokens report dark = %v and %v; want false and true", light.Dark, dark.Dark)
	}
	// Accent variants derive from the family's color instead of keeping Lobe's.
	if dark.PrimaryHover == dark.Primary || dark.PrimaryFill == dark.Primary {
		t.Errorf("primary variants were not derived: hover %v, fill %v, base %v", dark.PrimaryHover, dark.PrimaryFill, dark.Primary)
	}
	if dark.TextSecondary == dark.Text {
		t.Error("secondary text kept the primary text color")
	}
	// Text on a filled accent button must not be the accent itself.
	if dark.PrimaryForeground != catppuccinMocha.Crust {
		t.Errorf("dark primary foreground = %v; want the crust color", dark.PrimaryForeground)
	}
	if light.PrimaryForeground != catppuccinLatte.Base {
		t.Errorf("light primary foreground = %v; want the base color", light.PrimaryForeground)
	}
}

func TestThemeConfigsResolveAndFallBack(t *testing.T) {
	if len(appThemes) < 2 {
		t.Fatalf("appThemes has %d entries; want at least the default and Catppuccin", len(appThemes))
	}
	for _, th := range appThemes {
		if th.key == "" || th.label == "" || th.config == nil || th.dots[0].A == 0 {
			t.Fatalf("incomplete theme entry: %+v", th)
		}
		if themeConfig(th.key) != th.config {
			t.Errorf("themeConfig(%q) did not return the registered config", th.key)
		}
	}
	if themeConfig("no-such-scheme") != theme.Default() {
		t.Error("unknown palette did not fall back to the default")
	}
	if themeConfig(themePaletteCatppuccin) == theme.Default() {
		t.Error("Catppuccin resolved to the default config")
	}
}

// laneHueDistance is the OKLab distance between two colors on a 0–100 scale.
// Unlike a lightness ratio it says whether two lane lines look like different
// colors, which is what decides how many lanes a scheme can show apart.
func laneHueDistance(a, b ui.Color) float64 {
	lab := func(c ui.Color) [3]float64 {
		lin := func(v uint8) float64 {
			x := float64(v) / 255
			if x <= 0.04045 {
				return x / 12.92
			}
			return math.Pow((x+0.055)/1.055, 2.4)
		}
		r, g, b := lin(c.R), lin(c.G), lin(c.B)
		l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
		m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
		s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
		return [3]float64{
			0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
			1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
			0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
		}
	}
	la, lb := lab(a), lab(b)
	dr, dg, db := la[0]-lb[0], la[1]-lb[1], la[2]-lb[2]
	return 100 * math.Sqrt(dr*dr+dg*dg+db*db)
}

// The lane palette has to keep its colors apart in every scheme: a repository
// with more than a handful of live branches draws several lanes at once, and
// two lanes that read as the same color defeat the graph. The thresholds come
// from the roles' OKLab separation (≥6.5 for any pair, ≥11 for neighboring
// slots); a new scheme whose accents collapse two slots fails here.
func TestLanePaletteStaysDistinctInEveryScheme(t *testing.T) {
	type paletteCase struct {
		name      string
		tk        *theme.Tokens
		lightened bool // light themes darken lanes with laneInk before drawing
	}
	cases := []paletteCase{{"fallback", nil, false}}
	for _, th := range appThemes {
		// SuperGit separates semantic status inks from history lanes. Its
		// registered twelve-color palette is checked by the ordered-palette
		// test and TestSuperGitSelectionAndStatusFamiliesStayConsistent;
		// lanePaletteOf is not used to render its history.
		if th.key == themePaletteSuperGit {
			continue
		}
		for _, dark := range []bool{false, true} {
			cases = append(cases, paletteCase{th.key, th.config.Tokens(dark), !dark})
		}
	}
	for _, p := range cases {
		var palette []ui.Color
		if p.tk == nil {
			palette = lanePalette
		} else {
			palette = lanePaletteOf(p.tk)
		}
		if len(palette) != len(lanePalette) {
			t.Errorf("%s: palette has %d colors; want %d", p.name, len(palette), len(lanePalette))
			continue
		}
		closest := 100.0
		neighbors := 100.0
		for i := range palette {
			for j := i + 1; j < len(palette); j++ {
				closest = math.Min(closest, laneHueDistance(palette[i], palette[j]))
			}
			j := (i + 1) % len(palette)
			neighbors = math.Min(neighbors, laneHueDistance(palette[i], palette[j]))
		}
		if closest < 6.5 {
			t.Errorf("%s: closest lane pair is %.1f; want at least 6.5", p.name, closest)
		}
		if neighbors < 11 {
			t.Errorf("%s: neighboring slots are %.1f apart; want at least 11", p.name, neighbors)
		}
		if p.tk == nil || !p.lightened {
			continue
		}
		// laneInk pulls light-theme lanes toward black; different hues must not
		// converge into one ink while gaining contrast.
		inks := make([]ui.Color, len(palette))
		for i, c := range palette {
			inks[i] = laneInk(p.tk, c)
		}
		inkClosest := 100.0
		for i := range inks {
			for j := i + 1; j < len(inks); j++ {
				inkClosest = math.Min(inkClosest, laneHueDistance(inks[i], inks[j]))
			}
		}
		if inkClosest < 6.5 {
			t.Errorf("%s: darkened lane pair is %.1f apart; want at least 6.5", p.name, inkClosest)
		}
	}
}

func TestGraphLaneColorsComeFromTokens(t *testing.T) {
	tk := catppuccinMocha.tokens(*theme.Dark())
	palette := lanePaletteOf(&tk)
	if len(palette) != len(lanePalette) {
		t.Fatalf("theme palette has %d colors; want %d", len(palette), len(lanePalette))
	}
	if palette[0] != tk.Blue || palette[2] != tk.Green {
		t.Errorf("lane palette does not follow the tokens: %v", palette)
	}
	graph := newCommitGraph("", []gitrepo.Commit{{SHA: "a", Refs: "HEAD -> main"}}, palette)
	if got := graph.LaneColor(graph.Rows[0].Color); got != palette[graph.Rows[0].Color] {
		t.Errorf("row color = %v; want palette slot %d", got, graph.Rows[0].Color)
	}
	if graph.Rows[0].Color >= len(palette) {
		t.Errorf("row color index %d is outside the palette", graph.Rows[0].Color)
	}
}

func TestPersistedPaletteApplies(t *testing.T) {
	a := &app{palette: themePaletteCatppuccin}
	for _, th := range appThemes {
		a.applySettings(persistedSettings{Palette: th.key})
		if a.palette != th.key {
			t.Errorf("persisted palette %q applied as %q", th.key, a.palette)
		}
	}
	a.applySettings(persistedSettings{Palette: "nope"})
	if a.palette != appThemes[len(appThemes)-1].key {
		t.Errorf("unknown persisted palette overwrote the choice with %q", a.palette)
	}
}

// TestPaletteTextTokensStayReadable guards every scheme's text roles: adding a
// scheme means adding a flavor here through appThemes, and a flavor whose text
// hierarchy loses contrast fails here instead of on screen.
func TestPaletteTextTokensStayReadable(t *testing.T) {
	flavors := []struct {
		name string
		pal  paletteColors
		dark bool
	}{
		{"latte", catppuccinLatte, false},
		{"mocha", catppuccinMocha, true},
		{"frappe", catppuccinFrappe, true},
		{"macchiato", catppuccinMacchiato, true},
	}
	for _, flavor := range flavors {
		base := theme.Light()
		if flavor.dark {
			base = theme.Dark()
		}
		tk := flavor.pal.tokens(*base)
		selected := tk.Blue.Alpha(0.14).Over(tk.BgContainer)
		if ratio := contrastRatio(tk.Text, tk.BgContainer); ratio < 7 {
			t.Errorf("%s: primary text = %.2f; want at least 7:1", flavor.name, ratio)
		}
		// A filled gray tag ("no status") is the neutral badge: its ink has to
		// clear the same floor as any small label on its own fill.
		grayGround := tk.GrayFillTertiary.Over(tk.BgContainer)
		if ratio := contrastRatio(tk.GrayActive, grayGround); ratio < 4.5 {
			t.Errorf("%s: gray tag ink = %.2f; want at least 4.5:1", flavor.name, ratio)
		}
		for _, c := range []struct {
			name  string
			color ui.Color
			min   float64
		}{
			{"secondary on the page", tk.TextSecondary, 4.5},
			{"secondary on a selected row", tk.TextSecondary, 4.5},
			// Catppuccin's subtext0 sits at 4.37 on Latte. It stays the
			// tertiary role so the palette keeps its published color, and the
			// few small labels that land on a tinted row use secondary text.
			{"tertiary on the page", tk.TextTertiary, 4.3},
		} {
			ground := tk.BgContainer
			if c.name == "secondary on a selected row" {
				ground = selected
			}
			if ratio := contrastRatio(c.color, ground); ratio < c.min {
				t.Errorf("%s: %s = %.2f; want at least %.1f:1", flavor.name, c.name, ratio, c.min)
			}
		}
	}
}

func TestDiffMetaTextReadsOnItsRow(t *testing.T) {
	light := theme.Light()
	a := &app{}
	for _, kind := range []string{"meta", "hunk"} {
		colors := a.diffColors(light, kind)
		if ratio := contrastRatio(colors.text, colors.background); ratio < 4.5 {
			t.Errorf("%s: text = %.2f; want at least 4.5:1", kind, ratio)
		}
	}
}

// hueOf is the color's hue in degrees, used to check that neighbouring lane
// slots do not wear look-alike colors.
func hueOf(c ui.Color) float64 {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	max, min := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	if max == min {
		return 0
	}
	var h float64
	switch max {
	case r:
		h = math.Mod((g-b)/(max-min), 6)
	case g:
		h = (b-r)/(max-min) + 2
	default:
		h = (r-g)/(max-min) + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h
}

// hueGap is the shorter way round the circle between two hues.
func hueGap(a, b float64) float64 {
	d := math.Abs(a - b)
	return math.Min(d, 360-d)
}

// The graph hands lane colors out in order, so neighbouring slots are the
// pairs a reader sees side by side. Each scheme's order has to keep those
// apart, and widening the palette must not add a duplicate. The default Lobe
// scheme uses ten families: Gold and Lime sit too close to Orange and Yellow
// to earn a slot of their own.
func TestLanePalettesAreWideAndOrdered(t *testing.T) {
	for _, th := range appThemes {
		want := 12
		if th.key == themePaletteDefault {
			want = 10
		}
		for _, set := range []struct {
			name   string
			colors []ui.Color
		}{
			{"light", th.lanesLight},
			{"dark", th.lanesDark},
		} {
			if len(set.colors) != want {
				t.Errorf("%s %s: %d lane colors, want %d", th.key, set.name, len(set.colors), want)
			}
			seen := make(map[ui.Color]bool, len(set.colors))
			for _, color := range set.colors {
				if seen[color] {
					t.Errorf("%s %s: %v repeats a slot", th.key, set.name, color)
				}
				seen[color] = true
			}
			for i, color := range set.colors {
				next := set.colors[(i+1)%len(set.colors)]
				if gap := hueGap(hueOf(color), hueOf(next)); gap < 60 {
					t.Errorf("%s %s: slots %d and %d are only %.0f° apart",
						th.key, set.name, i, (i+1)%len(set.colors), gap)
				}
			}
		}
	}
}
