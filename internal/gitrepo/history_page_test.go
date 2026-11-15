package gitrepo

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func importHistory(t *testing.T, dir string, count int) {
	t.Helper()
	importHistoryWithPadding(t, dir, count, 0)
}

func importHistoryWithPadding(t *testing.T, dir string, count, padding int) {
	t.Helper()
	var input strings.Builder
	for i := 0; i < count; i++ {
		message := fmt.Sprintf("Paged commit %04d", i) + strings.Repeat("x", padding)
		fmt.Fprintf(&input, "commit refs/heads/main\ncommitter Test Developer <test@example.invalid> %d +0000\ndata %d\n%s\n\n", 1700000000+i, len(message), message)
	}
	input.WriteString("done\n")
	if _, err := run(context.Background(), dir, input.String(), "fast-import", "--quiet"); err != nil {
		t.Fatal(err)
	}
}

func TestHistoryExcludesStashCommits(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	write(t, dir, "base.txt", "stashed change\n")
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.SaveStash(ctx, "hidden stash", false); err != nil {
		t.Fatal(err)
	}
	page, err := r.ReadHistory(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Commits) != 1 || page.Commits[0].Subject != "initial" {
		t.Fatalf("history includes non-branch commits: %#v", page.Commits)
	}
	for _, commit := range page.Commits {
		if strings.Contains(commit.Refs, "refs/stash") {
			t.Fatalf("stash commit shown as history: %#v", commit)
		}
	}
}

func TestHistoryPagesContinueBeyondTwoThousandAndPinTips(t *testing.T) {
	dir := fixture(t)
	importHistory(t, dir, 2105)
	r := openTest(t, dir)
	ctx := context.Background()
	first, err := r.HistoryPage(ctx, nil, 0, 300)
	if err != nil || len(first.Commits) != 300 || !first.HasMore || len(first.Tips) != 1 {
		t.Fatalf("first page: %d %+v %v", len(first.Commits), first.Tips, err)
	}
	expected := strings.Fields(git(t, dir, "log", "--topo-order", "--all", "HEAD", "--format=%H"))
	// Ref movement must not shift offsets of a previously started traversal.
	git(t, dir, "commit", "--allow-empty", "-m", "new tip after first page")
	pinned := append([]string(nil), first.Tips...)
	all := append([]Commit(nil), first.Commits...)
	for first.HasMore {
		first, err = r.HistoryPage(ctx, pinned, len(all), 300)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, first.Commits...)
	}
	var got []string
	for _, commit := range all {
		got = append(got, commit.SHA)
	}
	if !reflect.DeepEqual(got, expected) || len(all) != 2105 || len(first.Commits) != 5 {
		t.Fatalf("pinned pages lost order or membership: got %d records, tail %d", len(all), len(first.Commits))
	}
	fresh, err := r.ReadHistory(ctx, 2400)
	if err != nil || len(fresh.Commits) != 2106 || fresh.HasMore || fresh.Commits[0].Subject != "new tip after first page" {
		t.Fatalf("fresh aggregate: len=%d more=%v err=%v", len(fresh.Commits), fresh.HasMore, err)
	}
	// Returned tips must not alias the caller's cursor.
	fresh, err = r.HistoryPage(ctx, pinned, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	fresh.Tips[0] = "changed"
	if pinned[0] == "changed" {
		t.Fatal("page exposed caller's tip storage")
	}
}

func TestHistoryPagesAggregateUsesPageSizedOutputs(t *testing.T) {
	dir := fixture(t)
	// Aggregate metadata exceeds the command cap, while each 300-row page fits.
	importHistoryWithPadding(t, dir, 1000, 5000)
	r := openTest(t, dir)
	page, err := r.ReadHistory(context.Background(), 1200)
	if err != nil || len(page.Commits) != 1000 || page.HasMore {
		t.Fatalf("bounded aggregate: records=%d more=%v err=%v", len(page.Commits), page.HasMore, err)
	}
}

func TestHistoryPagesMatchMergeTagsAndDetachedHead(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "checkout", "-b", "side")
	git(t, dir, "commit", "--allow-empty", "-m", "side change")
	git(t, dir, "tag", "-a", "annotated", "-m", "annotation")
	git(t, dir, "checkout", "main")
	git(t, dir, "commit", "--allow-empty", "-m", "main change")
	git(t, dir, "merge", "--no-ff", "side", "-m", "merge side")
	git(t, dir, "checkout", "--detach")
	git(t, dir, "commit", "--allow-empty", "-m", "detached tip")
	expected := strings.Fields(git(t, dir, "log", "--topo-order", "--all", "HEAD", "--format=%H"))
	r := openTest(t, dir)
	var got []string
	var tips []string
	for {
		page, err := r.HistoryPage(context.Background(), tips, len(got), 1)
		if err != nil {
			t.Fatal(err)
		}
		tips = page.Tips
		for _, c := range page.Commits {
			got = append(got, c.SHA)
		}
		if !page.HasMore {
			break
		}
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("merge/tag/detached ordering: %v want %v", got, expected)
	}
}

func TestHistoryPagesEmptyEndArgumentsAndCancellation(t *testing.T) {
	dir := fixture(t)
	r := openTest(t, dir)
	ctx := context.Background()
	empty, err := r.HistoryPage(ctx, nil, 0, 300)
	if err != nil || len(empty.Commits) != 0 || empty.HasMore || empty.Tips == nil {
		t.Fatalf("unborn cursor: %+v %v", empty, err)
	}
	importHistory(t, dir, 4)
	stillEmpty, err := r.HistoryPage(ctx, empty.Tips, 0, 300)
	if err != nil || len(stillEmpty.Commits) != 0 {
		t.Fatal("empty pinned cursor fell back to new HEAD", err)
	}
	first, err := r.HistoryPage(ctx, nil, 0, 2)
	if err != nil || !first.HasMore {
		t.Fatal("first exact page", err)
	}
	last, err := r.HistoryPage(ctx, first.Tips, 2, 2)
	if err != nil || last.HasMore || len(last.Commits) != 2 {
		t.Fatalf("exact final page: %+v %v", last, err)
	}
	for _, args := range [][2]int{{-1, 1}, {0, 0}, {0, 2001}} {
		if _, err := r.HistoryPage(ctx, first.Tips, args[0], args[1]); err == nil {
			t.Fatalf("accepted invalid offset/limit %v", args)
		}
	}
	if _, err := r.HistoryPage(ctx, []string{"--all"}, 0, 1); err == nil {
		t.Fatal("accepted revision option instead of immutable OID")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.HistoryPage(cancelled, first.Tips, 0, 2); err == nil {
		t.Fatal("cancelled read succeeded")
	}
}

// A repository with more refs than a page holds starts its walk from every
// tip. Paging must still return each commit once, in the same topological
// order a single walk gives, across pages far smaller than the ref count.
func TestHistoryWithManyRefsPagesTheSameMembersAndOrder(t *testing.T) {
	dir := fixture(t)
	importHistory(t, dir, 200)
	shas := strings.Fields(git(t, dir, "rev-list", "--reverse", "main"))
	if len(shas) != 200 {
		t.Fatalf("fixture has %d commits; want 200", len(shas))
	}
	for i := 0; i < 120; i++ {
		git(t, dir, "branch", fmt.Sprintf("ref%03d", i), shas[i])
	}
	expected := strings.Fields(git(t, dir, "log", "--topo-order", "--all", "HEAD", "--format=%H"))
	r := openTest(t, dir)
	ctx := context.Background()

	first, err := r.HistoryPage(ctx, nil, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Tips) < 120 {
		t.Fatalf("the walk starts from %d tips; want every branch", len(first.Tips))
	}
	var tips []string
	var got []string
	page := first
	for {
		if page.HasMore && len(page.Commits) != 7 {
			t.Fatalf("page at offset %d holds %d commits; want a full page of 7", len(got), len(page.Commits))
		}
		tips = page.Tips
		for _, commit := range page.Commits {
			got = append(got, commit.SHA)
		}
		if !page.HasMore {
			break
		}
		page, err = r.HistoryPage(ctx, tips, len(got), 7)
		if err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("paged walk over many refs lost order or membership: got %d commits, want %d", len(got), len(expected))
	}
}

// The history list may be scoped to another branch, so the amend box cannot
// take HEAD's subject from its first row. This read must answer for HEAD no
// matter which branch is newest.
func TestHeadSubjectReadsHeadEvenWhenAnotherBranchIsNewer(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "commit", "--allow-empty", "-m", "second on main")
	git(t, dir, "switch", "-c", "feature")
	git(t, dir, "commit", "--allow-empty", "-m", "newer on feature")
	git(t, dir, "switch", "main")
	got, err := openTest(t, dir).HeadSubject(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "second on main" {
		t.Errorf("HeadSubject = %q; want the commit HEAD points to", got)
	}
}

func TestHeadSubjectOfAnEmptyRepositoryIsEmpty(t *testing.T) {
	got, err := openTest(t, fixture(t)).HeadSubject(context.Background())
	if err != nil || got != "" {
		t.Errorf("HeadSubject of an empty repository = %q, %v; want empty, no error", got, err)
	}
}
