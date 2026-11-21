package lobedesk

import (
	_ "embed"
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const codeFontMin, codeFontMax = 11, 19

// Supported monospace code font presets. JetBrains Mono is the default; the
// bundled OFL fonts register with MyGo on first use so every machine gets the
// exact typeface out of the box, while SF Mono and Menlo resolve through the
// platform's native font system.
const (
	codeFontJetBrainsMono = "jetbrains-mono"
	codeFontFiraCode      = "fira-code"
	codeFontIBMPlexMono   = "ibm-plex-mono"
	codeFontGeistMono     = "geist-mono"
	codeFontSFMono        = "sf-mono"
	codeFontMenlo         = "menlo"
)

type codeFontOption struct {
	Key     string
	Label   string
	Family  string
	Advance float32 // ASCII glyph advance as a fraction of font size, rounded slightly up
}

var codeFontOptions = []codeFontOption{
	{Key: codeFontJetBrainsMono, Label: "JetBrains Mono", Family: "SuperGit JetBrains Mono", Advance: 0.61},
	{Key: codeFontFiraCode, Label: "Fira Code", Family: "SuperGit Fira Code", Advance: 0.61},
	{Key: codeFontIBMPlexMono, Label: "IBM Plex Mono", Family: "SuperGit IBM Plex Mono", Advance: 0.61},
	{Key: codeFontGeistMono, Label: "Geist Mono", Family: "SuperGit Geist Mono", Advance: 0.61},
	{Key: codeFontSFMono, Label: "SF Mono", Family: "monospace", Advance: 0.64},
	{Key: codeFontMenlo, Label: "Menlo", Family: "Menlo", Advance: 0.62},
}

//go:embed fonts/JetBrainsMono-Regular.ttf
var jetBrainsMonoRegularTTF []byte

//go:embed fonts/JetBrainsMono-Bold.ttf
var jetBrainsMonoBoldTTF []byte

//go:embed fonts/FiraCode-Regular.ttf
var firaCodeRegularTTF []byte

//go:embed fonts/FiraCode-Bold.ttf
var firaCodeBoldTTF []byte

//go:embed fonts/IBMPlexMono-Regular.ttf
var ibmPlexMonoRegularTTF []byte

//go:embed fonts/IBMPlexMono-Bold.ttf
var ibmPlexMonoBoldTTF []byte

//go:embed fonts/GeistMono-Regular.ttf
var geistMonoRegularTTF []byte

//go:embed fonts/GeistMono-Bold.ttf
var geistMonoBoldTTF []byte

var (
	bundledFontsOnce sync.Once
	activeCodeFont   atomic.Value // stores string preset key
)

func init() {
	activeCodeFont.Store(codeFontJetBrainsMono)
}

func ensureBundledCodeFonts() {
	bundledFontsOnce.Do(func() {
		for _, font := range []struct {
			family string
			data   []byte
		}{
			{"SuperGit JetBrains Mono", jetBrainsMonoRegularTTF},
			{"SuperGit JetBrains Mono", jetBrainsMonoBoldTTF},
			{"SuperGit Fira Code", firaCodeRegularTTF},
			{"SuperGit Fira Code", firaCodeBoldTTF},
			{"SuperGit IBM Plex Mono", ibmPlexMonoRegularTTF},
			{"SuperGit IBM Plex Mono", ibmPlexMonoBoldTTF},
			{"SuperGit Geist Mono", geistMonoRegularTTF},
			{"SuperGit Geist Mono", geistMonoBoldTTF},
		} {
			_ = ui.RegisterFont(font.data, font.family)
		}
	})
}

func codeFontKeyOf(key string) string {
	for _, opt := range codeFontOptions {
		if opt.Key == key {
			return opt.Key
		}
	}
	return codeFontJetBrainsMono
}

func codeFontFamilyFor(key string) string {
	ensureBundledCodeFonts()
	key = codeFontKeyOf(key)
	for _, opt := range codeFontOptions {
		if opt.Key == key {
			return opt.Family
		}
	}
	return codeFontOptions[0].Family
}

func setCodeFontKey(key string) {
	activeCodeFont.Store(codeFontKeyOf(key))
}

func codeFontKeyValue() string {
	if v, ok := activeCodeFont.Load().(string); ok && v != "" {
		return codeFontKeyOf(v)
	}
	return codeFontJetBrainsMono
}

func codeFontFamily() string {
	return codeFontFamilyFor(codeFontKeyValue())
}

func codeFontAdvanceFor(key string) float32 {
	key = codeFontKeyOf(key)
	for _, opt := range codeFontOptions {
		if opt.Key == key && opt.Advance > 0 {
			return opt.Advance
		}
	}
	return codeFontOptions[0].Advance
}

func codeFontAdvance() float32 {
	return codeFontAdvanceFor(codeFontKeyValue())
}

// codeFontSetting omits the default JetBrains Mono from the persisted file,
// matching fontScaleSetting so future defaults stay clean.
func codeFontSetting(key string) string {
	key = codeFontKeyOf(key)
	if key == codeFontJetBrainsMono {
		return ""
	}
	return key
}

func (a *app) codeFontKey() string {
	if a != nil && a.codeFont != "" {
		return codeFontKeyOf(a.codeFont)
	}
	return codeFontKeyValue()
}

func (a *app) patchFontFamily() string {
	return codeFontFamilyFor(a.codeFontKey())
}

func validCodeFontSize(size float32) bool {
	return size >= codeFontMin && size <= codeFontMax
}

func (a *app) patchFontSize() float32 {
	if a.codeFontSize > 0 {
		return a.codeFontSize
	}
	return scaledFont(diffCodeSize)
}

func (a *app) patchRowHeight() float32 {
	return max(diffRowH, a.patchFontSize()+8)
}

func (a *app) currentCodeFontOption() codeFontOption {
	key := a.codeFontKey()
	for _, opt := range codeFontOptions {
		if opt.Key == key {
			return opt
		}
	}
	return codeFontOptions[0]
}

// codeFontFamilySetting presents the monospace presets as a single-row dropdown
// matching paletteRow, keeping the appearance card compact at minimum window
// size while rendering the active font name in its own typeface.
func (a *app) codeFontFamilySetting(c *ui.Context) {
	tk := theme.TokensOf(c)
	current := a.currentCodeFontOption()
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		display.Text(c, a.t("codeFontFamily"), display.TextProps{FontSize: scaledFont(12)}).Width(96).Shrink(0)
		items := make([]overlay.MenuItem, 0, len(codeFontOptions))
		for _, opt := range codeFontOptions {
			item := overlay.MenuItem{Key: opt.Key, Label: opt.Label}
			if opt.Key == current.Key {
				item.Extra = "✓"
			}
			items = append(items, item)
		}
		chosen := overlay.DropdownMenu(c, &a.codeFontOpen, overlay.DropdownMenuProps{Items: items}, func(trigger ui.Element) {
			styleWorkbenchButton(c, trigger, buttonNormal, false, a.codeFontOpen)
			trigger.Role(ui.RoleButton).Label(a.t("codeFontFamily")).Description(current.Label)
			trigger.Gap(6).Children(func() {
				ui.Text(c, current.Label).Font(current.Family).FontSize(scaledFont(12)).SingleLine()
				ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(11)).TextColor(tk.TextTertiary)
			})
		})
		if chosen != "" && chosen != current.Key {
			resolved := codeFontKeyOf(chosen)
			a.codeFont = resolved
			setCodeFontKey(resolved)
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
	})
}

func (a *app) codeFontSizeSetting(c *ui.Context) {
	size := a.patchFontSize()
	down := max(codeFontMin, float32(math.Ceil(float64(size)))-1)
	up := min(codeFontMax, float32(math.Floor(float64(size)))+1)
	next := a.codeFontSize
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		display.Text(c, a.t("codeFontSize"), display.TextProps{FontSize: scaledFont(12)}).Width(96).Shrink(0)
		ui.Row(c).Grow(1).Basis(0).MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
				if workbenchIconButton(c, lucide.Minus(), a.t("decreaseCodeFont"), buttonNormal, size <= codeFontMin).Clicked() {
					next = down
				}
				display.Text(c, fmt.Sprintf("%.1f pt", size), display.TextProps{FontSize: scaledFont(12)}).Width(58).TextAlign(ui.Center)
				if workbenchIconButton(c, lucide.Plus(), a.t("increaseCodeFont"), buttonNormal, size >= codeFontMax).Clicked() {
					next = up
				}
			})
			if workbenchButton(c, a.t("codeFontFollowInterface"), buttonNormal, next == 0).Clicked() {
				next = 0
			}
		})
	})
	if next != a.codeFontSize {
		a.codeFontSize = next
		a.saveSettingsFor(toastSettings)
		a.invalidate()
	}
}
