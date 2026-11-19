package lobedesk

import (
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
	fileRowDirSize  float32 = 9
)

// fileRowHeight is one file row at the reader's font scale; the gap under it
// stays a point, as the list's own spacing does.
func fileRowHeight() float32 { return scaledFont(fileRowBaseH) }

// fileRowFolderMin is the row width below which the folder is left out of a
// file row: under it the name needs the room more than the folder does.
const fileRowFolderMin = 260

// fileRowShowsFolder reports whether a row that many points wide has room for
// the folder beside the file name.
func fileRowShowsFolder(room float32) bool { return room >= fileRowFolderMin }

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

// fileRowPath keeps the current name primary and its directory secondary.
// When space permits, a rename source precedes the current name with an
// arrow; it does not replace the destination directory. Narrow rows keep
// the primary name. The row's label already carries the complete path.
func fileRowPath(c *ui.Context, row ui.Element, path, originalPath string, selected bool) {
	name, dir := splitRepoPath(path)
	room := row.Bounds().W
	ui.Row(c).Grow(1).Basis(0).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
		if originalPath != "" && fileRowShowsFolder(room) {
			ui.Row(c).Shrink(0).MaxWidthPercent(30).MinWidth(0).Gap(4).AlignItems(ui.Center).Children(func() {
				display.Text(c, originalPath, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(fileRowDirSize)}).
					SingleLine().Grow(1).Basis(0).MinWidth(0).AccessHidden(true)
				display.Text(c, "→", display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(fileRowDirSize)}).
					SingleLine().Shrink(0).AccessHidden(true)
			})
		}
		display.Text(c, name, display.TextProps{FontSize: scaledFont(fileRowNameSize), Strong: selected}).
			SingleLine().Grow(1).Basis(0).MinWidth(0).AccessHidden(true)
		if dir != "" && fileRowShowsFolder(room) {
			maxWidth := float32(40)
			if originalPath != "" {
				maxWidth = 20
			}
			display.Text(c, dir, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(fileRowDirSize)}).
				SingleLine().Shrink(0).MaxWidthPercent(maxWidth).AccessHidden(true)
		}
	})
}
