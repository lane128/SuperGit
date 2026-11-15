package gitrepo

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// BlameLine is one line of a file with the commit that last changed it.
type BlameLine struct {
	SHA, Author, Date, Subject, Text string
	Line                             int
}

// Blame reads line attribution for a tracked path at the worktree, which is
// what the workspace's file menu means.
func (r *Repository) Blame(ctx context.Context, path string, ignoreWhitespace bool) ([]BlameLine, error) {
	return r.BlameAt(ctx, "", path, ignoreWhitespace)
}

// BlameAt reads line attribution for a path as of rev. An empty rev blames the
// worktree, including edits not committed yet; a revision blames the version
// that commit held, which is what a file in a commit's patch asks for -- the
// worktree copy may be a different file by now.
//
// --line-porcelain repeats metadata for every line, so callers can render each
// row without joins. ignoreWhitespace asks Git to attribute the last change
// that altered non-whitespace content, which collapses reformatting commits.
func (r *Repository) BlameAt(ctx context.Context, rev, path string, ignoreWhitespace bool) ([]BlameLine, error) {
	if err := r.path(path); err != nil {
		return nil, err
	}
	args := []string{"blame", "--line-porcelain"}
	if ignoreWhitespace {
		args = append(args, "-w")
	}
	if rev != "" {
		args = append(args, rev)
	}
	args = append(args, "--", path)
	out, err := run(ctx, r.Root, "", args...)
	if err != nil {
		return nil, err
	}
	return parseBlame(out), nil
}

func parseBlame(out string) []BlameLine {
	var result []BlameLine
	var cur BlameLine
	hasHeader := false
	for _, raw := range strings.SplitAfter(out, "\n") {
		line := strings.TrimSuffix(raw, "\n")
		switch {
		case strings.HasPrefix(line, "\t"):
			cur.Text = line[1:]
			result = append(result, cur)
			cur = BlameLine{}
			hasHeader = false
		case strings.HasPrefix(line, "author "):
			cur.Author = strings.TrimPrefix(line, "author ")
		case strings.HasPrefix(line, "author-time "):
			if sec, err := strconv.ParseInt(strings.TrimPrefix(line, "author-time "), 10, 64); err == nil {
				cur.Date = time.Unix(sec, 0).Format(time.RFC3339)
			}
		case strings.HasPrefix(line, "summary "):
			cur.Subject = strings.TrimPrefix(line, "summary ")
		default:
			if hasHeader {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 3 && oidRE.MatchString(fields[0]) {
				if n, err := strconv.Atoi(fields[2]); err == nil {
					cur.SHA, cur.Line, hasHeader = fields[0], n, true
				}
			}
		}
	}
	return result
}
