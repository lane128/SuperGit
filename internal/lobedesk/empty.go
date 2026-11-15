package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
)

// sectionEmptyState is the message a section shows when it has no rows: one
// line naming the state and, where the reader can act, a second line saying
// how rows get there. Everything is secondary ink so the message reads as a
// note under the section header rather than as content.
func sectionEmptyState(c *ui.Context, title, hint string, size float32) {
	display.Text(c, title, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(size)}).
		Padding(12, 12, 2, 12)
	if hint != "" {
		display.Text(c, hint, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(size), Rows: 2}).
			Padding(2, 12, 12, 12)
	}
}

// listEmptyState is the message a list pane shows when it has no rows: an
// icon, a title and a sentence of guidance. The file list and the stash list
// both use it, so an empty pane reads the same wherever it appears: centred
// across the pane and set at the top, where its first row would be.
//
// The ink is passed in because the two cases mean different things: a clean
// work tree is a success state and wears the success color, while an empty
// stash list is neutral.
func listEmptyState(c *ui.Context, icon *ui.SVG, ink ui.Color, title, hint string) {
	ui.Column(c).FillWidth().Padding(18).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon).FontSize(scaledFont(22)).TextColor(ink)
		display.Text(c, title, display.TextProps{Strong: true, FontSize: scaledFont(12)})
		if hint != "" {
			display.Text(c, hint, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(11), Rows: 3}).
				TextAlign(ui.Center).MaxWidth(360)
		}
	})
}
