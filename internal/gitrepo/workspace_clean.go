package gitrepo

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// WorkspaceActionError lets interfaces localize safety checks and distinguish
// preflight refusals from failures after mutation may have begun. English does
// not include Cause; callers may show the original diagnostic separately.
type WorkspaceActionError struct {
	Phase    string
	Path     string
	Partial  bool
	StashOID string
	English  string
	Cause    error
	message  string
}

func (e *WorkspaceActionError) Error() string {
	if e.Cause != nil {
		return e.message + ": " + e.Cause.Error()
	}
	return e.message
}
func (e *WorkspaceActionError) Unwrap() error { return e.Cause }

func workspaceRefusal(english, chinese string, args ...any) error {
	return &WorkspaceActionError{Phase: "preflight", English: fmt.Sprintf(english, args...), message: fmt.Sprintf(chinese, args...)}
}
func workspaceFailure(phase, name, oid string, cause error) error {
	labels := map[string][2]string{
		"delete":  {"删除失败，工作区可能部分完成", "Deletion failed; workspace cleanup may be partially complete"},
		"unstage": {"撤销暂存失败，工作区可能部分完成", "Unstaging failed; workspace cleanup may be partially complete"},
		"restore": {"恢复失败，工作区可能部分完成", "Restoration failed; workspace cleanup may be partially complete"},
		"verify":  {"结果校验失败，工作区可能部分完成", "Result verification failed; the workspace may be partially changed"},
		"stash":   {"Stash 未能确认完成，请检查工作区和 Stash 列表", "Stash completion could not be confirmed; inspect the workspace and stash list"},
	}
	label := labels[phase]
	e := &WorkspaceActionError{Phase: phase, Path: name, StashOID: oid, Partial: true, English: label[1], message: label[0], Cause: cause}
	if name != "" {
		e.English += fmt.Sprintf(" (%q)", name)
		e.message += fmt.Sprintf(" (%q)", name)
	}
	if oid != "" {
		e.English += "; saved Stash " + oid
		e.message += "；已保存 Stash " + oid
	}
	return e
}

// WorkspaceCleanPreview describes the entire checkout. Its private snapshot is
// the authority: editing exported display fields invalidates the confirmation.
type WorkspaceCleanPreview struct {
	Head         string
	RestorePaths []string
	NewPaths     []string
	// IndexOnlyPaths is a subset of NewPaths whose working files are missing.
	// Cleaning discards their staged contents; it neither preserves nor deletes
	// a working file. Interfaces must disclose this instead of promising retention.
	IndexOnlyPaths []string
	DeleteNew      bool
	CanExecute     bool
	snapshot       *workspaceSnapshot
}

type workspaceTreeEntry struct{ mode, oid string }
type workspaceFile struct {
	mode   fs.FileMode
	digest string
	info   fs.FileInfo
}
type workspaceSnapshot struct {
	root, head, branch, stamp                  string
	files                                      map[string]workspaceFile
	headTree                                   map[string]workspaceTreeEntry
	restore, newPaths, removeFiles, removeDirs []string
	deleteNew, canExecute                      bool
	indexOnly                                  []string
}

func (r *Repository) PrepareWorkspaceClean(ctx context.Context, deleteNew bool) (WorkspaceCleanPreview, error) {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	return r.prepareWorkspaceClean(ctx, deleteNew)
}

func (r *Repository) prepareWorkspaceClean(ctx context.Context, deleteNew bool) (WorkspaceCleanPreview, error) {
	s, err := r.workspaceSnapshot(ctx, deleteNew)
	if err != nil {
		return WorkspaceCleanPreview{}, err
	}
	return WorkspaceCleanPreview{Head: s.head, RestorePaths: slices.Clone(s.restore), NewPaths: slices.Clone(s.newPaths), IndexOnlyPaths: slices.Clone(s.indexOnly), DeleteNew: deleteNew, CanExecute: s.canExecute, snapshot: s}, nil
}

func (r *Repository) workspaceSnapshot(ctx context.Context, deleteNew bool) (*workspaceSnapshot, error) {
	op, err := r.Operation(ctx)
	if err != nil {
		return nil, err
	}
	if op.Kind != "" {
		return nil, workspaceRefusal("A %s operation is in progress; finish or abort it first", "已有进行中的 %s，请先继续或中止", op.Kind)
	}
	// A sequencer can remain active between cherry-picks without CHERRY_PICK_HEAD.
	for _, name := range []string{"sequencer", "BISECT_LOG"} {
		p, err := r.gitPath(ctx, name)
		if err != nil {
			return nil, err
		}
		if _, err = os.Stat(p); err == nil {
			return nil, workspaceRefusal("A Git operation is in progress: %s", "进行中的 Git 操作不支持工作区快捷操作: %s", name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	head, err := r.resolve(ctx, "HEAD")
	if err != nil {
		return nil, &WorkspaceActionError{Phase: "preflight", English: "Workspace actions require an existing commit", message: "工作区需要已有提交", Cause: err}
	}
	branch, err := run(ctx, r.Root, "", "rev-parse", "--symbolic-full-name", "HEAD")
	if err != nil {
		return nil, err
	}
	canonical, err := filepath.EvalSymlinks(r.Root)
	if err != nil {
		return nil, err
	}
	s := &workspaceSnapshot{root: canonical, head: head, branch: branch, deleteNew: deleteNew}
	config, err := run(ctx, r.Root, "", "config", "--null", "--list")
	if err != nil {
		return nil, err
	}
	sparse, err := run(ctx, r.Root, "", "config", "--bool", "--get", "core.sparseCheckout")
	if err != nil && !workspaceExitOne(err) {
		return nil, err
	}
	if strings.TrimSpace(sparse) == "true" {
		return nil, workspaceRefusal("Sparse checkout is not supported by workspace actions", "稀疏检出不支持工作区快捷操作")
	}
	flags, err := run(ctx, r.Root, "", "ls-files", "-v", "-z")
	if err != nil {
		return nil, err
	}
	for _, record := range workspaceRecords(flags) {
		if len(record) < 3 || record[0] != 'H' {
			return nil, workspaceRefusal("Special index flags (skip-worktree or assume-unchanged) and conflicts are not supported", "特殊索引标记或冲突不支持工作区快捷操作")
		}
	}
	index, err := run(ctx, r.Root, "", "ls-files", "--stage", "-z")
	if err != nil {
		return nil, err
	}
	indexTree, err := workspaceParseTree(index, true)
	if err != nil {
		return nil, err
	}
	tree, err := run(ctx, r.Root, "", "ls-tree", "-r", "-z", head)
	if err != nil {
		return nil, err
	}
	s.headTree, err = workspaceParseTree(tree, false)
	if err != nil {
		return nil, err
	}
	status, err := r.Status(ctx)
	if err != nil {
		return nil, err
	}
	for _, e := range status.Entries {
		if e.Conflict || e.IsSubmodule() {
			return nil, workspaceRefusal("Resolve conflicts and save submodule changes first", "冲突或子模块改动不支持工作区快捷操作")
		}
	}
	submodules := map[string]bool{}
	for name, e := range s.headTree {
		if e.mode == "160000" {
			if indexTree[name] != e {
				return nil, workspaceRefusal("The submodule index entry has changed: %q", "子模块索引已变化: %q", name)
			}
			submodules[name] = true
		}
	}
	for name, e := range indexTree {
		if e.mode == "160000" && !submodules[name] {
			return nil, workspaceRefusal("New submodules are not supported by workspace actions: %q", "新增子模块不支持工作区快捷操作: %q", name)
		}
	}
	// Explicit status ignores user submodule-ignore configuration.
	rawStatus, err := run(ctx, r.Root, "", "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return nil, err
	}
	for name := range submodules {
		out, err := run(ctx, r.Root, "", "status", "--porcelain=v1", "-z", "--ignore-submodules=none", "--", name)
		if err != nil {
			return nil, err
		}
		if out != "" {
			return nil, workspaceRefusal("Save changes inside the submodule first: %q", "子模块有改动: %q", name)
		}
	}
	s.files, err = workspaceScan(ctx, canonical, submodules)
	if err != nil {
		return nil, err
	}
	currentIgnored, err := r.workspaceIgnored(ctx, "", s.files)
	if err != nil {
		return nil, err
	}
	futureIgnored, err := r.workspaceHeadIgnored(ctx, s)
	if err != nil {
		return nil, err
	}
	protected := map[string]bool{}
	for name := range s.files {
		_, inIndex := indexTree[name]
		_, inHead := s.headTree[name]
		if (currentIgnored[name] && !inIndex) || (futureIgnored[name] && !inHead) {
			protected[name] = true
		}
		if deleteNew && currentIgnored[name] && !inIndex && !futureIgnored[name] {
			return nil, workspaceRefusal("Restoring ignore rules would expose a previously ignored path; a clean workspace cannot be guaranteed: %q", "恢复忽略规则会暴露原先忽略的路径，无法安全清空: %q", name)
		}
	}
	headDirs := map[string]bool{".": true}
	for name := range s.headTree {
		for d := path.Dir(name); d != "."; d = path.Dir(d) {
			headDirs[d] = true
		}
	}
	protectedParents := map[string]bool{}
	for name := range protected {
		for d := path.Dir(name); d != "."; d = path.Dir(d) {
			protectedParents[d] = true
		}
	}
	candidates := map[string]bool{}
	for name, f := range s.files {
		if name == "." || submodules[name] || protected[name] {
			continue
		}
		_, inHead := s.headTree[name]
		if f.mode.IsDir() {
			if headDirs[name] {
				continue
			}
			if !protectedParents[name] {
				candidates[name] = true
			}
		} else if !inHead {
			candidates[name] = true
		}
	}
	// Both staged and unstaged differences matter; a staged deletion followed by
	// an identical untracked replacement still needs its index restored.
	changed, err := run(ctx, r.Root, "", "diff", "--name-only", "-z", "--no-renames", head, "--")
	if err != nil {
		return nil, err
	}
	changedSet := map[string]bool{}
	for _, name := range workspaceRecords(changed) {
		changedSet[name] = true
	}
	for name, e := range s.headTree {
		if indexTree[name] != e {
			changedSet[name] = true
		}
	}
	for name := range s.headTree {
		if submodules[name] {
			continue
		}
		if !changedSet[name] {
			continue
		}
		s.restore = append(s.restore, name)
		// Restoring a tracked child leaves an existing directory ancestor intact.
		// Protected leaves and descendants still obstruct restoration.
		for other := range protected {
			if s.files[other].mode.IsDir() && strings.HasPrefix(name, other+"/") {
				continue
			}
			if workspaceOverlap(name, other) {
				return nil, workspaceRefusal("Restoration would overwrite a protected ignored path: %q", "恢复会覆盖需要保留的忽略路径: %q", other)
			}
		}
		for other, f := range s.files {
			obstruction := (other == name && f.mode.IsDir()) || strings.HasPrefix(other, name+"/") || (strings.HasPrefix(name, other+"/") && !f.mode.IsDir())
			if obstruction && (!deleteNew || !candidates[other]) {
				return nil, workspaceRefusal("Restoration would overwrite a new path that must be preserved: %q", "恢复会覆盖需要保留的新增路径: %q", other)
			}
		}
	}
	for name := range candidates {
		s.newPaths = append(s.newPaths, name)
		if s.files[name].mode.IsDir() {
			s.removeDirs = append(s.removeDirs, name)
		} else {
			s.removeFiles = append(s.removeFiles, name)
		}
	}
	// An addition can exist only in the index (its working file was deleted).
	// It still belongs in the preview because this action discards its staging,
	// but there is no filesystem entry to approve for deletion.
	for name := range indexTree {
		if _, inHead := s.headTree[name]; inHead {
			continue
		}
		if _, present := s.files[name]; !present {
			s.newPaths = append(s.newPaths, name)
			s.indexOnly = append(s.indexOnly, name)
		}
	}
	slices.Sort(s.restore)
	slices.Sort(s.newPaths)
	slices.Sort(s.indexOnly)
	slices.Sort(s.removeFiles)
	slices.SortFunc(s.removeDirs, func(a, b string) int {
		if d := strings.Count(b, "/") - strings.Count(a, "/"); d != 0 {
			return d
		}
		return strings.Compare(b, a)
	})
	indexChanged := len(indexTree) != len(s.headTree)
	for name, e := range indexTree {
		if s.headTree[name] != e {
			indexChanged = true
		}
	}
	s.canExecute = indexChanged || len(s.restore) > 0 || (deleteNew && len(s.newPaths) > 0)
	// Bind raw index extensions/flags as well as entries, all configuration and
	// external ignore files. Reads never refresh or rewrite the user's index.
	indexPath, err := r.gitPath(ctx, "index")
	if err != nil {
		return nil, err
	}
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	excludes, err := r.workspaceExcludes(ctx)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	enc := json.NewEncoder(h)
	for _, v := range []any{canonical, head, branch, config, flags, index, tree, rawStatus, sha256.Sum256(indexBytes), excludes, currentIgnored, futureIgnored, deleteNew, s.restore, s.newPaths} {
		if err := enc.Encode(v); err != nil {
			return nil, err
		}
	}
	names := make([]string, 0, len(s.files))
	for name := range s.files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		f := s.files[name]
		if err := enc.Encode([]any{name, f.mode, f.digest}); err != nil {
			return nil, err
		}
	}
	s.stamp = fmt.Sprintf("%x", h.Sum(nil))
	return s, nil
}

func workspaceRecords(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\x00"), "\x00")
}
func workspaceExitOne(err error) bool {
	var e *exec.ExitError
	return errors.As(err, &e) && e.ExitCode() == 1
}
func workspaceOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

func workspaceParseTree(raw string, index bool) (map[string]workspaceTreeEntry, error) {
	result := map[string]workspaceTreeEntry{}
	for _, line := range workspaceRecords(raw) {
		meta, name, ok := strings.Cut(line, "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) != 3 || !fs.ValidPath(name) || name == "." {
			return nil, workspaceRefusal("The complete Git tree or index could not be parsed", "无法解析完整的 Git 树或索引")
		}
		if index && fields[2] != "0" {
			return nil, workspaceRefusal("Resolve index conflicts first", "请先处理索引冲突")
		}
		oid := fields[2]
		if index {
			oid = fields[1]
		}
		result[name] = workspaceTreeEntry{fields[0], oid}
	}
	return result, nil
}

// Scan through os.Root without following symlinks. Include parent identities and
// missing-path changes by comparing the complete inventory on execution.
func workspaceScan(ctx context.Context, dir string, submodules map[string]bool) (map[string]workspaceFile, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	files := map[string]workspaceFile{}
	err = fs.WalkDir(root.FS(), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if name == ".git" {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(path.Base(name), ".git") {
			return workspaceRefusal("Nested Git repositories are not supported by workspace actions: %q", "嵌套 Git 仓库不支持工作区快捷操作: %q", name)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		f := workspaceFile{mode: info.Mode(), info: info}
		if submodules[name] {
			files[name] = f
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		switch {
		case info.IsDir():
			// Bare repositories have no .git entry and must not become clean targets.
			if _, err := root.Lstat(path.Join(name, "HEAD")); err == nil {
				if objects, e := root.Lstat(path.Join(name, "objects")); e == nil && objects.IsDir() {
					return workspaceRefusal("Nested bare repositories are not supported by workspace actions: %q", "嵌套裸仓库不支持工作区快捷操作: %q", name)
				}
			}
		case info.Mode()&os.ModeSymlink != 0, info.Mode().IsRegular():
			f.digest, err = workspaceDigest(ctx, root, name, info)
		default:
			return workspaceRefusal("Special files cannot be handled safely: %q", "不支持安全处理特殊文件: %q", name)
		}
		if err != nil {
			return err
		}
		files[name] = f
		return nil
	})
	return files, err
}

func workspaceDigest(ctx context.Context, root *os.Root, name string, info fs.FileInfo) (string, error) {
	if info.Mode()&os.ModeSymlink != 0 {
		return root.Readlink(name)
	}
	if !info.Mode().IsRegular() {
		return "", workspaceRefusal("Special files cannot be handled safely: %q", "不支持安全处理特殊文件: %q", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return "", workspaceRefusal("The path changed while it was being read: %q", "读取时路径已变化: %q", name)
	}
	h := sha256.New()
	buf := make([]byte, 128<<10)
	for {
		if err = ctx.Err(); err != nil {
			return "", err
		}
		n, err := file.Read(buf)
		h.Write(buf[:n])
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
	}
	after, err := file.Stat()
	if err != nil {
		return "", err
	}
	if after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
		return "", workspaceRefusal("The path changed while it was being read: %q", "读取时路径已变化: %q", name)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (r *Repository) workspaceIgnored(ctx context.Context, worktree string, files map[string]workspaceFile) (map[string]bool, error) {
	names := make([]string, 0, len(files))
	for name, f := range files {
		if name != "." {
			if f.mode.IsDir() {
				name += "/"
			}
			names = append(names, "./"+name)
		}
	}
	slices.Sort(names)
	result := map[string]bool{}
	if len(names) == 0 {
		return result, nil
	}
	// check-ignore rejects inherited literal pathspec magic. Its stdin paths
	// are prefixed with ./ so names beginning with ':' stay ordinary paths.
	args := []string{"--no-literal-pathspecs"}
	if worktree != "" {
		args = append(args, "--work-tree="+worktree)
	}
	args = append(args, "check-ignore", "--no-index", "-z", "--stdin")
	out, err := run(ctx, r.Root, strings.Join(names, "\x00")+"\x00", args...)
	if err != nil && !workspaceExitOne(err) {
		return nil, err
	}
	for _, name := range workspaceRecords(out) {
		result[strings.TrimPrefix(strings.TrimSuffix(name, "/"), "./")] = true
	}
	return result, nil
}

func (r *Repository) workspaceHeadIgnored(ctx context.Context, s *workspaceSnapshot) (map[string]bool, error) {
	dir, err := os.MkdirTemp("", "supergit-head-ignore-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	for name, e := range s.headTree {
		if path.Base(name) != ".gitignore" || e.mode == "120000" {
			continue
		}
		if e.mode != "100644" && e.mode != "100755" {
			return nil, workspaceRefusal("Unsupported ignore-file type: %q", "不支持的忽略文件: %q", name)
		}
		data, err := run(ctx, r.Root, "", "cat-file", "blob", e.oid)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return nil, err
		}
		if err = os.WriteFile(target, []byte(data), 0600); err != nil {
			return nil, err
		}
	}
	return r.workspaceIgnored(ctx, dir, s.files)
}

func (r *Repository) workspaceExcludes(ctx context.Context) (map[string]string, error) {
	result := map[string]string{}
	info, err := r.gitPath(ctx, "info/exclude")
	if err != nil {
		return nil, err
	}
	global, err := run(ctx, r.Root, "", "config", "--path", "--get", "core.excludesFile")
	if err != nil && !workspaceExitOne(err) {
		return nil, err
	}
	names := []string{info}
	if global != "" {
		names = append(names, strings.TrimSuffix(global, "\n"))
	} else {
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			base = filepath.Join(home, ".config")
		}
		names = append(names, filepath.Join(base, "git", "ignore"))
	}
	for _, name := range names {
		if !filepath.IsAbs(name) {
			name = filepath.Join(r.Root, name)
		}
		data, err := os.ReadFile(name)
		if errors.Is(err, os.ErrNotExist) {
			result[name] = "missing"
			continue
		}
		if err != nil {
			return nil, err
		}
		result[name] = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	return result, nil
}

func workspaceSame(a, b *workspaceSnapshot) bool {
	if a == nil || b == nil || a.stamp != b.stamp || a.root != b.root || len(a.files) != len(b.files) {
		return false
	}
	for name, f := range a.files {
		other, ok := b.files[name]
		if !ok || !os.SameFile(f.info, other.info) {
			return false
		}
	}
	return true
}

func (r *Repository) CleanWorkspaceChecked(ctx context.Context, preview WorkspaceCleanPreview) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	expected := preview.snapshot
	if expected == nil || preview.Head != expected.head || preview.DeleteNew != expected.deleteNew || preview.CanExecute != expected.canExecute || !slices.Equal(preview.RestorePaths, expected.restore) || !slices.Equal(preview.NewPaths, expected.newPaths) || !slices.Equal(preview.IndexOnlyPaths, expected.indexOnly) {
		return &WorkspaceActionError{Phase: "validate", English: "The workspace preview is invalid; prepare a new preview", message: "工作区预览已失效，请重新确认"}
	}
	now, err := r.workspaceSnapshot(ctx, preview.DeleteNew)
	if err != nil {
		return err
	}
	if !workspaceSame(expected, now) {
		return &WorkspaceActionError{Phase: "validate", English: "The workspace, index, or branch changed; prepare a new preview", message: "工作区、索引或分支已变化，请重新确认"}
	}
	if !now.canExecute {
		return workspaceRefusal("There are no changes to clean with this option", "没有可清理的改动")
	}
	root, err := os.OpenRoot(now.root)
	if err != nil {
		return err
	}
	defer root.Close()
	removal := CleanPreview{identities: map[string]fs.FileInfo{}}
	for name, f := range now.files {
		removal.identities[name] = f.info
	}
	remove := func(name string) error {
		if err := ctx.Err(); err != nil {
			return workspaceFailure("delete", name, "", err)
		}
		for dir := path.Dir(name); ; dir = path.Dir(dir) {
			info, err := root.Lstat(dir)
			if err != nil {
				return workspaceFailure("delete", name, "", err)
			}
			original, ok := now.files[dir]
			if !ok || !info.IsDir() || !os.SameFile(original.info, info) {
				return workspaceFailure("delete", name, "", workspaceRefusal("A parent directory changed after validation", "校验后父目录已变化"))
			}
			if dir == "." {
				break
			}
		}
		if f := now.files[name]; !f.mode.IsDir() {
			info, err := root.Lstat(name)
			if err != nil {
				return workspaceFailure("delete", name, "", err)
			}
			digest, err := workspaceDigest(ctx, root, name, info)
			if err != nil {
				return workspaceFailure("delete", name, "", err)
			}
			if info.Mode() != f.mode || digest != f.digest || !os.SameFile(info, f.info) {
				return workspaceFailure("delete", name, "", workspaceRefusal("The path changed after validation", "校验后路径已变化"))
			}
		}
		if err := removal.removeEntry(root, name); err != nil {
			return workspaceFailure("delete", name, "", err)
		}
		return nil
	}
	blockers := map[string]bool{}
	if now.deleteNew {
		for _, name := range append(slices.Clone(now.removeFiles), now.removeDirs...) {
			for _, restore := range now.restore {
				if workspaceOverlap(name, restore) {
					blockers[name] = true
					break
				}
			}
		}
		for _, name := range append(slices.Clone(now.removeFiles), now.removeDirs...) {
			if blockers[name] {
				if err := remove(name); err != nil {
					return err
				}
			}
		}
	}
	// read-tree changes only the index, retaining HEAD-absent working files.
	if _, err = run(ctx, r.Root, "", "read-tree", now.head); err != nil {
		return workspaceFailure("unstage", "", "", err)
	}
	// Index restoration invokes Git; recheck the filesystem afterwards before
	// letting Git overwrite any approved path. External commands do not share
	// the in-process write lock.
	submodules := map[string]bool{}
	for name, e := range now.headTree {
		if e.mode == "160000" {
			submodules[name] = true
		}
	}
	currentFiles, err := workspaceScan(ctx, now.root, submodules)
	if err != nil {
		return workspaceFailure("restore", "", "", err)
	}
	if len(currentFiles) != len(now.files)-len(blockers) {
		return workspaceFailure("restore", "", "", workspaceRefusal("Paths changed after validation", "校验后路径已变化"))
	}
	for name, f := range now.files {
		if blockers[name] {
			continue
		}
		other, ok := currentFiles[name]
		if !ok || f.mode != other.mode || f.digest != other.digest || !os.SameFile(f.info, other.info) {
			return workspaceFailure("restore", name, "", workspaceRefusal("The path changed after validation", "校验后路径已变化"))
		}
	}
	if len(now.restore) > 0 {
		// NUL-delimited stdin keeps every approved literal path in one Git
		// invocation without command-line size limits or inter-chunk windows.
		if _, err = run(ctx, r.Root, strings.Join(now.restore, "\x00")+"\x00", "restore", "--source="+now.head, "--worktree", "--pathspec-from-file=-", "--pathspec-file-nul"); err != nil {
			return workspaceFailure("restore", strings.Join(now.restore, ", "), "", err)
		}
	}
	if now.deleteNew {
		for _, name := range append(slices.Clone(now.removeFiles), now.removeDirs...) {
			if !blockers[name] {
				if err := remove(name); err != nil {
					return err
				}
			}
		}
	}
	if err := r.workspaceVerifyResult(ctx, now); err != nil {
		return workspaceFailure("verify", "", "", err)
	}
	return nil
}

func (r *Repository) workspaceVerifyResult(ctx context.Context, s *workspaceSnapshot) error {
	head, err := r.resolve(ctx, "HEAD")
	if err != nil {
		return err
	}
	branch, err := run(ctx, r.Root, "", "rev-parse", "--symbolic-full-name", "HEAD")
	if err != nil {
		return err
	}
	if head != s.head || branch != s.branch {
		return workspaceRefusal("HEAD or the branch changed", "HEAD 或分支已变化")
	}
	index, err := run(ctx, r.Root, "", "diff", "--cached", "--name-only", "-z", s.head, "--")
	if err != nil {
		return err
	}
	if index != "" {
		return workspaceRefusal("The index still contains changes", "暂存区仍有改动")
	}
	changed, err := run(ctx, r.Root, "", "diff", "--name-only", "-z", "--ignore-submodules=none", "--")
	if err != nil {
		return err
	}
	if changed != "" {
		return workspaceRefusal("Tracked files still contain changes", "已跟踪文件仍有改动")
	}
	if s.deleteNew {
		status, err := run(ctx, r.Root, "", "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none")
		if err != nil {
			return err
		}
		if status != "" {
			return workspaceRefusal("The workspace still contains uncommitted changes", "工作区仍有待提交改动")
		}
	}
	return nil
}
