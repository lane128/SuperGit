package lobedesk

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

// The settings card and the window have to agree on the keys, so every
// window-level binding comes from shortcutTable. A stray OnWindowShortcut in
// a page would work but stay missing from the card; this reads the package's
// sources and fails on one.
func TestEveryWindowShortcutComesFromTheTable(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	registrars := []string{}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("OnWindowShortcut(")) {
			registrars = append(registrars, name)
		}
	}
	if len(registrars) != 1 || registrars[0] != "shortcuts.go" {
		t.Fatalf("window keys are registered in %v; want only shortcuts.go", registrars)
	}
}

// Every row of the table has to be usable: an id of its own, a key to press, a
// wording in both languages, a group the card draws, and an action unless a
// text field owns the key. Two window bindings on one key would make the
// second unreachable, since the runtime hands a key to one handler.
func TestEveryShortcutIsListedAndRegistered(t *testing.T) {
	ids := map[string]bool{}
	bound := map[string]string{}
	for _, s := range shortcutTable() {
		if s.ID == "" || ids[s.ID] {
			t.Errorf("shortcut %q has an empty or repeated id", s.ID)
		}
		ids[s.ID] = true
		if s.Keys == "" {
			t.Errorf("%s has no key text for the settings card", s.ID)
		}
		if !containsString(shortcutGroupOrder, s.Group) {
			t.Errorf("%s is in group %q, which the card does not draw", s.ID, s.Group)
		}
		for _, loc := range []locale{localeChinese, localeEnglish} {
			if messages[loc][s.Label] == "" {
				t.Errorf("%s has no %q wording in %s", s.ID, s.Label, loc)
			}
		}
		switch {
		case s.Field && s.Run != nil:
			t.Errorf("%s is a field binding but also has a window action", s.ID)
		case !s.Field && s.Run == nil:
			t.Errorf("%s is registered as a window key but has no action", s.ID)
		}
		if s.Field {
			continue
		}
		sig := fmt.Sprintf("%d/%d", s.Mod, s.Key)
		if other, ok := bound[sig]; ok {
			t.Errorf("%s binds the same key as %s; one of them can never run", s.ID, other)
		}
		bound[sig] = s.ID
	}
}

func containsString(list []string, want string) bool {
	for _, value := range list {
		if value == want {
			return true
		}
	}
	return false
}

// The card reads the table, so every action and key it declares is on the
// settings page, under a heading for its group and with the page names
// heading the page groups.
func TestTheSettingsCardListsEveryShortcut(t *testing.T) {
	a := &app{locale: localeChinese}
	tt := ui.NewTester(func(c *ui.Context) {
		a.settingsPage(c)
	}, 1000, 2000)
	tt.Frame()
	for _, s := range shortcutTable() {
		if !tt.HasText(a.t(s.Label)) {
			t.Errorf("the card does not list %s (%q)", s.ID, a.t(s.Label))
		}
		if !tt.HasText(s.Keys) {
			t.Errorf("the card does not show the %s key %q", s.ID, s.Keys)
		}
	}
	for _, group := range shortcutGroupOrder {
		listed := false
		for _, s := range shortcutTable() {
			listed = listed || s.Group == group
		}
		if listed && !tt.HasText(a.t(shortcutGroupLabel(group))) {
			t.Errorf("the card has no heading for the %s group", group)
		}
	}
}

// The working tree's patch lives on the history page now, and its line
// selection keeps the copy key: ⌘C appears only while a line is selected in
// the patch in front, as it does for a commit's patch.
func TestCopyLinesIsAvailableForTheWorkingTreePatch(t *testing.T) {
	a := &app{
		page: "history", worktreeSelected: true, selectedPath: "a.go", detailTab: detailTabFile,
		lineSel: map[int]bool{0: true},
	}
	for _, s := range shortcutTable() {
		if s.ID != "diff.copyLines" {
			continue
		}
		if s.Available == nil || !s.Available(a) {
			t.Fatal("⌘C is not available while the working tree's patch has a line selection")
		}
		a.detailTab = detailTabSummary
		if s.Available(a) {
			t.Error("⌘C is available on the working tree's summary tab")
		}
		return
	}
	t.Fatal("the table has no copy binding")
}

// The dispatcher still reaches the table's actions: the page keys switch the
// page, so moving the bindings into the table did not cut them off.
func TestTheTableStillDispatchesThePageKeys(t *testing.T) {
	a := &app{repo: &gitrepo.Repository{Root: "/repos/alpha"}, page: "stash", locale: localeChinese}
	tt := ui.NewTester(func(c *ui.Context) {
		a.registerShortcuts(c)
		ui.Column(c).Fill()
	}, 400, 300)
	tt.Key(ui.Cmd, ui.Key1)
	if a.page != "history" {
		t.Fatalf("⌘1 left the page %q; want history", a.page)
	}
	tt.Key(ui.Cmd, ui.Key2)
	if a.page != "branches" {
		t.Fatalf("⌘2 left the page %q; want branches", a.page)
	}
}
