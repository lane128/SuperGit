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
	"path"
	"slices"
	"strconv"
	"strings"
)

// A compare-and-delete on refs/stash also removes its reflog. Unlike repeated
// drops, a rejected or interrupted command cannot leave a half-cleared list.
func (r *Repository) ClearStashesChecked(ctx context.Context, expected []Stash) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	current, err := r.Stashes(ctx)
	if err != nil {
		return err
	}
	if len(current) == 0 || !slices.Equal(current, expected) {
		return errors.New("Stash 列表已变化，请重新确认")
	}
	_, err = run(ctx, r.Root, "", "update-ref", "-d", "refs/stash", current[0].SHA)
	return err
}

type CleanPreview struct {
	Paths              []string
	root, stamp        string
	files, directories []string
	identities         map[string]fs.FileInfo
}

func (r *Repository) PrepareCleanUntracked(ctx context.Context) (CleanPreview, error) {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	return r.cleanPreview(ctx)
}

func (r *Repository) cleanPreview(ctx context.Context) (CleanPreview, error) {
	// Git decides which paths clean -fd would touch, including empty
	// directories and exceptions for ignored paths and nested repositories.
	out, err := run(ctx, r.Root, "", "clean", "-nfd", "--")
	if err != nil {
		return CleanPreview{}, err
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return CleanPreview{}, err
	}
	defer root.Close()
	preview := CleanPreview{root: r.Root, identities: make(map[string]fs.FileInfo)}
	hash := sha256.New()
	encoder := json.NewEncoder(hash)
	for line := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "" || strings.HasPrefix(line, "Would skip repository ") {
			continue
		}
		value, ok := strings.CutPrefix(line, "Would remove ")
		if !ok {
			return CleanPreview{}, fmt.Errorf("unexpected git clean preview: %s", line)
		}
		if strings.HasPrefix(value, "\"") {
			value, err = strconv.Unquote(value)
			if err != nil {
				return CleanPreview{}, err
			}
		}
		value = strings.TrimSuffix(value, "/")
		if !fs.ValidPath(value) || value == "." {
			return CleanPreview{}, errors.New("invalid clean path")
		}
		preview.Paths = append(preview.Paths, value)
		parent := path.Dir(value)
		if _, seen := preview.identities[parent]; !seen {
			preview.identities[parent], err = root.Stat(parent)
			if err != nil {
				return CleanPreview{}, err
			}
		}
		visit := func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if strings.EqualFold(path.Base(name), ".git") {
				return errors.New("refusing to clean nested Git metadata")
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			preview.identities[name] = info
			if err := encoder.Encode([]any{name, info.Mode()}); err != nil {
				return err
			}
			switch {
			case entry.IsDir():
				preview.directories = append(preview.directories, name)
				return nil
			case info.Mode()&os.ModeSymlink != 0:
				link, err := root.Readlink(name)
				if err != nil {
					return err
				}
				if err := encoder.Encode(link); err != nil {
					return err
				}
			case info.Mode().IsRegular():
				file, err := root.Open(name)
				if err != nil {
					return err
				}
				sum := sha256.New()
				_, err = io.Copy(sum, file)
				closeErr := file.Close()
				if err != nil {
					return err
				}
				if closeErr != nil {
					return closeErr
				}
				if err := encoder.Encode(sum.Sum(nil)); err != nil {
					return err
				}
			default:
				return fmt.Errorf("不能安全删除特殊文件 %q", name)
			}
			preview.files = append(preview.files, name)
			return nil
		}
		// fs.WalkDir follows its starting path if that path is a symlink.
		// Clean removes the link itself, including links outside the checkout.
		info, err := root.Lstat(value)
		if err != nil {
			return CleanPreview{}, err
		}
		if info.IsDir() {
			err = fs.WalkDir(root.FS(), value, visit)
		} else {
			err = visit(value, fs.FileInfoToDirEntry(info), nil)
		}
		if err != nil {
			return CleanPreview{}, err
		}
	}
	preview.stamp = fmt.Sprintf("%x", hash.Sum(nil))
	return preview, nil
}

func (r *Repository) CleanUntrackedChecked(ctx context.Context, expected CleanPreview) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	now, err := r.cleanPreview(ctx)
	if err != nil {
		return err
	}
	if expected.root != r.Root || len(now.Paths) == 0 || now.stamp != expected.stamp || !slices.Equal(now.Paths, expected.Paths) {
		return errors.New("未跟踪文件已改变，请重新确认")
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	// Never recursively delete an approved directory: a newly arrived child
	// must survive and cause removal of its parent to fail.
	slices.Reverse(now.directories)
	for _, name := range append(now.files, now.directories...) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := now.removeEntry(root, name); err != nil {
			return err
		}
	}
	return nil
}

// Anchor unlink to the directory that was reviewed. A directory replaced
// with a symlink must never redirect the removal into another tracked tree.
func (p *CleanPreview) removeEntry(root *os.Root, name string) error {
	dir := path.Dir(name)
	parent, err := root.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer parent.Close()
	info, err := parent.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(p.identities[dir], info) {
		return errors.New("文件夹已变化，请重新确认")
	}
	base := path.Base(name)
	info, err = parent.Lstat(base)
	if err != nil {
		return err
	}
	if !os.SameFile(p.identities[name], info) {
		return errors.New("未跟踪文件已变化，请重新确认")
	}
	return parent.Remove(base)
}
