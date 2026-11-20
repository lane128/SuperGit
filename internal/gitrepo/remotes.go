package gitrepo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Remote struct{ Name, URL, PushURL, Fingerprint string }

var ErrRemoteTargetChanged = errors.New("remote configuration changed before write")

// checkRemoteTarget compares the effective fetch and push destinations captured
// by Remotes. The repository mutex cannot serialize external Git config edits.
func (r *Repository) checkRemoteTarget(ctx context.Context, expected Remote) error {
	if !remoteNameRE.MatchString(expected.Name) || expected.Fingerprint == "" {
		return ErrRemoteTargetChanged
	}
	fetch, err := run(ctx, r.Root, "", "remote", "get-url", "--all", "--", expected.Name)
	if err != nil {
		return errors.Join(ErrRemoteTargetChanged, err)
	}
	push, err := run(ctx, r.Root, "", "remote", "get-url", "--push", "--all", "--", expected.Name)
	if err != nil {
		return errors.Join(ErrRemoteTargetChanged, err)
	}
	if fmt.Sprintf("%x", sha256.Sum256([]byte(fetch+"\x00"+push))) != expected.Fingerprint {
		return ErrRemoteTargetChanged
	}
	return nil
}

var remoteNameRE = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)

func validURL(value string) error {
	if value == "" || strings.HasPrefix(value, "-") || strings.ContainsAny(value, "\x00\r\n") {
		return errors.New("请输入有效的 Git 地址")
	}
	if strings.Contains(value, "::") {
		return errors.New("不支持外部命令传输协议")
	}
	if strings.Contains(value, "://") {
		u, err := url.Parse(value)
		if err != nil {
			return err
		}
		switch u.Scheme {
		case "https", "http", "ssh", "git", "file":
		default:
			return errors.New("不支持此 Git 地址协议")
		}
		if u.User != nil {
			if _, ok := u.User.Password(); ok {
				return errors.New("地址中不能包含密码或令牌，请使用系统凭据助手")
			}
		}
	}
	return nil
}
func newDestination(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(abs); !errors.Is(err, os.ErrNotExist) {
		return "", errors.New("目标目录已经存在，请选择尚不存在的新目录")
	}
	return abs, nil
}
func Init(ctx context.Context, path string) (*Repository, error) {
	abs, err := newDestination(path)
	if err != nil {
		return nil, err
	}
	if _, err := run(ctx, "", "", "init", "-b", "main", "--", abs); err != nil {
		return nil, err
	}
	return Open(ctx, abs)
}
func Clone(ctx context.Context, address, path string) (*Repository, error) {
	if err := validURL(address); err != nil {
		return nil, err
	}
	abs, err := newDestination(path)
	if err != nil {
		return nil, err
	}
	if _, err := run(ctx, "", "", "clone", "--no-recurse-submodules", "--", address, abs); err != nil {
		return nil, err
	}
	return Open(ctx, abs)
}
func (r *Repository) Remotes(ctx context.Context) ([]Remote, error) {
	out, err := run(ctx, r.Root, "", "remote")
	if err != nil {
		return nil, err
	}
	var result []Remote
	for _, name := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if name == "" {
			continue
		}
		fetch, e := run(ctx, r.Root, "", "remote", "get-url", "--all", "--", name)
		if e != nil {
			return nil, e
		}
		push, e := run(ctx, r.Root, "", "remote", "get-url", "--push", "--all", "--", name)
		if e != nil {
			return nil, e
		}
		result = append(result, Remote{Name: name, URL: strings.SplitN(strings.TrimSuffix(fetch, "\n"), "\n", 2)[0], PushURL: strings.SplitN(strings.TrimSuffix(push, "\n"), "\n", 2)[0], Fingerprint: fmt.Sprintf("%x", sha256.Sum256([]byte(fetch+"\x00"+push)))})
	}
	return result, nil
}
func (r *Repository) checkRemote(ctx context.Context, name string, push bool) error {
	if !remoteNameRE.MatchString(name) {
		return errors.New("无效的远端名称")
	}
	args := []string{"remote", "get-url"}
	if push {
		args = append(args, "--push")
	}
	args = append(args, "--all", "--", name)
	out, err := run(ctx, r.Root, "", args...)
	if err != nil {
		return err
	}
	addresses := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(addresses) != 1 {
		return errors.New("此远端有多个地址，请先在终端明确单一目标后再使用同步操作")
	}
	for _, address := range addresses {
		if err := validURL(address); err != nil {
			return err
		}
	}
	return nil
}
func (r *Repository) AddRemote(ctx context.Context, name, address string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if !remoteNameRE.MatchString(name) {
		return errors.New("无效的远端名称")
	}
	if err := validURL(address); err != nil {
		return err
	}
	_, err := run(ctx, r.Root, "", "remote", "add", "--", name, address)
	return err
}
func (r *Repository) RemoteBranches(ctx context.Context) ([]Branch, error) {
	out, err := run(ctx, r.Root, "", "for-each-ref", "--format=%(refname:strip=2)%00%(objectname)%00%(symref)%00%(subject)%00%(authorname)%00%(authordate:iso-strict)", "refs/remotes")
	if err != nil {
		return nil, err
	}
	var result []Branch
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		p := strings.Split(line, "\x00")
		if len(p) == 6 && p[2] == "" {
			result = append(result, Branch{Name: p[0], SHA: p[1],
				Commit: Commit{SHA: p[1], Subject: p[3], Author: p[4], Date: p[5], Day: localDay(p[5])}})
		}
	}
	return result, nil
}

var ErrRemoteDeleteTargetChanged = errors.New("remote branch changed or deletion was refused")

// DeleteRemoteBranchChecked uses an explicit expected OID, so a background
// fetch cannot silently authorize deletion of a newer remote branch tip.
func (r *Repository) DeleteRemoteBranchChecked(ctx context.Context, remote Remote, branch, expectedSHA string) error {
	if !oidRE.MatchString(expectedSHA) {
		return ErrRemoteDeleteTargetChanged
	}
	return r.deleteRemoteBranch(ctx, remote.Name, branch, &remote, expectedSHA)
}

// DeleteRemoteBranch removes a branch from the remote. Git also drops the
// matching remote-tracking ref as part of the successful push.
func (r *Repository) DeleteRemoteBranch(ctx context.Context, remote, branch string) error {
	return r.deleteRemoteBranch(ctx, remote, branch, nil, "")
}

func (r *Repository) deleteRemoteBranch(ctx context.Context, remote, branch string, expectedRemote *Remote, expectedSHA string) error {
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
	if err := r.branchName(ctx, branch); err != nil {
		return err
	}
	args := []string{"push", "--no-recurse-submodules"}
	if expectedSHA != "" {
		args = append(args, "--force-with-lease=refs/heads/"+branch+":"+expectedSHA)
	}
	args = append(args, "--", remote, ":refs/heads/"+branch)
	_, err := run(ctx, r.Root, "", args...)
	if err != nil && expectedSHA != "" {
		return errors.Join(ErrRemoteDeleteTargetChanged, err)
	}
	return err
}

type FetchMode uint8

const (
	FetchDefault FetchMode = iota
	FetchWithPrune
	FetchWithTags
)

func (r *Repository) Fetch(ctx context.Context, remote string) error {
	return r.FetchRemote(ctx, remote, FetchDefault)
}

func (r *Repository) FetchPrune(ctx context.Context, remote string) error {
	return r.FetchRemote(ctx, remote, FetchWithPrune)
}

func (r *Repository) FetchRemote(ctx context.Context, remote string, mode FetchMode) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.checkRemote(ctx, remote, false); err != nil {
		return err
	}
	args := []string{"fetch", "--no-recurse-submodules"}
	switch mode {
	case FetchWithPrune:
		args = append(args, "--prune")
	case FetchWithTags:
		args = append(args, "--tags")
	case FetchDefault:
	default:
		return errors.New("unknown fetch option")
	}
	_, err := run(ctx, r.Root, "", append(args, "--", remote)...)
	return err
}

func (r *Repository) PullFF(ctx context.Context) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.clean(ctx); err != nil {
		return err
	}
	s, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if s.Upstream == "" {
		return errors.New("当前分支没有上游，请先推送并设置上游或创建远端跟踪分支")
	}
	remote, err := run(ctx, r.Root, "", "config", "--get", "branch."+s.Branch+".remote")
	if err != nil {
		return err
	}
	if err := r.checkRemote(ctx, strings.TrimSpace(remote), false); err != nil {
		return err
	}
	if _, err = run(ctx, r.Root, "", "fetch", "--no-recurse-submodules", "--", strings.TrimSpace(remote)); err != nil {
		return err
	}
	sha, err := r.resolve(ctx, s.Upstream)
	if err != nil {
		return err
	}
	_, err = run(ctx, r.Root, "", "merge", "--ff-only", "--no-edit", "--no-autostash", "--no-overwrite-ignore", "--", sha)
	return err
}

var ErrPushTargetChanged = errors.New("source branch changed before push")
var ErrPushUpstreamFailed = errors.New("push succeeded but upstream setup failed")

type pushExpectation struct {
	remote    Remote
	sourceSHA string
}

// PushChecked publishes only the source commit selected by the confirmation.
func (r *Repository) PushChecked(ctx context.Context, remote Remote, source Branch, destination string, setUpstream bool) error {
	if !oidRE.MatchString(source.SHA) {
		return ErrPushTargetChanged
	}
	return r.push(ctx, remote.Name, source.Name, destination, "", setUpstream, &pushExpectation{remote: remote, sourceSHA: source.SHA})
}

func (r *Repository) Push(ctx context.Context, remote, source, destination string, setUpstream bool) error {
	return r.push(ctx, remote, source, destination, "", setUpstream, nil)
}

// PushBranch publishes a local branch without either checking it out or
// resolving it through HEAD, which is what the branch list's Push action
// needs for a branch the user is not standing on.
func (r *Repository) PushBranch(ctx context.Context, remote, branch, destination string, setUpstream bool) error {
	return r.push(ctx, remote, branch, destination, "", setUpstream, nil)
}

var ErrForcePushTargetChanged = errors.New("remote branch changed or force push was refused")

// PushForceChecked binds the lease to the confirmation, independently of any
// fetch that updates local tracking refs before the push starts.
func (r *Repository) PushForceChecked(ctx context.Context, remote Remote, source Branch, destination, expectedRemoteSHA string) error {
	if !oidRE.MatchString(source.SHA) {
		return ErrPushTargetChanged
	}
	if expectedRemoteSHA != "" && !oidRE.MatchString(expectedRemoteSHA) {
		return ErrForcePushTargetChanged
	}
	lease := "--force-with-lease=refs/heads/" + destination + ":" + expectedRemoteSHA
	return r.push(ctx, remote.Name, source.Name, destination, lease, false, &pushExpectation{remote: remote, sourceSHA: source.SHA})
}

// PushForce rewrites the remote branch with --force-with-lease, which refuses
// the push when the remote moved since the last fetch instead of discarding
// commits that were pushed from elsewhere.
func (r *Repository) PushForce(ctx context.Context, remote, source, destination string) error {
	return r.push(ctx, remote, source, destination, "--force-with-lease", false, nil)
}

func (r *Repository) push(ctx context.Context, remote, source, destination, force string, setUpstream bool, expected *pushExpectation) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if expected != nil {
		if err := r.checkRemoteTarget(ctx, expected.remote); err != nil {
			return err
		}
	}
	if err := r.checkRemote(ctx, remote, true); err != nil {
		return err
	}
	for _, name := range []string{source, destination} {
		if err := r.branchName(ctx, name); err != nil {
			return err
		}
	}
	sha, err := r.resolve(ctx, "refs/heads/"+source)
	if err != nil {
		if expected != nil {
			return errors.Join(ErrPushTargetChanged, err)
		}
		return err
	}
	if expected != nil && sha != expected.sourceSHA {
		return ErrPushTargetChanged
	}
	if err := r.checkPublishedSubmodules(ctx, sha); err != nil {
		return err
	}
	args := []string{"push", "--recurse-submodules=check"}
	if force != "" {
		args = append(args, force)
	} else {
		args = append(args, "--no-force")
	}
	if setUpstream && expected == nil {
		args = append(args, "--set-upstream")
	}
	sourceRef := "refs/heads/" + source
	if expected != nil {
		sourceRef = expected.sourceSHA
	}
	args = append(args, "--", remote, fmt.Sprintf("%s:refs/heads/%s", sourceRef, destination))
	_, err = run(ctx, r.Root, "", args...)
	if err != nil && force != "" {
		if expected != nil {
			return errors.Join(ErrForcePushTargetChanged, err)
		}
		if strings.Contains(err.Error(), "stale info") {
			return errors.New("远端自上次获取后已有新提交，请先获取再强制推送")
		}
	}
	if err == nil && expected != nil && setUpstream {
		// An OID source does not let `push --set-upstream` identify the local
		// branch. Store Git's remote/merge pair rather than assuming the remote
		// uses refs/remotes/<name>/* as its fetch mapping.
		for _, setting := range [][2]string{
			{"branch." + source + ".remote", remote},
			{"branch." + source + ".merge", "refs/heads/" + destination},
		} {
			if _, err := run(ctx, r.Root, "", "config", "--local", "--replace-all", setting[0], setting[1]); err != nil {
				return errors.Join(ErrPushUpstreamFailed, err)
			}
		}
	}
	return err
}
