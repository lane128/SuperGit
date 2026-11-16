package lobedesk

import (
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"supergit/internal/gitrepo"
)

func TestWorkspaceAlwaysShowsThreeFileGroups(t *testing.T) {
	for _, entries := range [][]gitrepo.Entry{nil, {{Path: "a.go", Index: 'M', Worktree: 'M'}}, {{Path: "new.txt", Index: '?', Worktree: '?'}}, {{Path: "conflict", Index: 'U', Worktree: 'U', Conflict: true}}} {
		a := &app{locale: localeEnglish, status: gitrepo.Status{Entries: entries}}
		tt := pageTester(a, a.workingFileSections)
		var previous float32 = -1
		for _, name := range []string{"Modified Files", "Untracked Files", "Staged Files"} {
			header, ok := tt.AccessByRole(ui.RoleButton, name)
			if !ok {
				t.Fatalf("entries %+v: missing permanent group %q", entries, name)
			}
			if header.Bounds.Y <= previous {
				t.Fatal("group order changed")
			}
			previous = header.Bounds.Y
		}
		// Folding cannot erase a zero-count group heading.
		a.filesCollapsed = map[string]bool{filesSectionUnstaged: true, filesSectionUntracked: true, filesSectionStaged: true}
		tt.Frame()
		for _, name := range []string{"Modified Files", "Untracked Files", "Staged Files"} {
			if _, ok := tt.AccessByRole(ui.RoleButton, name); !ok {
				t.Fatalf("folding removed %q", name)
			}
		}
	}
}

func TestOverviewAndFilesShareGroupOrder(t *testing.T) {
	a := &app{status: gitrepo.Status{Entries: []gitrepo.Entry{
		{Path: "staged", Index: 'M', Worktree: ' '},
		{Path: "new", Index: '?', Worktree: '?'},
		{Path: "both", Index: 'M', Worktree: 'M'},
	}}}
	want := []detailFileKey{{Path: "both"}, {Path: "new"}, {Path: "staged", Staged: true}, {Path: "both", Staged: true}}
	if got := a.overviewKeys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("overview order = %+v; want %+v", got, want)
	}
	var files []detailFileKey
	for _, ref := range a.fileRefs() {
		files = append(files, detailFileKey{Path: ref.path, Staged: ref.staged})
	}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("keyboard order = %+v; want %+v", files, want)
	}
	a.filesCollapsed = map[string]bool{filesSectionUntracked: true}
	var visible []detailFileKey
	for _, ref := range a.fileRefs() {
		visible = append(visible, detailFileKey{Path: ref.path, Staged: ref.staged})
	}
	if !reflect.DeepEqual(visible, []detailFileKey{{Path: "both"}, {Path: "staged", Staged: true}, {Path: "both", Staged: true}}) {
		t.Fatalf("folded rows remain in keyboard traversal: %+v", visible)
	}
	if got := a.overviewKeys(); !reflect.DeepEqual(got, want) {
		t.Fatal("folding the file column removed an overview target")
	}
	path, staged := firstWorkspaceEntry(a.status.Entries[:2])
	if path != "new" || staged {
		t.Fatalf("initial cursor = %q staged=%v; must follow new group order", path, staged)
	}
}
