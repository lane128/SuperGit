package gitrepo

import (
	"context"
	"errors"
	"strings"
)

// RenameRemoteChecked and RemoveRemoteChecked retain the selected destinations.
func (r *Repository) RenameRemoteChecked(ctx context.Context, remote Remote, name string) error {
	if !remoteNameRE.MatchString(name) {
		return errors.New("无效的远端名称")
	}
	return r.changeRemoteChecked(ctx, remote, "rename", name)
}

func (r *Repository) RemoveRemoteChecked(ctx context.Context, remote Remote) error {
	return r.changeRemoteChecked(ctx, remote, "remove")
}

func (r *Repository) changeRemoteChecked(ctx context.Context, remote Remote, operation string, args ...string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.checkRemoteTarget(ctx, remote); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", append([]string{"remote", operation, "--", remote.Name}, args...)...)
	return err
}

var ErrUpstreamTargetChanged = errors.New("branch or tracking configuration changed before updating upstream")

// ChangeUpstreamChecked checks the previous upstream even when unsetting it.
// A nil target removes tracking; setting it uses a fully qualified remote ref.
func (r *Repository) ChangeUpstreamChecked(ctx context.Context, branch Branch, target *Branch) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.branchName(ctx, branch.Name); err != nil {
		return err
	}
	branches, err := r.Branches(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, current := range branches {
		if current.Name == branch.Name && current.SHA == branch.SHA && current.Upstream == branch.Upstream {
			found = true
			break
		}
	}
	if !found {
		return ErrUpstreamTargetChanged
	}
	if target == nil {
		if branch.Upstream == "" {
			return ErrUpstreamTargetChanged
		}
		_, err = run(ctx, r.Root, "", "branch", "--unset-upstream", "--", branch.Name)
		return err
	}
	ref := "refs/remotes/" + target.Name
	if _, err := run(ctx, r.Root, "", "check-ref-format", ref); err != nil {
		return err
	}
	sha, err := r.resolve(ctx, ref)
	if err != nil || sha != target.SHA {
		return errors.Join(ErrUpstreamTargetChanged, err)
	}
	_, err = run(ctx, r.Root, "", "branch", "--set-upstream-to="+ref, "--", branch.Name)
	return err
}

// AdvertisedRefs reads heads from the fetch endpoint or tags from the push
// endpoint. Remote-only tags and annotated tag objects must be selectable.
func (r *Repository) AdvertisedRefs(ctx context.Context, remote Remote, tags bool) ([]Branch, error) {
	if err := r.checkRemoteTarget(ctx, remote); err != nil {
		return nil, err
	}
	if err := r.checkRemote(ctx, remote.Name, tags); err != nil {
		return nil, err
	}
	address, prefix := remote.Name, "refs/heads/"
	if tags {
		raw, err := run(ctx, r.Root, "", "remote", "get-url", "--push", "--", remote.Name)
		if err != nil {
			return nil, err
		}
		address, prefix = strings.TrimSuffix(raw, "\n"), "refs/tags/"
		if err := validURL(address); err != nil {
			return nil, err
		}
	}
	out, err := run(ctx, r.Root, "", "ls-remote", "--refs", "--", address, prefix+"*")
	if err != nil {
		return nil, err
	}
	var result []Branch
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		sha, ref, ok := strings.Cut(line, "\t")
		name, hasPrefix := strings.CutPrefix(ref, prefix)
		if !ok || !hasPrefix || name == "" || !oidRE.MatchString(sha) {
			return nil, errors.New("无效的远端引用")
		}
		result = append(result, Branch{Name: name, SHA: sha})
	}
	return result, nil
}

// DeleteRemoteTagChecked leases the advertised tag object on the push endpoint.
func (r *Repository) DeleteRemoteTagChecked(ctx context.Context, remote Remote, name, object string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if !oidRE.MatchString(object) || name == "" {
		return ErrTagTargetChanged
	}
	ref := "refs/tags/" + name
	if _, err := run(ctx, r.Root, "", "check-ref-format", ref); err != nil {
		return err
	}
	if err := r.checkRemoteTarget(ctx, remote); err != nil {
		return err
	}
	if err := r.checkRemote(ctx, remote.Name, true); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "push", "--no-follow-tags", "--no-recurse-submodules", "--force-with-lease="+ref+":"+object, "--", remote.Name, ":"+ref)
	return err
}

// CreateBranchFromTagChecked rejects a retagged name while the branch name
// was being entered. CreateBranch pins the peeled commit through checkout.
func (r *Repository) CreateBranchFromTagChecked(ctx context.Context, name string, tag Tag) error {
	if tag.Name == "" || !oidRE.MatchString(tag.ObjectSHA) || tag.TargetType != "commit" || !oidRE.MatchString(tag.TargetSHA) {
		return ErrSwitchTargetChanged
	}
	ref := "refs/tags/" + tag.Name
	object, err := run(ctx, r.Root, "", "show-ref", "--verify", "--hash", "--", ref)
	if err != nil || strings.TrimSpace(object) != tag.ObjectSHA {
		return errors.Join(ErrSwitchTargetChanged, err)
	}
	sha, err := r.resolve(ctx, tag.ObjectSHA)
	if err != nil || sha != tag.TargetSHA {
		return errors.Join(ErrSwitchTargetChanged, err)
	}
	return r.CreateBranch(ctx, name, sha, true)
}
