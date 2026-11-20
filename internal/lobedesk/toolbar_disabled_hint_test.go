package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

func TestRemoteActionsExplainWhyTheyAreDisabled(t *testing.T) {
	a := &app{locale: localeChinese, repo: &gitrepo.Repository{Root: t.TempDir()},
		status: gitrepo.Status{Branch: "main"}}
	tt := pageTester(a, a.titleBar)
	clickRepositoryTabsControl(t, tt, ui.RoleMenuButton, a.t("syncMenu"))

	for _, test := range []struct {
		title, reason string
	}{
		{a.t("fetch"), a.t("noRemote")},
		{a.t("pull"), a.t("noUpstream")},
		{a.t("pushBranchAction"), a.t("noRemote")},
	} {
		info, ok := tt.AccessByRole(ui.RoleMenuItem, test.title)
		if !ok || !info.Disabled || !strings.Contains(info.Description, test.reason) || !tt.HasText(test.reason) {
			t.Errorf("%q must be disabled with a visible and accessible reason %q: %+v", test.title, test.reason, info)
		}
	}
}

func TestPushWithoutABranchSaysSo(t *testing.T) {
	a := &app{locale: localeChinese, repo: &gitrepo.Repository{Root: t.TempDir()},
		remotes: []gitrepo.Remote{{Name: "origin"}}}
	tt := pageTester(a, a.titleBar)
	clickRepositoryTabsControl(t, tt, ui.RoleMenuButton, a.t("syncMenu"))
	info, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("pushBranchAction"))
	if !ok || !info.Disabled || !strings.Contains(info.Description, a.t("pushNoBranch")) {
		t.Fatalf("Push enabled without a branch or blamed the configured remote: %+v", info)
	}
}
