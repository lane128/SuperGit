package gitrepo

// parseStashDiff composes a read-only multi-file preview. The single-file
// parser retains each file's headers and hunk patch; only display indices are
// offset. A combined preview must never advertise partial write actions.
func parseStashDiff(text string) Diff {
	result := Diff{Text: text}
	for _, file := range splitFileDiffs(text) {
		offset := len(result.Hunks)
		for _, row := range file.Diff.Lines {
			if row.Hunk >= 0 {
				row.Hunk += offset
			}
			result.Lines = append(result.Lines, row)
		}
		result.Hunks = append(result.Hunks, file.Diff.Hunks...)
	}
	return result
}
