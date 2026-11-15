package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

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
