package gitrepo

import (
	"context"
	"strings"
)

// ReadVisibleHistory pins the tips that still contribute to the graph. Tags
// and detached HEAD remain independent roots. A hidden branch's ancestors
// stay visible wherever another visible tip still reaches them.
func (r *Repository) ReadVisibleHistory(ctx context.Context, limit int, hidden []string) (HistoryPage, error) {
	if len(hidden) == 0 {
		return r.ReadHistory(ctx, limit)
	}
	out, err := run(ctx, r.Root, "", "for-each-ref", "--format=%(refname)%09%(objectname)%09%(symref)", "refs/heads/", "refs/remotes/", "refs/tags/")
	if err != nil {
		return HistoryPage{}, err
	}
	excluded := make(map[string]bool, len(hidden))
	for _, ref := range hidden {
		excluded[ref] = true
	}
	var revs []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) == 3 && !excluded[fields[0]] && !excluded[fields[2]] {
			revs = append(revs, fields[1])
		}
	}
	headRef, _ := run(ctx, r.Root, "", "symbolic-ref", "--quiet", "HEAD")
	anchor := ""
	if !excluded[strings.TrimSpace(headRef)] {
		if head, err := r.ResolveRef(ctx, "HEAD"); err == nil {
			anchor = head
			revs = append([]string{head}, revs...)
		}
	}
	tips := []string{}
	if len(revs) > 0 {
		// Let Git peel tags and ignore tags pointing to blobs or trees.
		raw, err := run(ctx, r.Root, strings.Join(revs, "\n")+"\n", "log", "--no-walk=unsorted", "--format=%H", "--stdin", "--")
		if err != nil {
			return HistoryPage{}, err
		}
		tips = append(tips, strings.Fields(raw)...)
	}
	result := HistoryPage{Tips: tips}
	for len(result.Commits) < limit {
		page, err := r.historyPage(ctx, tips, len(result.Commits), min(300, limit-len(result.Commits)), hidden)
		if err != nil {
			return HistoryPage{}, err
		}
		result.Commits = append(result.Commits, page.Commits...)
		result.HasMore = page.HasMore
		if !page.HasMore {
			break
		}
	}
	prioritizeIndependentHead(result.Commits, anchor)
	return result, nil
}

func historyDecorationArgs(args, hidden []string) []string {
	if len(hidden) == 0 {
		return args
	}
	result := make([]string, 0, len(args)+len(hidden))
	result = append(result, args[0])
	for _, ref := range hidden {
		result = append(result, "--decorate-refs-exclude="+ref)
	}
	return append(result, args[1:]...)
}
