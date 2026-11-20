package gitrepo

import (
	"context"
	"strings"
	"testing"
)

func TestHistoryChangedFileCountsMatchFilesPane(t *testing.T) {
	dir := fixture(t)
	write(t, dir, "nested/file\nname", "unchanged\n")
	write(t, dir, "other", "keep\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-m", "root")
	git(t, dir, "checkout", "-b", "topic")
	git(t, dir, "mv", "nested/file\nname", "renamed")
	git(t, dir, "commit", "-m", "rename")
	git(t, dir, "checkout", "main")
	git(t, dir, "commit", "--allow-empty", "-m", "empty")
	git(t, dir, "merge", "--no-ff", "topic", "-m", "merge")
	repo := &Repository{Root: dir}
	var rows []Commit
	for _, line := range strings.Split(strings.TrimSpace(git(t, dir, "rev-list", "--parents", "--all")), "\n") {
		ids := strings.Fields(line)
		rows = append(rows, Commit{SHA: ids[0], Parents: ids[1:]})
	}
	counts, err := repo.ChangedFileCounts(context.Background(), rows)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		files, err := repo.CommitFiles(context.Background(), row.SHA)
		if err != nil {
			t.Fatal(err)
		}
		if n, ok := counts[row.SHA]; !ok || n != len(files) {
			t.Errorf("commit %s: count=%d present=%v, Files lists %d", row.SHA, n, ok, len(files))
		}
	}
}
