package gitrepo

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestPaletteHiddenBranchesWalkVisibleTips(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	git(t, dir, "tag", "base")
	git(t, dir, "switch", "-c", "topic")
	git(t, dir, "commit", "--allow-empty", "-m", "topic-only")
	topic := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	git(t, dir, "switch", "main")
	git(t, dir, "commit", "--allow-empty", "-m", "main-only")
	git(t, dir, "update-ref", "refs/remotes/origin/topic", topic)
	git(t, dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/topic")
	hidden := []string{"refs/heads/topic", "refs/remotes/origin/topic"}
	r, ctx := openTest(t, dir), context.Background()
	page, err := r.ReadVisibleHistory(ctx, 10, hidden)
	if err != nil || len(page.Commits) != 2 || page.Commits[0].Subject != "main-only" || page.Commits[1].Subject != "initial" {
		t.Fatalf("visible history %+v, err %v", page, err)
	}
	git(t, dir, "switch", "topic")
	page, err = r.ReadVisibleHistory(ctx, 10, hidden)
	if err != nil || len(page.Commits) != 2 || slices.ContainsFunc(page.Commits, func(c Commit) bool { return c.SHA == topic }) {
		t.Fatalf("hidden checked-out branch %+v, %v", page, err)
	}
	git(t, dir, "tag", "-a", "keep", "-m", "keep", topic)
	page, err = r.ReadVisibleHistory(ctx, 10, hidden)
	ix := slices.IndexFunc(page.Commits, func(c Commit) bool { return c.SHA == topic })
	if err != nil || ix < 0 || !strings.Contains(page.Commits[ix].Refs, "tag: keep") || strings.Contains(page.Commits[ix].Refs, "topic") {
		t.Fatalf("shared tag lost or leaked branch: %+v %v", page, err)
	}
	page, err = r.ReadVisibleHistory(ctx, 1, hidden)
	if err != nil || !page.HasMore {
		t.Fatalf("first page: %+v %v", page, err)
	}
	stream, err := r.OpenVisibleHistoryStream(ctx, page.Tips, 1, hidden)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	next, err := stream.Next(20)
	if err != nil || len(next.Commits) != 2 {
		t.Fatalf("stream %+v %v", next, err)
	}
	for _, c := range next.Commits {
		if strings.Contains(c.Refs, "topic") {
			t.Fatalf("stream exposed hidden label %+v", c)
		}
	}
	git(t, dir, "switch", "--detach", topic)
	git(t, dir, "tag", "-d", "keep")
	page, err = r.ReadVisibleHistory(ctx, 10, hidden)
	if err != nil || page.Commits[0].SHA != topic {
		t.Fatalf("detached HEAD should remain visible: %+v %v", page, err)
	}
}

func TestPaletteSnapshotTreeReadsImmutableFiles(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	p := "folder/旧\nname.txt"
	write(t, dir, p, "snapshot\n")
	git(t, dir, "add", "--", p)
	git(t, dir, "commit", "-m", "add nested file")
	rev := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	write(t, dir, p, "working changes\n")
	r, ctx := openTest(t, dir), context.Background()
	files, err := r.SnapshotFiles(ctx, rev)
	if err != nil || len(files) != 2 {
		t.Fatalf("all tree entries: %+v %v", files, err)
	}
	i := slices.IndexFunc(files, func(f SnapshotFile) bool { return f.Path == p })
	if i < 0 || files[i].Size != 9 || files[i].Kind != "blob" {
		t.Fatalf("snapshot metadata: %+v", files)
	}
	body, err := r.SnapshotBlob(ctx, files[i].OID)
	if err != nil || body != "snapshot\n" {
		t.Fatalf("immutable blob: %q %v", body, err)
	}
}

func TestPaletteDiffWhitespaceWorktreeIndexCommitAndStash(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r, ctx := openTest(t, dir), context.Background()
	write(t, dir, "base.txt", " original\n")
	for _, staged := range []bool{false, true} {
		if staged {
			git(t, dir, "add", "base.txt")
		}
		d, err := r.DisplayDiff(ctx, "base.txt", staged, true)
		if err != nil || d.Text != "" || d.CanStageHunks {
			t.Fatalf("whitespace-only staged=%t: %+v %v", staged, d, err)
		}
		canonical, err := r.Diff(ctx, "base.txt", staged)
		if err != nil || !canonical.CanStageHunks {
			t.Fatalf("canonical diff staged=%t: %+v %v", staged, canonical, err)
		}
	}
	git(t, dir, "commit", "-m", "whitespace")
	sha := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	files, err := r.DisplayCommitDiff(ctx, sha, true)
	if err != nil || len(files) != 0 {
		t.Fatalf("ignored commit diff %+v %v", files, err)
	}
	write(t, dir, "base.txt", "\toriginal\n")
	git(t, dir, "stash", "push", "-m", "indent")
	sha = strings.TrimSpace(git(t, dir, "rev-parse", "refs/stash"))
	d, err := r.DisplayStashDiff(ctx, sha, true)
	if err != nil || d.Text != "" {
		t.Fatalf("ignored stash diff %+v %v", d, err)
	}
	write(t, dir, "base.txt", "  original\nkeep\n")
	d, err = r.DisplayDiff(ctx, "base.txt", false, true)
	if err != nil || !strings.Contains(d.Text, "+keep") || d.CanStageHunks {
		t.Fatalf("filtered patch: %+v %v", d, err)
	}
	write(t, dir, "base.txt", " original\nkeep\n")
	d, err = r.DisplayDiff(ctx, "base.txt", false, true)
	if err != nil || !d.CanStageHunks {
		t.Fatalf("canonical patch should allow staging: %+v %v", d, err)
	}
}
