package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
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

func TestBranchSelectorShowsThinBottomProgressLineWithoutReplacingBranchName(t *testing.T) {
	a := &app{
		locale: localeChinese,
		repo:   &gitrepo.Repository{Root: t.TempDir()},
		status: gitrepo.Status{Branch: "main", Upstream: "origin/main", Ahead: 2, Behind: 1},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		a.toolbarCommandControl(c, true)
	}, 360, 44)

	// Idle: no progress bar inside the branch selector.
	if bar, ok := tt.AccessByRole(ui.RoleProgress, a.t("fetchingNow")); ok {
		t.Fatalf("idle branch selector unexpectedly rendered progress bar: %+v", bar)
	}
	if _, ok := tt.Find("main"); !ok {
		t.Fatal("idle branch selector missing branch name")
	}

	// Active determinate fetch: keeps branch name intact and renders a 2px bottom line.
	a.fetching = true
	a.opProgress = gitrepo.OperationProgress{Active: true, Phase: "Receiving objects", Percent: 68, Fraction: 0.64}
	tt.Frame()

	branch, ok := tt.AccessByRole(ui.RoleButton, a.t("branches"))
	if !ok {
		t.Fatal("missing branch selector during fetch")
	}
	if _, ok := tt.Find("main"); !ok {
		t.Fatal("active progress must keep the branch name visible")
	}
	bar, ok := tt.AccessByRole(ui.RoleProgress, a.t("fetchingNow"))
	if !ok {
		t.Fatal("expected thin progress line inside branch selector")
	}
	if bar.Bounds.H != branchProgressHeight {
		t.Fatalf("progress bar height = %v; want %v", bar.Bounds.H, branchProgressHeight)
	}
	if bar.Bounds.Y+bar.Bounds.H > branch.Bounds.Y+branch.Bounds.H || bar.Bounds.Y < branch.Bounds.Y+branch.Bounds.H-4 {
		t.Fatalf("progress bar %+v is not anchored at bottom of branch selector %+v", bar.Bounds, branch.Bounds)
	}
	if bar.Bounds.X < branch.Bounds.X || bar.Bounds.X+bar.Bounds.W > branch.Bounds.X+branch.Bounds.W {
		t.Fatalf("progress bar %+v overflows branch selector %+v", bar.Bounds, branch.Bounds)
	}

	// Indeterminate operation (no percentage yet): still renders 2px line.
	a.opProgress = gitrepo.OperationProgress{Active: true, Indeterminate: true}
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleProgress, a.t("fetchingNow")); !ok {
		t.Fatal("expected indeterminate thin progress line inside branch selector")
	}

	// Finished: thin progress line disappears.
	a.fetching = false
	a.opProgress = gitrepo.OperationProgress{}
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleProgress, a.t("fetchingNow")); ok {
		t.Fatal("progress line remained after operation finished")
	}
}

func TestBeginOperationWiresRealGitProgressAndClearsOnFinish(t *testing.T) {
	base := context.Background()
	repo, err := gitrepo.Init(base, filepath.Join(t.TempDir(), "work"))
	if err != nil {
		t.Fatalf("init work: %v", err)
	}
	cloneDir := filepath.Join(t.TempDir(), "clone")
	if err := os.WriteFile(filepath.Join(repo.Root, "f.txt"), []byte("hello progress\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := repo.Stage(base, "f.txt"); err != nil {
		t.Fatal(err)
	}
	for _, cfg := range [][2]string{{"user.name", "Test User"}, {"user.email", "test@example.com"}} {
		if out, err := exec.Command("git", "-C", repo.Root, "config", cfg[0], cfg[1]).CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v\n%s", cfg[0], err, out)
		}
	}
	if err := repo.Commit(base, "initial"); err != nil {
		t.Fatal(err)
	}
	// Clone work into a second checkout using beginOperation's context so real
	// git --progress streams straight into a.opProgress.
	a := &app{
		ctx:       base,
		locale:    localeEnglish,
		runOnMain: func(fn func()) { fn() },
	}
	opCtx, cancel := a.beginOperation(busyPush)
	defer cancel()
	if _, err := gitrepo.Clone(opCtx, "file://"+repo.Root, cloneDir); err != nil {
		t.Fatalf("clone with beginOperation ctx: %v", err)
	}
	if !a.opProgress.Active {
		t.Fatalf("expected beginOperation to receive live git progress, got %+v", a.opProgress)
	}
	a.finishOperation()
	if a.opProgress.Active || a.opProgress.Fraction != 0 {
		t.Fatalf("finishOperation did not reset opProgress: %+v", a.opProgress)
	}
}
