package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const (
	diffPaletteClassic = "git-classic"
	diffPaletteDelta   = "delta-layered"
	diffPaletteVivid   = "vivid"
)

type diffSemanticColors struct{ background, text, gutter ui.Color }
type diffPaletteOption struct {
	key, label  string
	light, dark [2]diffSemanticColors // addition, deletion
}

func diffSemantic(bg, text, gutter string) diffSemanticColors {
	return diffSemanticColors{ui.Hex(bg), ui.Hex(text), ui.Hex(gutter)}
}

var diffPaletteOptions = []diffPaletteOption{
	{diffPaletteClassic, "diffPaletteClassic",
		[2]diffSemanticColors{diffSemantic("#eaf6ed", "#14552c", "#14552c"), diffSemantic("#fceceb", "#8a2030", "#8a2030")},
		[2]diffSemanticColors{diffSemantic("#182b20", "#97e5a8", "#97e5a8"), diffSemantic("#342023", "#ffa6aa", "#ffa6aa")}},
	{diffPaletteDelta, "diffPaletteDelta",
		[2]diffSemanticColors{diffSemantic("#d9efdf", "#173c26", "#165b2c"), diffSemantic("#f8dce0", "#59242c", "#8c2338")},
		[2]diffSemanticColors{diffSemantic("#203e2c", "#d9f5e1", "#93e9ac"), diffSemantic("#4a2830", "#ffe1e5", "#ffb0be")}},
	{diffPaletteVivid, "diffPaletteVivid",
		[2]diffSemanticColors{diffSemantic("#c7ebd1", "#103e23", "#0d5229"), diffSemantic("#f7cbd0", "#641a2b", "#8c1832")},
		[2]diffSemanticColors{diffSemantic("#204f35", "#c6ffd6", "#a0ffbc"), diffSemantic("#602c3a", "#ffe1e8", "#ffc0d0")}},
}

func diffPaletteOptionFor(key string) diffPaletteOption {
	for _, option := range diffPaletteOptions {
		if option.key == key {
			return option
		}
	}
	return diffPaletteOptions[1]
}
func (a *app) diffPaletteKey() string { return diffPaletteOptionFor(a.diffPalette).key }

type diffLineColors struct{ background, text, gutter, selection ui.Color }

// Diff colors are local to the patch; changing them never recolors Git actions,
// file badges, or the application's success/error notifications.
func (a *app) diffColors(tk *theme.Tokens, kind string) diffLineColors {
	colors := diffLineColors{background: tk.BgContainer, text: tk.Text, selection: ui.Hex("#075cca")}
	if tk.Dark {
		colors.selection = ui.Hex("#9ccaff")
	}
	switch kind {
	case "add", "del":
		option := diffPaletteOptionFor(a.diffPalette)
		pair := option.light
		if tk.Dark {
			pair = option.dark
		}
		index := 0
		if kind == "del" {
			index = 1
		}
		colors.background, colors.text, colors.gutter = pair[index].background, pair[index].text, pair[index].gutter
	case "hunk":
		colors.background, colors.text = ui.Hex("#e5eef8"), ui.Hex("#254c70")
		if tk.Dark {
			colors.background, colors.text = ui.Hex("#25364a"), ui.Hex("#a8d2ff")
		}
	case "meta":
		colors.background = tk.FillQuaternary.Over(tk.BgContainer)
		colors.text = capsuleInk(tk.TextSecondary, colors.background, 4.5)
	}
	if kind != "add" && kind != "del" {
		colors.gutter = capsuleInk(tk.TextTertiary, colors.background, 4.5)
	}
	colors.selection = capsuleInk(colors.selection, colors.background, 3)
	return colors
}

// Draw selection over the row without washing out its semantic background.
// DrawOver leaves layout, hit testing, text selection and copying untouched.
func (a *app) paintDiffLine(row ui.Element, tk *theme.Tokens, kind string, selected, current bool) diffLineColors {
	colors := a.diffColors(tk, kind)
	row.Background(colors.background)
	hovered := row.Hovered()
	row.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if kind == "add" || kind == "del" {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 3, H: r.H}, colors.gutter, 0)
		}
		if !selected && !hovered {
			return
		}
		ink, thickness := colors.gutter, float32(1)
		if selected {
			ink = colors.selection
			if current {
				thickness = 2
			}
		}
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: thickness}, ink, 0)
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - thickness, W: r.W, H: thickness}, ink, 0)
		if selected {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: thickness, H: r.H}, ink, 0)
			p.Fill(ui.Rect{X: r.X + r.W - thickness, Y: r.Y, W: thickness, H: r.H}, ink, 0)
		}
	})
	return colors
}

func diffSearchColors(dark, active bool) (background, text ui.Color) {
	if dark {
		if active {
			return ui.Hex("#ffd26a"), ui.Hex("#2a210c")
		}
		return ui.Hex("#64501f"), ui.Hex("#fff2cb")
	}
	if active {
		return ui.Hex("#ffe08a"), ui.Hex("#352600")
	}
	return ui.Hex("#fff0c2"), ui.Hex("#352600")
}

func (a *app) diffPaletteRow(c *ui.Context) {
	tk := theme.TokensOf(c)
	current := diffPaletteOptionFor(a.diffPalette)
	settingsFormRow(c, a.t("diffPalette"), func() {
		items := make([]overlay.MenuItem, 0, len(diffPaletteOptions))
		for _, option := range diffPaletteOptions {
			item := overlay.MenuItem{Key: option.key, Label: a.t(option.label)}
			if option.key == current.key {
				item.Extra = "✓"
			}
			items = append(items, item)
		}
		chosen := overlay.DropdownMenu(c, &a.diffPaletteOpen, overlay.DropdownMenuProps{Items: items}, func(trigger ui.Element) {
			styleWorkbenchButton(c, trigger, buttonNormal, false, a.diffPaletteOpen)
			trigger.Role(ui.RoleButton).Label(a.t("diffPalette")).Description(a.t(current.label))
			trigger.Gap(6).MaxWidth(252).Shrink(1).Children(func() {
				pair := current.light
				if tk.Dark {
					pair = current.dark
				}
				for _, color := range pair {
					ui.Box(c).Size(6, 6).Radius(3).Background(color.gutter).AccessHidden(true)
				}
				ui.Text(c, a.t(current.label)).FontSize(scaledFont(12)).SingleLine().Shrink(1).MinWidth(0)
				ui.Icon(c, lucide.ChevronDown()).FontSize(scaledFont(11)).TextColor(tk.TextTertiary)
			})
		})
		if chosen != "" && chosen != current.key {
			a.diffPalette = diffPaletteOptionFor(chosen).key
			a.saveSettingsFor(toastSettings)
			a.invalidate()
		}
	})
}
