package gitrepo

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ChangeStamp observes local Git state without fetching or traversing ignored
// trees. It deliberately omits working file content: read the selected Diff
// separately to detect further edits to an already-dirty file. Config values
// contribute only to the hash and are never returned or logged.
func (r *Repository) ChangeStamp(ctx context.Context) (string, error) {
	// Keep all porcelain fields, including Index/conflict blob IDs and modes.
	status, err := r.statusText(ctx)
	if err != nil {
		return "", err
	}
	return r.ChangeStampFromStatus(ctx, status)
}

// ChangeStampFromStatus hashes the local state from a porcelain status the
// caller already read. The open path parses its rows from this same text, so
// the baseline describes exactly what the window shows and `git status` runs
// once instead of twice.
func (r *Repository) ChangeStampFromStatus(ctx context.Context, status string) (string, error) {
	local, err := r.LocalStampFromStatus(ctx, status)
	if err != nil {
		return "", err
	}
	refs, err := r.RefStamp(ctx)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	enc := json.NewEncoder(h)
	enc.Encode(local)
	enc.Encode(refs)
	// Parent porcelain only says that a child differs; it does not report the
	// child's current HEAD. Hash Git's local child-pointer inventory separately.
	if _, e := os.Lstat(filepath.Join(r.Root, ".gitmodules")); e == nil {
		out, e := run(ctx, r.Root, "", "submodule", "status")
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if err := enc.Encode(out); err != nil {
			return "", err
		}
		// Broken child metadata remains an observable state; it must not hide
		// unrelated parent refs. Snapshot reports submodule errors independently.
		if e != nil {
			if err := enc.Encode(e.Error()); err != nil {
				return "", err
			}
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return "", e
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (r *Repository) RefStamp(ctx context.Context) (string, error) {
	out, err := run(ctx, r.Root, "", "for-each-ref", "--sort=refname", "--format=%(refname)%00%(objectname)%00%(symref)")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(out))), err
}

// LocalStampFromStatus excludes tracking headers, which change after fetch.
func (r *Repository) LocalStampFromStatus(ctx context.Context, status string) (string, error) {
	id, err := r.CheckoutIdentity(ctx)
	if err != nil {
		return "", err
	}
	metadata, err := r.LocalMetadataStamp(ctx, id)
	if err != nil {
		return "", err
	}
	return LocalStatusStamp(metadata, status), nil
}

func LocalStatusStamp(metadata, status string) string {
	h := sha256.New()
	enc := json.NewEncoder(h)
	for strings.HasPrefix(status, "# ") {
		header, rest, _ := strings.Cut(status, "\x00")
		status = rest
		if !strings.HasPrefix(header, "# branch.ab ") && !strings.HasPrefix(header, "# branch.upstream ") {
			enc.Encode(header)
		}
	}
	enc.Encode(status)
	enc.Encode(metadata)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (r *Repository) LocalMetadataStamp(ctx context.Context, id CheckoutIdentity) (string, error) {
	h := sha256.New()
	enc := json.NewEncoder(h)
	for _, args := range [][]string{{"worktree", "list", "--porcelain", "-z"}, {"config", "--null", "--list", "--includes"}} {
		out, err := run(ctx, r.Root, "", args...)
		if err != nil {
			return "", err
		}
		enc.Encode(out)
	}
	enc.Encode(id)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CheckoutIdentity guards publication against checkout, index and operation changes.
type CheckoutIdentity struct {
	Head, Symbolic, Index string
	Operation             Operation
}

func (r *Repository) CheckoutIdentity(ctx context.Context) (id CheckoutIdentity, err error) {
	if id.Head, err = r.optionalRef(ctx, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		return
	}
	if id.Symbolic, err = r.optionalRef(ctx, "symbolic-ref", "--quiet", "HEAD"); err != nil {
		return
	}
	if id.Operation, err = r.Operation(ctx); err != nil {
		return
	}
	path, err := r.gitPath(ctx, "index")
	if err != nil {
		return
	}
	f, e := os.Open(path)
	if errors.Is(e, os.ErrNotExist) {
		return
	}
	if e != nil {
		return id, e
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err == nil {
		id.Index = fmt.Sprintf("%x", h.Sum(nil))
	}
	return
}

func (r *Repository) optionalRef(ctx context.Context, args ...string) (string, error) {
	out, e := run(ctx, r.Root, "", args...)
	var exit *exec.ExitError
	if errors.As(e, &exit) && exit.ExitCode() == 1 {
		return "", nil
	}
	return strings.TrimSuffix(out, "\n"), e
}
