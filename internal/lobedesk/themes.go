package lobedesk

import (
	"reflect"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// The app's color schemes. A scheme is a light and a dark set of Lobe UI
// tokens; the settings page picks one by key and the theme mode (system,
// light, dark) picks which of the pair to show. Adding a scheme means adding
// an entry to buildThemes, nothing else: the settings page, the persisted
// settings and the history graph read the list from there.
const (
	themePaletteSuperGit            = "supergit"
	themePaletteDefault             = "default"
	themePaletteCatppuccin          = "catppuccin"
	themePaletteCatppuccinFrappe    = "catppuccin-frappe"
	themePaletteCatppuccinMacchiato = "catppuccin-macchiato"
)

type appTheme struct {
	key    string
	label  string // i18n key
	config *theme.Config
	// dots preview the scheme's accents in the settings page. They are the
	// scheme's own colors, not the active theme's tokens: the point is to show
	// a palette that is not applied yet.
	dots [3]ui.Color
	// lanesLight and lanesDark are the history graph's line colors for the
	// scheme's two sets. They come from the scheme's own accents rather than
	// from the token set, because the tokens alias several families onto one
	// accent (Catppuccin's Lime is Green and Gold is Yellow); two lane slots
	// sharing a color would defeat the point of widening the palette.
	lanesLight, lanesDark []ui.Color
}

var appThemes = buildThemes()

func buildThemes() []appTheme {
	list := []appTheme{
		{key: themePaletteSuperGit, label: "paletteSuperGit"},
		{key: themePaletteDefault, label: "paletteDefault"},
		{key: themePaletteCatppuccin, label: "paletteCatppuccin"},
		{key: themePaletteCatppuccinFrappe, label: "paletteCatppuccinFrappe"},
		{key: themePaletteCatppuccinMacchiato, label: "paletteCatppuccinMacchiato"},
	}
	for i := range list {
		switch list[i].key {
		case themePaletteSuperGit:
			list[i].config = theme.NewConfig(supergitTokens(false), supergitTokens(true))
			list[i].dots = [3]ui.Color{ui.Hex("#8CB4FF"), ui.Hex("#8AD5AA"), ui.Hex("#F2A1AE")}
			// Keep the proven lane order while the workbench theme is introduced.
			list[i].lanesLight = catppuccinLatte.laneColors()
			list[i].lanesDark = catppuccinMocha.laneColors()
		case themePaletteDefault:
			list[i].config = theme.Default()
			list[i].dots = [3]ui.Color{ui.Hex("#1677ff"), ui.Hex("#722ed1"), ui.Hex("#52c41a")}
			list[i].lanesLight = lobeLaneColors(*theme.Light())
			list[i].lanesDark = lobeLaneColors(*theme.Dark())
		case themePaletteCatppuccin:
			list[i].config = theme.NewConfig(catppuccinLatte.tokens(*theme.Light()), catppuccinMocha.tokens(*theme.Dark()))
			list[i].dots = [3]ui.Color{catppuccinMocha.Blue, catppuccinMocha.Mauve, catppuccinMocha.Green}
			list[i].lanesLight = catppuccinLatte.laneColors()
			list[i].lanesDark = catppuccinMocha.laneColors()
		case themePaletteCatppuccinFrappe:
			list[i].config = theme.NewConfig(catppuccinLatte.tokens(*theme.Light()), catppuccinFrappe.tokens(*theme.Dark()))
			list[i].dots = [3]ui.Color{catppuccinFrappe.Blue, catppuccinFrappe.Mauve, catppuccinFrappe.Green}
			list[i].lanesLight = catppuccinLatte.laneColors()
			list[i].lanesDark = catppuccinFrappe.laneColors()
		case themePaletteCatppuccinMacchiato:
			list[i].config = theme.NewConfig(catppuccinLatte.tokens(*theme.Light()), catppuccinMacchiato.tokens(*theme.Dark()))
			list[i].dots = [3]ui.Color{catppuccinMacchiato.Blue, catppuccinMacchiato.Mauve, catppuccinMacchiato.Green}
			list[i].lanesLight = catppuccinLatte.laneColors()
			list[i].lanesDark = catppuccinMacchiato.laneColors()
		}
	}
	return list
}

// laneAccentRoles orders the palette's accents for successive graph lanes.
// The two lines a reader compares most often are on neighbouring lanes, which
// take neighbouring slots, so consecutive roles are at least ~115 degrees
// apart in hue in every flavor. The order was chosen once for both the light
// and dark sets rather than per flavor, so a scheme keeps its colors from one
// mode to the other.
var laneAccentRoles = [...]string{
	"Peach", "Teal", "Red", "Blue", "Maroon", "Lavender",
	"Green", "Mauve", "Yellow", "Sapphire", "Pink", "Sky",
}

// laneColors resolves the lane order to this scheme's accents.
func (p paletteColors) laneColors() []ui.Color {
	out := make([]ui.Color, 0, len(laneAccentRoles))
	for _, role := range laneAccentRoles {
		out = append(out, p.accent(role))
	}
	return out
}

// accent resolves one role name, so the lane order can name roles as strings
// and stay readable next to the table above.
func (p paletteColors) accent(role string) ui.Color {
	switch role {
	case "Peach":
		return p.Peach
	case "Teal":
		return p.Teal
	case "Red":
		return p.Red
	case "Blue":
		return p.Blue
	case "Maroon":
		return p.Maroon
	case "Lavender":
		return p.Lavender
	case "Green":
		return p.Green
	case "Mauve":
		return p.Mauve
	case "Yellow":
		return p.Yellow
	case "Sapphire":
		return p.Sapphire
	case "Pink":
		return p.Pink
	case "Sky":
		return p.Sky
	}
	return p.Blue
}

// lobeLaneColors is the default scheme's lane order. Lobe has three oranges
// and three greens; Gold (close to Orange) and Lime (close to Yellow) are left
// out rather than given slots that look like a neighbour's, which leaves ten
// families and raises the smallest gap between consecutive lanes to about 97°
// from about 74° with all twelve. The order is a cycle, so the last lane sits
// next to the first one too.
func lobeLaneColors(tk theme.Tokens) []ui.Color {
	return []ui.Color{
		tk.Yellow, tk.Geekblue, tk.Volcano, tk.Green, tk.Magenta,
		tk.Blue, tk.Red, tk.Cyan, tk.Orange, tk.Purple,
	}
}

// lanePalette is the graph's color order for the active scheme and the set
// the current frame is showing. A scheme with no stored order (or an unknown
// key) falls back to the token palette, which still gives every lane a color.
func (a *app) lanePalette(tk *theme.Tokens) []ui.Color {
	for _, th := range appThemes {
		if th.key != a.palette {
			continue
		}
		if tk.Dark && len(th.lanesDark) > 0 {
			return th.lanesDark
		}
		if !tk.Dark && len(th.lanesLight) > 0 {
			return th.lanesLight
		}
		break
	}
	return lanePaletteOf(tk)
}

// themeConfig resolves a palette key to its tokens, falling back to Lobe UI's
// own colors so an unknown or stale key never leaves the app unthemed.
func themeConfig(key string) *theme.Config {
	for _, t := range appThemes {
		if t.key == key && t.config != nil {
			return t.config
		}
	}
	return theme.Default()
}

// paletteColors is one color scheme's roles, as the app maps them onto Lobe UI
// tokens. The Catppuccin flavors fill it from the published palette
// (github.com/catppuccin/catppuccin); a scheme of one's own fills the same
// roles and the mapping below applies unchanged.
type paletteColors struct {
	// flavor names the scheme, for tests and logs.
	flavor string
	Dark   bool

	// Background layers, from the outermost to the most raised.
	Crust, Mantle, Base, Surface0, Surface1, Surface2 ui.Color
	// Text and lines, from the strongest to the weakest.
	Text, Subtext1, Subtext0, Overlay2, Overlay1, Overlay0 ui.Color
	// Accents.
	Blue, Lavender, Mauve, Pink, Red, Maroon, Peach, Yellow, Green, Teal, Sky, Sapphire ui.Color
}

var catppuccinLatte = paletteColors{
	flavor:   "latte",
	Dark:     false,
	Crust:    ui.Hex("#dce0e8"),
	Mantle:   ui.Hex("#e6e9ef"),
	Base:     ui.Hex("#eff1f5"),
	Surface0: ui.Hex("#ccd0da"),
	Surface1: ui.Hex("#bcc0cc"),
	Surface2: ui.Hex("#acb0be"),
	Text:     ui.Hex("#4c4f69"),
	Subtext1: ui.Hex("#5c5f77"),
	Subtext0: ui.Hex("#6c6f85"),
	Overlay2: ui.Hex("#7c7f93"),
	Overlay1: ui.Hex("#8c8fa1"),
	Overlay0: ui.Hex("#9ca0b0"),
	Blue:     ui.Hex("#1e66f5"),
	Lavender: ui.Hex("#7287fd"),
	Mauve:    ui.Hex("#8839ef"),
	Pink:     ui.Hex("#ea76cb"),
	Red:      ui.Hex("#d20f39"),
	Maroon:   ui.Hex("#e64553"),
	Peach:    ui.Hex("#fe640b"),
	Yellow:   ui.Hex("#df8e1d"),
	Green:    ui.Hex("#40a02b"),
	Teal:     ui.Hex("#179299"),
	Sky:      ui.Hex("#04a5e5"),
	Sapphire: ui.Hex("#209fb5"),
}

var catppuccinMocha = paletteColors{
	flavor:   "mocha",
	Dark:     true,
	Crust:    ui.Hex("#11111b"),
	Mantle:   ui.Hex("#181825"),
	Base:     ui.Hex("#1e1e2e"),
	Surface0: ui.Hex("#313244"),
	Surface1: ui.Hex("#45475a"),
	Surface2: ui.Hex("#585b70"),
	Text:     ui.Hex("#cdd6f4"),
	Subtext1: ui.Hex("#bac2de"),
	Subtext0: ui.Hex("#a6adc8"),
	Overlay2: ui.Hex("#9399b2"),
	Overlay1: ui.Hex("#7f849c"),
	Overlay0: ui.Hex("#6c7086"),
	Blue:     ui.Hex("#89b4fa"),
	Lavender: ui.Hex("#b4befe"),
	Mauve:    ui.Hex("#cba6f7"),
	Pink:     ui.Hex("#f5c2e7"),
	Red:      ui.Hex("#f38ba8"),
	Maroon:   ui.Hex("#eba0ac"),
	Peach:    ui.Hex("#fab387"),
	Yellow:   ui.Hex("#f9e2af"),
	Green:    ui.Hex("#a6e3a1"),
	Teal:     ui.Hex("#94e2d5"),
	Sky:      ui.Hex("#89dceb"),
	Sapphire: ui.Hex("#74c7ec"),
}

var catppuccinFrappe = paletteColors{
	flavor:   "frappe",
	Dark:     true,
	Crust:    ui.Hex("#232634"),
	Mantle:   ui.Hex("#292c3c"),
	Base:     ui.Hex("#303446"),
	Surface0: ui.Hex("#414559"),
	Surface1: ui.Hex("#51576d"),
	Surface2: ui.Hex("#626880"),
	Text:     ui.Hex("#c6d0f5"),
	Subtext1: ui.Hex("#b5bfe2"),
	Subtext0: ui.Hex("#a5adce"),
	Overlay2: ui.Hex("#949cbb"),
	Overlay1: ui.Hex("#838ba7"),
	Overlay0: ui.Hex("#737994"),
	Blue:     ui.Hex("#8caaee"),
	Lavender: ui.Hex("#babbf1"),
	Mauve:    ui.Hex("#ca9ee6"),
	Pink:     ui.Hex("#f4b8e4"),
	Red:      ui.Hex("#e78284"),
	Maroon:   ui.Hex("#ea999c"),
	Peach:    ui.Hex("#ef9f76"),
	Yellow:   ui.Hex("#e5c890"),
	Green:    ui.Hex("#a6d189"),
	Teal:     ui.Hex("#81c8be"),
	Sky:      ui.Hex("#99d1db"),
	Sapphire: ui.Hex("#85c1dc"),
}

var catppuccinMacchiato = paletteColors{
	flavor:   "macchiato",
	Dark:     true,
	Crust:    ui.Hex("#181926"),
	Mantle:   ui.Hex("#1e2030"),
	Base:     ui.Hex("#24273a"),
	Surface0: ui.Hex("#363a4f"),
	Surface1: ui.Hex("#494d64"),
	Surface2: ui.Hex("#5b6078"),
	Text:     ui.Hex("#cad3f5"),
	Subtext1: ui.Hex("#b8c0e0"),
	Subtext0: ui.Hex("#a5adcb"),
	Overlay2: ui.Hex("#939ab7"),
	Overlay1: ui.Hex("#8087a2"),
	Overlay0: ui.Hex("#6e738d"),
	Blue:     ui.Hex("#8aadf4"),
	Lavender: ui.Hex("#b7bdf8"),
	Mauve:    ui.Hex("#c6a0f6"),
	Pink:     ui.Hex("#f5bde6"),
	Red:      ui.Hex("#ed8796"),
	Maroon:   ui.Hex("#ee99a0"),
	Peach:    ui.Hex("#f5a97f"),
	Yellow:   ui.Hex("#eed49f"),
	Green:    ui.Hex("#a6da95"),
	Teal:     ui.Hex("#8bd5ca"),
	Sky:      ui.Hex("#91d7e3"),
	Sapphire: ui.Hex("#7dc4e4"),
}

// accentFamilies maps each Lobe UI accent family onto the palette role that
// fits it. Everything the app and the Lobe components draw with these colors,
// from buttons to tags to selection fills, follows the palette through it.
var accentFamilies = []struct {
	family string
	color  func(p paletteColors) ui.Color
}{
	{"Primary", func(p paletteColors) ui.Color { return p.Blue }},
	{"Blue", func(p paletteColors) ui.Color { return p.Blue }},
	{"Geekblue", func(p paletteColors) ui.Color { return p.Lavender }},
	{"Info", func(p paletteColors) ui.Color { return p.Sky }},
	{"Link", func(p paletteColors) ui.Color { return p.Blue }},
	{"Success", func(p paletteColors) ui.Color { return p.Green }},
	{"Green", func(p paletteColors) ui.Color { return p.Green }},
	{"Lime", func(p paletteColors) ui.Color { return p.Green }},
	{"Warning", func(p paletteColors) ui.Color { return p.Yellow }},
	{"Gold", func(p paletteColors) ui.Color { return p.Yellow }},
	{"Yellow", func(p paletteColors) ui.Color { return p.Yellow }},
	{"Error", func(p paletteColors) ui.Color { return p.Red }},
	{"Red", func(p paletteColors) ui.Color { return p.Red }},
	{"Volcano", func(p paletteColors) ui.Color { return p.Maroon }},
	{"Orange", func(p paletteColors) ui.Color { return p.Peach }},
	{"Purple", func(p paletteColors) ui.Color { return p.Mauve }},
	{"Magenta", func(p paletteColors) ui.Color { return p.Pink }},
	{"Cyan", func(p paletteColors) ui.Color { return p.Teal }},
}

var colorType = reflect.TypeOf(ui.Color{})

// tokens derives a full Lobe UI token set from the palette. Backgrounds, text
// and borders are assigned by role; each accent family is derived from its
// accent so hover, active, fill and border variants stay in the same hue
// without listing every one of Lobe's tokens by hand.
func (p paletteColors) tokens(base theme.Tokens) theme.Tokens {
	t := base

	// Surfaces: the window is the darkest (Latte: deepest) layer, the content
	// card sits on it, and dialogs and popovers are raised again.
	t.BgLayout = p.Crust
	t.BgBase = p.Base
	t.BgContainer = p.Base
	t.BgContainerSecondary = p.Mantle
	// Dark flavors lift raised surfaces toward surface0; Latte already has
	// base as its lightest layer, so raised surfaces stay on base, as Lobe's
	// light tokens do.
	t.BgElevated = p.Base
	t.BgSpotlight = p.Mantle
	if p.Dark {
		t.BgElevated = p.Surface0
		t.BgSpotlight = p.Surface1
	}
	t.BgBlur = p.Mantle.Alpha(0.85)
	t.BgMask = p.Crust.Alpha(0.55)
	t.BgContainerDisabled = p.Text.Alpha(0.04)
	// Solid buttons invert the text color, as in Lobe's own themes.
	t.BgSolid = p.Text
	t.BgSolidHover = p.Subtext1
	t.BgSolidActive = p.Subtext0
	t.BgTextHover = p.Text.Alpha(0.06)
	t.BgTextActive = p.Text.Alpha(0.12)

	t.Fill = p.Text.Alpha(0.12)
	t.FillSecondary = p.Text.Alpha(0.07)
	t.FillTertiary = p.Text.Alpha(0.04)
	t.FillQuaternary = p.Text.Alpha(0.025)
	t.FillAlter = p.Text.Alpha(0.02)
	t.FillContent = p.Text.Alpha(0.07)
	t.FillContentHover = p.Text.Alpha(0.12)

	t.Border = p.Surface1
	t.BorderSecondary = p.Surface0
	t.BorderDisabled = p.Surface1
	t.BorderBg = p.Base
	t.Split = p.Surface0

	t.Text = p.Text
	t.TextBase = p.Text
	t.TextHeading = p.Text
	t.TextLabel = p.Text
	t.TextSecondary = p.Subtext1
	t.TextTertiary = p.Subtext0
	t.TextQuaternary = p.Overlay0
	t.TextDescription = p.Subtext0
	t.TextPlaceholder = p.Overlay0
	t.TextDisabled = p.Surface2
	t.TextLightSolid = p.Crust

	t.Icon = p.Overlay1
	t.IconHover = p.Text
	t.Highlight = p.Red
	t.Gray = p.Overlay1
	t.Gray7 = p.Overlay0
	t.Gray9 = p.Overlay1
	t.Gray10 = p.Subtext0
	t.Gray11 = p.Text
	t.Shadow = ui.RGBA(0, 0, 0, 0.24)

	hover := func(c ui.Color) ui.Color { return c.Mix(p.Text, 0.16) }
	active := func(c ui.Color) ui.Color { return c.Mix(p.Text, 0.28) }
	textOf := func(c ui.Color) ui.Color { return c.Mix(p.Text, 0.12) }
	tint := func(c ui.Color) ui.Color { return c.Mix(p.Base, 0.9) }
	line := func(c ui.Color) ui.Color { return c.Mix(p.Base, 0.55) }

	for _, f := range accentFamilies {
		accent := f.color(p)
		v := reflect.ValueOf(&t).Elem()
		typ := v.Type()
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.Type != colorType || !strings.HasPrefix(field.Name, f.family) {
				continue
			}
			suffix := strings.TrimPrefix(field.Name, f.family)
			color := accent
			switch {
			case suffix == "":
				color = accent
			case suffix == "Foreground":
				// Text on a solid accent: dark ink in a dark flavor, light ink
				// in Latte, so a filled button's label always reads.
				if p.Dark {
					color = p.Crust
				} else {
					color = p.Base
				}
			case suffix == "Hover":
				color = hover(accent)
			case suffix == "Active":
				color = active(accent)
			case suffix == "TextHover":
				color = hover(accent)
			case suffix == "TextActive":
				color = active(accent)
			case suffix == "Text":
				color = textOf(accent)
			case suffix == "BgFilledHover":
				color = tint(accent).Mix(accent, 0.12)
			case suffix == "BgHover":
				color = tint(accent).Mix(accent, 0.1)
			case suffix == "BgActive":
				color = tint(accent).Mix(accent, 0.16)
			case suffix == "Bg":
				color = tint(accent)
			case suffix == "Fill":
				color = accent.Alpha(0.38)
			case suffix == "FillSecondary":
				color = accent.Alpha(0.28)
			case suffix == "FillTertiary":
				color = accent.Alpha(0.18)
			case suffix == "FillQuaternary":
				color = accent.Alpha(0.1)
			case suffix == "BorderHover":
				color = line(accent)
			case suffix == "Border":
				color = line(accent).Mix(p.Base, 0.2)
			case suffix == "Outline":
				color = accent.Alpha(0.06)
			case suffix == "Affix":
				color = accent
			default:
				// Numbered ramp entries (Geekblue8, Magenta9, …) share the
				// accent; the few that need a distinct tone are set above.
				color = accent
			}
			v.Field(i).Set(reflect.ValueOf(color))
		}
	}
	// Gray is the neutral tag family, and it is the one accent Lobe keeps to
	// itself: a filled gray tag drew its ink from the library's fixed mid-gray,
	// which reads about 4.2:1 on the light fill and 2.2:1 on the dark ones. The
	// three tokens such a tag draws from follow the palette's neutral instead,
	// so "no status" badges read like every other badge in the scheme.
	t.GrayFillTertiary = p.Overlay1.Alpha(0.18)
	t.GrayFillQuaternary = p.Overlay1.Alpha(0.1)
	t.GrayActive = capsuleInk(p.Overlay1, t.GrayFillTertiary.Over(p.Base), 4.5)
	// Selection (selected text) is the accent, not the yellow Lobe uses: it
	// reads on both the light and dark surfaces and matches the scheme. Set
	// after the loop, which would otherwise give Yellow9 the Yellow accent.
	t.Yellow9 = p.Blue.Alpha(0.35)
	return t
}

// String names the palette in test failures and logs.
func (p paletteColors) String() string {
	return "catppuccin-" + p.flavor
}
