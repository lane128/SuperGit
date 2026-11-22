package lobedesk

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Shared by rendered controls and virtual placeholder height estimates.
const workbenchButtonBaseHeight float32 = 29

type buttonRole uint8

const (
	buttonNormal buttonRole = iota
	buttonPrimary
	buttonDanger
	buttonDangerIcon
)

// workbenchButtonColors implements the Plan A+ GitHub Primer button grammar,
// separating primary green CTAs and danger fills from neutral surface controls
// and blue selection states.
func workbenchButtonColors(tk *theme.Tokens, role buttonRole, hovered, pressed, disabled, selected bool) (ui.Color, ui.Color) {
	if disabled {
		if role == buttonPrimary {
			if tk.Dark {
				return ui.Hex("#1A4329"), ui.Hex("#9EC7AB")
			}
			return ui.Hex("#94D3A2"), ui.Hex("#FFFFFF")
		}
		if tk.Dark {
			return ui.Hex("#1B2027"), ui.Hex("#8B949E")
		}
		return ui.Hex("#F6F8FA"), ui.Hex("#656D76")
	}
	if selected {
		if tk.Dark {
			return ui.Hex("#1B2D45"), ui.Hex("#79C0FF")
		}
		return ui.Hex("#EDF4FF"), ui.Hex("#0969DA")
	}
	colors := [3]string{"#F6F8FA", "#F3F4F6", "#EBECF0"}
	ink := tk.Text
	if tk.Dark {
		colors = [3]string{"#21262D", "#30363D", "#282E33"}
	}
	switch role {
	case buttonPrimary:
		colors, ink = [3]string{"#1F883D", "#1A7F37", "#187733"}, ui.Hex("#FFFFFF")
		if tk.Dark {
			colors = [3]string{"#1F7E33", "#238636", "#1A6D2C"}
		}
	case buttonDangerIcon:
		colors, ink = [3]string{"#FFEBE9", "#FFE0DD", "#FFD4D0"}, ui.Hex("#B61D28")
		if tk.Dark {
			colors, ink = [3]string{"#321F24", "#41262C", "#4F2C33"}, ui.Hex("#FF7B72")
		}
	case buttonDanger:
		colors, ink = [3]string{"#CF222E", "#B61D28", "#A41923"}, ui.Hex("#FFFFFF")
		if tk.Dark {
			colors = [3]string{"#C9312E", "#DA3633", "#B22927"}
		}
	}
	state := 0
	if pressed {
		state = 2
	} else if hovered {
		state = 1
	}
	return ui.Hex(colors[state]), ink
}

func workbenchButtonBorder(tk *theme.Tokens, role buttonRole, hovered, pressed, disabled, selected bool) ui.Color {
	if disabled {
		if role == buttonPrimary {
			if tk.Dark {
				return ui.Hex("#3FB950").Alpha(0.18)
			}
			return ui.Hex("#1F883D").Alpha(0.14)
		}
		if tk.Dark {
			return ui.Hex("#252B33")
		}
		return ui.Hex("#E5E8EC")
	}
	if selected {
		if tk.Dark {
			return ui.Hex("#58A6FF").Alpha(0.36)
		}
		return ui.Hex("#0969DA").Alpha(0.34)
	}
	switch role {
	case buttonPrimary, buttonDanger:
		if tk.Dark {
			return ui.Hex("#F0F6FC").Alpha(0.12)
		}
		return ui.Hex("#1F2328").Alpha(0.15)
	case buttonDangerIcon:
		if tk.Dark {
			return ui.Hex("#F85149").Alpha(0.28)
		}
		return ui.Hex("#CF222E").Alpha(0.24)
	default:
		if tk.Dark {
			if hovered || pressed {
				return ui.Hex("#545D68")
			}
			return ui.Hex("#363D45")
		}
		if hovered || pressed {
			return ui.Hex("#BAC2CA")
		}
		return ui.Hex("#D0D7DE")
	}
}

// styleWorkbenchButton works on ButtonBase and native menu triggers alike.
// Callers pass disabled before styling so its readable fill is applied without
// the SDK's additional half-opacity. Native focus, click and menu semantics stay.
func styleWorkbenchButton(c *ui.Context, b ui.Element, role buttonRole, disabled, selected bool) ui.Element {
	tk := theme.TokensOf(c)
	b.Disabled(disabled).DimDisabled(false)
	hovered, pressed, isDisabled := b.Hovered(), b.Pressed(), b.IsDisabled()
	bg, ink := workbenchButtonColors(tk, role, hovered, pressed, isDisabled, selected)
	border := workbenchButtonBorder(tk, role, hovered, pressed, isDisabled, selected)
	focus := ui.Hex("#0969DA")
	if tk.Dark {
		focus = ui.Hex("#58A6FF")
	}
	if !isDisabled {
		switch role {
		case buttonPrimary, buttonDanger:
			b.InsetShadow(0, 1, 0, 0, ui.Hex("#FFFFFF").Alpha(0.10))
		case buttonNormal:
			if !selected && !pressed {
				sheen := ui.Hex("#FFFFFF").Alpha(0.55)
				if tk.Dark {
					sheen = ui.Hex("#FFFFFF").Alpha(0.04)
				}
				b.InsetShadow(0, 1, 0, 0, sheen)
			}
		}
	}
	return b.Height(scaledFont(workbenchButtonBaseHeight)).PaddingX(10).Gap(6).Radius(6).
		FontSize(scaledFont(12)).FontWeight(500).
		Background(bg).Border(1, border).TextColor(ink).
		FocusRing(true).FocusRingColor(focus).Cursor(ui.CursorPointer).
		Transition(ui.ElementTransition{Duration: 100 * time.Millisecond, Colors: true})
}

func workbenchButton(c *ui.Context, label string, role buttonRole, disabled bool) ui.Element {
	b := styleWorkbenchButton(c, ui.ButtonBase(c), role, disabled, false)
	b.Children(func() { ui.Text(c, label).SingleLine() })
	return b
}

func workbenchIconLabelButton(c *ui.Context, icon *ui.SVG, label string, role buttonRole, disabled bool) ui.Element {
	b := styleWorkbenchButton(c, ui.ButtonBase(c).Label(label), role, disabled, false)
	b.Children(func() {
		ui.Icon(c, icon).FontSize(scaledFont(14)).Shrink(0)
		ui.Text(c, label).SingleLine()
	})
	return b
}

func workbenchIconButton(c *ui.Context, icon *ui.SVG, title string, role buttonRole, disabled bool) ui.Element {
	b := styleWorkbenchButton(c, ui.ButtonBase(c), role, disabled, false).
		Width(scaledFont(workbenchButtonBaseHeight)).PaddingX(0).Label(title).Tooltip(title)
	b.Children(func() { ui.Icon(c, icon).FontSize(scaledFont(15)) })
	return b
}

// compactActionIcon keeps the native square target and its tooltip. The glyph
// and its soft surface share the same hover, pressed and disabled state.
func compactActionIcon(c *ui.Context, icon *ui.SVG, title string, role buttonRole, disabled bool) ui.Element {
	b := ui.ButtonBase(c).Disabled(disabled).DimDisabled(false)
	action.StyleActionIcon(c, b, icon, action.ActionIconProps{
		Title: title, BlockSize: 24, GlyphSize: 14,
	})
	tk := theme.TokensOf(c)
	hovered, pressed, isDisabled := b.Hovered(), b.Pressed(), b.IsDisabled()
	bg, ink := workbenchButtonColors(tk, role, hovered, pressed, isDisabled, false)
	border := workbenchButtonBorder(tk, role, hovered, pressed, isDisabled, false)
	return b.Background(bg).Border(1, border).TextColor(ink).Radius(6).FocusRing(true).FocusRingColor(tk.Blue)
}

// Paint a single selection behind both the tab and its close target. The
// inset belongs to the paint, so the scroll extent and hit boxes stay fixed.
func paintWorkbenchTab(tab ui.Element, tk *theme.Tokens, active bool, verticalInset float32) ui.Color {
	if !active && !tab.Hovered() {
		return tk.TextSecondary
	}
	bg, ink := workbenchButtonColors(tk, buttonNormal, true, false, false, active)
	tab.Draw(func(p *ui.Painter, r ui.Rect) {
		r.Y += verticalInset
		r.H -= 2 * verticalInset
		p.Fill(r, bg, 6)
		if active {
			p.Stroke(r, tk.Blue.Alpha(0.22), 6, 1)
		}
	})
	return ink
}

func styleTabClose(b ui.Element, tk *theme.Tokens, ink ui.Color) {
	bg := ui.Transparent
	if b.IsDisabled() {
		_, ink = workbenchButtonColors(tk, buttonNormal, false, false, true, false)
	} else if b.Hovered() || b.Pressed() {
		bg, _ = workbenchButtonColors(tk, buttonNormal, b.Hovered(), b.Pressed(), false, false)
	}
	b.Radius(5).Background(bg).TextColor(ink).FocusRing(true).FocusRingColor(tk.Blue)
}
