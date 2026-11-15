package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// selectionBg is the tint that marks the selected row. Every list uses it --
// files, stashes, refs, branches, worktrees and the history rows -- so the
// same blue means "this is the selection" wherever the reader is. Hover stays
// neutral (FillTertiary), so hover and selection never trade places.
//
// surface is what the tint sits on: the page colour for lists, BgElevated
// inside popups. The tint is flattened against it because several inks are
// checked against the selected row's real color.
func selectionBg(tk *theme.Tokens, surface ui.Color) ui.Color {
	return tk.Blue.Alpha(0.14).Over(surface)
}

// tertiaryTextInk keeps small tertiary labels above 4.5:1 on the surface they
// sit on. Catppuccin Latte's Subtext0 token is 4.37:1 on the base surface and
// 3.65:1 on the selected-row tint, and Frappe's reaches only 4.28:1 on that
// tint, all below AA for the sidebar's 10-point counts and shortcut hints. The
// capsule ink search deepens the color on a light ground and lifts it toward
// white on a dark one, leaving a color that already clears the ratio alone.
func tertiaryTextInk(tk *theme.Tokens, ground ui.Color) ui.Color {
	return capsuleInk(tk.TextTertiary, ground, 4.5)
}
