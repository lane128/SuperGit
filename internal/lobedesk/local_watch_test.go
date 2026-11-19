package lobedesk

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalWatchRefreshesAndPreservesDraftWithoutRepeatedReloads(t *testing.T) {
	a, callbacks := overviewFixture(t)
	a.page = "history"
	a.selectedPath = "same.go"
	a.commitText = "keep title\n\nkeep body"
	a.loadWorktreeOverview()
	waitOverview(t, a, callbacks)
	stamp, err := readLocalWatchStamp(a.ctx, a.repo, a.status.Entries)
	if err != nil {
		t.Fatal(err)
	}
	a.localWatch.stamp = stamp
	if err := os.WriteFile(filepath.Join(a.repo.Root, "same.go"), []byte("external automatic save\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.pollLocalChanges(time.Now())
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for a.localWatch.reading || a.loading || a.diffLoading || !strings.Contains(a.diff.Text, "+external automatic save") {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("local change did not refresh patch")
		}
	}
	if a.commitText != "keep title\n\nkeep body" {
		t.Fatal("automatic refresh lost commit draft")
	}
	epoch := a.loadSeq
	a.pollLocalChanges(time.Now().Add(time.Minute))
	for a.localWatch.reading {
		select {
		case fn := <-callbacks:
			fn()
		case <-timer.C:
			t.Fatal("unchanged probe did not finish")
		}
	}
	if a.loadSeq != epoch {
		t.Fatal("unchanged repository repeatedly reloaded after automatic refresh")
	}
}

func TestLocalProbeCanceledByRefreshAndWrite(t *testing.T) {
	for _, write := range []bool{false, true} {
		t.Run(fmt.Sprint(write), func(t *testing.T) {
			a, _ := overviewFixture(t)
			_, ok := a.beginLocalProbe(time.Now())
			if !ok {
				t.Fatal("probe unavailable")
			}
			ctx, cancel := context.WithCancel(a.ctx)
			defer cancel()
			a.localWatch.cancel = cancel
			if write {
				_, stop := a.beginOperation(busyCommit)
				defer stop()
			} else {
				a.openRepository(a.repo.Root, false)
			}
			if ctx.Err() == nil || a.localWatch.reading {
				t.Fatal("refresh/write did not cancel the in-flight local probe")
			}
		})
	}
}

func TestLocalWatchDetectsFurtherDirtySavesWithoutChangingGitFlags(t *testing.T) {
	a, _ := overviewFixture(t)
	base, err := readLocalWatchStamp(a.ctx, a.repo, a.status.Entries)
	if err != nil {
		t.Fatal(err)
	}
	gitBefore, err := a.repo.ChangeStamp(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.repo.Root, "same.go")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("another working version\n"), 0600); err != nil {
		t.Fatal(err)
	}
	later := info.ModTime().Add(time.Second)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
	gitAfter, err := a.repo.ChangeStamp(a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if gitAfter != gitBefore {
		t.Fatal("fixture must retain identical Git metadata while dirty bytes change")
	}
	updated, err := readLocalWatchStamp(a.ctx, a.repo, a.status.Entries)
	if err != nil {
		t.Fatal(err)
	}
	if base == updated {
		t.Fatal("local watcher missed a further save of an already-dirty file")
	}
	again, err := readLocalWatchStamp(a.ctx, a.repo, a.status.Entries)
	if err != nil || again != updated {
		t.Fatal("unchanged local stamp is unstable")
	}
}

func TestLocalWatchSchedulingAndStaleWrites(t *testing.T) {
	a, _ := overviewFixture(t)
	a.commitText = "draft survives"
	a.localWatch.focused = true
	a.localWatch.stamp = "before"
	now := time.Now()
	req, ok := a.beginLocalProbe(now)
	if !ok {
		t.Fatal("idle repository did not start detection")
	}
	if _, ok := a.beginLocalProbe(now.Add(time.Minute)); ok {
		t.Fatal("overlapping probe started")
	}
	if a.acceptLocalProbe(req, "before", nil, now) {
		t.Fatal("unchanged repository refreshed")
	}
	if _, ok := a.beginLocalProbe(now.Add(time.Second)); ok {
		t.Fatal("active interval not enforced")
	}
	req, ok = a.beginLocalProbe(now.Add(3 * time.Second))
	if !ok {
		t.Fatal("active interval did not expire")
	}
	if !a.acceptLocalProbe(req, "after", nil, now) {
		t.Fatal("external state change did not request refresh")
	}
	if a.commitText != "draft survives" {
		t.Fatal("probe changed draft")
	}
	a.busy = true
	if _, ok := a.beginLocalProbe(now.Add(time.Minute)); ok {
		t.Fatal("write allowed a concurrent probe")
	}
	a.busy = false
	req, ok = a.beginLocalProbe(now.Add(time.Minute))
	if !ok {
		t.Fatal("probe did not recover after write")
	}
	a.cancelLocalProbe()
	if a.acceptLocalProbe(req, "late", nil, now) {
		t.Fatal("canceled probe was accepted")
	}
	req, ok = a.beginLocalProbe(now.Add(2 * time.Minute))
	if !ok {
		t.Fatal("new probe unavailable")
	}
	a.loadSeq++
	if a.acceptLocalProbe(req, "old epoch", nil, now) {
		t.Fatal("old refresh epoch was accepted")
	}
}

func TestLocalWatchDefersInFlightRefreshUntilFolderChooserCloses(t *testing.T) {
	a, _ := overviewFixture(t)
	now := time.Now()
	a.localWatch.stamp = "before"
	req, ok := a.beginLocalProbe(now)
	if !ok {
		t.Fatal("probe unavailable before chooser")
	}
	a.repositoryChooserOpen = true
	if a.acceptLocalProbe(req, "after", nil, now) {
		t.Fatal("in-flight probe can refresh while the chooser owns the window")
	}
	if _, ok := a.beginLocalProbe(now.Add(time.Minute)); ok {
		t.Fatal("new probe started while choosing a folder")
	}
	a.repositoryChooserOpen = false
	retry, ok := a.beginLocalProbe(now)
	if !ok || !a.acceptLocalProbe(retry, "after", nil, now) {
		t.Fatal("closing the chooser did not make the deferred refresh eligible again")
	}
}

func TestLocalWatchBackgroundErrorsAndClose(t *testing.T) {
	a, _ := overviewFixture(t)
	a.localWatch.stamp = "baseline"
	now := time.Now()
	req, ok := a.beginLocalProbe(now)
	if !ok {
		t.Fatal("background probe unavailable")
	}
	a.acceptLocalProbe(req, "", errors.New("offline"), now)
	if _, ok := a.beginLocalProbe(now.Add(4 * time.Second)); ok {
		t.Fatal("failed/background probe did not back off")
	}
	if len(a.notices[toastMain]) != 1 {
		t.Fatal("probe failure did not provide recovery feedback")
	}
	req, ok = a.beginLocalProbe(now.Add(15 * time.Second))
	if !ok {
		t.Fatal("background retry unavailable")
	}
	a.acceptLocalProbe(req, "", errors.New("offline"), now.Add(15*time.Second))
	if len(a.notices[toastMain]) != 1 {
		t.Fatal("repeated probe failure spammed notifications")
	}
	a.localWatch.closed = true
	if _, ok := a.beginLocalProbe(now.Add(time.Minute)); ok {
		t.Fatal("closed window started detection")
	}
}
