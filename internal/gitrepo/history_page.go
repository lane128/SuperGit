package gitrepo

import (
	"context"
	"fmt"
	"slices"
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
	return r.historyPage(ctx, tips, offset, limit, nil)
}

func (r *Repository) historyPage(ctx context.Context, tips []string, offset, limit int, hidden []string) (HistoryPage, error) {
	if offset < 0 || limit < 1 || limit > 2000 || tips == nil && offset != 0 {
		return HistoryPage{}, fmt.Errorf("无效的历史分页范围")
	}
	anchor := ""
	if tips == nil {
		// --all would also walk refs/stash and refs/notes, whose synthetic
		// commits are not history the user can browse as commits.
		args := []string{"log", "--no-walk=unsorted", "--branches", "--tags", "--remotes", "--format=%H"}
		out, err := run(ctx, r.Root, "", append(args, "--")...)
		if err != nil {
			return HistoryPage{}, err
		}
		tips = strings.Fields(out)
		// Prefer HEAD among independent tips while preserving child-before-parent order.
		if head, err := r.ResolveRef(ctx, "HEAD"); err == nil {
			anchor = head
			rest := make([]string, 0, len(tips)+1)
			rest = append(rest, head)
			for _, tip := range tips {
				if tip != head {
					rest = append(rest, tip)
				}
			}
			tips = rest
		}
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
	args := []string{"log", "-z", historyFormat, "--topo-order", "--skip=" + strconv.Itoa(offset), "-n", strconv.Itoa(limit + 1), "--stdin", "--"}
	args = historyDecorationArgs(args, hidden)
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
	prioritizeIndependentHead(page.Commits, anchor)
	return page, nil
}

// The graph draws edges from children down to their parents. If a visible tip
// descends from HEAD, its direct child will precede HEAD in Git's topo-order.
// In that case keep the order intact, including for non-first merge parents.
func prioritizeIndependentHead(commits []Commit, head string) {
	if head == "" {
		return
	}
	for i, commit := range commits {
		if slices.Contains(commit.Parents, head) {
			return
		}
		if commit.SHA == head {
			copy(commits[1:i+1], commits[:i])
			commits[0] = commit
			return
		}
	}
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
