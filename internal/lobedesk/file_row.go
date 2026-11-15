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
	fileRowBaseH    float32 = 26
	fileRowGap      float32 = 1
	fileRowNameSize float32 = 11
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

// fileRowPath draws the row's one line as the path would be written: the file
// name, then the folder it sits in. The name takes whatever the folder leaves
// and ends in an ellipsis when that is not enough; the folder is dropped in a
// pane too narrow to hold both, where it would only take room from the name.
// The row's label already names the path, so neither text is read again.
func fileRowPath(c *ui.Context, row ui.Element, path, originalPath string, selected bool) {
	name, dir := splitRepoPath(path)
	if originalPath != "" {
		dir = originalPath + " →"
	}
	room := row.Bounds().W
	ui.Row(c).Grow(1).Basis(0).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
		display.Text(c, name, display.TextProps{FontSize: scaledFont(fileRowNameSize), Strong: selected}).
			SingleLine().Grow(1).Basis(0).MinWidth(0).AccessHidden(true)
		if dir != "" && fileRowShowsFolder(room) {
			display.Text(c, dir, display.TextProps{Type: display.TextSecondary, FontSize: scaledFont(fileRowDirSize)}).
				SingleLine().Shrink(0).MaxWidthPercent(40).AccessHidden(true)
		}
	})
}
