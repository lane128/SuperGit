package desktop

import (
	"context"
	"fmt"
	"supergit/internal/gitrepo"
)

func readSnapshot(ctx context.Context, r *gitrepo.Repository, limit int) (snapshot, error) {
	s := snapshot{}
	var err error
	// Baseline precedes the reads: changes during the snapshot remain observable.
	s.changeStamp, err = r.ChangeStamp(ctx)
	if err != nil {
		s.warnings = append(s.warnings, fmt.Sprintf("变化检测：%v", err))
	}
	s.status, err = r.Status(ctx)
	if err != nil {
		return s, err
	}
	warn := func(label string, err error) {
		if err != nil {
			s.warnings = append(s.warnings, fmt.Sprintf("%s：%v", label, err))
		}
	}
	history, err := r.ReadHistory(ctx, limit)
	s.history, s.historyTips, s.historyMore = history.Commits, history.Tips, history.HasMore
	s.historyErr = err
	warn("历史", err)
	s.branches, err = r.Branches(ctx)
	warn("分支", err)
	s.tags, s.tagsErr = r.Tags(ctx)
	warn("标签", s.tagsErr)
	s.worktrees, err = r.Worktrees(ctx)
	warn("Worktrees", err)
	s.modules, err = r.Submodules(ctx)
	warn("子模块", err)
	s.remotes, err = r.Remotes(ctx)
	warn("远端", err)
	s.remoteBranches, err = r.RemoteBranches(ctx)
	warn("远端分支", err)
	s.stashes, err = r.Stashes(ctx)
	warn("Stash", err)
	s.integration, err = r.Operation(ctx)
	warn("进行中操作", err)
	return s, nil
}
