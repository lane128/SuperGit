package gitrepo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// ApplyPreparedCommitFileUndo publishes exactly one prepared result. App writes
// serialize here; external programs do not share writeMu. We recheck repository
// state and target identity/content immediately before the atomic replacement,
// but cannot provide a filesystem compare-and-swap against external writers.
// A successful rename is the success boundary, including with late cancellation.
func (r *Repository) ApplyPreparedCommitFileUndo(ctx context.Context, p *PreparedCommitFileUndo) (err error) {
	if p == nil {
		return ErrIntegrationTargetChanged
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if p.directory == "" || p.original == nil || p.root != r.Root {
		return ErrIntegrationTargetChanged
	}
	if err := r.checkCommitFileUndo(ctx, p.source, p.path, p.expected, ""); err != nil {
		return err
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	dir, base := filepath.Dir(filepath.FromSlash(p.path)), filepath.Base(filepath.FromSlash(p.path))
	parent, err := root.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer parent.Close()
	parentInfo, err := parent.Stat(".")
	if err != nil {
		return err
	}
	if err := p.checkOriginal(ctx, parent, base); err != nil {
		return err
	}
	candidate, err := os.Open(p.result)
	if err != nil {
		return err
	}
	defer candidate.Close()
	candidateInfo, err := candidate.Stat()
	if err != nil {
		return err
	}
	if !candidateInfo.Mode().IsRegular() {
		return ErrIntegrationTargetChanged
	}
	name := ".supergit-undo-" + rand.Text()
	temp, err := parent.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	replaced := false
	defer func() {
		if !replaced {
			removeErr := parent.Remove(name)
			if !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, removeErr)
			}
		}
	}()
	_, copyErr := io.Copy(temp, undoContextReader{ctx: ctx, reader: candidate})
	modeErr := temp.Chmod(p.original.Mode().Perm())
	syncErr := temp.Sync()
	closeErr := temp.Close()
	if err := errors.Join(copyErr, modeErr, syncErr, closeErr); err != nil {
		return err
	}
	tempPath := filepath.ToSlash(filepath.Join(dir, name))
	if err := r.checkCommitFileUndo(ctx, p.source, p.path, p.expected, tempPath); err != nil {
		return err
	}
	// Verify the anchored directory still occupies the captured repository path.
	currentParent, err := root.Stat(dir)
	if err != nil {
		return err
	}
	if !os.SameFile(parentInfo, currentParent) {
		return ErrIntegrationTargetChanged
	}
	if err := p.checkOriginal(ctx, parent, base); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := parent.Rename(name, base); err != nil {
		return err
	}
	replaced = true
	p.original = nil // A prepared result is single-use.
	return nil
}

func (p *PreparedCommitFileUndo) checkOriginal(ctx context.Context, parent *os.Root, name string) error {
	info, err := parent.Lstat(name)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || !os.SameFile(p.original, info) || info.Mode() != p.original.Mode() {
		return ErrIntegrationTargetChanged
	}
	file, err := parent.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(info, opened) || opened.Mode() != info.Mode() {
		return ErrIntegrationTargetChanged
	}
	digest := sha256.New()
	if _, err := io.Copy(digest, undoContextReader{ctx: ctx, reader: file}); err != nil {
		return err
	}
	var got [32]byte
	copy(got[:], digest.Sum(nil))
	after, err := parent.Lstat(name)
	if err != nil {
		return err
	}
	if got != p.digest || !os.SameFile(info, after) || after.Mode() != info.Mode() {
		return ErrIntegrationTargetChanged
	}
	return nil
}
