package gitrepo

import (
	"context"
	"reflect"
	"testing"
)

func TestCommitAuthorIdentitySurvivesAllHistoryReaders(t *testing.T) {
	root := fixture(t)
	const name = "作者 Name With Spaces"
	const email = "MiXeD+tag@example.invalid"
	t.Setenv("GIT_AUTHOR_NAME", name)
	t.Setenv("GIT_AUTHOR_EMAIL", email)
	t.Setenv("GIT_COMMITTER_NAME", "Different Committer")
	t.Setenv("GIT_COMMITTER_EMAIL", "committer@example.invalid")
	write(t, root, "selected.txt", "base\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "root")
	git(t, root, "branch", "side")
	git(t, root, "mv", "selected.txt", "moved.txt")
	git(t, root, "commit", "-m", "rename")
	write(t, root, "moved.txt", "later\n")
	git(t, root, "commit", "-am", "later")
	git(t, root, "checkout", "side")
	write(t, root, "side.txt", "side\n")
	git(t, root, "add", "--all")
	git(t, root, "commit", "-m", "side")
	git(t, root, "checkout", "main")
	git(t, root, "merge", "--no-ff", "side", "-m", "merge")
	r := openTest(t, root)
	ctx := context.Background()
	check := func(commits []Commit) {
		t.Helper()
		if len(commits) == 0 {
			t.Fatal("identity fixture produced no commits")
		}
		for _, commit := range commits {
			value := reflect.ValueOf(commit).FieldByName("AuthorEmail")
			if !value.IsValid() {
				t.Fatal("commit author email is unavailable")
			}
			if commit.Author != name || value.String() != email || commit.Day == "" || commit.Date == "" {
				t.Fatalf("author identity/date misparsed: %+v", commit)
			}
		}
	}
	all, err := r.History(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	check(all)
	hasRoot, hasMerge := false, false
	for _, commit := range all {
		hasRoot = hasRoot || len(commit.Parents) == 0
		hasMerge = hasMerge || len(commit.Parents) == 2
	}
	if !hasRoot || !hasMerge {
		t.Fatal("fixture lost root or merge")
	}
	file, err := r.FileHistory(ctx, "moved.txt", 100)
	if err != nil {
		t.Fatal(err)
	}
	check(file)
	if len(file) != 3 {
		t.Fatalf("rename-following fixture has %d commits", len(file))
	}
	var tips []string
	var paged []Commit
	for {
		page, err := r.HistoryPage(ctx, tips, len(paged), 1)
		if err != nil {
			t.Fatal(err)
		}
		tips = page.Tips
		paged = append(paged, page.Commits...)
		if !page.HasMore {
			break
		}
	}
	check(paged)
	if len(paged) != len(all) {
		t.Fatal("paged history lost records")
	}
	stream, err := r.OpenHistoryStream(ctx, tips, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var streamed []Commit
	for {
		page, err := stream.Next(1)
		if err != nil {
			t.Fatal(err)
		}
		streamed = append(streamed, page.Commits...)
		if !page.HasMore {
			break
		}
	}
	check(streamed)
	if len(streamed) != len(all) {
		t.Fatal("stream framing lost records")
	}
	tips = nil
	var filePaged []Commit
	for {
		page, err := r.FileHistoryPage(ctx, "moved.txt", tips, len(filePaged), 1)
		if err != nil {
			t.Fatal(err)
		}
		tips = page.Tips
		filePaged = append(filePaged, page.Commits...)
		if !page.HasMore {
			break
		}
	}
	check(filePaged)
	if len(filePaged) != len(file) {
		t.Fatal("file history pagination lost rename records")
	}
}
