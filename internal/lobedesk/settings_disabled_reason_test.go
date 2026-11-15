package lobedesk

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// A disabled checkbox that reads as merely unavailable looks broken. The last
// details block and pruning (when automatic fetch is off) now say why they
// cannot be changed, in the tooltip and to assistive technology.
// tallSettingsTester renders the settings page in a viewport tall enough to
// hold it. A tooltip follows the pointer, and the page's lower cards sit past
// the usual 650-high test window.
func tallSettingsTester(a *app) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(900).Height(1400).Children(func() { a.settingsPage(c) })
	}, 900, 1400)
}

func TestDetailsKeepsOneBlockWithAReason(t *testing.T) {
	a := &app{
		locale:               localeChinese,
		detailsInfoHidden:    false,
		detailsChangesHidden: true,
		detailsDiffHidden:    true,
	}
	tt := tallSettingsTester(a)
	info, ok := tt.Access(a.t("detailsInfo"))
	if !ok {
		t.Fatal("the details block checkbox is not in the accessibility tree")
	}
	if !info.Disabled {
		t.Fatal("the last standing block is not marked unavailable")
	}
	if info.Description != a.t("detailsKeepOneBlock") {
		t.Errorf("the disabled block reads %q; want the reason %q", info.Description, a.t("detailsKeepOneBlock"))
	}
	// The reason has to reach the eye too: hovering the dimmed box shows it.
	tt.Move(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if _, ok := tt.Find(a.t("detailsKeepOneBlock")); !ok {
		t.Error("hovering the disabled block shows no reason")
	}
}

func TestPruningSaysItNeedsAutomaticFetch(t *testing.T) {
	a := &app{locale: localeChinese, autoFetch: false}
	tt := tallSettingsTester(a)
	prune, ok := tt.Access(a.t("autoFetchPrune"))
	if !ok {
		t.Fatal("the prune checkbox is not in the accessibility tree")
	}
	if !prune.Disabled {
		t.Fatal("pruning is enabled with automatic fetch off")
	}
	if prune.Description != a.t("autoFetchPruneNeedsFetch") {
		t.Errorf("the disabled prune box reads %q; want the reason %q", prune.Description, a.t("autoFetchPruneNeedsFetch"))
	}
	tt.Move(prune.Bounds.X+prune.Bounds.W/2, prune.Bounds.Y+prune.Bounds.H/2)
	tt.Advance(time.Second)
	tt.Frame()
	if _, ok := tt.Find(a.t("autoFetchPruneNeedsFetch")); !ok {
		t.Error("hovering the disabled prune box shows no reason")
	}
}
