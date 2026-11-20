package gitrepo

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const MaxPreviewBytes = 16 << 20

var ErrPreviewTooLarge = errors.New("file exceeds image preview limit")

// BlobPreview reads an immutable object. Check size before starting the blob transfer.
func (r *Repository) BlobPreview(ctx context.Context, oid string) ([]byte, error) {
	if !oidRE.MatchString(oid) {
		return nil, errors.New("invalid preview object ID")
	}
	raw, err := run(ctx, r.Root, "", "cat-file", "-s", oid)
	if err != nil {
		return nil, err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return nil, err
	}
	if size > MaxPreviewBytes {
		return nil, ErrPreviewTooLarge
	}
	b := limitedBuffer{limit: MaxPreviewBytes}
	if err := runOutput(ctx, r.Root, "", &b, "cat-file", "blob", oid); err != nil {
		return nil, err
	}
	if b.truncated {
		return nil, ErrPreviewTooLarge
	}
	return b.buffer.Bytes(), nil
}

const PreviewIndex = ":index"
const PreviewWorktree = ":worktree"

// ReadPreview reads the exact side requested by the patch. Index and tree
// paths resolve to a blob first, so both size and content refer to one object.
func (r *Repository) ReadPreview(ctx context.Context, revision, path string) ([]byte, error) {
	if err := r.path(path); err != nil {
		return nil, err
	}
	if revision == PreviewWorktree {
		root, err := os.OpenRoot(r.Root)
		if err != nil {
			return nil, err
		}
		defer root.Close()
		parent, err := conflictParent(root, path)
		if err != nil {
			return nil, err
		}
		defer parent.Close()
		name := filepath.Base(path)
		info, err := parent.Lstat(name)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, errors.New("preview requires a regular file")
		}
		if info.Size() > MaxPreviewBytes {
			return nil, ErrPreviewTooLarge
		}
		file, err := parent.Open(name)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		opened, err := file.Stat()
		if err != nil {
			return nil, err
		}
		if !os.SameFile(info, opened) {
			return nil, errors.New("file changed while opening preview")
		}
		data, err := io.ReadAll(io.LimitReader(file, MaxPreviewBytes+1))
		if len(data) > MaxPreviewBytes {
			return nil, ErrPreviewTooLarge
		}
		return data, err
	}
	spec := ":0:" + path
	if revision != PreviewIndex {
		if !oidRE.MatchString(revision) {
			return nil, errors.New("invalid preview revision")
		}
		spec = revision + ":" + path
	}
	oid, err := run(ctx, r.Root, "", "rev-parse", "--verify", "--end-of-options", spec)
	if err != nil {
		return nil, err
	}
	return r.BlobPreview(ctx, strings.TrimSpace(oid))
}
