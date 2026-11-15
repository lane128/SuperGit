package lobedesk

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func remoteActionsTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Width(300).Height(40).Children(func() { a.remoteActions(c) })
	}, 300, 40)
}

// A disabled toolbar icon used to be just grey: nothing said what to configure
// first, and MyGo hides a disabled element's own tooltip. Each icon now says
// why on hover, and the reason also reaches the accessibility tree.
func TestRemoteActionsExplainWhyTheyAreDisabled(t *testing.T) {
	a := &app{locale: localeChinese}
	tt := remoteActionsTester(a)

	for _, test := range []struct {
		title, reason string
	}{
		{a.t("fetch"), a.t("noRemote")},
		{a.t("pull"), a.t("noUpstream")},
		{a.t("pushButton"), a.t("noRemote")},
	} {
		info, ok := tt.AccessByRole(ui.RoleButton, test.title)
		if !ok {
			t.Fatalf("no %q button", test.title)
		}
		if !info.Disabled {
			t.Errorf("%q is enabled without a repository", test.title)
		}
		if !strings.Contains(info.Description, test.reason) {
			t.Errorf("%q describes itself as %q; want it to mention %q", test.title, info.Description, test.reason)
		}
	}

	fetch, _ := tt.AccessByRole(ui.RoleButton, a.t("fetch"))
	tt.Move(fetch.Bounds.X+fetch.Bounds.W/2, fetch.Bounds.Y+fetch.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if _, ok := tt.Find(a.t("noRemote")); !ok {
		t.Errorf("hovering the disabled fetch button shows no %q tooltip", a.t("noRemote"))
	}
}

// With a remote but no checked-out branch, push has its own reason: "no
// remotes" would send the reader to the wrong setting.
func TestPushWithoutABranchSaysSo(t *testing.T) {
	a := &app{
		locale:  localeChinese,
		remotes: []gitrepo.Remote{{Name: "origin"}},
		status:  gitrepo.Status{Entries: nil},
	}
	tt := remoteActionsTester(a)
	info, ok := tt.AccessByRole(ui.RoleButton, a.t("pushButton"))
	if !ok {
		t.Fatal("no push button")
	}
	if !info.Disabled {
		t.Error("push is enabled without a branch")
	}
	if !strings.Contains(info.Description, a.t("pushNoBranch")) {
		t.Errorf("push describes itself as %q; want %q", info.Description, a.t("pushNoBranch"))
	}
	if strings.Contains(info.Description, a.t("noRemote")) {
		t.Errorf("push blames the remote although one is configured: %q", info.Description)
	}
}
