package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func TestCommitDiffHunkHeaderFitsNarrowPaneAndCopiesHunk(t *testing.T) {
	const header = "@@ -1 +1 @@"
	patch := gitrepo.Diff{Lines: []gitrepo.DiffLine{
		{Kind: "hunk", Text: header},
		{Kind: "del", Text: "-before", Old: 1},
		{Kind: "add", Text: "+after", New: 1},
	}}
	a := &app{locale: localeEnglish, page: "history", selectedSHA: "selected-commit"}
	a.commitDiff, a.commitSections = buildCommitDiff([]gitrepo.FileDiff{{Path: "sample.txt", Diff: patch}})
	const narrow, wide = 300, 420
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Fill().Children(func() { a.commitDiffView(c) })
	}, narrow, 200)

	var bounds ui.Rect
	for _, width := range []int{narrow, wide, narrow} {
		tt.SetSize(width, 200)
		tt.Frame()
		row, found := tt.AccessByRole(ui.RoleButton, header)
		if !found {
			t.Fatal("missing History hunk header")
		}
		bounds = row.Bounds
		// The button and its text share a label, so Find would return the
		// button's bounds even when the glyphs are clipped.
		img := tt.Image()
		left, right := width, 0
		for y := int(bounds.Y) + 3; y < int(bounds.Y+bounds.H)-3; y++ {
			ground := img.RGBAAt(2, y)
			for x := 4; x < width-4; x++ {
				if img.RGBAAt(x, y) != ground {
					left, right = min(left, x), max(right, x)
				}
			}
		}
		if left > 32 || right-left < 45 || right > 110 {
			t.Fatalf("hunk title ink at width %d spans x=%d..%d, want complete text in the leading part of the pane", width, left, right)
		}
	}

	copied := ""
	restore := writeClipboard
	writeClipboard = func(value string) error { copied = value; return nil }
	t.Cleanup(func() { writeClipboard = restore })
	tt.RightClickAt(bounds.X+bounds.W/2, bounds.Y+bounds.H/2)
	item, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("copyHunk"))
	if !ok {
		t.Fatal("missing Copy Hunk on the History header")
	}
	tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
	if want := header + "\n-before\n+after"; copied != want {
		t.Errorf("copied hunk = %q, want %q", copied, want)
	}
}

// File-history actions can replace the live patch during the current frame.
func TestCommitDiffFileHistoryMenuCanReplacePatchDuringRender(t *testing.T) {
	for _, singleFile := range []bool{false, true} {
		for _, header := range []bool{true, false} {
			name := "body"
			if header {
				name = "header"
			}
			if singleFile {
				name = "file-tab/" + name
			} else {
				name = "summary/" + name
			}
			t.Run(name, func(t *testing.T) {
				a := &app{ctx: canceledContext(), runOnMain: func(func()) {}, repo: &gitrepo.Repository{Root: t.TempDir()}, locale: localeEnglish, page: "history", selectedSHA: "selected-commit"}
				t.Cleanup(func() { a.resetFileHistory() })
				patch := gitrepo.Diff{Lines: []gitrepo.DiffLine{{Kind: "hunk", Text: "@@ -0,0 +1,2 @@"}, {Kind: "add", Text: "+first payload", New: 1}, {Kind: "add", Text: "+second payload", New: 2}}}
				a.commitDiff, a.commitSections = buildCommitDiff([]gitrepo.FileDiff{{Path: "first.txt", Diff: patch}, {Path: "second.txt", Diff: patch}})
				if singleFile {
					a.detailTabs().Open(detailFileKey{Path: "first.txt"})
				}
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					ui.Column(c).Width(700).Height(500).Children(func() { a.commitDiffView(c) })
				}, 700, 500)
				var bounds ui.Rect
				if header {
					row, ok := tt.AccessByRole(ui.RoleButton, "first.txt")
					if !ok {
						t.Fatal("missing patch header")
					}
					bounds = row.Bounds
				} else {
					row, ok := tt.Find("+first payload")
					if !ok {
						t.Fatal("missing first patch line")
					}
					bounds = row
				}
				x := bounds.X + bounds.W/2
				if !header {
					x = a.diffWidth(a.commitDiff) - 8
				} // Use row space, outside selectable text's own context menu.
				tt.RightClickAt(x, bounds.Y+bounds.H/2)
				item, ok := tt.AccessByRole(ui.RoleMenuItem, a.t("fileHistory"))
				if !ok {
					t.Fatalf("missing file history action at %+v; visible text: %v", bounds, tt.Texts())
				}
				tt.ClickAt(item.Bounds.X+item.Bounds.W/2, item.Bounds.Y+item.Bounds.H/2)
				tt.Frame()
				if a.fileFilter != "first.txt" || len(a.commitDiff.Lines) != 0 {
					t.Fatal("patch menu did not switch to its own file history")
				}
			})
		}
	}
}

// A diff row's menu describes the file its section belongs to. The details
// pane highlights one file, but a row of another file's patch can be
// right-clicked without choosing that file first; copy path, history and
// blame must follow the row, not the highlight.
func TestCommitDiffLineMenuUsesTheSectionsPath(t *testing.T) {
	copied := ""
	restore := writeClipboard
	writeClipboard = func(value string) error {
		copied = value
		return nil
	}
	defer func() { writeClipboard = restore }()

	a := &app{locale: localeChinese, page: "history", commitPath: "other.go", selectedSHA: "aaaa"}
	line := gitrepo.DiffLine{Kind: "add", Text: "package section", New: 1}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(600).Height(400).Children(func() {
			a.commitDiffLine(c, line, "dir/section.go", 600, 0)
		})
	}, 600, 400)
	r, ok := tt.Find("package section")
	if !ok {
		t.Fatal("the diff line is not laid out")
	}
	tt.RightClickAt(r.X+r.W/2, r.Y+r.H/2)
	item, ok := tt.Find(a.t("copyPath"))
	if !ok {
		t.Fatalf("right-clicking the diff row shows no %q item", a.t("copyPath"))
	}
	tt.ClickAt(item.X+item.W/2, item.Y+item.H/2)
	if copied != "dir/section.go" {
		t.Errorf("copy path copied %q; want the section's path dir/section.go", copied)
	}
}
