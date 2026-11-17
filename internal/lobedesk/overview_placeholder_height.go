package lobedesk

import "github.com/egoist/mygo/ui"

// overviewSectionHeightFor adds wrapping placeholder heights to the fixed patch
// heights. Conflict guidance retains ordinary layout because it has more than
// one independent text/action block.
func (a *app) overviewSectionHeightFor(c *ui.Context, entry worktreeOverviewEntry, width float32) (float32, bool) {
	if height, exact := a.overviewSectionHeight(entry); exact {
		return height, true
	}
	if _, conflict := a.conflictPreviewEntry(entry.Key.Path); conflict || width <= 24 {
		return 0, false
	}
	message := a.t("diffEmpty")
	errorBody := false
	switch {
	case entry.Loading:
		message = a.t("diffLoading")
	case entry.Err != "":
		message, errorBody = entry.Err, true
	}
	// Measure with the same base size as display.Text, not a smaller span
	// inside the surrounding theme's larger default line metrics.
	previous := c.Theme()
	measuring := *previous
	measuring.FontSize = scaledFont(11)
	c.SetTheme(&measuring)
	_, height := c.MeasureText(width-24, ui.Span{Text: message})
	if errorBody {
		_, line := c.MeasureText(0, ui.Span{Text: "Mg"})
		height = min(height, 4*line)
		// Error body: four-line text, six-DIP gap and shared Retry height.
		height += 6 + scaledFont(workbenchButtonBaseHeight)
	}
	c.SetTheme(previous)
	return 35 + 24 + height, true
}
