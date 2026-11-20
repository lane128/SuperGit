package gitrepo

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	h := sha256.New()
	enc := json.NewEncoder(h)
	if err := enc.Encode(status); err != nil {
		return "", err
	}
	for _, args := range [][]string{
		{"for-each-ref", "--sort=refname", "--format=%(refname)%00%(objectname)%00%(symref)"},
		{"worktree", "list", "--porcelain", "-z"},
		{"config", "--null", "--list", "--includes"},
	} {
		out, err := run(ctx, r.Root, "", args...)
		if err != nil {
			return "", err
		}
		if err := enc.Encode(out); err != nil {
			return "", err
		}
	}
	op, err := r.Operation(ctx)
	if err != nil {
		return "", err
	}
	if err := enc.Encode(op); err != nil {
		return "", err
	}
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
