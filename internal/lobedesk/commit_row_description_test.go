package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"

	"supergit/internal/gitrepo"
)

func commitDescriptionTester(a *app, commit gitrepo.Commit) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Width(700).Height(120).Children(func() {
			a.commitRow(c, commit, 0, 0, 1, &commitGraph{})
		})
	}, 700, 120)
}

// The graph draws a merge as a different node and the row has no text for
// it, so a screen reader heard a merge exactly like an ordinary commit.
func TestMergeCommitSaysItIsAMerge(t *testing.T) {
	a := &app{locale: localeChinese}
	merge := gitrepo.Commit{SHA: "aaaa", Subject: "Merge branch 'feat'", Parents: []string{"bbbb", "cccc"}}
	tt := commitDescriptionTester(a, merge)
	row, ok := tt.AccessByRole(ui.RoleOption, merge.Subject)
	if !ok {
		t.Fatal("the merge commit row is not an option")
	}
	if !strings.Contains(row.Description, a.t("mergeCommitRow")) {
		t.Errorf("the merge reads %+v; want a description saying %q", row, a.t("mergeCommitRow"))
	}
}

func TestOrdinaryCommitHasNoMergeDescription(t *testing.T) {
	a := &app{locale: localeChinese}
	commit := gitrepo.Commit{SHA: "aaaa", Subject: "Add the parser", Parents: []string{"bbbb"}}
	tt := commitDescriptionTester(a, commit)
	row, ok := tt.AccessByRole(ui.RoleOption, commit.Subject)
	if !ok {
		t.Fatal("the commit row is not an option")
	}
	if strings.Contains(row.Description, a.t("mergeCommitRow")) {
		t.Errorf("an ordinary commit reads %+v; want no merge description", row)
	}
}
