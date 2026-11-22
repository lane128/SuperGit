package lobedesk

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"supergit/internal/gitrepo"
)

func TestCommitSummaryParentLinksNavigate(t *testing.T) {
	parents := []string{strings.Repeat("ab", 20), strings.Repeat("cd", 20)}
	for _, parent := range parents {
		label := parent[:8]
		for _, input := range []string{"click", "enter"} {
			t.Run(label+"/"+input, func(t *testing.T) {
				a := browsingApp(t)
				merge := gitrepo.Commit{SHA: a.selectedSHA, Subject: "Merge branches", Parents: parents}
				a.commits = []gitrepo.Commit{merge, {SHA: parents[0]}, {SHA: parents[1]}}
				a.detailsDiffHidden = true
				view := pageTester(a, a.detailOverview)
				link, ok := view.AccessByRole(ui.RoleLink, label)
				if !ok {
					t.Fatalf("missing parent link %q", label)
				}
				if input == "enter" {
					for range len(view.AccessAll()) {
						view.Key(0, ui.KeyTab)
						view.Frame()
						if view.Focused(label) {
							break
						}
					}
					if !view.Focused(label) || a.selectedSHA != merge.SHA {
						t.Fatalf("Tab must focus %q while the merge is still selected", label)
					}
					view.Key(0, ui.KeyEnter)
				} else {
					r := link.Bounds
					view.ClickAt(r.X+r.W/2, r.Y+r.H/2)
				}
				view.Frame()
				if a.selectedSHA != parent {
					t.Fatalf("%s on %q selected %q, want %q", input, label, a.selectedSHA, parent)
				}
				if _, ok := view.Find(parent); !ok {
					t.Fatalf("root commit %q is missing from the summary", parent)
				}
				if _, ok := view.Find(a.t("parent")); ok {
					t.Fatal("root commit still displays a parent row")
				}
				if link, ok := view.AccessByRole(ui.RoleLink, ""); ok {
					t.Fatalf("root commit still exposes link %q", link.Name)
				}
			})
		}
	}
}

func TestCommitSummaryStashSnapshotParentsAreReadOnly(t *testing.T) {
	base, index, untracked := strings.Repeat("ab", 20), strings.Repeat("cd", 20), strings.Repeat("ef", 20)
	stash := gitrepo.Stash{SHA: strings.Repeat("12", 20), Ref: "stash@{0}", ParentSHAs: strings.Join([]string{base, index, untracked}, " ")}
	a := browsingApp(t)
	a.stashes = []gitrepo.Stash{stash}
	a.selectedStash, a.selectedSHA = stash.SHA, ""
	// Include the snapshot objects so an erroneous jump can actually select them.
	a.commits = []gitrepo.Commit{{SHA: base}, {SHA: index}, {SHA: untracked}}
	a.detailsDiffHidden = true
	view := pageTester(a, a.detailOverview)
	for _, sha := range []string{index, untracked} {
		label := sha[:8]
		text, ok := view.AccessByRole(ui.RoleText, label)
		if !ok || text.Focusable || text.Offscreen {
			t.Fatalf("stash snapshot %q must be visible static text, got %+v", label, text)
		}
		for _, role := range []ui.Role{ui.RoleLink, ui.RoleButton} {
			if _, ok := view.AccessByRole(role, label); ok {
				t.Fatalf("stash snapshot %q exposes an action", label)
			}
		}
		r := text.Bounds
		view.ClickAt(r.X+r.W/2, r.Y+r.H/2)
		view.Frame()
		if a.selectedStash != stash.SHA || a.selectedSHA != "" {
			t.Fatalf("clicking snapshot %q changed selection to stash %q, commit %q", label, a.selectedStash, a.selectedSHA)
		}
	}
	link, ok := view.AccessByRole(ui.RoleLink, base[:8])
	if !ok {
		t.Fatalf("missing stash base link %q", base[:8])
	}
	r := link.Bounds
	view.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	view.Frame()
	if a.selectedStash != "" || a.selectedSHA != base {
		t.Fatalf("stash base link selected stash %q, commit %q; want commit %q", a.selectedStash, a.selectedSHA, base)
	}
}
