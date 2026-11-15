package lobedesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A label that opens a dialog ends with an ellipsis; the same words used as a
// dialog title or a confirm button do not. Both forms live in i18n so the
// suffix is written once per language rather than appended at each call site,
// and these pairs are what keeps the two forms from drifting apart.
var dialogLabelPairs = []struct {
	base, action string
}{
	{"mergeIntoCurrent", "mergeIntoCurrentAction"},
	{"rebaseOnto", "rebaseOntoAction"},
	{"pushDialogTitle", "pushBranchAction"},
	{"stashFile", "stashFileAction"},
	{"saveStash", "saveStashAction"},
	{"dropStash", "dropStashAction"},
	{"newBranchTitle", "newBranchAction"},
	{"newBranchNoCheckout", "newBranchNoCheckoutAction"},
	{"forceDeleteBranch", "forceDeleteBranchAction"},
	{"deleteTag", "deleteTagAction"},
	{"resetSettings", "resetSettingsAction"},
	{"forcePush", "forcePushAction"},
}

func TestActionLabelsCarryTheirEllipsisInI18n(t *testing.T) {
	for _, locale := range []locale{localeChinese, localeEnglish} {
		a := &app{locale: locale}
		for _, pair := range dialogLabelPairs {
			base, action := a.t(pair.base), a.t(pair.action)
			if strings.HasSuffix(base, "…") {
				t.Errorf("%s: title %q already ends with an ellipsis", locale, base)
			}
			if !strings.HasSuffix(action, "…") {
				t.Errorf("%s: action %q does not end with an ellipsis", locale, action)
			}
			if strings.TrimSuffix(action, "…") != base {
				t.Errorf("%s: action %q is not %q plus an ellipsis", locale, action, base)
			}
		}
	}
}

// The check above only notices the one spelling it was written for. A call
// site that stashes the suffix in a variable and appends that would slip past
// it, so the strong form bans the literal outright: i18n owns the ellipsis a
// label ends with, and elideMiddle owns the one in the middle of a clipped
// name. Anywhere else, the string came from a call site building its own.
func TestEllipsisLiteralsLiveInI18nOrTheElisionHelper(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"i18n.go": true, "model.go": true}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || allowed[path] {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for n, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if strings.Contains(line, `"…"`) {
				t.Errorf("%s:%d builds an ellipsis outside i18n: %s", path, n+1, trimmed)
			}
		}
	}
}

// The original, narrower check: a menu label must not chain a.t(...) with a
// literal suffix. The scan above is the general rule; this one names the exact
// mistake it was written for, so a failure here points straight at the shape
// that came back.
func TestMenuLabelsNeverBuildTheirOwnEllipsis(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || path == "i18n.go" {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for n, line := range strings.Split(string(src), "\n") {
			if strings.Contains(line, `a.t(`) && strings.Contains(line, `+ "…"`) {
				t.Errorf("%s:%d appends the ellipsis at the call site: %s", path, n+1, strings.TrimSpace(line))
			}
		}
	}
}
