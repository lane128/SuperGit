package lobedesk

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// Shared by rendered controls and virtual placeholder height estimates.
const workbenchButtonBaseHeight float32 = 29

type buttonRole uint8

const (
	buttonNormal buttonRole = iota
	buttonPrimary
	buttonDanger
)

// workbenchButtonColors separates action fills from selection and status inks.
// Selection blue must never turn green when the primary action is styled.
func workbenchButtonColors(tk *theme.Tokens, role buttonRole, hovered, pressed, disabled, selected bool) (ui.Color, ui.Color) {
	if disabled {
		if tk.Dark {
			return ui.Hex("#252A33"), ui.Hex("#8996AA")
		}
		return ui.Hex("#EFF2F6"), ui.Hex("#627084")
	}
	if selected {
		if tk.Dark {
			return ui.Hex("#213653"), ui.Hex("#8CB4FF")
		}
		return ui.Hex("#E8F0FF"), ui.Hex("#2456BD")
	}
	colors := [3]string{"#E8EDF5", "#DCE5F2", "#D2DDEE"}
	ink := tk.Text
	if tk.Dark {
		colors = [3]string{"#2C333F", "#384252", "#424F62"}
	}
	switch role {
	case buttonPrimary:
		colors, ink = [3]string{"#176840", "#1B7547", "#125736"}, ui.Hex("#FFFFFF")
		if tk.Dark {
			colors = [3]string{"#196C43", "#1B7448", "#145C39"}
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
	return b.Height(scaledFont(workbenchButtonBaseHeight)).PaddingX(10).Gap(6).Radius(6).
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
	b.Children(func() { ui.Icon(c, icon).FontSize(scaledFont(14)) })
	return b
}
