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
			if !tt.HasText(a.t("loadingRepo")) {
				t.Fatal("slow repository read has no visible progress")
			}
			if _, ok := tt.AccessByRole(ui.RoleButton, a.t("cancel")); ok {
				t.Fatal("repository read offers a cancel action that only stops writes")
			}
			refresh, ok := tt.AccessByRole(ui.RoleButton, a.t("refresh"))
			if !ok || !refresh.Disabled {
				t.Fatal("active read can start another refresh")
			}
			a.loading = false
			tt.Frame()
			if tt.HasText(a.t("loadingRepo")) {
				t.Fatal("completed read kept title progress")
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
