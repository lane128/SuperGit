package lobedesk

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestSlowRepositoryReadHasVisibleTitleBarProgress(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			a := &app{locale: language, repo: &gitrepo.Repository{Root: t.TempDir()}, loading: true, loadSince: time.Now().Add(-time.Second)}
			tt := titleBarTester(a)
			if _, ok := tt.AccessByRole(ui.RoleText, a.t("loadingRepo")); ok {
				t.Fatal("slow repository read should not render duplicate left progress text when branch selector is visible")
			}
			if _, ok := tt.AccessByRole(ui.RoleProgress, a.t("loadingRepo")); !ok {
				t.Fatal("slow repository read has no visible branch progress bar")
			}
			if _, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel")); ok {
				t.Fatal("repository read offers a cancel action that only stops writes")
			}
			sync, ok := tt.AccessByRole(ui.RoleMenuButton, a.t("syncMenu"))
			if !ok || !sync.Disabled || a.canRefreshShortcut() {
				t.Fatal("active read allows a sync or another refresh")
			}
			a.loading = false
			tt.Frame()
			if _, ok := tt.AccessByRole(ui.RoleProgress, a.t("loadingRepo")); ok {
				t.Fatal("completed read kept branch progress bar")
			}
		})
	}
}

func TestRepositoryReadProgressWaitsAndKeepsOperationPriority(t *testing.T) {
	a := &app{locale: localeEnglish, repo: &gitrepo.Repository{Root: t.TempDir()}, loading: true, loadSince: time.Now().Add(time.Hour)}
	if got := a.progressText(); got != "" {
		t.Fatal("short read flashes progress")
	}
	a.loadSince = time.Now().Add(-time.Second)
	if got := a.progressText(); got != a.t("loadingRepo") {
		t.Fatalf("slow read says %q", got)
	}
	a.busy, a.busyKey = true, string(busyPush)
	if got := a.progressText(); got != a.t(string(busyPush)) {
		t.Fatal("read hides operation progress")
	}
	a.fetching = true
	if got := a.progressText(); got != a.t("fetchingNow") {
		t.Fatal("read hides fetch progress")
	}
	a.busy, a.fetching, a.loading = false, false, false
	if got := a.progressText(); got != "" {
		t.Fatal("idle retained progress")
	}
}
