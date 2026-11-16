package gitrepo

import (
	"context"
	"errors"
	"path/filepath"
)

func (r *Repository) AddWorktree(ctx context.Context, path, branch, start string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.idle(ctx); err != nil {
		return err
	}
	abs, err := newDestination(path)
	if err != nil {
		return err
	}
	if err := r.branchName(ctx, branch); err != nil {
		return err
	}
	sha, err := r.resolve(ctx, start)
	if err != nil {
		return err
	}
	_, err = run(ctx, r.Root, "", "worktree", "add", "-b", branch, "--", abs, sha)
	return err
}
func canonical(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved
	}
	abs, _ := filepath.Abs(path)
	return abs
}

var ErrWorktreeTargetChanged = errors.New("worktree changed before removal")

// RemoveWorktreeChecked verifies the checkout displayed before confirmation.
// External Git processes do not share this repository's write lock.
func (r *Repository) RemoveWorktreeChecked(ctx context.Context, expected Worktree) error {
	if !oidRE.MatchString(expected.HEAD) {
		return ErrWorktreeTargetChanged
	}
	return r.removeWorktree(ctx, expected.Path, &expected)
}

func (r *Repository) RemoveWorktree(ctx context.Context, path string) error {
	return r.removeWorktree(ctx, path, nil)
}

func (r *Repository) removeWorktree(ctx context.Context, path string, expected *Worktree) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if canonical(abs) == canonical(r.Root) {
		return errors.New("不能移除当前打开的工作区")
	}
	list, err := r.Worktrees(ctx)
	if err != nil {
		return err
	}
	for _, w := range list {
		if canonical(w.Path) == canonical(abs) {
			if expected != nil && (w.HEAD != expected.HEAD || w.Branch != expected.Branch || w.Locked != expected.Locked || w.Bare != expected.Bare) {
				return ErrWorktreeTargetChanged
			}
			if w.Locked || w.Bare {
				return errors.New("工作区已锁定或不可移除")
			}
			child, err := Open(ctx, w.Path)
			if err != nil {
				return err
			}
			if err := child.clean(ctx); err != nil {
				return err
			}
			if err := child.noIgnored(ctx); err != nil {
				return err
			}
			_, err = run(ctx, r.Root, "", "worktree", "remove", "--", w.Path)
			return err
		}
	}
	if expected != nil {
		return ErrWorktreeTargetChanged
	}
	return errors.New("工作区不在当前仓库列表中")
}

var ErrSubmoduleTargetChanged = errors.New("submodule target changed before update")

// ErrDirtySubmodule preserves the original refusal for existing callers while
// allowing interfaces to provide recovery guidance in the current language.
var ErrDirtySubmodule = errors.New("子模块内部有改动，请先提交或 Stash；更新不会丢弃内容")

// UpdateSubmoduleChecked verifies the displayed Index and checkout identities.
// External Git processes do not share the repository write lock.
func (r *Repository) UpdateSubmoduleChecked(ctx context.Context, expected Submodule) error {
	if !oidRE.MatchString(expected.Index) {
		return ErrSubmoduleTargetChanged
	}
	return r.updateSubmodule(ctx, expected.Path, &expected)
}

func (r *Repository) UpdateSubmodule(ctx context.Context, path string) error {
	return r.updateSubmodule(ctx, path, nil)
}

func (r *Repository) updateSubmodule(ctx context.Context, path string, expected *Submodule) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.path(path); err != nil {
		return err
	}
	if err := r.idle(ctx); err != nil {
		return err
	}
	modules, err := r.Submodules(ctx)
	if err != nil {
		return err
	}
	for _, m := range modules {
		if m.Path == path {
			if expected != nil && (m.Name != expected.Name || m.Index != expected.Index || m.Recorded != expected.Recorded || m.Current != expected.Current || m.Branch != expected.Branch || m.Initialized != expected.Initialized) {
				return ErrSubmoduleTargetChanged
			}
			if m.Dirty {
				return ErrDirtySubmodule
			}
			if m.Error != "" {
				return errors.New(m.Error)
			}
			if m.Initialized {
				child, err := Open(ctx, filepath.Join(r.Root, path))
				if err != nil {
					return err
				}
				if err := child.idle(ctx); err != nil {
					return err
				}
				if m.Index == "" {
					return errors.New("子模块没有有效的 Index 指针")
				}
				if err := child.protectIgnored(ctx, m.Index); err != nil {
					return err
				}
			}
			_, err := run(ctx, r.Root, "", "submodule", "update", "--init", "--checkout", "--", path)
			return err
		}
	}
	if expected != nil {
		return ErrSubmoduleTargetChanged
	}
	return errors.New("此路径不是当前仓库配置的子模块")
}
