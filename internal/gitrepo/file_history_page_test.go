package gitrepo

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Catch truncated traversal, moving-HEAD offsets and lost rename ancestry.
func TestFileHistoryPagesFollowRenamesBeyondTwoThousandAndPinHead(t *testing.T) {
	dir := fixture(t)
	path := ":(glob) renamed\t文件.txt"
	var input strings.Builder
	for i := 0; i < 2405; i++ {
		// Earlier metadata exceeds 4 MiB; only the requested page may be retained.
		message := fmt.Sprintf("File commit %04d", i) + strings.Repeat("x", 2000)
		fmt.Fprintf(&input, "commit refs/heads/main\ncommitter Test Developer <test@example.invalid> %d +0000\ndata %d\n%s\n", 1700000000+i, len(message), message)
		name := "base.txt"
		if i == 1200 {
			fmt.Fprintf(&input, "R base.txt %s\n", strconv.Quote(path))
		}
		if i >= 1200 {
			name = path
		}
		revision := i
		if i == 1200 {
			revision-- // Keep rename contents identical so Git recognizes its ancestry.
		}
		content := fmt.Sprintf("line %04d\n", revision)
		fmt.Fprintf(&input, "M 100644 inline %s\ndata %d\n%s\n", strconv.Quote(name), len(content), content)
	}
	input.WriteString("done\n")
	if _, err := run(context.Background(), dir, input.String(), "fast-import", "--quiet"); err != nil {
		t.Fatal(err)
	}
	r := openTest(t, dir)
	ctx := context.Background()
	first, err := r.FileHistoryPage(ctx, path, nil, 0, 200)
	if err != nil || len(first.Commits) != 200 || !first.HasMore || len(first.Tips) != 1 {
		t.Fatalf("first page: %+v %v", first, err)
	}
	expected := strings.Fields(git(t, dir, "--literal-pathspecs", "log", "--follow", "--format=%H", "HEAD", "--", path))
	// Move HEAD after the first page, without changing the pinned traversal.
	git(t, dir, "reset", "--hard", "HEAD~500")
	all := append([]Commit(nil), first.Commits...)
	for first.HasMore {
		first, err = r.FileHistoryPage(ctx, path, first.Tips, len(all), 200)
		if err != nil {
			t.Fatal(err)
		}
		if len(first.Commits) > 200 {
			t.Fatal("unbounded page")
		}
		all = append(all, first.Commits...)
	}
	var got []string
	for _, c := range all {
		got = append(got, c.SHA)
	}
	if len(all) != 2405 || !reflect.DeepEqual(got, expected) {
		t.Fatalf("rename traversal: got %d want %d", len(all), len(expected))
	}
}

func TestFileHistoryPagesFollowDeletedRenamedPath(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	initial := git(t, dir, "rev-parse", "HEAD")
	path := ":(glob) renamed 文件.txt"
	write(t, dir, "base.txt", "before rename\n")
	git(t, dir, "commit", "-qam", "change old path")
	beforeRename := git(t, dir, "rev-parse", "HEAD")
	git(t, dir, "mv", "--", "base.txt", path)
	git(t, dir, "commit", "-qm", "rename")
	write(t, dir, path, "after rename\n")
	git(t, dir, "commit", "-qam", "change new path")
	write(t, dir, "unrelated.txt", "unrelated\n")
	git(t, dir, "add", "--", "unrelated.txt")
	git(t, dir, "commit", "-qm", "unrelated")
	unrelated := git(t, dir, "rev-parse", "HEAD")
	git(t, dir, "--literal-pathspecs", "rm", "--", path)
	git(t, dir, "commit", "-qm", "delete renamed path")
	tip := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	expected := strings.Fields(git(t, dir, "--literal-pathspecs", "log", "--follow", "--format=%H", tip, "--", path))
	plain := strings.Fields(git(t, dir, "--literal-pathspecs", "log", "--format=%H", tip, "--", path))
	if len(expected) != 5 || expected[0] != strings.TrimSpace(tip) || reflect.DeepEqual(expected, plain) {
		t.Fatal("fixture does not distinguish followed deleted history")
	}
	r := openTest(t, dir)
	page, err := r.FileHistoryPage(context.Background(), path, nil, 0, 2)
	if err != nil || len(page.Commits) != 2 || !page.HasMore || len(page.Tips) != 1 || page.Tips[0] != strings.TrimSpace(tip) {
		t.Fatalf("first deleted-path page: %+v %v", page, err)
	}
	tips := append([]string(nil), page.Tips...)
	// Recreate the deleted path at a newer HEAD after the first accepted page.
	write(t, dir, path, "new HEAD recreation\n")
	git(t, dir, "--literal-pathspecs", "add", "--", path)
	git(t, dir, "commit", "-qm", "recreate after snapshot")
	var got []string
	for {
		if !reflect.DeepEqual(page.Tips, tips) || len(page.Commits) > 2 || page.HasMore && len(page.Commits) == 0 {
			t.Fatal("page lost pinned bounded cursor")
		}
		for _, commit := range page.Commits {
			got = append(got, commit.SHA)
		}
		if !page.HasMore {
			break
		}
		if len(got) > len(expected) {
			t.Fatal("traversal did not finish")
		}
		page, err = r.FileHistoryPage(context.Background(), path, tips, len(got), 2)
		if err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("deleted rename sequence: got %v want %v", got, expected)
	}
	joined := "|" + strings.Join(got, "|") + "|"
	for _, sha := range []string{initial, beforeRename} {
		if !strings.Contains(joined, "|"+strings.TrimSpace(sha)+"|") {
			t.Fatal("lost old-path ancestry")
		}
	}
	if strings.Contains(joined, "|"+strings.TrimSpace(unrelated)+"|") {
		t.Fatal("included unrelated commit")
	}
}

func TestFileHistoryPagesEmptyArgumentsAndCancellation(t *testing.T) {
	dir := fixture(t)
	r := openTest(t, dir)
	ctx := context.Background()
	empty, err := r.FileHistoryPage(ctx, "base.txt", nil, 0, 2)
	if err != nil || empty.Tips == nil || len(empty.Commits) != 0 || empty.HasMore {
		t.Fatalf("unborn: %+v %v", empty, err)
	}
	seed(t, dir)
	pinned, err := r.FileHistoryPage(ctx, "base.txt", empty.Tips, 0, 2)
	if err != nil || len(pinned.Commits) != 0 {
		t.Fatalf("empty cursor followed new HEAD: %+v %v", pinned, err)
	}
	first, err := r.FileHistoryPage(ctx, "base.txt", nil, 0, 1)
	if err != nil || len(first.Commits) != 1 || first.HasMore {
		t.Fatalf("exact last page: %+v %v", first, err)
	}
	for _, args := range [][2]int{{-1, 1}, {0, 0}, {0, 2001}, {1, 1}} {
		if _, err := r.FileHistoryPage(ctx, "base.txt", nil, args[0], args[1]); err == nil {
			t.Fatalf("accepted range %v", args)
		}
	}
	for _, tips := range [][]string{{"--all"}, {first.Tips[0], first.Tips[0]}} {
		if _, err := r.FileHistoryPage(ctx, "base.txt", tips, 0, 1); err == nil {
			t.Fatalf("accepted tips %v", tips)
		}
	}
	if _, err := r.FileHistoryPage(ctx, "../escape", nil, 0, 1); err == nil {
		t.Fatal("accepted escaped path")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.FileHistoryPage(canceled, "base.txt", nil, 0, 1); err == nil {
		t.Fatal("cancellation reported as empty success")
	}
	page, err := r.FileHistoryPage(ctx, "base.txt", first.Tips, 10, 1)
	if err != nil || len(page.Commits) != 0 || page.HasMore {
		t.Fatalf("past end: %+v %v", page, err)
	}
	page.Tips[0] = "changed"
	if first.Tips[0] == "changed" {
		t.Fatal("cursor aliases caller storage")
	}
}
