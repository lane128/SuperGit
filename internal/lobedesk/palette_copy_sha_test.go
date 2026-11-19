package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

const paletteCopySHASHA = "0123456789abcdef0123456789abcdef01234567"

// copySHAPaletteFixture is a History selection with a commit that can be
// resolved, and a clipboard that records instead of writing the system's.
func copySHAPaletteFixture(t *testing.T) (*app, *[]string) {
	t.Helper()
	a := &app{
		page: "history", locale: localeEnglish,
		repo:        &gitrepo.Repository{Root: t.TempDir()},
		commits:     []gitrepo.Commit{{SHA: paletteCopySHASHA, Subject: "fixture"}},
		selectedSHA: paletteCopySHASHA,
	}
	var copied []string
	original := writeClipboard
	writeClipboard = func(value string) error {
		copied = append(copied, value)
		return nil
	}
	t.Cleanup(func() { writeClipboard = original })
	return a, &copied
}

// The Summary pane's Copy SHA was reachable only by aiming at a small icon.
// The palette names the same action, so searching for it finds a command.
func TestThePaletteOffersCopySHAForTheSelectedCommit(t *testing.T) {
	a, copied := copySHAPaletteFixture(t)
	a.openCommandPalette()
	a.commandPalette.query = "copy sha"
	entries := a.commandPaletteEntries()
	var entry appShortcut
	found := false
	for _, s := range entries {
		if s.ID == "commit.copySHA" {
			entry, found = s, true
		}
	}
	if !found {
		t.Fatalf("searching copy sha found no command: %+v", entries)
	}
	if reason := a.commandPaletteUnavailable(entry.ID); reason != "" {
		t.Fatalf("Copy SHA is unavailable with a selected commit: %q", reason)
	}
	generation := a.commandPalette.generation
	a.runPaletteCommand(entry.ID, generation)
	if len(*copied) != 1 || (*copied)[0] != paletteCopySHASHA {
		t.Fatalf("Copy SHA copied %v; want the full commit hash", *copied)
	}
	if a.commandPalette.open {
		t.Fatal("running the command left the palette open")
	}
}

// Without a commit, the row stays listed and says why it cannot run, rather
// than silently copying nothing.
func TestCopySHANeedsASelectedCommit(t *testing.T) {
	a, copied := copySHAPaletteFixture(t)
	a.selectedSHA = ""
	a.openCommandPalette()
	a.commandPalette.query = "copy sha"
	entry := appShortcut{}
	for _, s := range a.commandPaletteEntries() {
		if s.ID == "commit.copySHA" {
			entry = s
		}
	}
	if entry.ID == "" {
		t.Fatal("Copy SHA disappeared from the palette without a selection")
	}
	if reason := a.commandPaletteUnavailable(entry.ID); reason == "" {
		t.Fatal("Copy SHA reports no reason without a selected commit")
	}
	entry.Run(a)
	if len(*copied) != 0 {
		t.Fatalf("Copy SHA copied without a commit: %v", *copied)
	}
}

// The palette row must use the same label as the Summary button, so the two
// names cannot drift apart in either language.
func TestThePaletteCopySHARowIsNamedAndRendered(t *testing.T) {
	for _, language := range []locale{localeEnglish, localeChinese} {
		t.Run(string(language), func(t *testing.T) {
			a, _ := copySHAPaletteFixture(t)
			a.locale = language
			a.openCommandPalette()
			a.commandPalette.query = "copy"
			tt := paletteTester(a, false, 600, 500)
			row, ok := tt.AccessByRole(ui.RoleOption, a.t("copySHA"))
			if !ok {
				t.Fatalf("the palette does not render a %q row", a.t("copySHA"))
			}
			if row.Disabled {
				t.Fatalf("the %q row is disabled with a selected commit", a.t("copySHA"))
			}
		})
	}
}
