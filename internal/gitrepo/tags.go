package gitrepo

import (
	"context"
	"errors"
	"strings"
)

var ErrTagTargetChanged = errors.New("tag object changed before deletion")

// DeleteTagChecked uses the original reference object, not its peeled commit.
// Git compares the old value atomically while deleting only the named ref.
func (r *Repository) DeleteTagChecked(ctx context.Context, name, expectedObjectSHA string) error {
	if !oidRE.MatchString(expectedObjectSHA) {
		return ErrTagTargetChanged
	}
	return r.deleteTag(ctx, name, expectedObjectSHA)
}

// DeleteTag removes a local tag only if it still points where it pointed when
// checked, so a tag re-created by someone else is never deleted by mistake.
func (r *Repository) DeleteTag(ctx context.Context, name string) error {
	return r.deleteTag(ctx, name, "")
}

func (r *Repository) deleteTag(ctx context.Context, name, expectedObjectSHA string) error {
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n") {
		return errors.New("无效的标签名称")
	}
	if _, err := run(ctx, r.Root, "", "check-ref-format", "refs/tags/"+name); err != nil {
		return err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	checked := expectedObjectSHA != ""
	if !checked {
		out, err := run(ctx, r.Root, "", "show-ref", "--verify", "refs/tags/"+name)
		if err != nil {
			return errors.New("标签不存在或已变化，请刷新列表")
		}
		fields := strings.Fields(out)
		if len(fields) != 2 || !oidRE.MatchString(fields[0]) {
			return errors.New("无效的标签引用")
		}
		expectedObjectSHA = fields[0]
	}
	_, err := run(ctx, r.Root, "", "update-ref", "--no-deref", "-d", "refs/tags/"+name, expectedObjectSHA)
	if err != nil && checked {
		return errors.Join(ErrTagTargetChanged, err)
	}
	return err
}

var ErrTagPushTargetChanged = errors.New("tag object changed before push")

// PushTagChecked retains the original tag object, including its annotation,
// and refuses a changed remote configuration before publishing it.
func (r *Repository) PushTagChecked(ctx context.Context, remote Remote, tag Tag) error {
	if !oidRE.MatchString(tag.ObjectSHA) {
		return ErrTagPushTargetChanged
	}
	return r.pushTag(ctx, remote.Name, tag.Name, &remote, tag.ObjectSHA)
}

// PushTag publishes a local tag to a remote without force.
func (r *Repository) PushTag(ctx context.Context, remote, name string) error {
	return r.pushTag(ctx, remote, name, nil, "")
}

func (r *Repository) pushTag(ctx context.Context, remote, name string, expectedRemote *Remote, expectedObjectSHA string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if expectedRemote != nil {
		if err := r.checkRemoteTarget(ctx, *expectedRemote); err != nil {
			return err
		}
	}
	if err := r.checkRemote(ctx, remote, true); err != nil {
		return err
	}
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n") {
		return errors.New("无效的标签名称")
	}
	if _, err := run(ctx, r.Root, "", "check-ref-format", "refs/tags/"+name); err != nil {
		return err
	}
	refspec := "refs/tags/" + name
	if expectedObjectSHA != "" {
		out, err := run(ctx, r.Root, "", "show-ref", "--verify", refspec)
		if err != nil {
			return errors.Join(ErrTagPushTargetChanged, err)
		}
		fields := strings.Fields(out)
		if len(fields) != 2 || fields[0] != expectedObjectSHA {
			return ErrTagPushTargetChanged
		}
		// Pin the source even if another process changes the local tag after
		// this check. An OID preserves annotated tag objects without peeling.
		refspec = expectedObjectSHA + ":" + refspec
	} else if _, err := r.resolve(ctx, refspec); err != nil {
		return errors.New("标签不存在或已变化，请刷新列表")
	}
	_, err := run(ctx, r.Root, "", "push", "--no-force", "--no-recurse-submodules", "--", remote, refspec)
	return err
}

// CreateAnnotatedTag writes a tag object at an immutable commit OID. Signing
// is explicitly disabled so creation never blocks on a passphrase prompt.
func (r *Repository) CreateAnnotatedTag(ctx context.Context, name, sha, message string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n") {
		return errors.New("无效的标签名称")
	}
	if _, err := run(ctx, r.Root, "", "check-ref-format", "refs/tags/"+name); err != nil {
		return err
	}
	if !oidRE.MatchString(sha) {
		return errors.New("无效的目标提交 SHA")
	}
	resolved, err := r.resolve(ctx, sha)
	if err != nil {
		return err
	}
	if resolved != sha {
		return errors.New("目标必须是提交对象 SHA")
	}
	if strings.TrimSpace(message) == "" {
		return errors.New("请填写附注说明")
	}
	_, err = run(ctx, r.Root, "", "tag", "-a", "--no-sign", "-m", message, "--", name, resolved)
	return err
}

// CreateTag creates only a local lightweight tag at an immutable commit OID.
// System Git performs the non-forced ref creation and its reference hooks.
func (r *Repository) CreateTag(ctx context.Context, name, sha string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n") {
		return errors.New("无效的标签名称")
	}
	if _, err := run(ctx, r.Root, "", "check-ref-format", "refs/tags/"+name); err != nil {
		return err
	}
	if !oidRE.MatchString(sha) {
		return errors.New("无效的目标提交 SHA")
	}
	resolved, err := r.resolve(ctx, sha)
	if err != nil {
		return err
	}
	if resolved != sha {
		return errors.New("目标必须是提交对象 SHA")
	}
	// A normal `git tag` follows dangling symbolic refs. The create transaction
	// must instead refuse any existing ref, including a dangling symbolic tag.
	// Updating the ref directly also keeps this explicitly lightweight: signing
	// and annotation configuration cannot turn it into a different operation.
	_, err = run(ctx, r.Root, "create refs/tags/"+name+" "+sha+"\n", "update-ref", "--no-deref", "--stdin")
	return err
}
