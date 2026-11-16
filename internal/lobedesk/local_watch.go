package lobedesk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"supergit/internal/gitrepo"
)

const localProbeInterval = 3 * time.Second
const localBackgroundInterval = 15 * time.Second

type localRepositoryWatch struct {
	stamp                    string
	seq                      uint64
	reading, focused, closed bool
	next                     time.Time
	lastError                string
	cancel                   context.CancelFunc
}

type localWatchRequest struct {
	repo       *gitrepo.Repository
	epoch, seq uint64
	entries    []gitrepo.Entry
}

// ChangeStamp observes Git metadata, but deliberately excludes working bytes.
// Track metadata of all already-dirty files too, so further saves outside the
// selected file invalidate the overview. Do not walk ignored or untracked trees.
func readLocalWatchStamp(parent context.Context, repo *gitrepo.Repository, entries []gitrepo.Entry) (string, error) {
	ctx, cancel := context.WithTimeout(parent, localBackgroundInterval)
	defer cancel()
	stamp, err := repo.ChangeStamp(ctx)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	enc := json.NewEncoder(hash)
	if err := enc.Encode(stamp); err != nil {
		return "", err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if !filepath.IsLocal(entry.Path) {
			return "", fmt.Errorf("invalid repository file path: %q", entry.Path)
		}
		path := filepath.Join(repo.Root, entry.Path)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			if err := enc.Encode([]string{entry.Path, "missing"}); err != nil {
				return "", err
			}
			continue
		}
		if err != nil {
			return "", err
		}
		observation := struct {
			Path           string
			Size, Modified int64
			Mode           uint32
			Link           string
		}{Path: entry.Path, Size: info.Size(), Modified: info.ModTime().UnixNano(), Mode: uint32(info.Mode())}
		if info.Mode()&os.ModeSymlink != 0 {
			observation.Link, err = os.Readlink(path)
			if err != nil {
				return "", err
			}
		}
		if err := enc.Encode(observation); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (a *app) cancelLocalProbe() {
	if a.localWatch.cancel != nil {
		a.localWatch.cancel()
	}
	a.localWatch.cancel = nil
	a.localWatch.seq++
	a.localWatch.reading = false
}

func (a *app) beginLocalProbe(now time.Time) (localWatchRequest, bool) {
	w := &a.localWatch
	if w.closed || a.repo == nil || a.loading || a.busy || w.reading || now.Before(w.next) ||
		a.confirm.open || a.prompt.open || a.worktreeForm.open || a.reset.open {
		return localWatchRequest{}, false
	}
	interval := localBackgroundInterval
	if w.focused {
		interval = localProbeInterval
	}
	w.next = now.Add(interval)
	w.reading = true
	w.seq++
	return localWatchRequest{repo: a.repo, epoch: a.loadSeq, seq: w.seq, entries: slices.Clone(a.status.Entries)}, true
}

func (a *app) acceptLocalProbe(req localWatchRequest, stamp string, err error, now time.Time) bool {
	w := &a.localWatch
	if w.seq != req.seq || a.repo != req.repo || a.loadSeq != req.epoch {
		return false
	}
	w.reading = false
	w.cancel = nil
	if w.closed || a.busy || a.loading {
		return false
	}
	if err != nil {
		w.next = now.Add(localBackgroundInterval)
		if w.lastError != err.Error() {
			w.lastError = err.Error()
			a.queue("error", a.t("localWatchFailed"), err.Error())
			a.invalidate()
		}
		return false
	}
	w.lastError = ""
	return stamp != "" && stamp != w.stamp
}

func (a *app) pollLocalChanges(now time.Time) {
	req, ok := a.beginLocalProbe(now)
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.localWatch.cancel = cancel
	go func() {
		defer cancel()
		stamp, err := readLocalWatchStamp(ctx, req.repo, req.entries)
		a.hopToMain(func() {
			if a.acceptLocalProbe(req, stamp, err, time.Now()) {
				a.openRepository(req.repo.Root, false)
			}
		})
	}()
}

func (a *app) startLocalWatching() {
	ctx, cancel := context.WithCancel(a.ctx)
	a.localWatch.focused = true
	a.window.OnFocus(func() {
		a.localWatch.focused = true
		a.localWatch.next = time.Time{}
		a.pollLocalChanges(time.Now())
	})
	a.window.OnBlur(func() { a.localWatch.focused = false })
	a.window.OnClosed(func() {
		a.localWatch.closed = true
		a.cancelLocalProbe()
		cancel()
	})
	go func() {
		ticker := time.NewTicker(localProbeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.hopToMain(func() { a.pollLocalChanges(time.Now()) })
			}
		}
	}()
}
