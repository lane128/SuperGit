package gitrepo

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// DiscardTarget always distinguishes deleting an untracked path from restoring
// a tracked working file. Batch discard must never silently imply deletion.
type DiscardTarget struct {
	Path      string
	Untracked bool
}

// PreparedDiscard is an opaque snapshot of exactly what a confirmation covers.
// It includes raw file bytes (including binary/untracked files), Index records,
// status and HEAD. No file content is returned to the UI or written to disk.
type PreparedDiscard struct {
	root    string
	targets []DiscardTarget
	stamp   string
}

func (r *Repository) PrepareDiscard(ctx context.Context, targets []DiscardTarget) (PreparedDiscard, error) {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if len(targets) == 0 {
		return PreparedDiscard{}, errors.New("没有可放弃的文件")
	}
	targets = append([]DiscardTarget(nil), targets...)
	stamp, err := r.discardStamp(ctx, targets)
	if err != nil {
		return PreparedDiscard{}, err
	}
	return PreparedDiscard{root: r.Root, targets: targets, stamp: stamp}, nil
}

func (r *Repository) DiscardPrepared(ctx context.Context, prepared PreparedDiscard) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if prepared.root != r.Root || prepared.stamp == "" || len(prepared.targets) == 0 {
		return errors.New("文件操作目标已改变，请重新确认")
	}
	stamp, err := r.discardStamp(ctx, prepared.targets)
	if err != nil {
		return err
	}
	if stamp != prepared.stamp {
		return errors.New("文件或暂存区已改变，请重新确认")
	}
	// Validate every target before the first write. Keep the write lock through
	// execution so another operation from this repository instance cannot slip
	// between validation and restore/delete. External tools remain independent.
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return err
	}
	defer root.Close()
	tracked := make([]string, 0, len(prepared.targets))
	for _, target := range prepared.targets {
		if target.Untracked {
			if err := root.Remove(target.Path); err != nil {
				return err
			}
		} else {
			tracked = append(tracked, target.Path)
		}
	}
	for start := 0; start < len(tracked); start += stageChunk {
		if _, err := run(ctx, r.Root, "", append([]string{"restore", "--worktree", "--"}, tracked[start:min(start+stageChunk, len(tracked))]...)...); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) discardStamp(ctx context.Context, targets []DiscardTarget) (string, error) {
	status, err := r.Status(ctx)
	if err != nil {
		return "", err
	}
	entries := make(map[string]Entry, len(status.Entries))
	for _, entry := range status.Entries {
		entries[entry.Path] = entry
	}
	root, err := os.OpenRoot(r.Root)
	if err != nil {
		return "", err
	}
	defer root.Close()
	h := sha256.New()
	enc := json.NewEncoder(h)
	if err := enc.Encode(status.Head); err != nil {
		return "", err
	}
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if err := r.path(target.Path); err != nil {
			return "", err
		}
		entry, ok := entries[target.Path]
		if !ok || entry.Conflict || entry.IsSubmodule() || !entry.Unstaged() || target.Untracked != (entry.Index == '?' && entry.Worktree == '?') {
			return "", errors.New("文件状态已改变或不支持放弃，请刷新后重新确认")
		}
		index, err := run(ctx, r.Root, "", "ls-files", "--stage", "-z", "--", target.Path)
		if err != nil {
			return "", err
		}
		if err := enc.Encode(struct {
			Target DiscardTarget
			Entry  Entry
			Index  string
		}{target, entry, index}); err != nil {
			return "", err
		}
		info, err := root.Lstat(target.Path)
		if errors.Is(err, os.ErrNotExist) && !target.Untracked {
			if err := enc.Encode("missing"); err != nil {
				return "", err
			}
			continue
		}
		if err != nil {
			return "", err
		}
		if err := enc.Encode(info.Mode()); err != nil {
			return "", err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			link, err := root.Readlink(target.Path)
			if err != nil {
				return "", err
			}
			if err := enc.Encode(link); err != nil {
				return "", err
			}
		case info.Mode().IsRegular():
			file, err := root.Open(target.Path)
			if err != nil {
				return "", err
			}
			contents := sha256.New()
			buffer := make([]byte, 128<<10)
			for {
				if err = ctx.Err(); err != nil {
					break
				}
				var n int
				n, err = file.Read(buffer)
				_, _ = contents.Write(buffer[:n])
				if err != nil {
					break
				}
			}
			closeErr := file.Close()
			if err != nil && !errors.Is(err, io.EOF) {
				return "", err
			}
			if closeErr != nil {
				return "", closeErr
			}
			if err := enc.Encode(fmt.Sprintf("%x", contents.Sum(nil))); err != nil {
				return "", err
			}
		default:
			return "", errors.New("此文件类型不支持安全放弃")
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
