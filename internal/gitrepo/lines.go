package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ApplyLines changes only Index. Selection uses indices in the displayed Diff;
// reverse selects changes to remove from the staged HEAD -> Index diff.
func (r *Repository) ApplyLines(ctx context.Context, path string, expected Diff, selection []int, reverse bool) error {
	if err := r.path(path); err != nil {
		return err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	current, err := r.Diff(ctx, path, reverse)
	if err != nil {
		return err
	}
	if current.Text != expected.Text {
		return errors.New("Diff 已在外部变化，请刷新并重新选择行")
	}
	status, err := r.Status(ctx)
	if err != nil {
		return err
	}
	allowed := false
	for _, entry := range status.Entries {
		if entry.Path == path {
			allowed = !entry.Conflict && !entry.IsSubmodule() && entry.Index != '?' && entry.OriginalPath == ""
			break
		}
	}
	if !allowed || !current.CanStageHunks {
		return errors.New("此文件需要文件级暂存，不能按行操作")
	}
	patch, err := selectedPatch(current, selection, reverse)
	if err != nil {
		return err
	}
	args := []string{"apply", "--cached", "--whitespace=nowarn"}
	if _, err := run(ctx, r.Root, patch, append(args, "--check")...); err != nil {
		return err
	}
	_, err = run(ctx, r.Root, patch, args...)
	return err
}

// DiscardLines reverts the selected changed lines of an unstaged diff in the
// working tree and leaves the index alone. It is ApplyLines' worktree twin:
// the same partial patch is applied in reverse to the file on disk. The applied
// patch is returned so the caller can offer an undo.
func (r *Repository) DiscardLines(ctx context.Context, path string, expected Diff, selection []int) (string, error) {
	if err := r.path(path); err != nil {
		return "", err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	current, err := r.Diff(ctx, path, false)
	if err != nil {
		return "", err
	}
	if current.Text != expected.Text {
		return "", errors.New("Diff 已在外部变化，请刷新并重新选择行")
	}
	status, err := r.Status(ctx)
	if err != nil {
		return "", err
	}
	allowed := false
	for _, entry := range status.Entries {
		if entry.Path == path {
			allowed = !entry.Conflict && !entry.IsSubmodule() && entry.Index != '?' &&
				entry.OriginalPath == "" && entry.Worktree != 'D' && entry.Unstaged()
			break
		}
	}
	if !allowed || !current.CanStageHunks {
		return "", errors.New("此文件需要文件级放弃，不能按行操作")
	}
	// A working-tree patch has to use the working tree as its old side:
	// unselected changes in the same hunk are still present on disk. The
	// forward diff's context comes from the index, so discard builds its
	// patch from the reverse diff (worktree -> index) instead.
	reverseOut, err := run(ctx, r.Root, "", "diff", "-R", "--no-ext-diff", "--no-textconv",
		"--no-color", "--unified=3", "--submodule=short", "--", path)
	if err != nil {
		return "", err
	}
	reverse := parseDiff(reverseOut)
	reverseChoice, err := reverseSelection(current, reverse, selection)
	if err != nil {
		return "", err
	}
	patch, err := selectedPatch(reverse, reverseChoice, false)
	if err != nil {
		return "", err
	}
	args := []string{"apply", "--whitespace=nowarn"}
	if _, err := run(ctx, r.Root, patch, append(args, "--check")...); err != nil {
		return "", err
	}
	if _, err := run(ctx, r.Root, patch, args...); err != nil {
		return "", err
	}
	return patch, nil
}

// UndoDiscard reverses the patch a successful DiscardLines applied. Git's own
// context check rejects the undo when the file changed after the discard.
func (r *Repository) UndoDiscard(ctx context.Context, path, patch string) error {
	if err := r.path(path); err != nil {
		return err
	}
	if strings.TrimSpace(patch) == "" {
		return errors.New("没有可撤销的放弃操作")
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	args := []string{"apply", "-R", "--whitespace=nowarn"}
	if _, err := run(ctx, r.Root, patch, append(args, "--check")...); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, patch, args...)
	return err
}

// reverseSelection maps lines chosen in the worktree diff to the same lines in
// the reverse diff, where additions become deletions and the line numbers swap
// sides. The number identifies the line's position on that side; the text must
// match as well so repeated lines cannot pair with the wrong change.
func reverseSelection(current, reverse Diff, selection []int) ([]int, error) {
	used := make(map[int]bool, len(selection))
	out := make([]int, 0, len(selection))
	for _, index := range selection {
		if index < 0 || index >= len(current.Lines) {
			return nil, errors.New("只能选择新增或删除的差异行")
		}
		line := current.Lines[index]
		var kind string
		var number int
		switch line.Kind {
		case "add":
			kind, number = "del", line.New
		case "del":
			kind, number = "add", line.Old
		default:
			return nil, errors.New("只能选择新增或删除的差异行")
		}
		text := line.Text
		if kind == "del" {
			text = "-" + text[1:]
		} else {
			text = "+" + text[1:]
		}
		found := -1
		for i, other := range reverse.Lines {
			if used[i] || other.Kind != kind || other.Text != text {
				continue
			}
			if kind == "del" && other.Old != number {
				continue
			}
			if kind == "add" && other.New != number {
				continue
			}
			found = i
			break
		}
		if found < 0 {
			return nil, errors.New("无法在反向差异中定位所选行，请刷新后重试")
		}
		used[found] = true
		out = append(out, found)
	}
	return out, nil
}
func selectedPatch(d Diff, selection []int, reverse bool) (string, error) {
	chosen := make(map[int]bool, len(selection))
	for _, i := range selection {
		if i < 0 || i >= len(d.Lines) || (d.Lines[i].Kind != "add" && d.Lines[i].Kind != "del") {
			return "", errors.New("只能选择新增或删除的差异行")
		}
		chosen[i] = true
	}
	if len(chosen) == 0 {
		return "", errors.New("请先选择需要操作的差异行")
	}
	var patch strings.Builder
	start := 0
	for start < len(d.Lines) && d.Lines[start].Kind != "hunk" {
		line := d.Lines[start]
		// A partial patch has different blob identities from the full diff.
		if !strings.HasPrefix(line.Text, "index ") {
			patch.WriteString(line.Text + "\n")
		}
		start++
	}
	delta := 0
	for start < len(d.Lines) {
		header := d.Lines[start]
		if header.Kind != "hunk" {
			return "", errors.New("无效的差异片段")
		}
		end := start + 1
		for end < len(d.Lines) && d.Lines[end].Kind != "hunk" {
			end++
		}
		total, selected, noNewline := 0, 0, false
		for i := start + 1; i < end; i++ {
			line := d.Lines[i]
			if line.Kind == "add" || line.Kind == "del" {
				total++
				if chosen[i] {
					selected++
				}
			}
			if strings.HasPrefix(line.Text, "\\ No newline at end of file") {
				noNewline = true
			}
		}
		if selected == 0 {
			start = end
			continue
		}
		if noNewline && selected != total {
			return "", errors.New("此片段包含无结尾换行的内容，请选择该片段全部更改或使用片段暂存")
		}
		m := hunkRE.FindStringSubmatch(header.Text)
		if m == nil {
			return "", errors.New("无效的差异行范围")
		}
		sourceStart, _ := strconv.Atoi(m[1])
		if reverse {
			sourceStart, _ = strconv.Atoi(m[2])
		}
		sourceCount, targetCount := 0, 0
		var body strings.Builder
		emitted := false
		for _, i := range orderedSelectionLines(d, start+1, end, chosen, reverse) {
			line := d.Lines[i]
			prefix := byte(0)
			switch line.Kind {
			case "context":
				prefix = ' '
			case "del":
				if reverse {
					if chosen[i] {
						prefix = '+'
					}
				} else {
					prefix = '-'
					if !chosen[i] {
						prefix = ' '
					}
				}
			case "add":
				if reverse {
					prefix = '-'
					if !chosen[i] {
						prefix = ' '
					}
				} else if chosen[i] {
					prefix = '+'
				}
			case "meta":
				if emitted && strings.HasPrefix(line.Text, "\\ No newline at end of file") {
					body.WriteString(line.Text + "\n")
				}
				continue
			}
			emitted = prefix != 0
			if !emitted {
				continue
			}
			if prefix != '+' {
				sourceCount++
			}
			if prefix != '-' {
				targetCount++
			}
			body.WriteByte(prefix)
			body.WriteString(line.Text[1:])
			body.WriteByte('\n')
		}
		targetStart := sourceStart + delta
		if sourceCount == 0 {
			targetStart++
		}
		if targetCount == 0 {
			targetStart--
		}
		fmt.Fprintf(&patch, "@@ -%d,%d +%d,%d @@\n", sourceStart, sourceCount, targetStart, targetCount)
		patch.WriteString(body.String())
		delta += targetCount - sourceCount
		start = end
	}
	return patch.String(), nil
}

// Keep replacement pairs at their original positions: Git groups all removals
// before all additions, which would otherwise move a selected first replacement
// below the unselected second line when that removal is converted to context.
func orderedSelectionLines(d Diff, start, end int, chosen map[int]bool, reverse bool) []int {
	var order []int
	appendLine := func(i int) {
		if i < 0 {
			return
		}
		order = append(order, i)
		if i+1 < end && d.Lines[i+1].Kind == "meta" && strings.HasPrefix(d.Lines[i+1].Text, "\\ No newline at end of file") {
			order = append(order, i+1)
		}
	}
	for i := start; i < end; {
		kind := d.Lines[i].Kind
		if kind != "add" && kind != "del" {
			order = append(order, i)
			i++
			continue
		}
		var removed, added []int
		j := i
		for j < end {
			line := d.Lines[j]
			if line.Kind == "del" {
				removed = append(removed, j)
			} else if line.Kind == "add" {
				added = append(added, j)
			} else if line.Kind != "meta" || !strings.HasPrefix(line.Text, "\\ No newline at end of file") {
				break
			}
			j++
		}
		for k := 0; k < max(len(removed), len(added)); k++ {
			old, next := -1, -1
			if k < len(removed) {
				old = removed[k]
			}
			if k < len(added) {
				next = added[k]
			}
			if reverse && chosen[next] {
				appendLine(next)
				appendLine(old)
			} else {
				appendLine(old)
				appendLine(next)
			}
		}
		i = j
	}
	return order
}
