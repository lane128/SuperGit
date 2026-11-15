package lobedesk

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"supergit/internal/gitrepo"
)

func TestSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")
	hidden := false
	autoFetch, prune := true, true
	split, stashSplit, branchSplit := float32(0.42), float32(0.35), float32(0.28)
	hide := true
	fontScale := fontScaleLarge
	showInfo := false
	hideAuthor, hideTime := true, false
	every := 15
	if err := writeSettings(path, persistedSettings{
		Theme: "dark", Language: "en", ShowGraph: &hidden,
		AutoFetch: &autoFetch, AutoFetchPrune: &prune,
		HistorySplit: &split, StashSplit: &stashSplit,
		BranchDetailsSplit: &branchSplit,
		HideDetails:        &hide, FontScale: &fontScale,
		DetailsInfo: &showInfo,
		HistoryLine: "thick", HistoryNode: "large", HideAuthor: &hideAuthor, HideTime: &hideTime,
		AutoFetchEvery: &every,
	}); err != nil {
		t.Fatal(err)
	}
	loaded := readSettings(path)
	if loaded.Theme != "dark" || loaded.Language != "en" || loaded.ShowGraph == nil || *loaded.ShowGraph {
		t.Fatalf("loaded %+v; want dark/en/hidden graph", loaded)
	}
	if loaded.AutoFetch == nil || !*loaded.AutoFetch || loaded.AutoFetchPrune == nil || !*loaded.AutoFetchPrune {
		t.Fatalf("loaded %+v; want auto fetch + prune", loaded)
	}
	if loaded.HistorySplit == nil || *loaded.HistorySplit != split ||
		loaded.StashSplit == nil || *loaded.StashSplit != stashSplit ||
		loaded.BranchDetailsSplit == nil || *loaded.BranchDetailsSplit != branchSplit {
		t.Fatalf("loaded splits %+v; want %v/%v/%v", loaded, split, stashSplit, branchSplit)
	}
	a := &app{themeMode: "system", locale: localeChinese, showGraph: true}
	a.applySettings(loaded)
	if a.themeMode != "dark" || a.locale != localeEnglish || a.showGraph {
		t.Fatalf("applied theme=%q locale=%q graph=%t", a.themeMode, a.locale, a.showGraph)
	}
	if !a.autoFetch || !a.autoFetchPrune {
		t.Fatalf("applied fetch=%t prune=%t; want both on", a.autoFetch, a.autoFetchPrune)
	}
	if a.historySplit.share != split || a.stashSplit.share != stashSplit || a.branchSplit.share != branchSplit {
		t.Fatalf("applied splits %v/%v/%v; want %v/%v/%v",
			a.historySplit.share, a.stashSplit.share, a.branchSplit.share,
			split, stashSplit, branchSplit)
	}
	if !a.historyDetailsHidden {
		t.Fatal("applied the settings but not the hidden details pane")
	}
	if loaded.FontScale == nil || *loaded.FontScale != fontScaleLarge {
		t.Fatalf("loaded font scale %+v; want %v", loaded.FontScale, fontScaleLarge)
	}
	if got := uiFontScaleValue(); got != fontScaleLarge {
		t.Fatalf("applied font scale %v; want %v", got, fontScaleLarge)
	}
	if got := scaledFont(10); got != 10*fontScaleLarge {
		t.Fatalf("scaledFont(10) = %v; want %v", got, 10*fontScaleLarge)
	}
	if !a.detailsInfoHidden || a.detailsChangesHidden || a.detailsDiffHidden {
		t.Fatalf("applied details blocks info=%t changes=%t diff=%t; want only info hidden",
			a.detailsInfoHidden, a.detailsChangesHidden, a.detailsDiffHidden)
	}
	if a.historyLine != graphLineThick || a.historyNode != graphNodeLarge {
		t.Fatalf("applied graph style line=%v node=%v; want %v/%v",
			a.historyLine, a.historyNode, graphLineThick, graphNodeLarge)
	}
	if !a.hideAuthor || a.hideTime {
		t.Fatalf("applied columns author=%t time=%t; want author hidden", a.hideAuthor, a.hideTime)
	}
	if a.autoFetchEvery != 15 {
		t.Fatalf("applied fetch interval %d; want 15", a.autoFetchEvery)
	}
	setUIFontScale(fontScaleStandard)
}

func TestReadSettingsFallsBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if got := readSettings(path); got.Theme != "" || got.Language != "" || got.ShowGraph != nil {
		t.Fatalf("missing file = %+v; want empty settings", got)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readSettings(path); got.Theme != "" || got.ShowGraph != nil {
		t.Fatalf("corrupt file = %+v; want empty settings", got)
	}
}

func TestApplySettingsKeepsDefaultsForUnknownValues(t *testing.T) {
	a := &app{themeMode: "dark", locale: localeEnglish, showGraph: true}
	badEvery := 7
	a.applySettings(persistedSettings{Theme: "neon", Language: "fr", AutoFetchEvery: &badEvery})
	if a.themeMode != "dark" || a.locale != localeEnglish || !a.showGraph || a.autoFetch || a.autoFetchPrune || a.historySplit.share != 0 {
		t.Fatalf("unknown values changed theme=%q locale=%q graph=%t fetch=%t prune=%t split=%v",
			a.themeMode, a.locale, a.showGraph, a.autoFetch, a.autoFetchPrune, a.historySplit.share)
	}
	if a.autoFetchEvery != 0 {
		t.Fatalf("an interval outside the presets was kept: %d", a.autoFetchEvery)
	}
}

// Resetting preferences must land on exactly the values Run starts with,
// whichever of them the reader had changed.
func TestApplyDefaultsRestoresStartupValues(t *testing.T) {
	a := &app{
		themeMode: "dark", palette: themePaletteCatppuccinMacchiato, locale: localeEnglish,
		showGraph: false, autoFetch: true, autoFetchPrune: true,
	}
	a.historySplit.share = 0.3
	a.stashSplit.share = 0.4
	a.branchSplit.share = 0.33
	a.historyDetailsHidden = true
	a.detailsInfoHidden = true
	a.detailsChangesHidden = true
	a.detailsDiffHidden = true
	a.historyLine = graphLineThin
	a.historyNode = graphNodeSmall
	a.hideAuthor = true
	a.hideTime = true
	a.autoFetchEvery = 30
	a.applyDefaults()
	if a.themeMode != "system" || a.palette != themePaletteCatppuccin ||
		a.locale != localeChinese || !a.showGraph || a.autoFetch || a.autoFetchPrune {
		t.Fatalf("theme=%q palette=%q locale=%q graph=%t fetch=%t prune=%t",
			a.themeMode, a.palette, a.locale, a.showGraph, a.autoFetch, a.autoFetchPrune)
	}
	if a.historySplit.share != 0 || a.stashSplit.share != 0 || a.branchSplit.share != 0 {
		t.Fatalf("splits %v/%v/%v; want every divider back at its default",
			a.historySplit.share, a.stashSplit.share, a.branchSplit.share)
	}
	if a.historyDetailsHidden {
		t.Fatal("the reset left the details pane hidden")
	}
	if a.detailsInfoHidden || a.detailsChangesHidden || a.detailsDiffHidden {
		t.Fatal("the reset left a details pane hidden")
	}
	if a.historyLine != graphLineStandard || a.historyNode != graphNodeStandard {
		t.Fatalf("reset left the graph at line=%v node=%v", a.historyLine, a.historyNode)
	}
	if a.hideAuthor || a.hideTime {
		t.Fatal("the reset left a history column hidden")
	}
	if a.autoFetchEvery != 0 {
		t.Fatalf("the reset left the fetch interval at %d", a.autoFetchEvery)
	}
	if defaults := defaultSettings(); defaults.Theme != a.themeMode || defaults.Palette != a.palette {
		t.Fatalf("defaults %+v disagree with the reset values", defaults)
	}
}

// The reset restores the look and the Git behavior, but leaves the reader in
// the language they chose; dropping an English reader back into Chinese is
// not what "reset the look" should mean.
func TestResetDefaultsKeepTheLanguage(t *testing.T) {
	got := resetDefaults(persistedSettings{Language: string(localeEnglish)})
	if got.Theme != "system" || got.Palette != themePaletteCatppuccin || !ptrValue(got.ShowGraph) {
		t.Fatalf("reset %+v did not restore the look", got)
	}
	if got.Language != string(localeEnglish) {
		t.Fatalf("language %q, want the current %q", got.Language, localeEnglish)
	}
	// A reset leaves the dividers out of the file, so each page opens at its
	// own default width rather than a width saved against another window size.
	if got.HistorySplit != nil || got.StashSplit != nil || got.BranchDetailsSplit != nil || got.HideDetails != nil || got.FontScale != nil {
		t.Fatalf("reset kept layout %+v; want none", got)
	}
}

// The text size offers three presets, round-trips them through the settings
// file, and refuses a hand-edited value that would shrink the app past the
// smallest preset.
func TestFontScalePresets(t *testing.T) {
	defer setUIFontScale(fontScaleStandard)
	for _, key := range []string{"small", "standard", "large"} {
		if got := fontScaleKey(fontScaleOf(key)); got != key {
			t.Errorf("fontScaleKey(fontScaleOf(%q)) = %q", key, got)
		}
	}
	if fontScaleOf("huge") != fontScaleStandard {
		t.Error("an unknown preset should fall back to standard")
	}
	if fontScaleSetting(fontScaleStandard) != nil {
		t.Error("the standard scale should stay out of the settings file")
	}
	if got := fontScaleSetting(fontScaleLarge); got == nil || *got != fontScaleLarge {
		t.Errorf("large scale setting = %v; want %v", got, fontScaleLarge)
	}
	setUIFontScale(3)
	if got := uiFontScaleValue(); got != fontScaleStandard {
		t.Errorf("out-of-range scale = %v; want standard", got)
	}
	setUIFontScale(fontScaleSmall)
	a := &app{}
	a.applyDefaults()
	if got := uiFontScaleValue(); got != fontScaleStandard {
		t.Errorf("reset left the text scale at %v; want standard", got)
	}
}

func ptrValue(b *bool) bool { return b != nil && *b }

// The interval row round-trips its presets, refuses anything else, and the
// periodic clock only fires when both the switch and the interval are set.
func TestAutoFetchInterval(t *testing.T) {
	for _, minutes := range []int{0, 5, 15, 30, 60} {
		if got := fetchIntervalValue(itoa(minutes)); got != minutes {
			t.Errorf("fetchIntervalValue(%q) = %d; want %d", itoa(minutes), got, minutes)
		}
	}
	if got := fetchIntervalValue("7"); got != 0 {
		t.Errorf("fetchIntervalValue(7) = %d; want off", got)
	}
	if setting := fetchIntervalSetting(0); setting != nil {
		t.Errorf("off interval should stay out of the file, got %v", *setting)
	}
	if setting := fetchIntervalSetting(30); setting == nil || *setting != 30 {
		t.Errorf("30-minute interval = %v; want 30", setting)
	}
	last := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	if !autoFetchDue(true, 5, last, last.Add(5*time.Minute)) {
		t.Error("five minutes after the last fetch, the next one is due")
	}
	if autoFetchDue(true, 5, last, last.Add(4*time.Minute)) {
		t.Error("the interval fired early")
	}
	if autoFetchDue(false, 5, last, last.Add(time.Hour)) {
		t.Error("the clock fired while automatic fetch was off")
	}
	if autoFetchDue(true, 0, last, last.Add(time.Hour)) {
		t.Error("the clock fired with no interval chosen")
	}
}

func TestPersistSettingsWritesTheBranchSplit(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &app{branchSplit: splitDrag{share: 0.28}}
	a.persistSettings()

	path, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	loaded := readSettings(path)
	if loaded.BranchDetailsSplit == nil || *loaded.BranchDetailsSplit != 0.28 {
		t.Fatalf("persisted branch split = %v; want 0.28", loaded.BranchDetailsSplit)
	}

	// A default share is left out of the file so a future default still applies.
	a.branchSplit.share = 0
	a.persistSettings()
	if loaded = readSettings(path); loaded.BranchDetailsSplit != nil {
		t.Fatalf("default branch split persisted as %v; want absent", *loaded.BranchDetailsSplit)
	}
}

// Folding a workspace section is remembered per repository and survives a
// restart: the click saves it, and the next start opens that repository with
// the same sections folded. Another repository starts unfolded, and the reset
// button forgets both the folds and the other repositories' folds.
func TestFoldedWorkspaceSectionsSurviveARestart(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &app{repo: &gitrepo.Repository{Root: "/repos/alpha"}}
	a.toggleFilesSection(filesSectionStaged)

	restarted := &app{themeMode: "system", locale: localeChinese}
	restarted.applySettings(loadSettings())
	folded := restarted.foldedFor("/repos/alpha")
	if !folded[filesSectionStaged] || folded[filesSectionUnstaged] || folded[filesSectionUntracked] {
		t.Fatalf("restarted with %v folded; want only the staged section", folded)
	}
	if other := restarted.foldedFor("/repos/beta"); len(other) != 0 {
		t.Fatalf("another repository opened with %v folded; want nothing", other)
	}

	// Unfolding the last section leaves no entry behind, so the file does not
	// fill up with repositories the reader has since reopened.
	a.toggleFilesSection(filesSectionStaged)
	if loaded := loadSettings(); len(loaded.WorkspaceFolded) != 0 {
		t.Fatalf("an unfolded repository left %+v in the file", loaded.WorkspaceFolded)
	}

	a.toggleFilesSection(filesSectionStaged)
	a.applyDefaults()
	if len(a.filesFolded) != 0 || len(a.filesCollapsed) != 0 {
		t.Fatalf("resetting the layout kept folds %+v / %+v", a.filesFolded, a.filesCollapsed)
	}
}

// The recency list keeps repositories apart: folding one moves it to the
// front without touching the others, an unfolded repository leaves no entry,
// and the list stops at the cap.
func TestFoldedRepositoriesStayApartAndBounded(t *testing.T) {
	list := []foldedRepo{
		{Root: "/repos/alpha", Sections: []string{filesSectionStaged}},
		{Root: "/repos/beta", Sections: []string{filesSectionUntracked}},
	}
	list = withFolds(list, "/repos/beta", []string{filesSectionUnstaged, filesSectionStaged})
	if len(list) != 2 || list[0].Root != "/repos/beta" || !slices.Equal(list[0].Sections, []string{filesSectionUnstaged, filesSectionStaged}) || list[1].Root != "/repos/alpha" {
		t.Fatalf("after folding beta the list is %+v", list)
	}
	list = withFolds(list, "/repos/beta", nil)
	if len(list) != 1 || list[0].Root != "/repos/alpha" {
		t.Fatalf("unfolding beta left %+v", list)
	}
	for i := 0; i < foldedReposMax+5; i++ {
		list = withFolds(list, fmt.Sprintf("/repos/%d", i), []string{filesSectionStaged})
	}
	if len(list) != foldedReposMax {
		t.Fatalf("the list grew to %d entries; want at most %d", len(list), foldedReposMax)
	}
	if newest := fmt.Sprintf("/repos/%d", foldedReposMax+4); list[0].Root != newest {
		t.Fatalf("the newest repository is %q; want %q", list[0].Root, newest)
	}
}

// A hand-edited file cannot fold a section the workspace does not draw, repeat
// a section, or list one repository twice: the first entry for a root wins and
// the sections come back in the order the page draws them.
func TestFoldedSettingsOnlyKeepWhatTheWorkspaceDraws(t *testing.T) {
	list := cleanFolded([]foldedRepo{
		{Root: " /repos/alpha ", Sections: []string{filesSectionUntracked, "conflicted", filesSectionStaged, filesSectionStaged}},
		{Root: "/repos/alpha", Sections: []string{filesSectionUntracked}},
		{Root: "", Sections: []string{filesSectionStaged}},
		{Root: "/repos/beta", Sections: []string{"nope"}},
	})
	want := []foldedRepo{{Root: "/repos/alpha", Sections: []string{filesSectionStaged, filesSectionUntracked}}}
	if !slices.EqualFunc(list, want, func(a, b foldedRepo) bool {
		return a.Root == b.Root && slices.Equal(a.Sections, b.Sections)
	}) {
		t.Fatalf("cleanFolded = %+v; want %+v", list, want)
	}
}
