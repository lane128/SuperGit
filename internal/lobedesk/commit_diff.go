package lobedesk

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

// The commit details diff shows every file the commit touched, one section per
// file. Sublime Merge's commit view shows a single file at a time; reading a
// change usually means jumping between the files it touches, so here they are
// stacked, each with a header that collapses it, and the changed-files list
// scrolls the diff to the file it names.
const (
	// diffFileHeaderH is one file section's header row.
	diffFileHeaderH float32 = 26
	// diffRowH is one line of a patch and diffHunkH a hunk header, matching
	// what the rows draw.
	diffRowH  float32 = 23
	diffHunkH float32 = 24
)

// commitDiffSection is one file's part of the stacked diff. Start and Lines
// place it in the combined diff the reader selects and copies from; Rows are
// the indexes that get drawn, which leave out the `diff --git`, `index` and
// `---`/`+++` lines the header already says.
type commitDiffSection struct {
	Binary bool
	Path   string
	Start  int
	Lines  int
	Hunks  int
	Rows   []int
	// Submodule marks a gitlink: the "160000" mode in the patch header. A
	// submodule pointer has no text at that revision, so it cannot be blamed.
	Submodule bool
	// NewFile marks the side of a type change that creates the path at this
	// commit. A gitlink replaced by a regular file leaves two sections under
	// one path, and only the new side says what can be read there now.
	NewFile bool
}

// commitHiddenSel is what a closed section hid from the selection: the rows
// it held and the cursor if it was one of them (-1 when it was not).
type commitHiddenSel struct {
	Rows   []int
	Cursor int
}

// resetCommitDiff drops the loaded patches and their sections together, so no
// frame draws rows of a commit that is no longer selected.
func (a *app) resetCommitDiff() {
	a.commitDiff = gitrepo.Diff{}
	a.commitSections = nil
	a.detailContexts.History.patchWidths = historyPatchWidthCache{}
	// Row indexes only mean anything against the patch they came from, so the
	// selections a closed section remembers go with it.
	a.commitCollapsedSel = nil
}

// buildCommitDiff joins the files' patches into one diff and records where each
// file starts. One diff means the line selection, copy and width all work on
// the whole commit instead of on one file.
func buildCommitDiff(files []gitrepo.FileDiff) (gitrepo.Diff, []commitDiffSection) {
	var combined gitrepo.Diff
	texts := make([]string, 0, len(files))
	sections := make([]commitDiffSection, 0, len(files))
	for _, file := range files {
		sec := commitDiffSection{
			Path: file.Path, Binary: file.Diff.Binary,
			Start: len(combined.Lines),
			Lines: len(file.Diff.Lines),
			Hunks: len(file.Diff.Hunks),
		}
		for i, line := range file.Diff.Lines {
			index := sec.Start + i
			combined.Lines = append(combined.Lines, line)
			if line.Kind == "meta" && strings.Contains(line.Text, " 160000") {
				sec.Submodule = true
			}
			if line.Kind == "meta" && strings.HasPrefix(strings.TrimSpace(line.Text), "new file mode ") {
				sec.NewFile = true
			}
			// The patch header says the same thing as the section header, in
			// paths taken from git's own vocabulary. A file whose patch has no
			// hunks (a binary change, a mode change) has nothing else, so its
			// meta rows stay.
			if line.Kind != "meta" || sec.Hunks == 0 {
				sec.Rows = append(sec.Rows, index)
			}
		}
		sections = append(sections, sec)
		texts = append(texts, file.Diff.Text)
	}
	combined.Text = strings.Join(texts, "")
	return combined, sections
}

// visibleCommitSections projects sections without renumbering their lines.
// A type change may contribute several sections for the same file.
func (a *app) visibleCommitSections() []commitDiffSection {
	tab := a.detailTabs().Active
	if tab.Summary {
		return a.commitSections
	}
	var out []commitDiffSection
	for _, sec := range a.commitSections {
		if sec.Path == tab.File.Path {
			out = append(out, sec)
		}
	}
	return out
}

type historyPatchWidthCache struct {
	first *gitrepo.DiffLine
	font  float32
	split bool
	rows  map[int]patchWidths
}

const historyPatchInset = 4

// Size the canvas from the file sections the active tab actually draws.
// Ordinary lines use the shared estimate. Measure unusually wide lines so
// their tails neither clip nor scroll entirely past the right edge.
func (a *app) historyPatchWidths(c *ui.Context, cache *historyPatchWidthCache, sections []commitDiffSection, lines []gitrepo.DiffLine) patchWidths {
	font := a.patchFontSize()
	if cache.first != &lines[0] || cache.font != font || cache.split != a.splitDiff {
		*cache = historyPatchWidthCache{first: &lines[0], font: font, split: a.splitDiff, rows: make(map[int]patchWidths)}
	}
	viewport := a.commitDiffViewport.W - 2*historyPatchInset
	widths := patchWidths{canvas: viewport}
	if a.splitDiff {
		widths.columns = max(280, viewport)
	} else if viewport <= 0 {
		widths.canvas = 560 // The first pass has no measured pane yet.
	}
	for _, section := range sections {
		widths.canvas = max(widths.canvas, diffEstimatedAdvance(section.Path)*max(font, scaledFont(10))+diffCodeReserve)
		if a.commitCollapsed[section.Path] {
			continue
		}
		sectionWidths, found := cache.rows[section.Start]
		if !found {
			for _, index := range section.Rows {
				line := lines[index]
				advance := diffEstimatedAdvance(line.Text) * font
				if advance > 4000 {
					advance, _ = c.MeasureText(0, ui.Span{Text: strings.ReplaceAll(line.Text, "\t", "    "), Font: "Menlo", Size: font})
				}
				if a.splitDiff && splitPatchCodeLine(line.Kind) {
					// History has no staging action at the end of a code row.
					sectionWidths.columns = max(sectionWidths.columns, 2*(advance+splitPatchReserve-diffHoverActionReserve))
				} else {
					sectionWidths.canvas = max(sectionWidths.canvas, advance+diffCodeReserve)
				}
			}
			cache.rows[section.Start] = sectionWidths
		}
		widths.canvas = max(widths.canvas, sectionWidths.canvas)
		widths.columns = max(widths.columns, sectionWidths.columns)
	}
	widths.canvas = max(widths.canvas, widths.columns)
	return widths
}

func (a *app) commitSectionPatchRows(lines []gitrepo.DiffLine, sec commitDiffSection) patchRows {
	if a.commitCollapsed[sec.Path] || len(sec.Rows) == 0 {
		return patchRows{}
	}
	return newPatchRows(lines, sec.Rows, a.splitDiff)
}

func (a *app) commitSectionHeight(sec commitDiffSection) float32 {
	if !a.commitCollapsed[sec.Path] && previewFor(sec.Path, sec.Binary) != textPreview {
		return diffFileHeaderH + previewSummaryHeight
	}
	height := diffFileHeaderH
	rows := a.commitSectionPatchRows(a.commitDiff.Lines, sec)
	for i := range rows.len() {
		height += rows.height(a, i)
	}
	return height
}

// commitDiffHeight is how tall the whole stacked diff draws: a header per file
// and the rows of every section that is open.
func (a *app) commitDiffHeight() float32 {
	height := float32(0)
	for _, sec := range a.visibleCommitSections() {
		height += a.commitSectionHeight(sec)
	}
	return height
}

// commitDiffOffset is where a file's header sits in the diff, or -1 when the
// commit does not touch that file. The file list uses it to scroll the diff to
// the file the reader picked.
func (a *app) commitDiffOffset(path string) float32 {
	y := float32(0)
	for _, sec := range a.commitSections {
		if sec.Path == path {
			return y
		}
		y += a.commitSectionHeight(sec)
	}
	return -1
}

// toggleCommitSection collapses or opens one file of the diff. Rows it hides
// are dropped from the line selection, so a copy never takes text the reader
// cannot see; opening the section again brings them back.
func (a *app) toggleCommitSection(path string) {
	if a.commitCollapsed == nil {
		a.commitCollapsed = make(map[string]bool)
	}
	if a.commitCollapsed[path] {
		a.expandCommitSection(path)
		a.invalidate()
		return
	}
	a.collapseCommitSection(path)
	a.invalidate()
}

// collapseCommitSection closes one file and remembers the rows it hid. The
// cursor is dropped along with its row, so no frame draws a selection handle
// on a line that is not there, and the hidden cursor is restored on expand.
func (a *app) collapseCommitSection(path string) {
	a.commitCollapsed[path] = true
	hidden := commitHiddenSel{Cursor: -1}
	for _, sec := range a.commitSections {
		if sec.Path != path {
			continue
		}
		for _, index := range sec.Rows {
			if !a.commitSel[index] {
				continue
			}
			delete(a.commitSel, index)
			hidden.Rows = append(hidden.Rows, index)
			if index == a.commitCursor {
				hidden.Cursor = index
			}
		}
	}
	if hidden.Cursor >= 0 {
		a.commitCursor = -1
	}
	if hidden.Rows == nil {
		return
	}
	if a.commitCollapsedSel == nil {
		a.commitCollapsedSel = make(map[string]commitHiddenSel)
	}
	a.commitCollapsedSel[path] = hidden
}

// expandCommitSection opens one file and puts back the rows it hid, so a
// collapse is a peek rather than a way to lose a selection.
func (a *app) expandCommitSection(path string) {
	delete(a.commitCollapsed, path)
	hidden, ok := a.commitCollapsedSel[path]
	if !ok {
		return
	}
	delete(a.commitCollapsedSel, path)
	if a.commitSel == nil {
		a.commitSel = make(map[int]bool)
	}
	for _, index := range hidden.Rows {
		a.commitSel[index] = true
	}
	if hidden.Cursor >= 0 {
		a.commitCursor = hidden.Cursor
	}
}

// scrollCommitDiffToFile moves the diff so the chosen file's header is at the
// top, opening the section first so the file's rows are what the reader sees.
func (a *app) scrollCommitDiffToFile(path string) {
	if a.commitCollapsed[path] {
		a.expandCommitSection(path)
	}
	y := a.commitDiffOffset(path)
	if y < 0 {
		return
	}
	// The scroll area clamps to the content it lays out; MaxY here is still the
	// previous commit's, so a commit that just opened would clamp a file to the
	// top.
	a.commitDiffScroll.Y = y
}
