package lobedesk

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// The working tree's files and a commit's files are the same kind of row in
// the same 300-point pane, so they are drawn by the same code. Two densities
// there made the reader re-learn the list when they moved between the working
// tree and a commit, and the working tree's fixed 30-point row stayed put
// while the commit's followed the font preset, so at the large preset the
// working tree's rows were the tighter of the two.
const (
	fileRowBaseH    float32 = 29
	fileRowGap      float32 = 1
	fileRowNameSize float32 = 12
)

// fileRowHeight is one file row at the reader's font scale; the gap under it
// stays a point, as the list's own spacing does.
func fileRowHeight() float32 { return scaledFont(fileRowBaseH) }

// fileRowShell builds the button both file lists start from: named by the
// file's path, one line high, with the selected and hover grounds. It is a
// button so it keeps Tab, Shift+F10 and the accessibility role. Each list adds
// its own description and tooltip: the working tree's rows also say which
// section they belong to, and a tooltip set here would be read as that
// description before the list could say it.
func fileRowShell(c *ui.Context, tk *theme.Tokens, path string, selected bool) ui.Element {
	row := ui.ButtonBase(c).Label(path).Row().FillWidth().Height(fileRowHeight()).Padding(0, 6).
		Gap(8).Radius(tk.BorderRadius).AlignItems(ui.Center).
		Role(ui.RoleOption).AccessSelected(selected)
	switch {
	case selected:
		row.Background(selectionBg(tk, tk.BgContainer))
	case row.Hovered():
		row.Background(tk.FillTertiary)
	}
	return row
}

// Paths use the same repository-relative spelling as the patch headers.
func fileRowPath(c *ui.Context, path string, selected bool) {
	slot := ui.Row(c).Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Center)
	slot.Children(func() {
		size := scaledFont(fileRowNameSize)
		weight := 400
		if selected {
			weight = 600
		}
		label := path
		if width := slot.Bounds().W; width > 0 {
			// Drop whole leading directories first so the filename stays readable.
			for start := 0; ; {
				measured, _ := c.MeasureText(0, ui.Span{Text: label, Size: size, Weight: weight})
				if measured <= width {
					break
				}
				slash := strings.IndexByte(path[start:], '/')
				if slash < 0 {
					break
				}
				start += slash + 1
				label = "…/" + path[start:]
			}
		}
		display.Text(c, label, display.TextProps{FontSize: size, Strong: selected}).SingleLine().FillWidth().AccessHidden(true)
	})
}
