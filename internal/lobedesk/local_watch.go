package lobedesk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"supergit/internal/gitrepo"
)

const localProbeInterval = 3 * time.Second
const localBackgroundInterval = 15 * time.Second

type localRepositoryWatch struct {
	initializing             bool
	detailsPending           bool
	refStamp                 string
	stamp                    string
	seq                      uint64
	reading, focused, closed bool
	next                     time.Time
	lastError                string
	cancel                   context.CancelFunc
	submoduleRound           int
}

type localWatchRequest struct {
	repo       *gitrepo.Repository
	epoch, seq uint64
	entries    []gitrepo.Entry
	submodules []gitrepo.Submodule
	baseline   string
}

// ChangeStamp observes Git metadata, but deliberately excludes working bytes.
// Track metadata of all already-dirty files too, so further saves outside the
// selected file invalidate the overview. Do not walk ignored or untracked trees.
func readLocalWatchStamp(parent context.Context, repo *gitrepo.Repository, entries []gitrepo.Entry) (string, error) {
	ctx, cancel := context.WithTimeout(parent, localBackgroundInterval)
	status, err := repo.StatusText(ctx)
	cancel()
	if err != nil {
		return "", err
	}
	return readLocalWatchStampWithStatus(parent, repo, entries, status)
}

// readLocalWatchStampWithStatus takes the porcelain status an open already
// read for display, so the slowest probe read is shared rather than repeated.
func readLocalWatchStampWithStatus(parent context.Context, repo *gitrepo.Repository, entries []gitrepo.Entry, status string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, localBackgroundInterval)
	defer cancel()
	stamp, err := repo.LocalStampFromStatus(ctx, status)
	if err != nil {
		return "", err
	}
	return watchFileStamp(ctx, repo, entries, stamp)
}
func watchFileStamp(ctx context.Context, repo *gitrepo.Repository, entries []gitrepo.Entry, stamp string) (string, error) {
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
		// The directory's mtime describes files inside the child, not a parent
		// diff. Keep the hash shape stable across child-only edits.
		if entry.IsSubmodule() && info.IsDir() {
			continue
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

const submoduleProbeBatch = 4
const submoduleProbeTimeout = 2 * time.Second

// The parent porcelain carries flags for child edits, not a count. Prioritize
// those children and any displayed badge (the parent and child snapshots can
// straddle an edit). Rotate through clean children too because nested ignore
// configuration can hide a child's gitlink changes from the parent's flags.
func (a *app) nextSubmoduleProbes() []gitrepo.Submodule {
	if len(a.submodules) == 0 {
		return nil
	}
	dirty := make(map[string]bool)
	for _, entry := range a.status.Entries {
		if entry.SubmoduleHasLocalEdits() {
			dirty[entry.Path] = true
		}
	}
	var priority, clean []gitrepo.Submodule
	seen := make(map[string]bool)
	for _, module := range a.submodules {
		if module.Pending || seen[module.Path] {
			continue
		}
		seen[module.Path] = true
		if dirty[module.Path] || module.ChangedFiles > 0 || module.Error != "" {
			priority = append(priority, module)
		} else {
			clean = append(clean, module)
		}
	}
	round := a.localWatch.submoduleRound
	a.localWatch.submoduleRound++
	var probes []gitrepo.Submodule
	for _, group := range [][]gitrepo.Submodule{priority, clean} {
		for i := range min(len(group), submoduleProbeBatch) {
			probes = append(probes, group[(round*submoduleProbeBatch+i)%len(group)])
		}
	}
	return probes
}

func readSubmoduleUpdates(ctx context.Context, repo *gitrepo.Repository, modules []gitrepo.Submodule) []gitrepo.Submodule {
	if len(modules) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, submoduleProbeTimeout)
	defer cancel()
	var mu sync.Mutex
	var updates []gitrepo.Submodule
	repo.InspectSubmodules(ctx, probeInventory(modules), func(after gitrepo.Submodule) { mu.Lock(); updates = append(updates, after); mu.Unlock() })
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		seen := map[string]bool{}
		for _, m := range updates {
			seen[m.Path] = true
		}
		for _, m := range modules {
			if !seen[m.Path] {
				updates = append(updates, gitrepo.Submodule{Path: m.Path, Error: ctx.Err().Error()})
				seen[m.Path] = true
			}
		}
	}
	return updates
}

// A previous checkout error is precisely the row a periodic probe should retry.
func probeInventory(modules []gitrepo.Submodule) []gitrepo.Submodule {
	rows := slices.Clone(modules)
	for i := range rows {
		rows[i].Error = ""
	}
	return rows
}

// A count or branch label can change while the parent snapshot stays valid.
// Patch those rows in place so parent history and file selections stay live.
func (a *app) applySubmoduleUpdates(updates []gitrepo.Submodule) (reload bool) {
	changed := false
	for _, after := range updates {
		for i, before := range a.submodules {
			if before.Path != after.Path {
				continue
			}
			if after.Error != "" {
				before.Error = after.Error
				before.Pending = false
				a.submodules[i] = before
				changed = true
				continue
			}
			// A refresh marks rows pending but keeps their last known detail.
			// A moved checkout still needs reconciliation then: the parent
			// gitlink diff was read before this observation landed.
			known := before.CheckoutKnown || before.Initialized || before.Current != "" || (!before.Pending && before.Error == "")
			reconcile := known && (before.Initialized != after.Initialized || before.Current != after.Current)
			if !reconcile && !before.Pending && before.ChangedFiles == after.ChangedFiles && before.Branch == after.Branch && before.Dirty == after.Dirty &&
				(before.Error == "") == (after.Error == "") {
				continue
			}
			after.CheckoutKnown = true
			after.Name, after.Recorded, after.Index = before.Name, before.Recorded, before.Index
			a.submodules[i] = after
			changed = true
			reload = reload || reconcile
		}
	}
	if changed {
		a.invalidate()
	}
	return reload
}

// applySubmoduleDetail publishes one asynchronous child observation from a
// parent snapshot. A moved checkout keeps the parent snapshot valid but makes
// any displayed gitlink diff stale, so re-read the working-tree details.
func (a *app) applySubmoduleDetail(seq uint64, repo *gitrepo.Repository, m gitrepo.Submodule) {
	if a.repo != repo || a.loadSeq != seq || a.localWatch.closed {
		return
	}
	if a.applySubmoduleUpdates([]gitrepo.Submodule{m}) {
		a.reloadWorkingTreeDetails()
	}
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
	if w.initializing || w.closed || a.repo == nil || a.loading || a.busy || a.repositoryChooserOpen || w.reading || now.Before(w.next) ||
		(a.repositoryTabs != nil && (a.repositoryTabs.active != a || a.repositoryTabs.opening != nil)) ||
		a.hasPendingDialog(toastMain) {
		return localWatchRequest{}, false
	}
	interval := localBackgroundInterval
	if w.focused {
		interval = localProbeInterval
	}
	w.next = now.Add(interval)
	w.reading = true
	w.seq++
	return localWatchRequest{
		repo: a.repo, epoch: a.loadSeq, seq: w.seq, entries: slices.Clone(a.status.Entries),
		submodules: a.nextSubmoduleProbes(), baseline: w.stamp,
	}, true
}

func (a *app) acceptLocalProbe(req localWatchRequest, stamp string, err error, now time.Time, updates []gitrepo.Submodule) bool {
	w := &a.localWatch
	if w.seq != req.seq || a.repo != req.repo || a.loadSeq != req.epoch {
		return false
	}
	w.reading = false
	w.cancel = nil
	if a.repositoryChooserOpen || a.hasPendingDialog(toastMain) {
		// An external change can arrive between keystrokes or picker stages.
		// Retry once the dialog closes so a reload cannot discard its input.
		w.next = time.Time{}
		return false
	}
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
	return stamp != "" && stamp != w.stamp || a.applySubmoduleUpdates(updates)
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
		var refs string
		if err == nil {
			refs, err = req.repo.RefStamp(ctx)
		}
		var updates []gitrepo.Submodule
		if err == nil && stamp == req.baseline {
			updates = readSubmoduleUpdates(ctx, req.repo, req.submodules)
		}
		a.hopToMain(func() {
			if a.acceptLocalProbe(req, stamp, err, time.Now(), updates) {
				affected := map[string]bool{}
				for _, m := range updates {
					affected[m.Path] = true
				}
				a.readRepositoryScope(req.repo.Root, false, "", affected)
			} else if err == nil && req.seq == a.localWatch.seq && req.epoch == a.loadSeq && !a.busy && !a.loading && !a.localWatch.closed && refs != a.localWatch.refStamp && !a.hasPendingDialog(toastMain) && !a.repositoryChooserOpen {
				a.refreshRefs()
			}
		})
	}()
}

func (a *app) startLocalWatching() {
	parent := a.ctx
	if a.repositoryTabs != nil {
		parent = a.repositoryTabs.ctx
	}
	ctx, cancel := context.WithCancel(parent)
	a.localWatch.focused = true
	a.window.OnFocus(func() {
		active := a.activeRepositorySession()
		active.localWatch.focused = true
		active.localWatch.next = time.Time{}
		active.pollLocalChanges(time.Now())
	})
	a.window.OnBlur(func() { a.activeRepositorySession().localWatch.focused = false })
	a.window.OnClosed(func() {
		if a.repositoryTabs != nil {
			a.repositoryTabs.saveSession()
			a.repositoryTabs.shutdown()
		} else {
			a.localWatch.closed = true
			a.cancelLocalProbe()
		}
		cancel()
		// Settings uses this workbench's sessions. Close it too so an
		// open-folder event can relaunch a workbench instead of reaching
		// a process that only has a Settings window left.
		if settings := a.settingsWindowSlot().Load(); settings != nil && !settings.IsDestroyed() {
			settings.Close()
		}
	})
	go func() {
		ticker := time.NewTicker(localProbeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.hopToMain(func() {
					if tabs := a.repositoryTabs; tabs != nil && tabs.openNextRepositoryFolder() {
						return
					}
					a.activeRepositorySession().pollLocalChanges(time.Now())
				})
			}
		}
	}()
}

func (a *app) installWatchBaseline(seq uint64, repo *gitrepo.Repository, baseline <-chan string) {
	a.localWatch.stamp = ""
	a.localWatch.initializing = true
	installBaseline := func(stamp string) bool {
		if seq == a.loadSeq && a.repo == repo && !a.localWatch.closed {
			a.localWatch.stamp = stamp
			a.localWatch.initializing = false
			a.localWatch.next = time.Time{}
			if stamp != "" {
				a.localWatch.lastError = ""
			}
			return true
		}
		return false
	}
	select {
	case stamp := <-baseline:
		installBaseline(stamp)
	default:
		go func() {
			stamp := <-baseline
			a.hopToMain(func() {
				if installBaseline(stamp) && a.localWatch.detailsPending {
					// Working-tree details were read before this baseline
					// existed. A save between those reads and this observation
					// would match the installed baseline, so read them again.
					a.localWatch.detailsPending = false
					a.reloadWorkingTreeDetails()
				}
			})
		}()
	}
}

func (a *app) publishSubmoduleInventory(ctx context.Context, seq uint64, repo *gitrepo.Repository, submodules []gitrepo.Submodule, reset, localChange bool, affected map[string]bool) {
	if !reset {
		previous := map[string]gitrepo.Submodule{}
		for _, m := range a.submodules {
			previous[m.Path] = m
		}
		for i, m := range submodules {
			if old, ok := previous[m.Path]; ok && m.Pending {
				if localChange && !old.Pending && !affected[m.Path] && m.Index == old.Index && m.Recorded == old.Recorded && old.Error == "" {
					old.Name = m.Name
					submodules[i] = old
					continue
				}
				old.Name, old.Index, old.Recorded, old.Pending, old.Error = m.Name, m.Index, m.Recorded, true, ""
				submodules[i] = old
			}
		}
	}
	a.submodules = submodules
	if len(submodules) > 0 {
		var inventory []gitrepo.Submodule
		for _, m := range submodules {
			if m.Pending {
				inventory = append(inventory, m)
			}
		}
		go repo.InspectSubmodules(ctx, inventory, func(m gitrepo.Submodule) {
			a.hopToMain(func() { a.applySubmoduleDetail(seq, repo, m) })
		})
	}
}
