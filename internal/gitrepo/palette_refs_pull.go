package gitrepo

import (
	"context"
	"errors"
)

// PullBranchChecked fetches the selected advertised branch and integrates its
// displayed commit. A concurrent fetch cannot retarget the operation through
// FETCH_HEAD, and a checkout during transport is detected before integration.
func (r *Repository) PullBranchChecked(ctx context.Context, remote Remote, branch Branch, current Status, mode string) error {
	switch mode {
	case "ff-only", "merge", "rebase":
	default:
		return errors.New("无效的拉取方式")
	}
	if current.Branch == "" || !oidRE.MatchString(current.Head) || !oidRE.MatchString(branch.SHA) {
		return ErrIntegrationTargetChanged
	}
	if err := r.fetchPullBranch(ctx, remote, branch, current); err != nil {
		return err
	}
	if mode != "ff-only" {
		return r.IntegrateChecked(ctx, mode, branch.SHA, current)
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.checkPullHead(ctx, current); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "merge", "--ff-only", "--no-edit", "--no-autostash", "--no-overwrite-ignore", "--", branch.SHA)
	return err
}

func (r *Repository) checkPullHead(ctx context.Context, expected Status) error {
	if err := r.idle(ctx); err != nil {
		return err
	}
	current, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if current.Head != expected.Head || current.Branch != expected.Branch {
		return ErrIntegrationTargetChanged
	}
	if len(current.Entries) != 0 {
		return ErrDirtyWorkingTree
	}
	return nil
}

func (r *Repository) fetchPullBranch(ctx context.Context, remote Remote, branch Branch, current Status) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, branch.Name); err != nil {
		return err
	}
	if err := r.checkPullHead(ctx, current); err != nil {
		return err
	}
	if err := r.checkRemoteTarget(ctx, remote); err != nil {
		return err
	}
	if err := r.checkRemote(ctx, remote.Name, false); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "fetch", "--no-write-fetch-head", "--no-recurse-submodules", "--", remote.Name, "refs/heads/"+branch.Name)
	if err != nil {
		return err
	}
	// Verify the advertised branch still names the confirmed commit, even if
	// that commit already existed locally before fetching a changed remote.
	out, err := run(ctx, r.Root, "", "ls-remote", "--refs", "--", remote.Name, "refs/heads/"+branch.Name)
	if err != nil {
		return err
	}
	if out != branch.SHA+"\trefs/heads/"+branch.Name+"\n" {
		return ErrIntegrationTargetChanged
	}
	return nil
}
