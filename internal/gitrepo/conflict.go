package gitrepo

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ErrConflictMarkers is returned when a conflicted working file still contains
// unresolved Git conflict markers (<<<<<<<, =======, >>>>>>> or |||||||).
var ErrConflictMarkers = errors.New("结果仍包含冲突标记，请先完成编辑")

type Conflict struct {
	Path, Base, Ours, Theirs, Working string
	Root                              string
	OIDs                              [3]string
	Operation                         Operation
	Mode                              os.FileMode
}

// Pin the parent directory and refuse symlink ancestors and nested repositories.
// os.Root alone prevents escaping the root, but permits links elsewhere inside it.
func conflictParent(root *os.Root, path string) (*os.Root, error) {
	parent := filepath.Dir(path)
	current := ""
	if parent != "." {
		for _, part := range strings.Split(parent, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			info, err := root.Lstat(current)
			if err != nil {
				return nil, err
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return nil, errors.New("冲突路径包含符号链接或非目录，请重新载入")
			}
			if _, err := root.Lstat(filepath.Join(current, ".git")); err == nil {
				return nil, errors.New("冲突路径进入了另一个 Git 仓库")
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
	}
	return root.OpenRoot(parent)
}
func boundedFile(root *os.Root, path string) (string, os.FileMode, error) {
	info, err := root.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", 0644, nil
	}
	if err != nil {
		return "", 0, err
	}
	if !info.Mode().IsRegular() {
		return "", 0, errors.New("此冲突不是普通文本文件，请使用外部工具处理")
	}
	if info.Size() > 1<<20 {
		return "", 0, errors.New("冲突文件超过 1 MiB，请使用外部编辑器")
	}
	f, err := root.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
	if err != nil {
		return "", 0, err
	}
	if len(b) > 1<<20 || !utf8.Valid(b) || strings.IndexByte(string(b), 0) >= 0 {
		return "", 0, errors.New("二进制或过大的冲突文件，请使用外部工具")
	}
	return string(b), info.Mode().Perm(), nil
}

// TakeConflictSide replaces the working file with our side or their side of
// the conflict, then stages it as resolved. During a rebase Git swaps which
// side is which, as git checkout --ours/--theirs does.
func (r *Repository) TakeConflictSide(ctx context.Context, path, side string) error {
	if side != "ours" && side != "theirs" {
		return fmt.Errorf("无效的冲突版本: %q", side)
	}
	if err := r.path(path); err != nil {
		return err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	missing, err := r.conflictSideMissing(ctx, path, side)
	if err != nil {
		return err
	}
	// A side that deleted the file has nothing to check out; accepting it
	// means removing the path, as checking out that side would leave it.
	if missing {
		_, err = run(ctx, r.Root, "", "rm", "--", path)
		return err
	}
	if _, err := run(ctx, r.Root, "", "checkout", "--"+side, "--", path); err != nil {
		return err
	}
	_, err = run(ctx, r.Root, "", "add", "--", path)
	return err
}

// conflictSideMissing reports whether the given side of an unmerged path has
// no entry, which happens when that side deleted the file.
func (r *Repository) conflictSideMissing(ctx context.Context, path, side string) (bool, error) {
	out, err := run(ctx, r.Root, "", "ls-files", "--unmerged", "-z", "--", path)
	if err != nil {
		return false, err
	}
	stage := "3"
	if side == "ours" {
		stage = "2"
	}
	for _, record := range strings.Split(out, "\x00") {
		meta, _, ok := strings.Cut(record, "\t")
		f := strings.Fields(meta)
		if ok && len(f) == 3 && f[2] == stage {
			return false, nil
		}
	}
	return true, nil
}

func (r *Repository) Conflict(ctx context.Context, path string) (Conflict, error) {
	c := Conflict{Path: path, Root: canonical(r.Root)}
	if err := r.path(path); err != nil {
		return c, err
	}
	out, err := run(ctx, r.Root, "", "ls-files", "--unmerged", "-z", "--", path)
	if err != nil {
		return c, err
	}
	for _, record := range strings.Split(out, "\x00") {
		meta, name, ok := strings.Cut(record, "\t")
		f := strings.Fields(meta)
		if !ok || name != path || len(f) != 3 {
			continue
		}
		if f[0] != "100644" && f[0] != "100755" {
			return c, errors.New("此冲突需要文件级处理，请使用外部工具")
		}
		stage, _ := strconv.Atoi(f[2])
		if stage >= 1 && stage <= 3 {
			c.OIDs[stage-1] = f[1]
		}
	}
	if c.OIDs == [3]string{} {
		return c, errors.New("该文件不再处于未解决冲突状态，请刷新")
	}
	values := []*string{&c.Base, &c.Ours, &c.Theirs}
	for i, sha := range c.OIDs {
		if sha != "" {
			value, err := run(ctx, r.Root, "", "cat-file", "blob", sha)
			if err != nil {
				return c, err
			}
			if len(value) > 1<<20 || !utf8.ValidString(value) || strings.IndexByte(value, 0) >= 0 {
				return c, errors.New("冲突包含二进制或过大版本，请使用外部工具")
			}
			*values[i] = value
		}
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return c, err
	}
	defer root.Close()
	parent, err := conflictParent(root, path)
	if err != nil {
		return c, err
	}
	defer parent.Close()
	c.Working, c.Mode, err = boundedFile(parent, filepath.Base(path))
	if err != nil {
		return c, err
	}
	c.Operation, err = r.Operation(ctx)
	return c, err
}
func isConflictMarkerPrefix(prefix string) bool {
	return prefix == "<<<<<<<" || prefix == "=======" || prefix == ">>>>>>>" || prefix == "|||||||"
}

func conflictMarkers(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if len(line) >= 7 && isConflictMarkerPrefix(line[:7]) {
			return true
		}
	}
	return false
}

// fileHasConflictMarkers scans a regular text file in the working tree for
// unresolved Git conflict markers. Deleted files, non-regular entries and
// binary files (containing a NUL byte in the first 8 KiB) return false so they
// can still be staged after external resolution.
func fileHasConflictMarkers(rootPath, path string) (bool, error) {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return false, err
	}
	defer root.Close()
	parent, err := conflictParent(root, path)
	if err != nil {
		return false, err
	}
	defer parent.Close()
	base := filepath.Base(path)
	info, err := parent.Lstat(base)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}
	f, err := parent.Open(base)
	if err != nil {
		return false, err
	}
	defer f.Close()
	head := make([]byte, 8192)
	n, readErr := f.Read(head)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return false, readErr
	}
	head = head[:n]
	if bytes.IndexByte(head, 0) >= 0 {
		return false, nil
	}
	reader := bufio.NewReader(io.MultiReader(bytes.NewReader(head), f))
	atLineStart := true
	prefix := make([]byte, 0, 7)
	for {
		chunk, err := reader.ReadSlice('\n')
		if atLineStart {
			if len(prefix) < 7 {
				need := 7 - len(prefix)
				if len(chunk) < need {
					prefix = append(prefix, chunk...)
				} else {
					prefix = append(prefix, chunk[:need]...)
				}
			}
			if len(prefix) == 7 && isConflictMarkerPrefix(string(prefix)) {
				return true, nil
			}
			if len(prefix) >= 7 || err == nil || errors.Is(err, io.EOF) {
				atLineStart = false
			}
		}
		if err == nil {
			prefix = prefix[:0]
			atLineStart = true
			continue
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
}

// MarkConflictResolved stages a conflicted path after verifying that the
// working tree text file no longer contains unresolved Git conflict markers.
func (r *Repository) MarkConflictResolved(ctx context.Context, path string) error {
	if err := r.path(path); err != nil {
		return err
	}
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	hasMarkers, err := fileHasConflictMarkers(r.Root, path)
	if err != nil {
		return err
	}
	if hasMarkers {
		return ErrConflictMarkers
	}
	_, err = run(ctx, r.Root, "", "add", "--", path)
	return err
}
func (r *Repository) ResolveConflictDeletion(ctx context.Context, original Conflict) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	current, err := r.Conflict(ctx, original.Path)
	if err != nil {
		return err
	}
	if current.Root != original.Root || current.OIDs != original.OIDs || current.Working != original.Working || current.Operation != original.Operation || current.Mode != original.Mode {
		return errors.New("冲突或文件已在外部变化，请重新载入")
	}
	if current.OIDs[1] != "" && current.OIDs[2] != "" {
		return errors.New("此冲突没有删除版本")
	}
	_, err = run(ctx, r.Root, "", "rm", "--", original.Path)
	return err
}
func (r *Repository) ResolveConflict(ctx context.Context, original Conflict, text string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if text == "" && (original.OIDs[1] == "" || original.OIDs[2] == "") {
		return errors.New("删除冲突需明确选择保留文件或接受删除，不能保存为空文件")
	}
	if len(text) > 1<<20 || !utf8.ValidString(text) || strings.IndexByte(text, 0) >= 0 {
		return errors.New("结果必须为不超过 1 MiB 的文本")
	}
	if conflictMarkers(text) {
		return ErrConflictMarkers
	}
	current, err := r.Conflict(ctx, original.Path)
	if err != nil {
		return err
	}
	if current.Root != original.Root || current.OIDs != original.OIDs || current.Working != original.Working || current.Operation != original.Operation {
		return errors.New("冲突或文件已在外部变化，请重新载入后再保存")
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	parent, err := conflictParent(root, original.Path)
	if err != nil {
		return err
	}
	defer parent.Close()
	temp := fmt.Sprintf(".supergit-resolve-%x", rand.Text())
	f, err := parent.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, current.Mode)
	if err != nil {
		return err
	}
	defer parent.Remove(temp)
	if _, err = f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = parent.Rename(temp, filepath.Base(original.Path)); err != nil {
		return err
	}
	_, err = run(ctx, r.Root, "", "add", "--", original.Path)
	return err
}
