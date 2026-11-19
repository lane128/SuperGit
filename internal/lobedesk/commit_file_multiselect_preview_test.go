package lobedesk

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func TestHistoricalFilesDeselectAndReselectRetainsRealPatch(t *testing.T) {
	a, _, _ := commitPatchFixture(t)
	diffs, err := a.repo.ShowAll(a.ctx, a.selectedSHA)
	if err != nil {
		t.Fatal(err)
	}
	a.commitDiff, a.commitSections = buildCommitDiff(diffs)
	if len(a.commitSections) == 0 || len(a.commitDiff.Lines) == 0 {
		t.Fatal("owned fixture has no loaded patch")
	}
	path, marker := a.commitFiles[0].Path, ""
	for _, file := range diffs {
		if file.Path == path {
			for _, line := range file.Diff.Lines {
				if line.Kind == "add" {
					marker = line.Text
					break
				}
			}
		}
	}
	if marker == "" {
		t.Fatal("owned first file has no visible added line")
	}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Row(c).Fill().Children(func() {
			ui.Column(c).Width(280).Height(500).Children(func() { a.commitFileList(c) })
			ui.Column(c).Grow(1).MinWidth(0).Children(func() { a.commitDiffView(c) })
		})
	}, 1000, 500)
	clickHistoricalFile(t, tt, path, 0)
	if _, ok := tt.Find(marker); !ok {
		t.Fatal("loaded real patch is not visible before deselection")
	}
	clickHistoricalFile(t, tt, path, ui.Cmd)
	if !a.detailTabs().Active.Summary {
		t.Fatal("last toggle did not return to Summary")
	}
	clickHistoricalFile(t, tt, path, 0)
	if _, ok := tt.Find(marker); !ok {
		t.Fatal("deselect/reselect discarded the loaded commit patch")
	}
}
