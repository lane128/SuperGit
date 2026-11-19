package gitrepo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Keep every history producer and streaming/page record boundary in sync.
const historyFormat = "--format=%H%x00%P%x00%s%x00%an%x00%ae%x00%aI%x00%D"
const historyFieldCount = 7

// FileHistory lists commits that changed path, following renames. It is a
// separate query from the paged all-history traversal because --follow cannot
// be combined with the pinned multi-tip walk.
func (r *Repository) FileHistory(ctx context.Context, path string, limit int) ([]Commit, error) {
	if err := r.path(path); err != nil {
		return nil, err
	}
	// An empty repository has no history to walk, and `git log` fails on it
	// with a fatal error. No commits is the honest answer here.
	if !r.hasHead(ctx) {
		return nil, nil
	}
	if limit < 1 || limit > 2000 {
		limit = 200
	}
	out, err := run(ctx, r.Root, "", "log", "--follow", "-z", historyFormat, "-n", strconv.Itoa(limit), "--", path)
	if err != nil {
		return nil, err
	}
	return parseHistory(out)
}

func (r *Repository) History(ctx context.Context, limit int) ([]Commit, error) {
	page, err := r.ReadHistory(ctx, limit)
	return page.Commits, err
}

// HeadSubject is the subject of the commit HEAD points to. The history list
// can be scoped to another ref, so the amend box cannot read it from the first
// row: that row is the newest commit of whatever the reader is browsing.
func (r *Repository) HeadSubject(ctx context.Context) (string, error) {
	if !r.hasHead(ctx) {
		return "", nil
	}
	out, err := run(ctx, r.Root, "", "log", "-1", "--format=%s", "HEAD", "--")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func parseHistory(out string) ([]Commit, error) {
	fields := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	if out == "" {
		return nil, nil
	}
	if len(fields)%historyFieldCount != 0 {
		return nil, fmt.Errorf("无效的提交历史记录")
	}
	result := make([]Commit, 0, len(fields)/historyFieldCount)
	for i := 0; i < len(fields); i += historyFieldCount {
		result = append(result, Commit{SHA: fields[i], Parents: strings.Fields(fields[i+1]), Subject: fields[i+2], Author: fields[i+3], AuthorEmail: fields[i+4], Date: fields[i+5], Refs: fields[i+6], Day: localDay(fields[i+5])})
	}
	return result, nil
}

// localDay turns Git's RFC 3339 author date into the calendar day this machine
// would print for it, the same day the commit list shows. Git records the
// author's offset, so a commit made in Tokyo can land on yesterday here; the
// search must match the day the reader sees, not the one Git stores.
func localDay(date string) string {
	parsed, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return ""
	}
	return parsed.Local().Format("2006-01-02")
}

func (r *Repository) CommitFiles(ctx context.Context, sha string) ([]ChangedFile, error) {
	if !oidRE.MatchString(sha) {
		return nil, fmt.Errorf("无效的提交 SHA")
	}
	parents, err := run(ctx, r.Root, "", "show", "-s", "--format=%P", sha)
	if err != nil {
		return nil, err
	}
	p := strings.Fields(parents)
	args := []string{"diff-tree", "--no-commit-id", "--name-status", "-r", "-z", "-M"}
	if len(p) > 0 {
		args = append(args, p[0], sha)
	} else {
		args = append(args, "--root", sha)
	}
	args = append(args, "--")
	out, err := run(ctx, r.Root, "", args...)
	if err != nil {
		return nil, err
	}
	records := strings.Split(out, "\x00")
	var files []ChangedFile
	for i := 0; i < len(records) && records[i] != ""; {
		status := records[i]
		i++
		if i >= len(records)-1 {
			return nil, fmt.Errorf("无效的提交文件记录")
		}
		f := ChangedFile{Status: status, Path: records[i]}
		i++
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if i >= len(records)-1 {
				return nil, fmt.Errorf("无效的重命名文件记录")
			}
			f.OriginalPath = f.Path
			f.Path = records[i]
			i++
		}
		files = append(files, f)
	}
	return files, nil
}
