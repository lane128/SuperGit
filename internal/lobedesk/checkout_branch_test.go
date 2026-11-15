package lobedesk

import (
	"testing"

	"supergit/internal/gitrepo"
)

// The picker's rows and the branch Enter takes come from the same filtered
// list, so the filter keeps Git's order rather than re-sorting by rank: the
// first match is the first branch under the cursor's name prefix.
func TestFilterBranchesAtKeepsTheOrderAndIgnoresCase(t *testing.T) {
	branches := []gitrepo.Branch{
		{Name: "feature/login"},
		{Name: "fixup"},
		{Name: "main", Current: true},
		{Name: "Release/1.0"},
	}
	got := filterBranchesAt(branches, " fIx ")
	if len(got) != 1 || got[0].Name != "fixup" {
		t.Errorf("filter %q = %v; want just fixup", " fIx ", got)
	}
	// An empty query is every branch, not none.
	if all := filterBranchesAt(branches, "   "); len(all) != len(branches) {
		t.Errorf("blank query = %d rows; want all %d", len(all), len(branches))
	}
	if none := filterBranchesAt(branches, "nope"); len(none) != 0 {
		t.Errorf("an unmatched query = %v; want nothing", none)
	}
	slashes := filterBranchesAt(branches, "/")
	if len(slashes) != 2 || slashes[0].Name != "feature/login" || slashes[1].Name != "Release/1.0" {
		t.Errorf("filter %q = %v; want the two names with a slash in order", "/", slashes)
	}
}

// The dialog remembers the commit it was opened for, and a frame without it
// draws nothing: the menu's key opens it and the next frame's dialog reads it.
func TestCheckoutBranchDialogOpensForTheCommit(t *testing.T) {
	a := &app{locale: localeChinese}
	if a.checkoutBranch.open {
		t.Fatal("the dialog started open")
	}
	commit := gitrepo.Commit{SHA: "0123456789abcdef0123456789abcdef01234567", Subject: "Move the merge"}
	a.openCheckoutBranchDialog(commit)
	if !a.checkoutBranch.open || a.checkoutBranch.sha != commit.SHA || !a.checkoutBranch.reveal {
		t.Errorf("dialog = %#v; want it open on the commit with its highlight ready to show", a.checkoutBranch)
	}
}
