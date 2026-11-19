package gitrepo

import (
	"context"
	"errors"
	"strings"
)

// ErrDirtyWorkingTree identifies the shared clean-checkout precondition.
// Keep its legacy text for callers that render backend errors directly.
var ErrDirtyWorkingTree = errors.New("工作区或暂存区有改动，请先提交或保存 Stash")

func (r *Repository) clean(ctx context.Context) error {
	if err := r.idle(ctx); err != nil {
		return err
	}
	s, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if len(s.Entries) > 0 {
		return ErrDirtyWorkingTree
	}
	return nil
}
func (r *Repository) branchName(ctx context.Context, name string) error {
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n") {
		return errors.New("无效的分支名称")
	}
	_, err := run(ctx, r.Root, "", "check-ref-format", "--branch", name)
	return err
}
func (r *Repository) resolve(ctx context.Context, ref string) (string, error) {
	if ref == "" || strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, "\x00\r\n") {
		return "", errors.New("无效的提交引用")
	}
	out, err := run(ctx, r.Root, "", "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	return strings.TrimSpace(out), err
}

// ResolveRef turns a ref such as HEAD, a branch or an upstream name into the
// full SHA of the commit it points at, rejecting refs that could be read as
// flags.
func (r *Repository) ResolveRef(ctx context.Context, ref string) (string, error) {
	return r.resolve(ctx, ref)
}
func (r *Repository) CreateBranch(ctx context.Context, name, start string, checkout bool) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, name); err != nil {
		return err
	}
	sha, err := r.resolve(ctx, start)
	if err != nil {
		return err
	}
	if checkout {
		if err := r.clean(ctx); err != nil {
			return err
		}
		_, err = run(ctx, r.Root, "", "switch", "--no-overwrite-ignore", "-c", name, sha)
	} else {
		_, err = run(ctx, r.Root, "", "branch", "--", name, sha)
	}
	return err
}

var ErrSwitchTargetChanged = errors.New("branch target changed before checkout")

// SwitchBranchChecked keeps a branch selection tied to its displayed commit.
// External Git processes do not share the repository's write lock.
func (r *Repository) SwitchBranchChecked(ctx context.Context, name, expectedSHA string) error {
	if !oidRE.MatchString(expectedSHA) {
		return ErrSwitchTargetChanged
	}
	return r.switchBranch(ctx, name, expectedSHA)
}

func (r *Repository) SwitchBranch(ctx context.Context, name string) error {
	return r.switchBranch(ctx, name, "")
}

func (r *Repository) switchBranch(ctx context.Context, name, expectedSHA string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, name); err != nil {
		return err
	}
	if err := r.clean(ctx); err != nil {
		return err
	}
	if expectedSHA != "" {
		current, err := r.resolve(ctx, "refs/heads/"+name)
		if err != nil {
			return errors.Join(ErrSwitchTargetChanged, err)
		}
		if current != expectedSHA {
			return ErrSwitchTargetChanged
		}
	}
	_, err := run(ctx, r.Root, "", "switch", "--no-guess", "--no-overwrite-ignore", "--", name)
	return err
}

// CheckoutCommit puts the repository on a detached HEAD at sha. It requires a
// clean tree like a branch switch, so checking out an old commit can never
// drop uncommitted work.
func (r *Repository) CheckoutCommit(ctx context.Context, sha string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if !oidRE.MatchString(sha) {
		return errors.New("无效的提交 SHA")
	}
	if err := r.clean(ctx); err != nil {
		return err
	}
	if _, err := r.resolve(ctx, sha); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "switch", "--detach", "--no-guess", "--no-overwrite-ignore", "--", sha)
	return err
}

var ErrResetTargetChanged = errors.New("repository HEAD or branch changed before reset")

// ResetToChecked rechecks the original HEAD and branch under the repository's
// write lock. External Git processes do not share that lock.
func (r *Repository) ResetToChecked(ctx context.Context, sha, mode string, expected Status) error {
	return r.resetTo(ctx, sha, mode, &expected)
}

// ResetTo moves the current branch (or detached HEAD) to sha. Soft keeps the
// index and working tree, mixed keeps the working tree and unstages, hard
// discards both. The SHA is validated so it can never be read as an option.
func (r *Repository) ResetTo(ctx context.Context, sha, mode string) error {
	return r.resetTo(ctx, sha, mode, nil)
}

func (r *Repository) resetTo(ctx context.Context, sha, mode string, expected *Status) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if !oidRE.MatchString(sha) {
		return errors.New("无效的提交 SHA")
	}
	switch mode {
	case "soft", "mixed", "hard":
	default:
		return errors.New("不支持的重置模式")
	}
	if expected != nil {
		current, err := r.Status(ctx)
		if err != nil {
			return err
		}
		if current.Head != expected.Head || current.Branch != expected.Branch {
			return ErrResetTargetChanged
		}
	}
	if err := r.idle(ctx); err != nil {
		return err
	}
	if _, err := r.resolve(ctx, sha); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "reset", "--"+mode, sha)
	return err
}

var ErrRenameTargetChanged = errors.New("branch target changed before rename")

// RenameBranchChecked keeps the input attached to its original branch tip.
// External Git processes do not share this repository's write lock.
func (r *Repository) RenameBranchChecked(ctx context.Context, old, name, expectedSHA string) error {
	if !oidRE.MatchString(expectedSHA) {
		return ErrRenameTargetChanged
	}
	return r.renameBranch(ctx, old, name, expectedSHA)
}

func (r *Repository) RenameBranch(ctx context.Context, old, name string) error {
	return r.renameBranch(ctx, old, name, "")
}

func (r *Repository) renameBranch(ctx context.Context, old, name, expectedSHA string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, old); err != nil {
		return err
	}
	if err := r.branchName(ctx, name); err != nil {
		return err
	}
	if expectedSHA != "" {
		current, err := r.resolve(ctx, "refs/heads/"+old)
		if err != nil {
			return errors.Join(ErrRenameTargetChanged, err)
		}
		if current != expectedSHA {
			return ErrRenameTargetChanged
		}
	}
	_, err := run(ctx, r.Root, "", "branch", "-m", "--", old, name)
	return err
}

var ErrBranchTargetChanged = errors.New("branch target changed before deletion")

// DeleteBranchChecked preserves Git's branch/worktree safety checks while
// requiring the confirmation's original tip. External Git has its own locks.
func (r *Repository) DeleteBranchChecked(ctx context.Context, name, expectedSHA string, force bool) error {
	if !oidRE.MatchString(expectedSHA) {
		return ErrBranchTargetChanged
	}
	return r.deleteBranch(ctx, name, force, expectedSHA)
}

func (r *Repository) DeleteBranch(ctx context.Context, name string) error {
	return r.deleteBranch(ctx, name, false, "")
}

// ForceDeleteBranch deletes a branch even when Git considers it unmerged.
// Git still refuses to delete the branch checked out in a worktree.
func (r *Repository) ForceDeleteBranch(ctx context.Context, name string) error {
	return r.deleteBranch(ctx, name, true, "")
}

func (r *Repository) deleteBranch(ctx context.Context, name string, force bool, expectedSHA string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, name); err != nil {
		return err
	}
	if expectedSHA != "" {
		current, err := r.resolve(ctx, "refs/heads/"+name)
		if err != nil {
			return errors.Join(ErrBranchTargetChanged, err)
		}
		if current != expectedSHA {
			return ErrBranchTargetChanged
		}
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := run(ctx, r.Root, "", "branch", flag, "--", name)
	return err
}
func (r *Repository) TrackBranch(ctx context.Context, name, remoteRef string) error {
	return r.trackBranch(ctx, name, remoteRef, "")
}

// TrackBranchChecked rejects a remote-tracking ref whose tip no longer matches
// the displayed candidate. The instance lock does not serialize external Git.
func (r *Repository) TrackBranchChecked(ctx context.Context, name, remoteRef, expectedSHA string) error {
	if !oidRE.MatchString(expectedSHA) {
		return ErrSwitchTargetChanged
	}
	return r.trackBranch(ctx, name, remoteRef, expectedSHA)
}

func (r *Repository) trackBranch(ctx context.Context, name, remoteRef, expectedSHA string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, name); err != nil {
		return err
	}
	if strings.HasPrefix(remoteRef, "-") {
		return errors.New("无效的远端引用")
	}
	sha, err := r.resolve(ctx, "refs/remotes/"+remoteRef)
	if err != nil {
		if expectedSHA != "" {
			return errors.Join(ErrSwitchTargetChanged, err)
		}
		return err
	}
	if expectedSHA != "" && sha != expectedSHA {
		return ErrSwitchTargetChanged
	}
	if err := r.clean(ctx); err != nil {
		return err
	}
	_, err = run(ctx, r.Root, "", "switch", "--no-overwrite-ignore", "-c", name, "--track", "refs/remotes/"+remoteRef)
	return err
}
