package gitrepo

import (
	"context"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// PreparedCommitFileUndo owns an isolated candidate result. It has not changed
// the repository. Call Close when the result is no longer needed.
type PreparedCommitFileUndo struct {
	mu                 sync.Mutex
	directory, result  string
	root, source, path string
	expected           Status
	original           os.FileInfo
	digest             [32]byte
}

func (p *PreparedCommitFileUndo) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.directory == "" {
		return nil
	}
	if err := os.RemoveAll(p.directory); err != nil {
		return err
	}
	p.directory = ""
	return nil
}

// PrepareCommitFileUndo reverse-applies the patch only in an owned scratch
// repository. A later application must revalidate the target before replacing
// its single worktree file. The real repository is only read here.
func (r *Repository) PrepareCommitFileUndo(ctx context.Context, sha, path string, expected Status) (*PreparedCommitFileUndo, error) {
	if err := r.CheckCommitFileUndo(ctx, sha, path, expected); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "supergit-undo-*")
	if err != nil {
		return nil, err
	}
	p := &PreparedCommitFileUndo{directory: directory, root: r.Root, source: sha, path: path, expected: Status{Head: expected.Head, Branch: expected.Branch}}
	keep := false
	defer func() {
		if !keep {
			p.Close()
		}
	}()
	worktree := filepath.Join(directory, "worktree")
	p.result = filepath.Join(worktree, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(p.result), 0700); err != nil {
		return nil, err
	}
	if err := p.copyOriginal(ctx); err != nil {
		return nil, err
	}
	// A fresh empty repository avoids discovering any repository above TMPDIR.
	// No user templates/hooks are installed, and no commit or network is used.
	if _, err := run(ctx, worktree, "", "init", "--quiet", "--template="); err != nil {
		return nil, err
	}
	patch, err := os.CreateTemp(directory, "change-*.patch")
	if err != nil {
		return nil, err
	}
	generationErr := r.WriteCommitFilePatch(ctx, sha, path, patch)
	closeErr := patch.Close()
	if generationErr != nil {
		return nil, generationErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if _, err := run(ctx, worktree, "", "apply", "--reverse", "--whitespace=nowarn", "--", patch.Name()); err != nil {
		return nil, err
	}
	if err := r.CheckCommitFileUndo(ctx, sha, path, expected); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	keep = true
	return p, nil
}
func (p *PreparedCommitFileUndo) copyOriginal(ctx context.Context) error {
	root, err := os.OpenRoot(p.root)
	if err != nil {
		return err
	}
	defer root.Close()
	info, err := root.Lstat(p.path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return ErrUnsupportedCommitFileUndo
	}
	source, err := root.Open(p.path)
	if err != nil {
		return err
	}
	defer source.Close()
	opened, err := source.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(info, opened) || info.Mode() != opened.Mode() {
		return ErrIntegrationTargetChanged
	}
	destination, err := os.OpenFile(p.result, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	digest := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(destination, digest), undoContextReader{ctx: ctx, reader: source})
	modeErr := destination.Chmod(info.Mode().Perm())
	closeErr := destination.Close()
	if copyErr != nil {
		return copyErr
	}
	if modeErr != nil {
		return modeErr
	}
	if closeErr != nil {
		return closeErr
	}
	p.original = info
	copy(p.digest[:], digest.Sum(nil))
	return nil
}

type undoContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r undoContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
