package lobedesk

import (
	"bytes"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

// The status letter is a bundled face, not the platform's UI font, so the
// same glyphs ship on every machine instead of following System/Inter/Geist.
func TestStatusMarkerFontIsBundled(t *testing.T) {
	ensureStatusMarkerFont()
	for _, family := range ui.FontFamilies() {
		if family.Name != statusMarkerFontFamily {
			continue
		}
		if !family.Registered {
			t.Fatalf("%q is listed but not registered by the app", statusMarkerFontFamily)
		}
		return
	}
	t.Fatalf("%q is not a registered family", statusMarkerFontFamily)
}

// A status letter must not change when the code font setting does.
func TestStatusMarkerIgnoresTheCodeFontSetting(t *testing.T) {
	render := func(key string) []byte {
		previous := codeFontKeyValue()
		setCodeFontKey(key)
		defer setCodeFontKey(previous)
		entry := gitrepo.Entry{Path: "changed.go", Worktree: 'M'}
		a := &app{locale: localeEnglish, worktreeSelected: true, status: gitrepo.Status{Entries: []gitrepo.Entry{entry}}}
		view := pageTester(a, func(c *ui.Context) {
			ui.Column(c).Width(320).Height(60).Children(func() { a.fileRow(c, entry, false) })
		})
		view.Frame()
		return append([]byte(nil), view.Image().Pix...)
	}
	if !bytes.Equal(render(codeFontJetBrainsMono), render(codeFontFiraCode)) {
		t.Fatal("the status marker changed with the code font setting")
	}
}

func TestDetailTabStatusBadgeIgnoresTheCodeFontSetting(t *testing.T) {
	render := func(fontKey string) []byte {
		previous := codeFontKeyValue()
		setCodeFontKey(fontKey)
		defer setCodeFontKey(previous)
		entry := gitrepo.Entry{Path: "changed.go", Worktree: 'M'}
		a := &app{locale: localeEnglish, worktreeSelected: true, status: gitrepo.Status{Entries: []gitrepo.Entry{entry}}}
		tabKey := detailTabKey{File: detailFileKey{Path: entry.Path}}
		a.detailTabs().Open(tabKey.File)
		view := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			tk := theme.TokensOf(c)
			ui.Row(c).Height(36).Background(tk.BgLayout).Children(func() { a.detailTabButton(c, tk, tabKey) })
		}, 320, 60)
		view.Frame()
		return append([]byte(nil), view.Image().Pix...)
	}
	if !bytes.Equal(render(codeFontJetBrainsMono), render(codeFontFiraCode)) {
		t.Fatal("the detail tab status badge changed with the code font setting")
	}
}

func TestFileRowsShowLeadingStatusMarkers(t *testing.T) {
	cases := []struct {
		entry  gitrepo.Entry
		staged bool
		code   string
	}{
		{gitrepo.Entry{Path: "added.go", Index: 'A', Worktree: ' '}, true, "A"},
		{gitrepo.Entry{Path: "modified.go", Worktree: 'M'}, false, "M"},
		{gitrepo.Entry{Path: "gone.go", Worktree: 'D'}, false, "D"},
		{gitrepo.Entry{Path: "new.go", Worktree: '?'}, false, "?"},
		{gitrepo.Entry{Path: "renamed.go", Index: 'R', Worktree: ' '}, true, "R"},
		{gitrepo.Entry{Path: "merge.go", Worktree: 'U', Conflict: true}, false, "!"},
		{gitrepo.Entry{Path: "vendor/lib", Index: 'M', Worktree: ' ', Submodule: "S.M."}, true, "S"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			a := &app{locale: localeEnglish, worktreeSelected: true, status: gitrepo.Status{Entries: []gitrepo.Entry{tc.entry}}}
			view := pageTester(a, func(c *ui.Context) {
				ui.Column(c).Width(320).Height(60).Children(func() { a.fileRow(c, tc.entry, tc.staged) })
			})
			view.Frame()
			row, ok := view.AccessByRole(ui.RoleOption, tc.entry.Path)
			if !ok {
				t.Fatal("missing file row")
			}
			badge, ok := view.AccessByRole(ui.RoleImage, a.detailStatusMeaning(tc.code))
			if !ok {
				t.Fatalf("missing %q marker", tc.code)
			}
			if badge.Bounds.X < row.Bounds.X || badge.Bounds.X > row.Bounds.X+8 ||
				badge.Bounds.X+badge.Bounds.W > row.Bounds.X+row.Bounds.W ||
				badge.Bounds.Y < row.Bounds.Y || badge.Bounds.Y+badge.Bounds.H > row.Bounds.Y+row.Bounds.H {
				t.Fatalf("marker %+v sits outside row %+v", badge.Bounds, row.Bounds)
			}
			if d := badge.Bounds.W - scaledFont(18); d > .5 || d < -.5 {
				t.Errorf("marker width %.2f; want %.2f", badge.Bounds.W, scaledFont(18))
			}
			if d := row.Bounds.H - fileRowHeight(); d > .5 || d < -.5 {
				t.Errorf("row height %.2f; want %.2f", row.Bounds.H, fileRowHeight())
			}
		})
	}
}

// A conflict keeps its ! marker and still has no stage control.
func TestConflictRowKeepsMarkerWithoutStageControl(t *testing.T) {
	entry := gitrepo.Entry{Path: "merge.go", Worktree: 'U', Conflict: true}
	a := &app{locale: localeEnglish, worktreeSelected: true, status: gitrepo.Status{Entries: []gitrepo.Entry{entry}}}
	view := pageTester(a, func(c *ui.Context) {
		ui.Column(c).Width(320).Height(60).Children(func() { a.fileRow(c, entry, false) })
	})
	view.Frame()
	if _, ok := view.AccessByRole(ui.RoleImage, a.detailStatusMeaning("!")); !ok {
		t.Fatal("conflict row lost its marker")
	}
	for _, info := range view.AccessAll() {
		if info.Role == ui.RoleButton && (info.Name == a.t("stageFile") || info.Name == a.t("unstageFile")) {
			t.Fatalf("conflict row still offers %q", info.Name)
		}
	}
}

func TestStatusMarkerInkMeetsContrast(t *testing.T) {
	for _, palette := range []string{themePaletteSuperGit, themePaletteDefault, themePaletteCatppuccin, themePaletteCatppuccinFrappe, themePaletteCatppuccinMacchiato} {
		for _, dark := range []bool{false, true} {
			tk := themeConfig(palette).Tokens(dark)
			for _, code := range []string{"A", "M", "D", "?", "R", "C", "T", "!", "S"} {
				ink := statusMarkerInk(tk, code)
				minimum := 4.5
				switch code {
				case "A", "M", "D", "T", "!":
					minimum = 7
				}
				grounds := []ui.Color{tk.BgContainer, tk.BgLayout, selectionBg(tk, tk.BgContainer)}
				for _, ground := range grounds {
					for _, bg := range []ui.Color{ground, tk.FillTertiary.Over(ground), tk.Fill.Over(ground)} {
						if ratio := contrastRatio(ink, bg); ratio < minimum {
							t.Errorf("%s/%t/%s ink %.2f < %.1f", palette, dark, code, ratio, minimum)
						}
					}
				}
			}
		}
	}
}

func TestStatusMarkerFitsNarrowRows(t *testing.T) {
	for _, width := range []float32{130, 110, 96} {
		entry := gitrepo.Entry{Path: "internal/lobedesk/commit_details_select_test.go", Worktree: 'M'}
		a := &app{locale: localeEnglish, worktreeSelected: true, status: gitrepo.Status{Entries: []gitrepo.Entry{entry}}}
		view := pageTester(a, func(c *ui.Context) {
			ui.Column(c).Width(width).Height(60).Children(func() { a.fileRow(c, entry, false) })
		})
		view.Frame()
		row, ok := view.AccessByRole(ui.RoleOption, entry.Path)
		if !ok {
			t.Fatalf("%.0f: missing row", width)
		}
		badge, ok := view.AccessByRole(ui.RoleImage, a.detailStatusMeaning("M"))
		if !ok {
			t.Fatalf("%.0f: missing marker", width)
		}
		stage, ok := view.AccessByRole(ui.RoleButton, a.t("stageFile"))
		if !ok {
			t.Fatalf("%.0f: missing stage control", width)
		}
		if badge.Bounds.X+badge.Bounds.W > stage.Bounds.X {
			t.Errorf("%.0f: marker %+v overlaps stage control %+v", width, badge.Bounds, stage.Bounds)
		}
		if stage.Bounds.X+stage.Bounds.W > row.Bounds.X+row.Bounds.W+.5 {
			t.Errorf("%.0f: stage control leaves its row: %+v / %+v", width, stage.Bounds, row.Bounds)
		}
	}
}

func TestHistoryAndComparisonRowsShowStatusMarkers(t *testing.T) {
	files := []gitrepo.ChangedFile{{Path: "added.go", Status: "A"}, {Path: "modified.go", Status: "M"}}
	a := &app{locale: localeEnglish}
	view := pageTester(a, func(c *ui.Context) {
		ui.Column(c).Width(320).Height(200).Children(func() { a.commitFileRows(c, files) })
	})
	view.Frame()
	view.Frame()
	for _, want := range []string{"A", "M"} {
		if _, ok := view.AccessByRole(ui.RoleImage, a.detailStatusMeaning(want)); !ok {
			t.Errorf("commit rows are missing the %q marker", want)
		}
	}
	comparison := &comparisonView{files: files}
	compare := pageTester(a, func(c *ui.Context) {
		ui.Column(c).Width(320).Height(200).Children(func() { a.comparisonFileList(c, comparison) })
	})
	compare.Frame()
	compare.Frame()
	for _, want := range []string{"A", "M"} {
		if _, ok := compare.AccessByRole(ui.RoleImage, a.detailStatusMeaning(want)); !ok {
			t.Errorf("comparison rows are missing the %q marker", want)
		}
	}
}

func TestStatusMarkerDoesNotChangeTheRowMeaning(t *testing.T) {
	entry := gitrepo.Entry{Path: "changed.go", Worktree: 'M'}
	a := &app{locale: localeEnglish, worktreeSelected: true, status: gitrepo.Status{Entries: []gitrepo.Entry{entry}}}
	view := pageTester(a, func(c *ui.Context) {
		ui.Column(c).Width(320).Height(60).Children(func() {
			row := a.fileRow(c, entry, false)
			row.Description(a.t("fileGroupModified")).Tooltip(entry.Path)
		})
	})
	view.Frame()
	row, ok := view.AccessByRole(ui.RoleOption, entry.Path)
	if !ok || row.Description != a.t("fileGroupModified") {
		t.Fatalf("row description changed: %+v", row)
	}
}
