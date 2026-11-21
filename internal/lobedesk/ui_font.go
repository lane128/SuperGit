package lobedesk

import (
	_ "embed"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Supported UI interface font presets. The system default uses the platform's
// native UI font (SF Pro on macOS); the bundled OFL sans-serif families
// register with MyGo on first use with PingFang SC and system-ui fallbacks so
// CJK and Latin text remain crisp together.
const (
	uiFontSystem      = "system"
	uiFontInter       = "inter"
	uiFontGeist       = "geist"
	uiFontIBMPlexSans = "ibm-plex-sans"
)

type uiFontOption struct {
	Key      string
	Label    string // non-empty for fixed names; empty uses LabelKey via i18n
	LabelKey string
	Family   string
}

var uiFontOptions = []uiFontOption{
	{Key: uiFontSystem, LabelKey: "uiFontSystem", Family: ""},
	{Key: uiFontInter, Label: "Inter", Family: "SuperGit Inter, PingFang SC, system-ui"},
	{Key: uiFontGeist, Label: "Geist", Family: "SuperGit Geist, PingFang SC, system-ui"},
	{Key: uiFontIBMPlexSans, Label: "IBM Plex Sans", Family: "SuperGit IBM Plex Sans, PingFang SC, system-ui"},
}

//go:embed fonts/Inter-Regular.ttf
var interRegularTTF []byte

//go:embed fonts/Inter-SemiBold.ttf
var interSemiBoldTTF []byte

//go:embed fonts/Geist-Regular.ttf
var geistRegularTTF []byte

//go:embed fonts/Geist-SemiBold.ttf
var geistSemiBoldTTF []byte

//go:embed fonts/IBMPlexSans-Regular.ttf
var ibmPlexSansRegularTTF []byte

//go:embed fonts/IBMPlexSans-SemiBold.ttf
var ibmPlexSansSemiBoldTTF []byte

type paletteFontKey struct {
	palette string
	uiFont  string
}

var (
	bundledUIFontsOnce sync.Once
	activeUIFont       atomic.Value // stores string preset key
	themedFontConfigs  sync.Map     // paletteFontKey -> *theme.Config
)

func init() {
	activeUIFont.Store(uiFontSystem)
}

func ensureBundledUIFonts() {
	bundledUIFontsOnce.Do(func() {
		for _, font := range []struct {
			family string
			data   []byte
		}{
			{"SuperGit Inter", interRegularTTF},
			{"SuperGit Inter", interSemiBoldTTF},
			{"SuperGit Geist", geistRegularTTF},
			{"SuperGit Geist", geistSemiBoldTTF},
			{"SuperGit IBM Plex Sans", ibmPlexSansRegularTTF},
			{"SuperGit IBM Plex Sans", ibmPlexSansSemiBoldTTF},
		} {
			_ = ui.RegisterFont(font.data, font.family)
		}
	})
}

func uiFontKeyOf(key string) string {
	if key == "plex" {
		return uiFontIBMPlexSans
	}
	for _, opt := range uiFontOptions {
		if opt.Key == key {
			return opt.Key
		}
	}
	return uiFontSystem
}

func uiFontFamilyFor(key string) string {
	key = uiFontKeyOf(key)
	if key == uiFontSystem {
		return ""
	}
	ensureBundledUIFonts()
	for _, opt := range uiFontOptions {
		if opt.Key == key {
			return opt.Family
		}
	}
	return ""
}

func setUIFontKey(key string) {
	activeUIFont.Store(uiFontKeyOf(key))
}

func uiFontKeyValue() string {
	if v, ok := activeUIFont.Load().(string); ok && v != "" {
		return uiFontKeyOf(v)
	}
	return uiFontSystem
}

// uiFontSetting omits the system default from the persisted settings file.
func uiFontSetting(key string) string {
	key = uiFontKeyOf(key)
	if key == uiFontSystem {
		return ""
	}
	return key
}

func (a *app) uiFontKey() string {
	if a != nil && a.uiFont != "" {
		return uiFontKeyOf(a.uiFont)
	}
	return uiFontKeyValue()
}

func (a *app) uiFontFamily() string {
	return uiFontFamilyFor(a.uiFontKey())
}

func (a *app) uiFontOptionLabel(opt uiFontOption) string {
	if opt.LabelKey != "" {
		return a.t(opt.LabelKey)
	}
	return opt.Label
}

func (a *app) currentUIFontOption() uiFontOption {
	key := a.uiFontKey()
	for _, opt := range uiFontOptions {
		if opt.Key == key {
			return opt
		}
	}
	return uiFontOptions[0]
}

// themeConfigWithUIFont returns a cached theme.Config for the active palette
// and UI font family. Creating a dedicated Config per pair keeps theme.TokensOf
// pointer lookups intact while propagating Theme.Font across windows and overlays.
func themeConfigWithUIFont(palette, fontKey string) *theme.Config {
	fontKey = uiFontKeyOf(fontKey)
	base := themeConfig(palette)
	if fontKey == uiFontSystem {
		return base
	}
	family := uiFontFamilyFor(fontKey)
	k := paletteFontKey{palette: palette, uiFont: fontKey}
	if cached, ok := themedFontConfigs.Load(k); ok {
		return cached.(*theme.Config)
	}
	cfg := theme.NewConfig(*base.Tokens(false), *base.Tokens(true))
	cfg.Theme(false).Font = family
	cfg.Theme(true).Font = family
	actual, _ := themedFontConfigs.LoadOrStore(k, cfg)
	return actual.(*theme.Config)
}

func (a *app) uiFontFamilySetting(c *ui.Context) {
	tk := theme.TokensOf(c)
	current := a.currentUIFontOption()
	currentLabel := a.uiFontOptionLabel(current)
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		display.Text(c, a.t("uiFontFamily"), display.TextProps{FontSize: scaledFont(12)}).Width(96).Shrink(0)
		items := make([]overlay.MenuItem, 0, len(uiFontOptions))
		for _, opt := range uiFontOptions {
			item := overlay.MenuItem{Key: opt.Key, Label: a.uiFontOptionLabel(opt)}
			if opt.Key == current.Key {
				item.Extra = "✓"
			}
			items = append(items, item)
		}
		chosen := overlay.DropdownMenu(c, &a.uiFontOpen, overlay.DropdownMenuProps{Items: items}, func(trigger ui.Element) {
			styleWorkbenchButton(c, trigger, buttonNormal, false, a.uiFontOpen)
			trigger.Role(ui.RoleButton).Label(a.t("uiFontFamily")).Description(currentLabel)
			trigger.Gap(6).Children(func() {
				label := ui.Text(c, currentLabel).FontSize(scaledFont(12)).SingleLine()
				if current.Family != "" {
					label.Font(current.Family)
				}
				ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(11)).TextColor(tk.TextTertiary)
			})
		})
		if chosen != "" && chosen != current.Key {
			resolved := uiFontKeyOf(chosen)
			a.uiFont = resolved
			setUIFontKey(resolved)
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
	})
}

// fontPreviewBox renders a compact live preview of both the active UI font and
// the active monospace code font side by side in the appearance settings card.
func (a *app) fontPreviewBox(c *ui.Context) {
	tk := theme.TokensOf(c)
	uiFamily := a.uiFontFamily()
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(96).Shrink(0)
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Padding(8, 10).Gap(6).Radius(6).
			Background(tk.BgLayout).Border(1, tk.BorderSecondary).Children(func() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				sample := ui.Text(c, a.t("uiFontPreviewSample")).FontSize(scaledFont(12)).TextColor(tk.Text).SingleLine().Grow(1).Basis(0).MinWidth(0)
				if uiFamily != "" {
					sample.Font(uiFamily)
				}
				stamp := ui.Text(c, "2026-11-22 00:08").FontSize(scaledFont(11)).TextColor(tk.TextTertiary).SingleLine().Shrink(0)
				if uiFamily != "" {
					stamp.Font(uiFamily)
				}
			})
			ui.Box(c).FillWidth().Height(1).Background(tk.BorderSecondary)
			ui.Text(c, `func diff(a, b int) bool { return a != b && a <= 0xFF }`).
				Font(a.patchFontFamily()).FontSize(a.patchFontSize()).TextColor(tk.Text).SingleLine()
		})
	})
}
