package desktop

import (
	"context"
	"github.com/egoist/mygo"
	"supergit/internal/gitrepo"
	"time"
)

const probeInterval = 3 * time.Second
const backgroundProbeInterval = 15 * time.Second

type repositoryWatch struct {
	stamp     string
	reading   bool
	next      time.Time
	lastError string
}
type watchRequest struct {
	Ticket       ticket
	Path         string
	Staged       bool
	DiffSequence uint64
	Stamp        string
}

// All scheduling state belongs to the UI thread. The worker reads only the
// captured repository, context, path and group; old generations cannot publish.
func (a *app) beginProbe(now time.Time) (watchRequest, bool) {
	if a.closed || a.repo == nil || a.busy() || a.diffLoading || a.watch.reading || now.Before(a.watch.next) {
		return watchRequest{}, false
	}
	a.watch.reading = true
	interval := probeInterval
	if !a.focused {
		interval = backgroundProbeInterval
	}
	a.watch.next = now.Add(interval)
	return watchRequest{Ticket: a.guard.Begin("watch"), Path: a.path, Staged: a.staged, DiffSequence: a.guard.lanes["diff"], Stamp: a.watch.stamp}, true
}
func (a *app) acceptProbe(req watchRequest, stamp string, d gitrepo.Diff, err error, now time.Time) bool {
	if req.Ticket.repository != a.guard.repository {
		return false
	}
	a.watch.reading = false
	if a.closed || !a.guard.Accept(req.Ticket) || a.busy() {
		return false
	}
	// A valid metadata change takes precedence over a failed selected Diff.
	if stamp != "" && stamp != a.watch.stamp {
		a.watch.lastError = ""
		return true
	}
	if err != nil {
		if stamp != "" && req.Path != "" && req.Path == a.path && req.Staged == a.staged && req.DiffSequence == a.guard.lanes["diff"] && !a.diffLoading {
			a.failWorkspaceDiff(err)
			a.invalidate()
		}
		label := "自动 Diff 检测失败（可手动刷新）"
		if stamp == "" {
			a.watch.next = now.Add(backgroundProbeInterval)
			label = "自动检测失败（稍后重试，可手动刷新）"
		}
		key := label + err.Error()
		if a.watch.lastError != key {
			a.watch.lastError = key
			a.record(label, err)
			a.invalidate()
		}
		return false
	}
	a.watch.lastError = ""
	return req.Path != "" && req.Path == a.path && req.Staged == a.staged && req.DiffSequence == a.guard.lanes["diff"] && !a.diffLoading && (a.diffErr != nil || d.Text != a.diff.Text)
}
func (a *app) pollChanges() {
	req, ok := a.beginProbe(time.Now())
	if !ok {
		return
	}
	r := a.repo
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	go func() {
		defer cancel()
		stamp, err := r.ChangeStamp(ctx)
		var d gitrepo.Diff
		// A known state change already needs a snapshot; avoid delaying it for
		// a large/slow Diff, which is loaded independently by that snapshot.
		if err == nil && stamp == req.Stamp && req.Path != "" {
			d, err = r.Diff(ctx, req.Path, req.Staged)
		}
		mygo.RunOnMain(func() {
			if a.acceptProbe(req, stamp, d, err, time.Now()) {
				a.refreshSnapshot(true)
			}
		})
	}()
}
func (a *app) startWatching() {
	ctx, cancel := context.WithCancel(context.Background())
	a.watchCancel = cancel
	a.focused = true
	a.window.OnFocus(func() { a.focused = true; a.watch.next = time.Time{}; a.pollChanges() })
	a.window.OnBlur(func() { a.focused = false })
	a.window.OnClosed(func() { a.closed = true; cancel(); a.cancel() })
	go func() {
		ticker := time.NewTicker(probeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mygo.RunOnMain(func() { a.pollChanges() })
			}
		}
	}()
}
