package gitrepo

import (
	"context"
	"errors"
	"strings"
)

// DeleteTag removes a local tag only if it still points where it pointed when
// checked, so a tag re-created by someone else is never deleted by mistake.
func (r *Repository) DeleteTag(ctx context.Context, name string) error {
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n") {
		return errors.New("无效的标签名称")
	}
	if _, err := run(ctx, r.Root, "", "check-ref-format", "refs/tags/"+name); err != nil {
		return err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	out, err := run(ctx, r.Root, "", "show-ref", "--verify", "refs/tags/"+name)
	if err != nil {
		return errors.New("标签不存在或已变化，请刷新列表")
	}
	fields := strings.Fields(out)
	if len(fields) != 2 || !oidRE.MatchString(fields[0]) {
		return errors.New("无效的标签引用")
	}
	_, err = run(ctx, r.Root, "", "update-ref", "-d", "refs/tags/"+name, fields[0])
	return err
}

// PushTag publishes a local tag to a remote without force. The tag ref is
// pushed as-is, so an annotated tag keeps its object.
func (r *Repository) PushTag(ctx context.Context, remote, name string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.checkRemote(ctx, remote, true); err != nil {
		return err
	}
	if name == "" || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n") {
		return errors.New("无效的标签名称")
	}
	if _, err := run(ctx, r.Root, "", "check-ref-format", "refs/tags/"+name); err != nil {
		return err
	}
	if _, err := r.resolve(ctx, "refs/tags/"+name); err != nil {
		return errors.New("标签不存在或已变化，请刷新列表")
	}
	_, err := run(ctx, r.Root, "", "push", "--no-force", "--no-recurse-submodules", "--", remote, "refs/tags/"+name)
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
