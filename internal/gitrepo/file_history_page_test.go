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
