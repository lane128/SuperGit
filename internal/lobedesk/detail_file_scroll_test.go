package lobedesk

import (
	"supergit/internal/gitrepo"
	"testing"
)

func TestLoadedWorkingFileTabOwnsOneScrollContainer(t *testing.T) {
	a := &app{worktreeSelected: true, selectedPath: "a.go", diff: gitrepo.Diff{Text: "+loaded patch\n", Lines: []gitrepo.DiffLine{{Kind: "add", Text: "+loaded patch"}}}}
	a.detailTabs().Open(detailFileKey{Path: "a.go"})
	tt := pageTester(a, a.historyDiffPane)
	tt.Frame()
	if _, ok := tt.Find("+loaded patch"); !ok {
		t.Fatal("loaded file patch disappeared")
	}
}
