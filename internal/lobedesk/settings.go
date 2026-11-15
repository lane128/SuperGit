package lobedesk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// persistedSettings is the on-disk shape of the settings page. Missing fields
// keep the defaults, so an older or hand-edited file still loads.
type persistedSettings struct {
	Theme          string   `json:"theme,omitempty"`
	Palette        string   `json:"palette,omitempty"`
	Language       string   `json:"language,omitempty"`
	ShowGraph      *bool    `json:"showGraph,omitempty"`
	AutoFetch      *bool    `json:"autoFetch,omitempty"`
	AutoFetchPrune *bool    `json:"autoFetchPrune,omitempty"`
	AutoFetchEvery *int     `json:"autoFetchEvery,omitempty"`
	HistorySplit   *float32 `json:"historySplit,omitempty"`
	StashSplit     *float32 `json:"stashSplit,omitempty"`
	// BranchDetailsSplit is the branch page's list/detail divider, kept apart
	// from the other pages' so each page opens at its own width.
	BranchDetailsSplit *float32 `json:"branchDetailsSplit,omitempty"`
	HideDetails        *bool    `json:"hideHistoryDetails,omitempty"`
	FontScale          *float32 `json:"fontScale,omitempty"`
	// The history page's details panes: the commit's message, its files and
	// its patch are each shown by default and can be hidden from Settings.
	DetailsInfo    *bool `json:"detailsInfo,omitempty"`
	DetailsChanges *bool `json:"detailsChanges,omitempty"`
	DetailsDiff    *bool `json:"detailsDiff,omitempty"`
	// The history list's display preferences. The two presets persist as their
	// names; the columns persist as "hidden" so the defaults stay out of the
	// file.
	HistoryLine string `json:"historyLine,omitempty"`
	HistoryNode string `json:"historyNode,omitempty"`
	HideAuthor  *bool  `json:"hideAuthor,omitempty"`
	HideTime    *bool  `json:"hideTime,omitempty"`
	// WorkspaceFolded remembers which workspace sections are folded, per
	// repository root. A reader who puts a long staged list aside expects it
	// to stay aside after a restart; the recency list keeps the file from
	// growing with every repository ever opened.
	WorkspaceFolded []foldedRepo `json:"workspaceFolded,omitempty"`
}

// foldedRepo is one repository's folded workspace sections. The file keeps a
// short recency list: a save moves the repository to the front and the oldest
// falls off.
type foldedRepo struct {
	Root     string   `json:"root"`
	Sections []string `json:"sections"`
}

// foldedReposMax is how many repositories remember their folds. Twenty covers
// a reader's active projects while keeping the settings file small.
const foldedReposMax = 20

// defaultSettings is what a fresh install runs with. Run applies it at
// start-up and the settings page's reset button restores it, so the two can
// never drift apart.
func defaultSettings() persistedSettings {
	showGraph, autoFetch, prune := true, false, false
	every := 0
	info, changes, diff := true, true, true
	hide := false
	return persistedSettings{
		Theme: "system", Palette: themePaletteCatppuccin, Language: string(localeChinese),
		ShowGraph: &showGraph, AutoFetch: &autoFetch, AutoFetchPrune: &prune, AutoFetchEvery: &every,
		DetailsInfo: &info, DetailsChanges: &changes, DetailsDiff: &diff,
		HistoryLine: "standard", HistoryNode: "standard",
		HideAuthor: &hide, HideTime: &hide,
	}
}

// resetSettings restores the defaults and saves them, after the settings page
// has confirmed with the reader. The language is kept: it is how the reader
// reads the app, not part of the look being reset. Only preferences change;
// repositories and the working tree are untouched.
func (a *app) resetSettings() {
	a.applySettings(resetDefaults(persistedSettings{Language: string(a.locale)}))
	a.resetLayout()
	a.syncLocale()
	// Only claim success once the file is written: persistSettings queues the
	// failure, and a success beside it would replace the warning the reader
	// still has to act on.
	if a.saveSettings() {
		a.queueFor(a.settingsNoticeOwner(), "success", a.t("resetSettingsDone"), "")
	}
	a.invalidate()
}

// resetDefaults is what a reset restores: the fresh-install values with the
// language the reader already chose kept.
func resetDefaults(current persistedSettings) persistedSettings {
	next := defaultSettings()
	next.Language = current.Language
	return next
}

// applyDefaults is the part of the reset that touches no file, so a test can
// check the restored values without writing into the user's config.
func (a *app) applyDefaults() {
	a.applySettings(defaultSettings())
	a.resetLayout()
}

// resetLayout returns every divider to the width its page opens with and
// brings back any pane the reader had collapsed.
func (a *app) resetLayout() {
	a.historySplit.share = 0
	a.stashSplit.share = 0
	a.branchSplit.share = 0
	a.historyDetailsHidden = false
	// The details panes come back.
	a.detailsInfoHidden = false
	a.detailsChangesHidden = false
	a.detailsDiffHidden = false
	// The graph and the history row columns have defaults too.
	a.historyLine = graphLineStandard
	a.historyNode = graphNodeStandard
	a.hideAuthor = false
	a.hideTime = false
	// Folding a workspace section is view state like a divider: the reset
	// unfolds them and forgets what other repositories had folded, and the
	// save that follows writes the empty list back.
	a.filesCollapsed = nil
	a.filesFoldSel = nil
	a.filesFolded = nil
	a.autoFetchEvery = 0
	a.lastAutoFetch = time.Time{}
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "supergit", "settings.json"), nil
}

func loadSettings() persistedSettings {
	path, err := settingsPath()
	if err != nil {
		return persistedSettings{}
	}
	return readSettings(path)
}

// readSettings treats a missing or corrupt file as "no saved choices" instead
// of blocking start-up.
func readSettings(path string) persistedSettings {
	var settings persistedSettings
	data, err := os.ReadFile(path)
	if err != nil {
		return settings
	}
	_ = json.Unmarshal(data, &settings)
	return settings
}

func (a *app) applySettings(settings persistedSettings) {
	switch settings.Theme {
	case "system", "light", "dark":
		a.themeMode = settings.Theme
	}
	for _, t := range appThemes {
		if settings.Palette == t.key {
			a.palette = t.key
			break
		}
	}
	switch settings.Language {
	case string(localeChinese):
		a.locale = localeChinese
	case string(localeEnglish):
		a.locale = localeEnglish
	}
	if settings.ShowGraph != nil {
		a.showGraph = *settings.ShowGraph
	}
	if settings.AutoFetch != nil {
		a.autoFetch = *settings.AutoFetch
	}
	if settings.AutoFetchPrune != nil {
		a.autoFetchPrune = *settings.AutoFetchPrune
	}
	if settings.AutoFetchEvery != nil {
		a.autoFetchEvery = fetchIntervalOf(*settings.AutoFetchEvery)
	}
	if settings.HistorySplit != nil && *settings.HistorySplit > 0 {
		a.historySplit.share = *settings.HistorySplit
	}
	if settings.StashSplit != nil && *settings.StashSplit > 0 {
		a.stashSplit.share = *settings.StashSplit
	}
	if settings.BranchDetailsSplit != nil && *settings.BranchDetailsSplit > 0 {
		a.branchSplit.share = *settings.BranchDetailsSplit
	}
	if settings.HideDetails != nil {
		a.historyDetailsHidden = *settings.HideDetails
	}
	if settings.DetailsInfo != nil {
		a.detailsInfoHidden = !*settings.DetailsInfo
	}
	if settings.DetailsChanges != nil {
		a.detailsChangesHidden = !*settings.DetailsChanges
	}
	if settings.DetailsDiff != nil {
		a.detailsDiffHidden = !*settings.DetailsDiff
	}
	// The files column has nothing to open into without both the Summary and
	// the patch; a settings file that hid both came from the old stacked
	// layout, where that was allowed.
	if a.detailsInfoHidden && a.detailsDiffHidden {
		a.detailsInfoHidden = false
	}
	if settings.HistoryLine != "" {
		a.historyLine = graphLineOf(settings.HistoryLine)
	}
	if settings.HistoryNode != "" {
		a.historyNode = graphNodeOf(settings.HistoryNode)
	}
	if settings.HideAuthor != nil {
		a.hideAuthor = *settings.HideAuthor
	}
	if settings.HideTime != nil {
		a.hideTime = *settings.HideTime
	}
	a.filesFolded = cleanFolded(settings.WorkspaceFolded)
	scale := fontScaleStandard
	if settings.FontScale != nil {
		scale = *settings.FontScale
	}
	setUIFontScale(scale)
}

// writeSettings replaces the file atomically, so a half-written settings file
// is never read back on the next start.
func writeSettings(path string, settings persistedSettings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "settings-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// splitSetting writes a divider share only once the reader has dragged it; a
// zero share is the page's default and stays out of the file, so the setting
// file never pins a default the app may later change.
func splitSetting(share float32) *float32 {
	if share <= 0 {
		return nil
	}
	return &share
}

// foldKeysOf lists the folded sections in the order the workspace draws them,
// so the saved file never depends on map iteration order.
func foldKeysOf(folded map[string]bool) []string {
	var keys []string
	for _, key := range []string{filesSectionUnstaged, filesSectionStaged, filesSectionUntracked} {
		if folded[key] {
			keys = append(keys, key)
		}
	}
	return keys
}

// cleanFolded reads the file's recency list: unknown section keys and entries
// with nothing folded are dropped, the first entry for a root wins, and only
// the newest repositories are kept. A hand-edited file therefore cannot hide
// a section under a name the workspace does not draw.
func cleanFolded(list []foldedRepo) []foldedRepo {
	result := make([]foldedRepo, 0, min(len(list), foldedReposMax))
	seen := make(map[string]bool, len(list))
	for _, entry := range list {
		root := strings.TrimSpace(entry.Root)
		if root == "" || seen[root] {
			continue
		}
		sections := foldKeysOf(sectionSet(entry.Sections))
		if len(sections) == 0 {
			continue
		}
		seen[root] = true
		result = append(result, foldedRepo{Root: root, Sections: sections})
		if len(result) == foldedReposMax {
			break
		}
	}
	return result
}

func sectionSet(keys []string) map[string]bool {
	set := make(map[string]bool, len(keys))
	for _, key := range keys {
		set[key] = true
	}
	return set
}

// foldedFor is the fold set a repository opens with; a repository the file
// does not mention starts unfolded.
func (a *app) foldedFor(root string) map[string]bool {
	for _, entry := range a.filesFolded {
		if entry.Root == root {
			return sectionSet(entry.Sections)
		}
	}
	return nil
}

// rememberFilesFolds moves the current repository's folds to the front of the
// recency list and saves them, so the next start opens the same sections
// folded. It runs on every fold click, like the settings page's own toggles,
// rather than only when the app closes.
func (a *app) rememberFilesFolds() {
	if a.repo == nil || a.repo.Root == "" {
		return
	}
	a.filesFolded = withFolds(a.filesFolded, a.repo.Root, foldKeysOf(a.filesCollapsed))
	a.persistSettings()
}

// withFolds puts root's folds at the front, keeps the other repositories in
// their existing order, drops root's entry when nothing is folded, and trims
// the tail to the recency cap.
func withFolds(list []foldedRepo, root string, sections []string) []foldedRepo {
	result := make([]foldedRepo, 0, min(len(list)+1, foldedReposMax))
	if len(sections) > 0 {
		result = append(result, foldedRepo{Root: root, Sections: sections})
	}
	for _, entry := range list {
		if entry.Root == root {
			continue
		}
		if len(result) == foldedReposMax {
			break
		}
		result = append(result, entry)
	}
	return result
}

// fetchIntervalOf accepts only the presets the settings page offers, so a
// hand-edited file cannot start a fetch every few seconds.
func fetchIntervalOf(minutes int) int {
	switch minutes {
	case 5, 15, 30, 60:
		return minutes
	}
	return 0
}

// fetchIntervalValue and fetchIntervalKey map the settings page's choices to
// the interval. Keeping them here makes the round trip testable and leaves
// the row's string values in one place.
func fetchIntervalValue(key string) int {
	switch key {
	case "5", "15", "30", "60":
		return fetchIntervalOf(atoiString(key))
	}
	return 0
}

// atoiString reads the small non-negative integers the interval row uses.
func atoiString(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// fetchIntervalSetting keeps the default "only on open" out of the file.
func fetchIntervalSetting(minutes int) *int {
	minutes = fetchIntervalOf(minutes)
	if minutes == 0 {
		return nil
	}
	return &minutes
}

// showSetting keeps "block is shown", the fresh-install state, out of the
// file, as hideSetting does for whole panes.
func showSetting(shown bool) *bool {
	if shown {
		return nil
	}
	return &shown
}

// hideSetting keeps "pane is shown", the fresh-install state, out of the file.
func hideSetting(hidden bool) *bool {
	if !hidden {
		return nil
	}
	return &hidden
}

// persistSettings writes the settings page state to the user's config
// directory. Failures stay visible as a toast instead of blocking the change,
// because a settings toggle that works for this run is still useful when the
// file cannot be written.
func (a *app) persistSettings() {
	a.saveSettings()
}

// saveSettings writes the settings and reports whether it succeeded. It
// queues the failure so every caller says why the choice will not survive a
// restart; a caller with its own success message checks the result first.
func (a *app) saveSettings() bool {
	path, err := settingsPath()
	var settings persistedSettings
	if err == nil {
		showGraph := a.showGraph
		autoFetch := a.autoFetch
		autoFetchPrune := a.autoFetchPrune
		settings = persistedSettings{
			Theme: a.themeMode, Palette: a.palette, Language: string(a.locale),
			ShowGraph: &showGraph, AutoFetch: &autoFetch, AutoFetchPrune: &autoFetchPrune,
			AutoFetchEvery:     fetchIntervalSetting(a.autoFetchEvery),
			HistorySplit:       splitSetting(a.historySplit.share),
			StashSplit:         splitSetting(a.stashSplit.share),
			BranchDetailsSplit: splitSetting(a.branchSplit.share),
			HideDetails:        hideSetting(a.historyDetailsHidden),
			FontScale:          fontScaleSetting(uiFontScaleValue()),
			DetailsInfo:        showSetting(!a.detailsInfoHidden),
			DetailsChanges:     showSetting(!a.detailsChangesHidden),
			DetailsDiff:        showSetting(!a.detailsDiffHidden),
			HistoryLine:        graphLineKey(a.historyLine),
			HistoryNode:        graphNodeKey(a.historyNode),
			HideAuthor:         hideSetting(a.hideAuthor),
			HideTime:           hideSetting(a.hideTime),
			WorkspaceFolded:    a.filesFolded,
		}
		err = writeSettings(path, settings)
	}
	if err != nil {
		a.queueFor(a.settingsNoticeOwner(), "warning", a.t("settingsSaveFailed"), err.Error())
		a.invalidate()
		return false
	}
	return true
}
