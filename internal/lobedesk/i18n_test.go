package lobedesk

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every key has to exist in both catalogs: a missing translation silently
// falls back to nothing, so a page would show an empty label instead of a
// word. The check runs both ways so an English-only string cannot slip in.
func TestCatalogsCarryTheSameKeys(t *testing.T) {
	zh, en := messages[localeChinese], messages[localeEnglish]
	for key := range zh {
		if _, ok := en[key]; !ok {
			t.Errorf("en is missing %q", key)
		}
	}
	for key := range en {
		if _, ok := zh[key]; !ok {
			t.Errorf("zh is missing %q", key)
		}
	}
	for loc, catalog := range messages {
		for key, value := range catalog {
			if strings.TrimSpace(value) == "" {
				t.Errorf("%s: %q is empty", loc, key)
			}
		}
	}
}

// Loading messages name what is being waited on and trail off, so the label
// never reads as a finished statement or as "Loading…" with no object.
func TestLoadingMessagesSayWhatIsLoading(t *testing.T) {
	for loc, catalog := range messages {
		for key, value := range catalog {
			if !strings.HasPrefix(key, "loading") {
				continue
			}
			if !strings.HasSuffix(value, "…") {
				t.Errorf("%s: %q = %q does not end with an ellipsis", loc, key, value)
			}
		}
	}
}

// The Chinese catalog picks one verb for creating something. Branch and
// worktree commands used to say 新建 while tags and every success toast said
// 创建, so the same action read two ways on two pages; this keeps the choice
// from drifting back.
func TestChineseUsesOneVerbForCreating(t *testing.T) {
	for key, value := range messages[localeChinese] {
		if strings.Contains(value, "新建") {
			t.Errorf("zh: %q = %q uses 新建; the catalog says 创建 everywhere else", key, value)
		}
	}
}

// A Chinese setting or message should not fall back to the English Git term:
// a reader who picked 中文 should never have to know that fetching is called
// fetch, or pruning prune, in the command line.
func TestChineseHasNoEnglishGitTerms(t *testing.T) {
	for key, value := range messages[localeChinese] {
		for _, term := range []string{"fetch", "prune"} {
			if strings.Contains(strings.ToLower(value), term) {
				t.Errorf("zh: %q = %q keeps the English term %q", key, value, term)
			}
		}
	}
}

// Explain which settings affect list density and which affect code in patches.
func TestAppearanceHelpNamesListDensityAndCodeSize(t *testing.T) {
	zh := messages[localeChinese]["appearanceHelp"]
	for _, want := range []string{"界面字号", "行高", "文件列表", "代码字号", "差异行距", "冲突编辑器"} {
		if !strings.Contains(zh, want) {
			t.Errorf("zh appearanceHelp = %q does not mention %q", zh, want)
		}
	}
	en := messages[localeEnglish]["appearanceHelp"]
	for _, want := range []string{"Text size", "file rows", "Code font size", "line spacing", "conflict editor"} {
		if !strings.Contains(en, want) {
			t.Errorf("en appearanceHelp = %q does not mention %q", en, want)
		}
	}
}

// A failure message says the same verb as the loading label it replaces, so
// the wait and its failure read as one event rather than "loading" turning
// into "reading".
func TestLoadFailuresAgreeWithTheirLoadingText(t *testing.T) {
	zh := messages[localeChinese]
	for _, key := range []string{"diffLoading", "loadingRepo", "loadingMoreHistory", "diffFailed", "resetUpstreamFailed"} {
		if !strings.Contains(zh[key], "加载") || strings.Contains(zh[key], "读取") {
			t.Errorf("zh: %q = %q does not use 加载", key, zh[key])
		}
	}
}

// A failed load reads the same way wherever it happens. The history page once
// said "Failed to load history" while the diff and upstream failures said
// "Could not load …", so the same kind of error arrived in two shapes.
func TestLoadFailuresStartWithCouldNot(t *testing.T) {
	for key, english := range messages[localeEnglish] {
		if !strings.HasSuffix(key, "Failed") || !strings.Contains(strings.ToLower(english), "load") {
			continue
		}
		if !strings.HasPrefix(english, "Could not load") {
			t.Errorf("%q reads %q; a load failure should start with \"Could not load\"", key, english)
		}
	}
}

// The busy labels are the second place a key reaches t() through a variable:
// the title bar looks up whatever label the operation passed. Scanning the
// declarations keeps a renamed or forgotten key from showing up as
// "busyRebase" in the title bar while a rebase runs.
func TestEveryBusyLabelHasText(t *testing.T) {
	src, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatalf("read app.go: %v", err)
	}
	label := regexp.MustCompile(`(busy[A-Z][A-Za-z]*)\s+busyLabel\s*=\s*"([a-zA-Z]+)"`)
	found := label.FindAllStringSubmatch(string(src), -1)
	if len(found) < 10 {
		t.Fatalf("found %d busy labels in app.go; the scan is broken", len(found))
	}
	for _, match := range found {
		for _, loc := range []locale{localeChinese, localeEnglish} {
			if _, ok := messages[loc][match[2]]; !ok {
				t.Errorf("%s is missing the text %q of %s", loc, match[2], match[1])
			}
		}
	}
}

// An empty state names where the control that fills it lives. Tags are listed
// under Branches but created from a commit's context menu in History, so that
// hint has to name the page; worktrees and stashes are created by the button
// above their own lists; an empty history sends the reader to the workspace to
// commit. The test pins the pages so moving a control makes someone revisit the
// sentence rather than leaving a hint that points nowhere.
func TestEmptyStateHintsPointAtTheRightPlace(t *testing.T) {
	zh, en := messages[localeChinese], messages[localeEnglish]
	if !strings.Contains(zh["noTagsHint"], "历史") {
		t.Errorf("zh noTagsHint = %q does not say the commit menu lives in History", zh["noTagsHint"])
	}
	if !strings.Contains(en["noTagsHint"], "History") {
		t.Errorf("en noTagsHint = %q does not say the commit menu lives in History", en["noTagsHint"])
	}
	if !strings.Contains(zh["noHistoryHint"], "历史") {
		t.Errorf("zh noHistoryHint = %q does not send the reader to history", zh["noHistoryHint"])
	}
	if !strings.Contains(en["noHistoryHint"], "history") {
		t.Errorf("en noHistoryHint = %q does not send the reader to history", en["noHistoryHint"])
	}
	for _, test := range []struct {
		key, zhMarker, enMarker string
	}{
		{"noWorktreesHint", "右上角", "above"},
		{"noStashesHint", "分支", "branch"},
	} {
		if !strings.Contains(zh[test.key], test.zhMarker) {
			t.Errorf("zh %s = %q no longer points at the top-right button", test.key, zh[test.key])
		}
		if !strings.Contains(en[test.key], test.enMarker) {
			t.Errorf("en %s = %q no longer points at the button above", test.key, en[test.key])
		}
	}
}

// The palette names are the one place a key reaches t() through a table
// instead of a literal, so the call-site scan cannot see them. Every scheme
// listed for the reader must carry both names, or the dropdown silently shows
// "paletteCatppuccinFrappe" where a color scheme's name belongs.
func TestEveryPaletteHasItsName(t *testing.T) {
	for _, th := range appThemes {
		for _, loc := range []locale{localeChinese, localeEnglish} {
			if _, ok := messages[loc][th.label]; !ok {
				t.Errorf("%s is missing the name %q of palette %q", loc, th.label, th.key)
			}
		}
	}
}

// Every key the code asks for must exist in the catalogs. A renamed key left
// behind in a call site is not a compile error: t() returns the key itself, so
// the button quietly shows "push". This scans the call sites so the rename is
// caught by the tests rather than by a screenshot.
func TestEveryKeyUsedByTheCodeExists(t *testing.T) {
	call := regexp.MustCompile(`a\.tf?\("([a-zA-Z][a-zA-Z0-9]*)"`)
	zh, en := messages[localeChinese], messages[localeEnglish]
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || path == "i18n.go" {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			key := m[1]
			if seen[key] {
				continue
			}
			seen[key] = true
			if _, ok := zh[key]; !ok {
				t.Errorf("%s: key %q is not in the Chinese catalog", path, key)
			}
			if _, ok := en[key]; !ok {
				t.Errorf("%s: key %q is not in the English catalog", path, key)
			}
		}
	}
	if len(seen) < 100 {
		t.Fatalf("only %d keys scanned; the pattern probably stopped matching", len(seen))
	}
}

// The Chinese interface keeps a fixed set of words in English. The list and
// its reasoning live in chineseEnglishTerms and docs/git-terms.md; this guard
// fails on any other Latin word, so adding one is a decision rather than a
// translation someone forgot.
func TestChineseStringsOnlyKeepTheListedEnglishTerms(t *testing.T) {
	allowed := make(map[string]bool, len(chineseEnglishTerms))
	for _, term := range chineseEnglishTerms {
		allowed[term] = true
	}
	// .gitmodules starts with a dot, so the general “Latin word” pattern would
	// not see it whole.
	word := regexp.MustCompile(`\.gitmodules|[A-Za-z][A-Za-z0-9._-]+`)
	for key, text := range messages[localeChinese] {
		for _, found := range word.FindAllString(text, -1) {
			if !allowed[found] {
				t.Errorf("zh %s keeps %q in English; add it to chineseEnglishTerms and docs/git-terms.md, or translate it", key, found)
			}
		}
	}
}

// The app calls the other side of a clone 「远端」 everywhere (docs/git-terms.md
// fixes the term); 「远程」 was left in one group name and a couple of
// sentences, so the same thing had two names in the interface.
func TestChineseCopyCallsRemotesYuanDuan(t *testing.T) {
	for key, text := range messages[localeChinese] {
		if strings.Contains(text, "远程") {
			t.Errorf("zh %s says 远程 in %q; the interface calls it 远端", key, text)
		}
	}
}
