package lobedesk

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestLegacyDetailSettingsMapToTabbedLayout(t *testing.T) {
	show, hide := true, false
	for _, test := range []struct {
		name                                string
		settings                            persistedSettings
		infoHidden, filesHidden, diffHidden bool
	}{
		{name: "missing fields"},
		{name: "metadata hidden", settings: persistedSettings{DetailsInfo: &hide}, infoHidden: true},
		{name: "file column hidden", settings: persistedSettings{DetailsChanges: &hide}, filesHidden: true},
		{name: "patch hidden", settings: persistedSettings{DetailsInfo: &show, DetailsDiff: &hide}, diffHidden: true},
		{name: "all hidden repaired", settings: persistedSettings{DetailsInfo: &hide, DetailsChanges: &hide, DetailsDiff: &hide}, filesHidden: true, diffHidden: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := &app{selectedSHA: "aaaa"}
			a.openDetailFile(detailFileKey{Path: "a.go"})
			a.commitSel = map[int]bool{1: true}
			a.applySettings(test.settings)
			if a.detailsInfoHidden != test.infoHidden || a.detailsChangesHidden != test.filesHidden || a.detailsDiffHidden != test.diffHidden {
				t.Fatal("legacy fields changed their content mapping")
			}
			if !a.detailSummaryShown() {
				t.Fatal("legacy setting removed permanent summary tab")
			}
			if test.diffHidden && (!a.detailTabs().Active.Summary || len(a.commitSel) > 0) {
				t.Fatal("hiding patch retained active file or invisible selection")
			}
			if test.diffHidden {
				a.applySettings(persistedSettings{DetailsDiff: &show})
				a.openDetailFile(detailFileKey{Path: "a.go"})
				if a.detailTabs().Active.Summary {
					t.Fatal("re-enabled patch cannot open a file")
				}
			}
		})
	}
	a := &app{worktreeSelected: true}
	a.openDetailFile(detailFileKey{Path: "a.go"})
	a.applySettings(persistedSettings{DetailsDiff: &hide})
	if a.detailTabs().Active.Summary || !a.detailFileShown() {
		t.Fatal("historical patch setting hid working changes")
	}
	a.lineSel = map[int]bool{1: true}
	a.applySettings(persistedSettings{HideDetails: &show})
	if len(a.lineSel) > 0 {
		t.Fatal("hidden details retained a working patch selection")
	}
}

func TestFirstDetailTabNamesTheCurrentContext(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		for _, worktree := range []bool{true, false} {
			a := &app{locale: language, worktreeSelected: worktree}
			label := a.t("summaryTab")
			if worktree {
				label = a.t("commit")
			}
			tt := pageTester(a, a.historyDiffPane)
			if _, ok := tt.AccessByRole(ui.RoleTab, label); !ok {
				t.Fatalf("worktree=%v: missing first tab %q", worktree, label)
			}
		}
	}
}
