package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const outputLimit = 4 << 20

type Repository struct {
	Root    string
	writeMu sync.Mutex
}
type limitedBuffer struct {
	// A named field avoids promoting bytes.Buffer.ReadFrom, which io.Copy
	// would otherwise use to bypass Write's bound when draining subprocesses.
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) String() string { return b.buffer.String() }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	left := b.limit - b.buffer.Len()
	if len(p) > left {
		b.truncated = true
		p = p[:max(0, left)]
	}
	b.buffer.Write(p)
	return n, nil
}

func run(ctx context.Context, root string, input string, args ...string) (string, error) {
	stdout := limitedBuffer{limit: outputLimit}
	err := runOutput(ctx, root, input, &stdout, args...)
	if err == nil && stdout.truncated {
		err = errors.New("Git 输出超过 4 MiB 显示上限；操作可能已经完成，请刷新状态确认")
	}
	return stdout.String(), err
}

// runOutput shares repository routing, cancellation and bounded stderr between
// ordinary reads and streaming readers that retain only their requested page.
func runOutput(ctx context.Context, root string, input string, stdout io.Writer, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	// log.showSignature prints verification text into stdout, which every
	// parser here would read as history: a user with the option set would get
	// bogus SHAs instead of commits. Signature status is read separately by
	// the details pane, so the display setting never belongs in these reads.
	argv := []string{"--no-pager", "--literal-pathspecs", "-c", "color.ui=false", "-c", "core.quotepath=false", "-c", "log.showSignature=false"}
	if root != "" {
		argv = append(argv, "-C", root)
	}
	argv = append(argv, args...)
	cmd := exec.CommandContext(ctx, "git", argv...)
	cmd.WaitDelay = time.Second
	configureProcess(cmd)
	// A shell launched from a hook/worktree may carry overrides for a different
	// repository or index. Repository selection, not that inherited routing,
	// determines where every operation runs. User configuration is preserved.
	blocked := map[string]bool{"GIT_DIR": true, "GIT_WORK_TREE": true, "GIT_INDEX_FILE": true, "GIT_COMMON_DIR": true, "GIT_OBJECT_DIRECTORY": true, "GIT_ALTERNATE_OBJECT_DIRECTORIES": true, "GIT_NAMESPACE": true, "GIT_PREFIX": true}
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		if !blocked[name] {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	var stderr = limitedBuffer{limit: 256 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = &stderr
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("Git 操作已取消或超时: %w", ctx.Err())
		}
		return fmt.Errorf("git %s: %w\n%s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	if stderr.truncated {
		return errors.New("Git 输出超过 4 MiB 显示上限；操作可能已经完成，请刷新状态确认")
	}
	return nil
}
func Open(ctx context.Context, path string) (*Repository, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	out, err := run(ctx, abs, "", "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("无法打开 Git 工作区: %w", err)
	}
	root := strings.TrimSuffix(out, "\n")
	if root == "" {
		return nil, errors.New("该路径不是 Git 工作区")
	}
	return &Repository{Root: root}, nil
}
func (r *Repository) path(path string) error {
	if path == "" || strings.IndexByte(path, 0) >= 0 || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(filepath.Clean(path), ".."+string(filepath.Separator)) {
		return fmt.Errorf("不允许访问仓库外的路径: %q", path)
	}
	return nil
}
func (r *Repository) hasHead(ctx context.Context) bool {
	_, err := run(ctx, r.Root, "", "rev-parse", "--verify", "HEAD")
	return err == nil
}
func (r *Repository) Stage(ctx context.Context, path string) error {
	if err := r.path(path); err != nil {
		return err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	_, err := run(ctx, r.Root, "", "add", "--", path)
	return err
}
func (r *Repository) Unstage(ctx context.Context, path string) error {
	if err := r.path(path); err != nil {
		return err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	args := []string{"restore", "--staged", "--", path}
	if !r.hasHead(ctx) {
		args = []string{"rm", "--cached", "--force", "--ignore-unmatch", "--", path}
	}
	_, err := run(ctx, r.Root, "", args...)
	return err
}

// stageChunk is how many paths one git command carries. A big repository's
// stage-all can hold tens of thousands of paths, and one argument list would
// run into the platform's limit; 256 paths keeps every call well under it.
const stageChunk = 256

// StagePaths stages many paths with one git command per chunk. A chunk that
// fails falls back to one path at a time, so a path that vanished between the
// status read and the command does not keep the rest out of the index.
func (r *Repository) StagePaths(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	for _, path := range paths {
		if err := r.path(path); err != nil {
			return err
		}
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	var first error
	for start := 0; start < len(paths); start += stageChunk {
		chunk := paths[start:min(start+stageChunk, len(paths))]
		if _, err := run(ctx, r.Root, "", append([]string{"add", "--"}, chunk...)...); err == nil {
			continue
		}
		for _, path := range chunk {
			if _, err := run(ctx, r.Root, "", "add", "--", path); err != nil && first == nil {
				first = err
			}
		}
	}
	return first
}

// UnstagePaths unstages many paths with one git command per chunk, with the
// same one-at-a-time fallback StagePaths has. Renamed entries pass both their
// old and their new path, which is what one `git restore --staged` needs.
func (r *Repository) UnstagePaths(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	for _, path := range paths {
		if err := r.path(path); err != nil {
			return err
		}
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	base := []string{"restore", "--staged", "--"}
	if !r.hasHead(ctx) {
		base = []string{"rm", "--cached", "--force", "--ignore-unmatch", "--"}
	}
	var first error
	for start := 0; start < len(paths); start += stageChunk {
		chunk := paths[start:min(start+stageChunk, len(paths))]
		args := append(append([]string{}, base...), chunk...)
		if _, err := run(ctx, r.Root, "", args...); err == nil {
			continue
		}
		for _, path := range chunk {
			one := append(append([]string{}, base...), path)
			if _, err := run(ctx, r.Root, "", one...); err != nil && first == nil {
				first = err
			}
		}
	}
	return first
}

// Discard restores a tracked path's working tree from the index, or removes
// an untracked file. Untracked paths are only removed when the caller says
// so, after confirming it with the user.
func (r *Repository) Discard(ctx context.Context, path string, untracked bool) error {
	if err := r.path(path); err != nil {
		return err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if untracked {
		root, err := os.OpenRoot(r.Root)
		if err != nil {
			return err
		}
		defer root.Close()
		return root.Remove(path)
	}
	_, err := run(ctx, r.Root, "", "restore", "--worktree", "--", path)
	return err
}
func (r *Repository) Commit(ctx context.Context, message string) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("提交标题不能为空")
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	_, err := run(ctx, r.Root, "", "commit", "-m", message)
	return err
}

// UndoCommit moves HEAD to its first parent and leaves the index and working
// tree untouched, so the commit's changes stay staged.
var ErrUndoTargetChanged = errors.New("HEAD or branch changed before undo")

// UndoCommitChecked keeps Undo attached to the originally selected HEAD and
// branch. External Git processes do not share this repository's write lock.
func (r *Repository) UndoCommitChecked(ctx context.Context, expected Status) error {
	if !oidRE.MatchString(expected.Head) {
		return ErrUndoTargetChanged
	}
	return r.undoCommit(ctx, &expected)
}

func (r *Repository) UndoCommit(ctx context.Context) error {
	return r.undoCommit(ctx, nil)
}

func (r *Repository) undoCommit(ctx context.Context, expected *Status) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if expected != nil {
		current, err := r.Status(ctx)
		if err != nil {
			return err
		}
		if current.Head != expected.Head || current.Branch != expected.Branch {
			return ErrUndoTargetChanged
		}
		if err := r.idle(ctx); err != nil {
			return errors.Join(ErrUndoTargetChanged, err)
		}
	}
	if !r.hasHead(ctx) {
		return errors.New("还没有可撤销的提交")
	}
	parents, err := run(ctx, r.Root, "", "show", "-s", "--format=%P", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(parents) == "" {
		return errors.New("根提交不能撤销")
	}
	_, err = run(ctx, r.Root, "", "reset", "--soft", "HEAD^")
	return err
}

// AmendCommit replaces HEAD with a new commit that has message, keeping the
// staged tree. It works with or without newly staged changes.
func (r *Repository) AmendCommit(ctx context.Context, message string) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("提交标题不能为空")
	}
	if !r.hasHead(ctx) {
		return errors.New("还没有可修改的提交")
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	_, err := run(ctx, r.Root, "", "commit", "--amend", "-m", message)
	return err
}

// StageRename updates an already staged rename's destination. Reversing restores
// both index paths atomically; neither operation changes working files.
func (r *Repository) StageRename(ctx context.Context, path, original string, reverse bool) error {
	if err := r.path(path); err != nil {
		return err
	}
	if err := r.path(original); err != nil {
		return err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	args := []string{"add", "--", path}
	if reverse {
		args = []string{"restore", "--staged", "--", original, path}
		if !r.hasHead(ctx) {
			args = []string{"rm", "--cached", "--force", "--ignore-unmatch", "--", original, path}
		}
	}
	_, err := run(ctx, r.Root, "", args...)
	return err
}
func (r *Repository) ApplyHunk(ctx context.Context, patch string, reverse bool) error {
	if !strings.Contains(patch, "\n@@ ") {
		return errors.New("不是有效的文本差异片段")
	}
	path := patchPath(patch)
	if path == "" {
		return errors.New("不是有效的文本差异片段")
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	// Recheck the file's status the way ApplyLines does: a patch that was
	// stageable when it was drawn may describe a submodule, a conflict or a
	// rename by the time it runs, and those go through the file row. A dirty
	// submodule's pointer body in particular looks like a hunk but stages
	// nothing.
	status, err := r.Status(ctx)
	if err != nil {
		return err
	}
	allowed := false
	for _, entry := range status.Entries {
		if entry.Path == path {
			allowed = !entry.Conflict && !entry.IsSubmodule() && entry.OriginalPath == ""
			break
		}
	}
	if !allowed {
		return errors.New("此文件需要文件级暂存，不能按片段操作")
	}
	args := []string{"apply", "--cached", "--whitespace=nowarn"}
	if reverse {
		args = append(args, "--reverse")
	}
	if _, err := run(ctx, r.Root, patch, append(args, "--check")...); err != nil {
		return err
	}
	_, err = run(ctx, r.Root, patch, args...)
	return err
}
