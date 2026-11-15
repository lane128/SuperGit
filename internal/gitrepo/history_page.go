package gitrepo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// HistoryPage carries an immutable traversal's starting OIDs. A non-nil empty
// Tips slice pins an empty repository; nil starts a fresh traversal.
type HistoryPage struct {
	Tips    []string
	Commits []Commit
	HasMore bool
}

// HistoryPage reads a bounded output page. --skip still traverses earlier
// commits in Git; this bounds response size rather than traversal complexity.
func (r *Repository) HistoryPage(ctx context.Context, tips []string, offset, limit int) (HistoryPage, error) {
	if offset < 0 || limit < 1 || limit > 2000 || tips == nil && offset != 0 {
		return HistoryPage{}, fmt.Errorf("无效的历史分页范围")
	}
	if tips == nil {
		// --all would also walk refs/stash and refs/notes, whose synthetic
		// commits are not history the user can browse as commits.
		args := []string{"log", "--no-walk=unsorted", "--branches", "--tags", "--remotes", "--format=%H"}
		if r.hasHead(ctx) {
			args = append(args, "HEAD")
		}
		out, err := run(ctx, r.Root, "", append(args, "--")...)
		if err != nil {
			return HistoryPage{}, err
		}
		tips = strings.Fields(out)
	}
	page := HistoryPage{Tips: append([]string{}, tips...)}
	for _, tip := range tips {
		if !oidRE.MatchString(tip) {
			return HistoryPage{}, fmt.Errorf("无效的历史起始 SHA")
		}
	}
	if len(tips) == 0 {
		return page, nil
	}
	args := []string{"log", "-z", "--format=%H%x00%P%x00%s%x00%an%x00%aI%x00%D", "--topo-order", "--skip=" + strconv.Itoa(offset), "-n", strconv.Itoa(limit + 1), "--stdin", "--"}
	out, err := run(ctx, r.Root, strings.Join(tips, "\n")+"\n", args...)
	if err != nil {
		return HistoryPage{}, err
	}
	page.Commits, err = parseHistory(out)
	if err != nil {
		return HistoryPage{}, err
	}
	page.HasMore = len(page.Commits) > limit
	if page.HasMore {
		page.Commits = page.Commits[:limit]
	}
	return page, nil
}

// ReadHistory rebuilds a requested depth without a single unbounded Git output.
// The same pinned tips and topological order are used for every batch.
func (r *Repository) ReadHistory(ctx context.Context, limit int) (HistoryPage, error) {
	if limit < 1 {
		limit = 300
	}
	var result HistoryPage
	for len(result.Commits) < limit {
		page, err := r.HistoryPage(ctx, result.Tips, len(result.Commits), min(300, limit-len(result.Commits)))
		if err != nil {
			return HistoryPage{}, err
		}
		result.Tips, result.HasMore = page.Tips, page.HasMore
		result.Commits = append(result.Commits, page.Commits...)
		if !page.HasMore {
			break
		}
	}
	return result, nil
}
