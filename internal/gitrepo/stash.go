package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Stash struct {
	SHA, Ref, Message, Date        string
	Author, AuthorEmail, Tree, Day string
	ParentSHAs                     string
}

// StashApplyError keeps the legacy hint while allowing interfaces to localize
// the reason and expose Git's original output. Failed restores can change files.
type StashApplyError struct {
	Conflict bool
	cause    error
	hint     string
}

func (e *StashApplyError) Error() string { return e.hint }
func (e *StashApplyError) Unwrap() error { return e.cause }

func (r *Repository) Stashes(ctx context.Context) ([]Stash, error) {
	out, err := run(ctx, r.Root, "", "stash", "list", "-z", "--format=%H%x00%gd%x00%gs%x00%aI%x00%an%x00%ae%x00%T%x00%P")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	fields := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	if len(fields)%8 != 0 {
		return nil, errors.New("无效的 Stash 列表")
	}
	var list []Stash
	for i := 0; i < len(fields); i += 8 {
		list = append(list, Stash{
			SHA: fields[i], Ref: fields[i+1], Message: fields[i+2],
			Date: fields[i+3], Day: localDay(fields[i+3]),
			Author: fields[i+4], AuthorEmail: fields[i+5],
			Tree: fields[i+6], ParentSHAs: fields[i+7],
		})
	}
	return list, nil
}
func (r *Repository) findStash(ctx context.Context, sha string) (Stash, error) {
	if !oidRE.MatchString(sha) {
		return Stash{}, errors.New("无效的 Stash SHA")
	}
	list, err := r.Stashes(ctx)
	if err != nil {
		return Stash{}, err
	}
	for _, s := range list {
		if s.SHA == sha {
			return s, nil
		}
	}
	return Stash{}, errors.New("Stash 已变化或不存在，请刷新列表")
}
func (r *Repository) SaveStash(ctx context.Context, message string, untracked bool) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.idle(ctx); err != nil {
		return err
	}
	s, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if len(s.Entries) == 0 {
		return errors.New("没有需要保存的改动")
	}
	for _, e := range s.Entries {
		if e.Conflict {
			return errors.New("请先处理冲突")
		}
	}
	mods, err := r.Submodules(ctx)
	if err != nil {
		return err
	}
	for _, m := range mods {
		if m.Dirty {
			return errors.New("子模块内部有改动，请先在子模块内保存或提交；父仓库 Stash 不包含它们")
		}
	}
	// Git stash internally builds magic pathspecs for untracked cleanup.
	// No user pathspec is passed here; inherited literal mode breaks that cleanup.
	args := []string{"--no-literal-pathspecs", "stash", "push"}
	if strings.TrimSpace(message) != "" {
		args = append(args, "-m", message)
	}
	if untracked {
		args = append(args, "--include-untracked")
	}
	args = append(args, "--")
	_, err = run(ctx, r.Root, "", args...)
	return err
}

// SaveStashPaths stashes only the named paths, the right-click "stash this
// file" action. The paths come from the status list, but they are still
// validated and passed behind "--". A path with no pending change is an
// error instead of a silent no-op, because git leaves the stash list
// unchanged in that case.
func (r *Repository) SaveStashPaths(ctx context.Context, message string, untracked bool, paths []string) error {
	if len(paths) == 0 {
		return errors.New("请选择要保存的文件")
	}
	for _, path := range paths {
		if err := r.path(path); err != nil {
			return err
		}
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.idle(ctx); err != nil {
		return err
	}
	s, err := r.Status(ctx)
	if err != nil {
		return err
	}
	wanted := make(map[string]bool, len(paths))
	for _, path := range paths {
		wanted[path] = true
	}
	found := false
	for _, e := range s.Entries {
		if !wanted[e.Path] && !(e.OriginalPath != "" && wanted[e.OriginalPath]) {
			continue
		}
		if e.Conflict {
			return errors.New("请先处理冲突")
		}
		found = true
	}
	if !found {
		return errors.New("选中的文件没有可保存的改动")
	}
	args := []string{"--no-literal-pathspecs", "stash", "push", "-m", message}
	if untracked {
		args = append(args, "--include-untracked")
	}
	args = append(args, "--")
	args = append(args, paths...)
	_, err = run(ctx, r.Root, "", args...)
	return err
}

func (r *Repository) StashDiff(ctx context.Context, sha string) (Diff, error) {
	return r.DisplayStashDiff(ctx, sha, false)
}

func (r *Repository) DisplayStashDiff(ctx context.Context, sha string, ignoreWhitespace bool) (Diff, error) {
	if _, err := r.findStash(ctx, sha); err != nil {
		return Diff{}, err
	}
	args := []string{"stash", "show", "--include-untracked", "--patch", "--no-ext-diff", "--no-textconv", "--no-color"}
	if ignoreWhitespace {
		args = append(args, "--ignore-all-space")
	}
	args = append(args, sha)
	out, err := run(ctx, r.Root, "", args...)
	return parseStashDiff(out), err
}

type StashSnapshot struct {
	Tracked, Untracked         []ChangedFile
	TrackedDiff, UntrackedDiff []FileDiff
}

// StashChanges compares the stash tree with its base. Git records untracked
// files separately in a root commit referenced by the optional third parent.
func (r *Repository) StashChanges(ctx context.Context, sha string, ignoreWhitespace bool) (StashSnapshot, error) {
	stash, err := r.findStash(ctx, sha)
	if err != nil {
		return StashSnapshot{}, err
	}
	var snapshot StashSnapshot
	snapshot.Tracked, err = r.CommitFiles(ctx, sha)
	if err != nil {
		return StashSnapshot{}, err
	}
	snapshot.TrackedDiff, err = r.DisplayCommitDiff(ctx, sha, ignoreWhitespace)
	parents := strings.Fields(stash.ParentSHAs)
	if err != nil || len(parents) < 3 {
		return snapshot, err
	}
	snapshot.Untracked, err = r.CommitFiles(ctx, parents[2])
	if err != nil {
		return StashSnapshot{}, err
	}
	snapshot.UntrackedDiff, err = r.DisplayCommitDiff(ctx, parents[2], ignoreWhitespace)
	return snapshot, err
}

func (r *Repository) ApplyStash(ctx context.Context, sha string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.idle(ctx); err != nil {
		return err
	}
	// A dirty worktree is not a reason to refuse: git merges the stash into
	// it and only stops when the changes overlap, the same as Sublime Merge.
	if _, err := r.findStash(ctx, sha); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "stash", "apply", "--index", sha)
	return r.stashApplyError(ctx, err, "应用")
}

// stashApplyError turns the refusals git gives when it cannot bring a stash
// in cleanly into short hints; other failures keep git's message. A stash
// that lands with conflicts exits non-zero ("Index was not unstashed"), so
// the conflicted status is what decides that case.
func (r *Repository) stashApplyError(ctx context.Context, err error, verb string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "would be overwritten by merge"):
		return &StashApplyError{cause: err, hint: fmt.Sprintf("Stash 与工作区改动重叠，未能%s；请先提交或保存当前改动", verb)}
	case strings.Contains(msg, "CONFLICT"):
		return &StashApplyError{Conflict: true, cause: err, hint: "Stash 内容已写入工作区，但有冲突需要解决"}
	}
	if s, statusErr := r.Status(ctx); statusErr == nil {
		for _, entry := range s.Entries {
			if entry.Conflict {
				return &StashApplyError{Conflict: true, cause: err, hint: "Stash 内容已写入工作区，但有冲突需要解决"}
			}
		}
	}
	return err
}
func (r *Repository) dropStash(ctx context.Context, ref, sha string) error {
	list, err := r.Stashes(ctx)
	if err != nil {
		return err
	}
	for _, s := range list {
		if s.Ref == ref && s.SHA == sha {
			_, err := run(ctx, r.Root, "", "stash", "drop", ref)
			return err
		}
	}
	return errors.New("Stash 位置已变化，为避免删除其他条目，请刷新后重新选择")
}
func (r *Repository) DropStash(ctx context.Context, ref, sha string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	return r.dropStash(ctx, ref, sha)
}
func (r *Repository) PopStash(ctx context.Context, ref, sha string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.idle(ctx); err != nil {
		return err
	}
	// Like ApplyStash, a dirty worktree is left to git to merge; a conflict
	// keeps the entry because the drop only runs after a clean apply.
	s, err := r.findStash(ctx, sha)
	if err != nil {
		return err
	}
	if s.Ref != ref {
		return errors.New("Stash 位置已变化，请刷新")
	}
	if _, err := run(ctx, r.Root, "", "stash", "apply", "--index", sha); err != nil {
		return r.stashApplyError(ctx, err, "弹出")
	}
	return r.dropStash(ctx, ref, sha)
}
