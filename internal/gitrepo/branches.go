package gitrepo

import (
	"context"
	"errors"
	"strings"
)

func (r *Repository) clean(ctx context.Context) error {
	if err := r.idle(ctx); err != nil {
		return err
	}
	s, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if len(s.Entries) > 0 {
		return errors.New("工作区或暂存区有改动，请先提交或保存 Stash")
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
func (r *Repository) SwitchBranch(ctx context.Context, name string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, name); err != nil {
		return err
	}
	if err := r.clean(ctx); err != nil {
		return err
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

// ResetTo moves the current branch (or detached HEAD) to sha. Soft keeps the
// index and working tree, mixed keeps the working tree and unstages, hard
// discards both. The SHA is validated so it can never be read as an option.
func (r *Repository) ResetTo(ctx context.Context, sha, mode string) error {
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
	if err := r.idle(ctx); err != nil {
		return err
	}
	if _, err := r.resolve(ctx, sha); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "reset", "--"+mode, sha)
	return err
}

func (r *Repository) RenameBranch(ctx context.Context, old, name string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, old); err != nil {
		return err
	}
	if err := r.branchName(ctx, name); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "branch", "-m", "--", old, name)
	return err
}
func (r *Repository) DeleteBranch(ctx context.Context, name string) error {
	return r.deleteBranch(ctx, name, false)
}

// ForceDeleteBranch deletes a branch even when Git considers it unmerged.
// Git still refuses to delete the branch checked out in a worktree.
func (r *Repository) ForceDeleteBranch(ctx context.Context, name string) error {
	return r.deleteBranch(ctx, name, true)
}

func (r *Repository) deleteBranch(ctx context.Context, name string, force bool) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, name); err != nil {
		return err
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := run(ctx, r.Root, "", "branch", flag, "--", name)
	return err
}
func (r *Repository) TrackBranch(ctx context.Context, name, remoteRef string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, name); err != nil {
		return err
	}
	if strings.HasPrefix(remoteRef, "-") {
		return errors.New("无效的远端引用")
	}
	if _, err := r.resolve(ctx, "refs/remotes/"+remoteRef); err != nil {
		return err
	}
	if err := r.clean(ctx); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "switch", "--no-overwrite-ignore", "-c", name, "--track", "refs/remotes/"+remoteRef)
	return err
}
