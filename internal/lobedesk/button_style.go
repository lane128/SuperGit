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

// workbenchButtonColors separates action fills from selection and status inks.
// Selection blue must never turn green when the primary action is styled.
func workbenchButtonColors(tk *theme.Tokens, role buttonRole, hovered, pressed, disabled, selected bool) (ui.Color, ui.Color) {
	if disabled {
		if tk.Dark {
			return ui.Hex("#2C3040"), ui.Hex("#A6ADC0")
		}
		return ui.Hex("#E2E6ED"), ui.Hex("#596578")
	}
	if selected {
		if tk.Dark {
			return ui.Hex("#374766"), ui.Hex("#E5EBFF")
		}
		return ui.Hex("#D6E1F7"), ui.Hex("#244E9C")
	}
	colors := [3]string{"#DEE4EF", "#D2DCEC", "#C6D3E7"}
	ink := tk.Text
	if tk.Dark {
		colors = [3]string{"#333A4C", "#3D455B", "#464F67"}
	}
	switch role {
	case buttonPrimary:
		colors, ink = [3]string{"#115534", "#196641", "#0D442B"}, ui.Hex("#FFFFFF")
		if tk.Dark {
			colors = [3]string{"#2B5E40", "#356E4C", "#254F36"}
		}
	case buttonDangerIcon:
		colors, ink = [3]string{"#F3E4E9", "#ECD6DD", "#E4C8D0"}, ui.Hex("#94304A")
		if tk.Dark {
			colors, ink = [3]string{"#41333F", "#513A47", "#60414E"}, ui.Hex("#FFADB8")
		}
	case buttonDanger:
		colors, ink = [3]string{"#A53A47", "#AF3B4A", "#8D2D38"}, ui.Hex("#FFFFFF")
		if tk.Dark {
			colors = [3]string{"#A44451", "#AC4655", "#923642"}
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

// styleWorkbenchButton works on ButtonBase and native menu triggers alike.
// Callers pass disabled before styling so its readable fill is applied without
// the SDK's additional half-opacity. Native focus, click and menu semantics stay.
func styleWorkbenchButton(c *ui.Context, b ui.Element, role buttonRole, disabled, selected bool) ui.Element {
	tk := theme.TokensOf(c)
	b.Disabled(disabled).DimDisabled(false)
	bg, ink := workbenchButtonColors(tk, role, b.Hovered(), b.Pressed(), b.IsDisabled(), selected)
	focus := ui.Hex("#2A66DC")
	if tk.Dark {
		focus = ui.Hex("#79A5FF")
	}
	if role == buttonPrimary && !disabled {
		b.InsetShadow(0, 1, 0, 0, ui.Hex("#FFFFFF").Alpha(0.12))
	}
	return b.Height(scaledFont(workbenchButtonBaseHeight)).PaddingX(10).Gap(6).Radius(7).
		FontSize(scaledFont(12)).FontWeight(500).
		Background(bg).Border(0, ui.Transparent).TextColor(ink).
		FocusRing(true).FocusRingColor(focus).Cursor(ui.CursorPointer).
		Transition(ui.ElementTransition{Duration: 100 * time.Millisecond, Colors: true})
}

func workbenchButton(c *ui.Context, label string, role buttonRole, disabled bool) ui.Element {
	b := styleWorkbenchButton(c, ui.ButtonBase(c), role, disabled, false)
	b.Children(func() { ui.Text(c, label).SingleLine() })
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
		Title: title, BlockSize: 24, GlyphSize: 15,
	})
	tk := theme.TokensOf(c)
	bg, ink := workbenchButtonColors(tk, role, b.Hovered(), b.Pressed(), b.IsDisabled(), false)
	return b.Background(bg).TextColor(ink).Radius(6).FocusRing(true).FocusRingColor(tk.Blue)
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
