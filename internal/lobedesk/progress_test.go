package lobedesk

import (
	"testing"
	"time"
)

// The title bar reports what is running: a fetch names itself, any other busy
// operation shares the general line, and idle shows nothing. A fetch is busy
// too, so it must win over the general line or its own name never shows.
func TestProgressTextNamesTheWorkInFlight(t *testing.T) {
	a := &app{locale: localeChinese}
	if got := a.progressText(); got != "" {
		t.Fatalf("idle = %q; want empty", got)
	}
	a.busy = true
	if got := a.progressText(); got != "正在处理…" {
		t.Fatalf("busy = %q; want 正在处理…", got)
	}
	a.fetching = true
	if got := a.progressText(); got != "正在获取…" {
		t.Fatalf("fetching = %q; want 正在获取…", got)
	}
	a.busy = false
	if got := a.progressText(); got != "正在获取…" {
		t.Fatalf("fetching alone = %q; want 正在获取…", got)
	}
}

// The two lines must stay distinguishable in both languages, or a fetch would
// read as the general spinner it replaces.
func TestFetchAndGeneralProgressReadDifferently(t *testing.T) {
	for _, locale := range []locale{localeChinese, localeEnglish} {
		a := &app{locale: locale}
		fetch, general := a.t("fetchingNow"), a.t("busyNow")
		if fetch == "" || general == "" {
			t.Fatalf("%s: progress text missing: %q/%q", locale, fetch, general)
		}
		if fetch == general {
			t.Fatalf("%s: fetch and general progress share %q", locale, fetch)
		}
	}
}

// An operation that names itself is what the reader waiting on a push or a
// rebase sees; without a name the bar keeps its general line. A fetch still
// wins over both, because its own line is what the reader looks for.
func TestNamedOperationSaysWhatItIsDoing(t *testing.T) {
	a := &app{locale: localeChinese, busy: true, busyKey: string(busyPush)}
	if got := a.progressText(); got != "正在推送…" {
		t.Fatalf("named push = %q; want 正在推送…", got)
	}
	a.busyKey = string(busyRebase)
	if got := a.progressText(); got != "正在变基…" {
		t.Fatalf("named rebase = %q; want 正在变基…", got)
	}
	a.busyKey = ""
	if got := a.progressText(); got != "正在处理…" {
		t.Fatalf("unnamed = %q; want the general line", got)
	}
	a.busyKey = string(busyPush)
	a.fetching = true
	if got := a.progressText(); got != "正在获取…" {
		t.Fatalf("fetch during a named push = %q; want 正在获取…", got)
	}
}

// The progress line waits a moment so a fast operation does not flash a
// spinner and a cancel button. An untimed operation shows at once, which is
// what keeps the states above testable without a clock.
func TestProgressWaitsOutItsDelay(t *testing.T) {
	now := time.Now()
	if progressVisible(now, now.Add(progressDelay-time.Millisecond)) {
		t.Fatal("progress showed before its delay was over")
	}
	if !progressVisible(now, now.Add(progressDelay)) {
		t.Fatal("progress stayed hidden at the end of its delay")
	}
	if !progressVisible(time.Time{}, now) {
		t.Fatal("an untimed operation never showed")
	}

	a := &app{locale: localeChinese, busy: true, progressSince: now}
	if got := a.progressText(); got != "" {
		t.Fatalf("just-started operation said %q; want empty", got)
	}
	a.fetching = true
	if got := a.progressText(); got != "" {
		t.Fatalf("just-started fetch said %q; want empty", got)
	}
}
