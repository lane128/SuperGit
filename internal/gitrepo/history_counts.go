package gitrepo

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// ChangedFileCounts compares each row against its first parent, including
// merges and stashes. Git quotes newlines in names, so one output line is one
// path. A NUL-prefixed header cannot be confused with any filename.
func (r *Repository) ChangedFileCounts(ctx context.Context, rows []Commit) (map[string]int, error) {
	var input strings.Builder
	for _, row := range rows {
		if !oidRE.MatchString(row.SHA) {
			return nil, fmt.Errorf("invalid commit %q", row.SHA)
		}
		input.WriteString(row.SHA)
		if len(row.Parents) > 0 {
			if !oidRE.MatchString(row.Parents[0]) {
				return nil, fmt.Errorf("invalid parent %q", row.Parents[0])
			}
			input.WriteByte(' ')
			input.WriteString(row.Parents[0])
		}
		input.WriteByte('\n')
	}
	result := historyCountOutput{counts: make(map[string]int, len(rows))}
	err := runOutput(ctx, r.Root, input.String(), &result, "diff-tree", "--stdin", "--always", "--root", "-r", "-M", "--no-color", "--no-ext-diff", "--name-only", "--format=%x00%H")
	if err == nil {
		for _, row := range rows {
			if _, ok := result.counts[row.SHA]; !ok {
				return result.counts, fmt.Errorf("missing diff-tree result for %s", row.SHA)
			}
		}
	}
	return result.counts, err
}

type historyCountOutput struct {
	counts  map[string]int
	sha     string
	partial []byte
}

func (o *historyCountOutput) Write(p []byte) (int, error) {
	size := len(p)
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			o.partial = append(o.partial, p...)
			break
		}
		line := p[:i]
		if len(o.partial) > 0 {
			line = append(o.partial, line...)
			o.partial = o.partial[:0]
		}
		switch {
		case len(line) == 0:
		case line[0] == 0:
			o.sha = string(line[1:])
			o.counts[o.sha] = 0
		case o.sha != "":
			o.counts[o.sha]++
		}
		p = p[i+1:]
	}
	return size, nil
}
