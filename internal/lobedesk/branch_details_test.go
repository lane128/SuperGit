package lobedesk

import (
	"math"
	"testing"

	"supergit/internal/gitrepo"
)

func branchDetailFixture() *app {
	return &app{
		locale:  localeChinese,
		remotes: []gitrepo.Remote{{Name: "origin"}},
		branches: []gitrepo.Branch{{
			Name: "main", SHA: "1111111111111111111111111111111111111111", Current: true,
			Upstream: "origin/main", Ahead: 2, Behind: 1,
			Commit: gitrepo.Commit{
				SHA: "1111111111111111111111111111111111111111", Subject: "Latest work",
				Author: "Ada Lovelace", Date: "2026-11-14T10:00:00+08:00",
			},
		}},
		remoteBranches: []gitrepo.Branch{{
			Name: "origin/topic", SHA: "2222222222222222222222222222222222222222",
			Commit: gitrepo.Commit{
				SHA: "2222222222222222222222222222222222222222", Subject: "Remote work",
				Author: "Grace Hopper", Date: "2026-11-13T09:00:00+08:00",
			},
		}},
		tags: []gitrepo.Tag{
			{
				Name: "v1", ObjectSHA: "3333333333333333333333333333333333333333", ObjectType: "tag",
				TargetSHA: "1111111111111111111111111111111111111111", TargetType: "commit",
				Message: "Release notes",
				Commit: gitrepo.Commit{
					SHA: "1111111111111111111111111111111111111111", Subject: "Latest work",
					Author: "Ada Lovelace", Date: "2026-11-14T10:00:00+08:00",
				},
			},
			{
				Name: "light", ObjectSHA: "1111111111111111111111111111111111111111", ObjectType: "commit",
				TargetSHA: "1111111111111111111111111111111111111111", TargetType: "commit",
			},
		},
	}
}

func TestSelectedBranchDetailResolvesAllThreeKinds(t *testing.T) {
	a := branchDetailFixture()
	cases := []struct {
		key  string
		kind branchDetailKind
		name string
	}{
		{"local:main", branchDetailLocal, "main"},
		{"remote:origin/topic", branchDetailRemote, "topic"},
		{"tag:v1", branchDetailTag, "v1"},
	}
	for _, tc := range cases {
		a.branchSel = tc.key
		sel, ok := a.selectedBranchDetail()
		if !ok {
			t.Fatalf("%s: no detail selection", tc.key)
		}
		if sel.kind != tc.kind || sel.name != tc.name {
			t.Fatalf("%s: got kind=%v name=%q; want %v/%q", tc.key, sel.kind, sel.name, tc.kind, tc.name)
		}
	}
	a.branchSel = "remote:origin/topic"
	if sel, _ := a.selectedBranchDetail(); sel.remote != "origin" {
		t.Fatalf("remote half = %q; want origin", sel.remote)
	}
}

func TestSelectedBranchDetailKeepsTheListDataWhenFiltered(t *testing.T) {
	a := branchDetailFixture()
	a.branchSel = "local:main"
	a.branchQuery = "nothing matches this"
	// The filtered slices hide the row; the detail still reads the full list,
	// so a search does not blank the pane before sync moves the selection.
	if a.branches == nil || len(a.filteredBranches()) != 0 {
		t.Fatal("fixture does not hide the selected row")
	}
	sel, ok := a.selectedBranchDetail()
	if !ok || sel.name != "main" {
		t.Fatalf("filtered selection = %+v, %t; want main", sel, ok)
	}
	// Once the page syncs, the selection follows the first visible row and the
	// detail describes that row.
	a.syncBranchSelection()
	if a.branchSel != "" {
		t.Fatalf("branchSel = %q with no matching rows; want empty", a.branchSel)
	}
	if _, ok := a.selectedBranchDetail(); ok {
		t.Fatal("an empty selection still produced details")
	}
}

func TestSyncBranchSelectionResetsTheDetailScrollOnlyOnAChange(t *testing.T) {
	a := branchDetailFixture()
	a.branchSel = "local:main"
	a.branchDetailScroll.Y = 120
	a.syncBranchSelection()
	if a.branchDetailScroll.Y != 120 {
		t.Fatalf("scroll reset while the selection stayed put: %v", a.branchDetailScroll.Y)
	}
	a.branchQuery = "topic"
	a.syncBranchSelection()
	if a.branchSel != "remote:origin/topic" {
		t.Fatalf("branchSel = %q; want remote:origin/topic", a.branchSel)
	}
	if a.branchDetailScroll.Y != 0 {
		t.Fatalf("scroll = %v after the selection moved; want 0", a.branchDetailScroll.Y)
	}
}

func TestTagDetailMessageOnlyComesFromAnnotatedTags(t *testing.T) {
	a := branchDetailFixture()
	a.branchSel = "tag:v1"
	sel, ok := a.selectedBranchDetail()
	if !ok || sel.tag.ObjectType != "tag" || sel.tag.Message != "Release notes" {
		t.Fatalf("annotated tag = %+v, %t; want its message", sel.tag, ok)
	}
	a.branchSel = "tag:light"
	sel, ok = a.selectedBranchDetail()
	if !ok || sel.tag.ObjectType != "commit" || sel.tag.Message != "" {
		t.Fatalf("lightweight tag = %+v, %t; want no tag message", sel.tag, ok)
	}
}

func TestNoSelectionProducesNoDetail(t *testing.T) {
	a := branchDetailFixture()
	for _, key := range []string{"", "local:missing", "remote:missing", "tag:missing", "bogus:x", "local"} {
		a.branchSel = key
		if _, ok := a.selectedBranchDetail(); ok {
			t.Fatalf("%q produced a detail selection", key)
		}
	}
}

func TestCheckoutTagRefusesNonCommitTargets(t *testing.T) {
	a := &app{locale: localeChinese}
	a.checkoutTag(gitrepo.Tag{Name: "tree", TargetType: "tree", TargetSHA: "abc"})
	if a.confirm.open {
		t.Fatal("a tree tag opened a checkout confirmation")
	}
	a.checkoutTag(gitrepo.Tag{Name: "v1", TargetType: "commit", TargetSHA: "deadbeef"})
	if !a.confirm.open {
		t.Fatal("a commit tag did not open a checkout confirmation")
	}
	if a.confirm.actionName != a.t("checkout") {
		t.Fatalf("confirm action = %q; want %q", a.confirm.actionName, a.t("checkout"))
	}
}

func TestBranchDetailNameRunesFollowThePaneWidth(t *testing.T) {
	a := &app{}
	a.branchSplit.rowWidth = 980
	// Pin the divider so the details get their minimum width, then their
	// default: a narrow pane must elide sooner than a wide one.
	a.branchSplit.share = 1 - branchSplitMinDetail/980
	narrow := a.branchDetailNameRunes()
	a.branchSplit.share = 0
	wide := a.branchDetailNameRunes()
	if narrow >= wide {
		t.Fatalf("narrow pane keeps %d runes, wide keeps %d; want fewer", narrow, wide)
	}
	if wide > 40 || narrow < 12 {
		t.Fatalf("name rune budget = %d narrow / %d wide; want within [12, 40]", narrow, wide)
	}
	// The first frame has no measured row; the opening room keeps the default.
	a.branchSplit.rowWidth = 0
	if got := a.branchDetailNameRunes(); got != wide {
		t.Fatalf("unmeasured row keeps %d runes; want the default's %d", got, wide)
	}
}

func TestBranchShareDefaultsAndClamps(t *testing.T) {
	a := &app{}
	a.branchSplit.rowWidth = 1000
	if got, want := a.branchShare(), 1-shareForWidth(1000, branchPaneWidth); math.Abs(float64(got-want)) > 0.0001 {
		t.Fatalf("branchShare = %v; want %v", got, want)
	}
	room := float32(1000 - splitterWidth)
	a.branchSplit.share = 0.99
	if got, want := a.branchShare(), 1-branchSplitMinDetail/room; math.Abs(float64(got-want)) > 0.0001 {
		t.Fatalf("clamped branchShare = %v; want %v", got, want)
	}
}

func TestPromptCheckoutRemoteBranchReusesAnExistingTracker(t *testing.T) {
	a := branchDetailFixture()
	// origin/main is already the upstream of main; asking for a name would
	// make git fail with "a branch named 'main' already exists".
	a.promptCheckoutRemoteBranch("origin/main", "main")
	if a.prompt.open {
		t.Fatal("a tracked remote branch opened the create-branch prompt")
	}
	// A ref nothing tracks still goes through the prompt.
	a.promptCheckoutRemoteBranch("origin/other", "other")
	if !a.prompt.open {
		t.Fatal("an untracked remote branch did not open the create-branch prompt")
	}
}

func TestBranchDetailEmptyTextNamesWhyNothingIsShown(t *testing.T) {
	a := branchDetailFixture()
	if got, want := a.branchDetailEmptyText(), a.t("branchDetailHint"); got != want {
		t.Fatalf("with refs to select: %q; want %q", got, want)
	}
	a.branchQuery = "nothing matches"
	if got, want := a.branchDetailEmptyText(), a.t("searchBranchesNoMatch"); got != want {
		t.Fatalf("under a query: %q; want %q", got, want)
	}
	empty := &app{locale: localeChinese}
	if got, want := empty.branchDetailEmptyText(), empty.t("noBranchesHint"); got != want {
		t.Fatalf("without refs: %q; want %q", got, want)
	}
}
