package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
)

// Copy the complete displayed date on which the menu opened. Refreshing the
// summary must not silently change the target of an already-open menu.
func (a *app) commitDateMenu(c *ui.Context, date ui.Element, text string) {
	open := ui.Local(date, "contextMenu", func() bool { return false })
	captured := ui.Local(date, "copy-date", func() string { return "" })
	if !*open {
		*captured = text
	}
	if overlay.ContextMenu(c, date, overlay.DropdownMenuProps{Items: []overlay.MenuItem{
		{Key: "copy-date", Label: a.t("copyDate"), Icon: lucide.Copy(), Disabled: *captured == ""},
	}}) == "copy-date" {
		a.copyText(*captured)
	}
}
