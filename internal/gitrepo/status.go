package gitrepo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func (r *Repository) statusText(ctx context.Context) (string, error) {
	return run(ctx, r.Root, "", "status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all", "--ignore-submodules=none")
}
func (r *Repository) Status(ctx context.Context) (Status, error) {
	out, err := r.statusText(ctx)
	if err != nil {
		return Status{}, err
	}
	s := Status{}
	records := strings.Split(out, "\x00")
	for i := 0; i < len(records); i++ {
		record := records[i]
		if record == "" {
			continue
		}
		switch {
		case strings.HasPrefix(record, "# branch.head "):
			s.Branch = strings.TrimPrefix(record, "# branch.head ")
			// Porcelain calls a detached HEAD "(detached)"; the rest of the
			// app treats an empty branch as the detached state.
			if s.Branch == "(detached)" {
				s.Branch = ""
			}
		case strings.HasPrefix(record, "# branch.oid "):
			s.Head = strings.TrimPrefix(record, "# branch.oid ")
			if s.Head == "(initial)" {
				s.Head = ""
			}
		case strings.HasPrefix(record, "# branch.upstream "):
			s.Upstream = strings.TrimPrefix(record, "# branch.upstream ")
		case strings.HasPrefix(record, "# branch.ab "):
			parts := strings.Fields(strings.TrimPrefix(record, "# branch.ab "))
			if len(parts) == 2 {
				s.Ahead, _ = strconv.Atoi(strings.TrimPrefix(parts[0], "+"))
				s.Behind, _ = strconv.Atoi(strings.TrimPrefix(parts[1], "-"))
			}
		case strings.HasPrefix(record, "1 "):
			parts := strings.SplitN(record, " ", 9)
			if len(parts) != 9 || len(parts[1]) != 2 {
				return s, fmt.Errorf("无效的 Git 状态记录")
			}
			s.Entries = append(s.Entries, Entry{Path: parts[8], Index: parts[1][0], Worktree: parts[1][1], Submodule: parts[2]})
		case strings.HasPrefix(record, "2 "):
			parts := strings.SplitN(record, " ", 10)
			if len(parts) != 10 || len(parts[1]) != 2 || i+1 >= len(records) {
				return s, fmt.Errorf("无效的重命名状态记录")
			}
			i++
			s.Entries = append(s.Entries, Entry{Path: parts[9], OriginalPath: records[i], Index: parts[1][0], Worktree: parts[1][1], Submodule: parts[2]})
		case strings.HasPrefix(record, "u "):
			parts := strings.SplitN(record, " ", 11)
			if len(parts) != 11 || len(parts[1]) != 2 {
				return s, fmt.Errorf("无效的冲突状态记录")
			}
			s.Entries = append(s.Entries, Entry{Path: parts[10], Index: parts[1][0], Worktree: parts[1][1], Submodule: parts[2], Conflict: true})
		case strings.HasPrefix(record, "? "):
			s.Entries = append(s.Entries, Entry{Path: record[2:], Index: '?', Worktree: '?'})
		}
	}
	return s, nil
}
func (r *Repository) Branches(ctx context.Context) ([]Branch, error) {
	out, err := run(ctx, r.Root, "", "for-each-ref", "--format=%(refname:short)%00%(objectname)%00%(upstream:short)%00%(HEAD)%00%(upstream:track,nobracket)%00%(subject)%00%(authorname)%00%(authordate:iso-strict)", "refs/heads")
	if err != nil {
		return nil, err
	}
	var result []Branch
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		p := strings.Split(line, "\x00")
		if len(p) == 8 {
			branch := Branch{Name: p[0], SHA: p[1], Upstream: p[2], Current: p[3] == "*",
				Commit: Commit{SHA: p[1], Subject: p[5], Author: p[6], Date: p[7], Day: localDay(p[7])}}
			switch track := strings.TrimSpace(p[4]); track {
			case "":
			case "gone":
				branch.UpstreamGone = true
			default:
				// The track field reads "ahead 1, behind 2"; LC_ALL=C in run
				// keeps those words stable.
				for _, part := range strings.Split(track, ",") {
					part = strings.TrimSpace(part)
					switch {
					case strings.HasPrefix(part, "ahead "):
						branch.Ahead, _ = strconv.Atoi(strings.TrimPrefix(part, "ahead "))
					case strings.HasPrefix(part, "behind "):
						branch.Behind, _ = strconv.Atoi(strings.TrimPrefix(part, "behind "))
					}
				}
			}
			result = append(result, branch)
		}
	}
	return result, nil
}
func (r *Repository) Worktrees(ctx context.Context) ([]Worktree, error) {
	out, err := run(ctx, r.Root, "", "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	var result []Worktree
	var current *Worktree
	for _, p := range strings.Split(out, "\x00") {
		switch {
		case strings.HasPrefix(p, "worktree "):
			result = append(result, Worktree{Path: strings.TrimPrefix(p, "worktree ")})
			current = &result[len(result)-1]
		case current != nil && strings.HasPrefix(p, "HEAD "):
			current.HEAD = strings.TrimPrefix(p, "HEAD ")
		case current != nil && strings.HasPrefix(p, "branch "):
			current.Branch = strings.TrimPrefix(p, "branch refs/heads/")
		case current != nil && p == "bare":
			current.Bare = true
		case current != nil && strings.HasPrefix(p, "locked"):
			current.Locked = true
		}
	}
	// Git reports a submodule's main worktree as its absorbed git directory
	// (.git/modules/...) even when called from a linked checkout. Resolve that
	// first record through Git's own core.worktree interpretation. Linked
	// records already name their checkout; a bare record has no checkout.
	if len(result) > 0 && !result[0].Bare {
		main, err := Open(ctx, result[0].Path)
		if err == nil {
			result[0].Path = main.Root
		} else if ctx.Err() != nil {
			return nil, err
		}
		// An unavailable main directory must not hide reachable linked trees.
		// Keep Git's listed path when it cannot currently be opened.
	}
	return result, nil
}
