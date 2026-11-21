package lobedesk

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestWorktreeVisiblePatchRowsHidesBoilerplateHeadersOnly(t *testing.T) {
	lines := []gitrepo.DiffLine{
		{Kind: "meta", Hunk: -1, Text: "diff --git a/app.go b/app.go"},
		{Kind: "meta", Hunk: -1, Text: "old mode 100644"},
		{Kind: "meta", Hunk: -1, Text: "new mode 100755"},
		{Kind: "meta", Hunk: -1, Text: "index 1111111..2222222 100644"},
		{Kind: "meta", Hunk: -1, Text: "--- a/app.go"},
		{Kind: "meta", Hunk: -1, Text: "+++ b/app.go"},
		{Kind: "hunk", Hunk: 0, Text: "@@ -1,2 +1,2 @@"},
		{Kind: "del", Hunk: 0, Old: 1, Text: "-before"},
		{Kind: "add", Hunk: 0, New: 1, Text: "+after"},
		{Kind: "meta", Hunk: 0, Text: `\ No newline at end of file`},
	}
	got := worktreeVisiblePatchRows(lines)
	want := []int{1, 2, 6, 7, 8, 9}
	if !slices.Equal(got, want) {
		t.Fatalf("worktreeVisiblePatchRows = %v, want %v", got, want)
	}
}

func TestWorktreeVisiblePatchRowsKeepsHeaderOnlyPatchNonEmpty(t *testing.T) {
	lines := []gitrepo.DiffLine{
		{Kind: "meta", Hunk: -1, Text: "diff --git a/empty.txt b/empty.txt"},
		{Kind: "meta", Hunk: -1, Text: "index 0000000..e69de29"},
	}
	if got := worktreeVisiblePatchRows(lines); got != nil {
		t.Fatalf("header-only patch should stay visible, got %v", got)
	}
}

func TestWorktreeDiffBodyOmitsRawGitHeaderAndKeepsCanonicalIndexes(t *testing.T) {
	diff := gitrepo.Diff{
		Text: "patch",
		Hunks: []gitrepo.Hunk{
			{Header: "@@ -1,1 +1,1 @@"},
		},
		Lines: []gitrepo.DiffLine{
			{Kind: "meta", Hunk: -1, Text: "diff --git a/internal/lobedesk/app.go b/internal/lobedesk/app.go"},
			{Kind: "meta", Hunk: -1, Text: "index 281223a..c363b3d 100644"},
			{Kind: "meta", Hunk: -1, Text: "--- a/internal/lobedesk/app.go"},
			{Kind: "meta", Hunk: -1, Text: "+++ b/internal/lobedesk/app.go"},
			{Kind: "hunk", Hunk: 0, Text: "@@ -1,1 +1,1 @@"},
			{Kind: "del", Hunk: 0, Old: 1, Text: "-old line"},
			{Kind: "add", Hunk: 0, New: 1, Text: "+new line"},
		},
	}
	a := &app{
		locale:           localeEnglish,
		worktreeSelected: true,
		selectedPath:     "internal/lobedesk/app.go",
		diff:             diff,
		lineCursor:       6,
		linePressIndex:   -1,
		lineSel:          map[int]bool{6: true},
	}
	view := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() {
			a.worktreeDiffHeader(c, theme.TokensOf(c))
			a.diffBody(c)
		})
	}, 640, 280)

	for _, hidden := range []string{
		"diff --git a/internal/lobedesk/app.go b/internal/lobedesk/app.go",
		"index 281223a..c363b3d 100644",
		"--- a/internal/lobedesk/app.go",
		"+++ b/internal/lobedesk/app.go",
	} {
		if view.HasText(hidden) {
			t.Errorf("raw patch header %q should be hidden in workspace diff", hidden)
		}
	}
	for _, shown := range []string{"@@ -1,1 +1,1 @@", "-old line", "+new line"} {
		if !view.HasText(shown) {
			t.Errorf("patch line %q should remain visible in workspace diff", shown)
		}
	}
	if got := a.lineIndices(); !slices.Equal(got, []int{5, 6}) {
		t.Fatalf("lineIndices = %v, want canonical indexes [5 6]", got)
	}
}
