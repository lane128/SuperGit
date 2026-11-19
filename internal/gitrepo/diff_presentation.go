package gitrepo

import "strings"

// SplitDiffRow holds source indexes into Diff.Lines. A negative index denotes
// an empty cell. Context uses the same source index on both sides.
type SplitDiffRow struct {
	Shared, Left, Right int
}

// SplitDiffRows pairs adjacent removals and insertions for display. Pass nil
// for visible to use the whole patch, or pass one file's visible source
// indexes when presenting part of a combined patch.
func SplitDiffRows(lines []DiffLine, visible []int) []SplitDiffRow {
	count := len(lines)
	if visible != nil {
		count = len(visible)
	}
	index := func(i int) int {
		if visible != nil {
			return visible[i]
		}
		return i
	}
	var rows []SplitDiffRow
	var left, right, markers []int
	eofMarker := func(line DiffLine, hunk int) bool {
		return line.Hunk == hunk && line.Kind == "meta" && strings.HasPrefix(line.Text, `\ No newline at end of file`)
	}
	for i := 0; i < count; {
		at := index(i)
		switch lines[at].Kind {
		case "context":
			rows = append(rows, SplitDiffRow{Shared: -1, Left: at, Right: at})
			i++
		case "add", "del":
			left, right, markers = left[:0], right[:0], markers[:0]
			hunk, next := lines[at].Hunk, at
		changeRun:
			for i < count {
				lineIndex := index(i)
				line := lines[lineIndex]
				// History can omit the marker. Only skipped EOF markers may
				// bridge a gap in the visible source indexes.
				for next < lineIndex && eofMarker(lines[next], hunk) {
					next++
				}
				if next != lineIndex || line.Hunk != hunk {
					break
				}
				switch {
				case line.Kind == "del":
					left = append(left, lineIndex)
				case line.Kind == "add":
					right = append(right, lineIndex)
				case eofMarker(line, hunk):
					markers = append(markers, lineIndex)
				default:
					break changeRun
				}
				i++
				next = lineIndex + 1
			}
			for j := 0; j < max(len(left), len(right)); j++ {
				row := SplitDiffRow{Shared: -1, Left: -1, Right: -1}
				if j < len(left) {
					row.Left = left[j]
				}
				if j < len(right) {
					row.Right = right[j]
				}
				rows = append(rows, row)
			}
			for _, marker := range markers {
				rows = append(rows, SplitDiffRow{Shared: marker, Left: -1, Right: -1})
			}
		default:
			rows = append(rows, SplitDiffRow{Shared: at, Left: -1, Right: -1})
			i++
		}
	}
	return rows
}
