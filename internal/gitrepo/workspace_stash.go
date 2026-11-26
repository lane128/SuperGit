package gitrepo

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

// SaveWorkspaceStash saves tracked and untracked changes, preserving ignored
// paths. The generated message identifies our entry even if another linked
// worktree changes the shared stash stack while Git is running.
func (r *Repository) SaveWorkspaceStash(ctx context.Context) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	before, err := r.workspaceSnapshot(ctx, true)
	if err != nil {
		return err
	}
	status, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if len(status.Entries) == 0 {
		return workspaceRefusal("There are no changes to stash", "没有需要保存的改动")
	}
	// Git stash refuses intent-to-add entries. Detect them before invoking the
	// mutating command rather than reporting a failed save after Git starts.
	visible, err := run(ctx, r.Root, "", "diff", "--cached", "--name-only", "-z", "--ita-visible-in-index", "--")
	if err != nil {
		return err
	}
	invisible, err := run(ctx, r.Root, "", "diff", "--cached", "--name-only", "-z", "--ita-invisible-in-index", "--")
	if err != nil {
		return err
	}
	if visible != invisible {
		return workspaceRefusal("Stage or unstage intent-to-add files before saving a workspace Stash", "请先暂存或撤销 intent-to-add 文件，再保存工作区 Stash")
	}
	existing, err := r.Stashes(ctx)
	if err != nil {
		return err
	}
	old := map[string]bool{}
	for _, s := range existing {
		old[s.SHA] = true
	}
	token := make([]byte, 16)
	if _, err = rand.Read(token); err != nil {
		return err
	}
	message := fmt.Sprintf("SuperGit workspace %s [%x]", time.Now().Format("2006-01-02 15:04:05"), token)
	// Revalidate immediately before entering Git. This cannot make external Git
	// atomic, but prevents using a snapshot invalidated during expensive scans.
	now, err := r.workspaceSnapshot(ctx, true)
	if err != nil {
		return err
	}
	if !workspaceSame(before, now) {
		return workspaceRefusal("The workspace changed; retry Stash", "工作区已变化，请重试 Stash")
	}
	pushErr := r.saveStashLocked(ctx, message, true)
	// Even after cancellation, inspect the reflog: Git may already have saved a
	// recoverable entry. Never drop or retry a partially completed save.
	verifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	entries, listErr := r.Stashes(verifyCtx)
	own, identifyErr := workspaceOwnStash(entries, old, message)
	if listErr != nil {
		identifyErr = listErr
	}
	if pushErr != nil {
		if identifyErr == nil {
			return workspaceFailure("stash", "", own.SHA, pushErr)
		}
		return workspaceFailure("stash", "", "", pushErr)
	}
	if identifyErr != nil {
		return workspaceFailure("stash", "", "", identifyErr)
	}
	if err := r.workspaceVerifyResult(verifyCtx, before); err != nil {
		return workspaceFailure("verify", "", own.SHA, err)
	}
	return nil
}

func workspaceOwnStash(entries []Stash, old map[string]bool, message string) (Stash, error) {
	var own Stash
	for _, entry := range entries {
		if !old[entry.SHA] && strings.HasSuffix(entry.Message, ": "+message) {
			if own.SHA != "" {
				return Stash{}, workspaceRefusal("Multiple matching stash entries were found", "发现多个匹配的 Stash 记录")
			}
			own = entry
		}
	}
	if own.SHA == "" {
		return Stash{}, workspaceRefusal("This operation's new stash entry could not be found", "没有找到本次新增的 Stash 记录")
	}
	return own, nil
}
