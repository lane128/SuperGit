package gitrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBranchLifecycleAndDirtyCheckoutGuard(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.CreateBranch(ctx, "feature/native", "HEAD", true); err != nil {
		t.Fatal(err)
	}
	if s, _ := r.Status(ctx); s.Branch != "feature/native" {
		t.Fatal(s)
	}
	if err := r.RenameBranch(ctx, "feature/native", "feature/renamed"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "base.txt", "uncommitted\n")
	if err := r.SwitchBranch(ctx, "main"); err == nil {
		t.Fatal("dirty checkout accepted")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "base.txt")); string(b) != "uncommitted\n" {
		t.Fatal("dirty work lost")
	}
	git(t, dir, "restore", "--", "base.txt")
	if err := r.SwitchBranch(ctx, "main"); err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteBranch(ctx, "feature/renamed"); err != nil {
		t.Fatal(err)
	}
	if err := r.CreateBranch(ctx, "--force", "HEAD", false); err == nil {
		t.Fatal("option name accepted")
	}
}
func TestLocalRemoteCloneFetchPushAndFastForwardPull(t *testing.T) {
	ctx := context.Background()
	source := fixture(t)
	seed(t, source)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, source, "clone", "--bare", source, bare)
	clonePath := filepath.Join(t.TempDir(), "cloned")
	r, err := Clone(ctx, bare, clonePath)
	if err != nil {
		t.Fatal(err)
	}
	git(t, clonePath, "config", "user.name", "Test Developer")
	git(t, clonePath, "config", "user.email", "test@example.invalid")
	remotes, err := r.Remotes(ctx)
	if err != nil || len(remotes) != 1 || remotes[0].Name != "origin" {
		t.Fatalf("remotes %v %v", remotes, err)
	}
	if err := r.CreateBranch(ctx, "feature/push", "HEAD", true); err != nil {
		t.Fatal(err)
	}
	write(t, clonePath, "push.txt", "push\n")
	git(t, clonePath, "add", ".")
	git(t, clonePath, "commit", "-m", "push change")
	if err := r.Push(ctx, "origin", "feature/push", "feature/published", true); err != nil {
		t.Fatal(err)
	}
	if got := git(t, bare, "rev-parse", "refs/heads/feature/published"); strings.TrimSpace(got) != strings.TrimSpace(git(t, clonePath, "rev-parse", "HEAD")) {
		t.Fatal("push target mismatch")
	}
	peerPath := filepath.Join(t.TempDir(), "peer")
	peer, err := Clone(ctx, bare, peerPath)
	if err != nil {
		t.Fatal(err)
	}
	git(t, peerPath, "config", "user.name", "Test Developer")
	git(t, peerPath, "config", "user.email", "test@example.invalid")
	if err := peer.TrackBranch(ctx, "published", "origin/feature/published"); err != nil {
		t.Fatal(err)
	}
	write(t, clonePath, "next.txt", "next\n")
	git(t, clonePath, "add", ".")
	git(t, clonePath, "commit", "-m", "next")
	if err := r.Push(ctx, "origin", "feature/push", "feature/published", false); err != nil {
		t.Fatal(err)
	}
	if err := peer.PullFF(ctx); err != nil {
		t.Fatal(err)
	}
	if got := git(t, peerPath, "rev-parse", "HEAD"); got != git(t, clonePath, "rev-parse", "HEAD") {
		t.Fatal("pull did not fast forward")
	}
	write(t, peerPath, "peer.txt", "peer\n")
	git(t, peerPath, "add", ".")
	git(t, peerPath, "commit", "-m", "peer diverges")
	write(t, clonePath, "other.txt", "other\n")
	git(t, clonePath, "add", ".")
	git(t, clonePath, "commit", "-m", "source diverges")
	r.Push(ctx, "origin", "feature/push", "feature/published", false)
	before := git(t, peerPath, "rev-parse", "HEAD")
	if err := peer.PullFF(ctx); err == nil {
		t.Fatal("divergent pull accepted")
	}
	if git(t, peerPath, "rev-parse", "HEAD") != before {
		t.Fatal("divergent pull rewrote HEAD")
	}
	if err := peer.Fetch(ctx, "origin"); err != nil {
		t.Fatal(err)
	}
	refs, err := peer.RemoteBranches(ctx)
	if err != nil || len(refs) < 2 {
		t.Fatalf("remote refs %v %v", refs, err)
	}
	for _, ref := range refs {
		if ref.Commit.SHA != ref.SHA || ref.Commit.Subject == "" || ref.Commit.Author == "" || ref.Commit.Date == "" || ref.Commit.Day == "" {
			t.Fatalf("remote ref %q lost its tip metadata: %+v", ref.Name, ref)
		}
	}
	if _, err := Clone(ctx, bare, clonePath); err == nil {
		t.Fatal("clone reused existing directory")
	}
}
func TestDeleteRemoteBranchRemovesRemoteAndTrackingRef(t *testing.T) {
	ctx := context.Background()
	source := fixture(t)
	seed(t, source)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, source, "clone", "--bare", source, bare)
	r, err := Clone(ctx, bare, filepath.Join(t.TempDir(), "cloned"))
	if err != nil {
		t.Fatal(err)
	}
	git(t, r.Root, "config", "user.name", "Test Developer")
	git(t, r.Root, "config", "user.email", "test@example.invalid")
	if err := r.CreateBranch(ctx, "feature/remote-delete", "HEAD", true); err != nil {
		t.Fatal(err)
	}
	if err := r.Push(ctx, "origin", "feature/remote-delete", "feature/remote-delete", true); err != nil {
		t.Fatal(err)
	}
	if err := r.DeleteRemoteBranch(ctx, "origin", "feature/remote-delete"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, bare, "for-each-ref", "--format=%(refname)", "refs/heads/feature"); strings.TrimSpace(got) != "" {
		t.Fatalf("remote branch still exists: %q", got)
	}
	refs, err := r.RemoteBranches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range refs {
		if ref.Name == "origin/feature/remote-delete" {
			t.Fatal("remote-tracking ref kept after delete")
		}
	}
	if err := r.DeleteRemoteBranch(ctx, "--force", "feature/remote-delete"); err == nil {
		t.Fatal("option remote accepted")
	}
	if err := r.DeleteRemoteBranch(ctx, "origin", "--upload-pack=bad"); err == nil {
		t.Fatal("option branch accepted")
	}
}

func TestInitAndRemoteValidation(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "new repository")
	r, err := Init(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Init(ctx, path); err == nil {
		t.Fatal("init reused existing folder")
	}
	if err := r.AddRemote(ctx, "origin", "https://example.invalid/repo.git"); err != nil {
		t.Fatal(err)
	}
	if err := r.AddRemote(ctx, "--upload-pack=bad", "ssh://example.invalid/repo"); err == nil {
		t.Fatal("remote option accepted")
	}
	if err := r.AddRemote(ctx, "unsafe", "ext::echo bad"); err == nil {
		t.Fatal("external shell transport accepted")
	}
}

func TestForceDeleteBranchRemovesUnmergedBranch(t *testing.T) {
	dir := fixture(t)
	seed(t, dir)
	r := openTest(t, dir)
	ctx := context.Background()
	if err := r.CreateBranch(ctx, "unmerged", "HEAD", false); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "checkout", "-q", "unmerged")
	git(t, dir, "commit", "--allow-empty", "-qm", "only here")
	git(t, dir, "checkout", "-q", "main")
	if err := r.DeleteBranch(ctx, "unmerged"); err == nil {
		t.Fatal("unmerged branch deleted without force")
	}
	if err := r.ForceDeleteBranch(ctx, "unmerged"); err != nil {
		t.Fatal(err)
	}
	if git(t, dir, "branch", "--list", "unmerged") != "" {
		t.Fatal("branch still exists")
	}
	if err := r.ForceDeleteBranch(ctx, "main"); err == nil {
		t.Fatal("current branch force deleted")
	}
}

func TestPushForceRewindsRemoteAndKeepsLease(t *testing.T) {
	ctx := context.Background()
	source := fixture(t)
	seed(t, source)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, source, "clone", "--bare", source, bare)
	clonePath := filepath.Join(t.TempDir(), "cloned")
	r, err := Clone(ctx, bare, clonePath)
	if err != nil {
		t.Fatal(err)
	}
	git(t, clonePath, "config", "user.name", "Test Developer")
	git(t, clonePath, "config", "user.email", "test@example.invalid")
	if err := r.CreateBranch(ctx, "feature/force", "HEAD", true); err != nil {
		t.Fatal(err)
	}
	write(t, clonePath, "force.txt", "first\n")
	git(t, clonePath, "add", ".")
	git(t, clonePath, "commit", "-m", "first")
	if err := r.Push(ctx, "origin", "feature/force", "feature/force", true); err != nil {
		t.Fatal(err)
	}
	git(t, clonePath, "reset", "--hard", "HEAD~1")
	if err := r.Push(ctx, "origin", "feature/force", "feature/force", false); err == nil {
		t.Fatal("a rewound branch pushed without force")
	}
	if err := r.PushForce(ctx, "origin", "feature/force", "feature/force"); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(git(t, bare, "rev-parse", "refs/heads/feature/force")), strings.TrimSpace(git(t, clonePath, "rev-parse", "HEAD")); got != want {
		t.Fatalf("force push left %s; want %s", got, want)
	}
	// Another clone advances the branch. Without a fetch the lease is stale,
	// so a second force push must be refused instead of dropping that commit.
	write(t, source, "lease.txt", "remote move\n")
	git(t, source, "add", ".")
	git(t, source, "commit", "-m", "remote move")
	git(t, source, "push", bare, "refs/heads/main:refs/heads/feature/force")
	write(t, clonePath, "local.txt", "local\n")
	git(t, clonePath, "add", ".")
	git(t, clonePath, "commit", "-m", "local move")
	err = r.PushForce(ctx, "origin", "feature/force", "feature/force")
	if err == nil || !strings.Contains(err.Error(), "先获取") {
		t.Fatalf("stale lease push error = %v", err)
	}
	if got, want := strings.TrimSpace(git(t, bare, "rev-parse", "refs/heads/feature/force")), strings.TrimSpace(git(t, source, "rev-parse", "HEAD")); got != want {
		t.Fatal("stale lease push moved the remote")
	}
	if err := r.Fetch(ctx, "origin"); err != nil {
		t.Fatal(err)
	}
	if err := r.PushForce(ctx, "origin", "feature/force", "feature/force"); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.TrimSpace(git(t, bare, "rev-parse", "refs/heads/feature/force")), strings.TrimSpace(git(t, clonePath, "rev-parse", "HEAD")); got != want {
		t.Fatal("force push after fetch did not update the remote")
	}
}

func TestFetchPruneDropsStaleTrackingRefs(t *testing.T) {
	ctx := context.Background()
	source := fixture(t)
	seed(t, source)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, source, "clone", "--bare", source, bare)
	r, err := Clone(ctx, bare, filepath.Join(t.TempDir(), "cloned"))
	if err != nil {
		t.Fatal(err)
	}
	git(t, r.Root, "config", "user.name", "Test Developer")
	git(t, r.Root, "config", "user.email", "test@example.invalid")
	if err := r.CreateBranch(ctx, "stale/branch", "HEAD", true); err != nil {
		t.Fatal(err)
	}
	if err := r.Push(ctx, "origin", "stale/branch", "stale/branch", true); err != nil {
		t.Fatal(err)
	}
	refs, err := r.RemoteBranches(ctx)
	if err != nil || !remoteHas(refs, "origin/stale/branch") {
		t.Fatalf("remote refs %v %v", refs, err)
	}
	git(t, bare, "update-ref", "-d", "refs/heads/stale/branch")
	if err := r.Fetch(ctx, "origin"); err != nil {
		t.Fatal(err)
	}
	if refs, err = r.RemoteBranches(ctx); err != nil || !remoteHas(refs, "origin/stale/branch") {
		t.Fatalf("plain fetch dropped the tracking ref: %v %v", refs, err)
	}
	if err := r.FetchPrune(ctx, "origin"); err != nil {
		t.Fatal(err)
	}
	if refs, err = r.RemoteBranches(ctx); err != nil || remoteHas(refs, "origin/stale/branch") {
		t.Fatalf("prune kept the tracking ref: %v %v", refs, err)
	}
}

func TestPushBranchPublishesWithoutCheckout(t *testing.T) {
	ctx := context.Background()
	source := fixture(t)
	seed(t, source)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, source, "clone", "--bare", source, bare)
	r, err := Clone(ctx, bare, filepath.Join(t.TempDir(), "cloned"))
	if err != nil {
		t.Fatal(err)
	}
	git(t, r.Root, "config", "user.name", "Test Developer")
	git(t, r.Root, "config", "user.email", "test@example.invalid")
	if err := r.CreateBranch(ctx, "side", "HEAD", false); err != nil {
		t.Fatal(err)
	}
	if s, _ := r.Status(ctx); s.Branch != "main" {
		t.Fatalf("branch %q; want to stay on main", s.Branch)
	}
	if err := r.PushBranch(ctx, "origin", "side", "side", true); err != nil {
		t.Fatal(err)
	}
	if got := git(t, bare, "rev-parse", "refs/heads/side"); strings.TrimSpace(got) != strings.TrimSpace(git(t, r.Root, "rev-parse", "side")) {
		t.Fatal("pushed side does not match the local branch")
	}
	branches, err := r.Branches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range branches {
		if branch.Name == "side" && branch.Upstream != "origin/side" {
			t.Fatalf("upstream = %q; want origin/side", branch.Upstream)
		}
	}
	if err := r.PushBranch(ctx, "origin", "side", "published", false); err != nil {
		t.Fatal(err)
	}
	if got := git(t, bare, "rev-parse", "refs/heads/published"); strings.TrimSpace(got) != strings.TrimSpace(git(t, r.Root, "rev-parse", "side")) {
		t.Fatal("renamed destination does not match the local branch")
	}
	if err := r.PushBranch(ctx, "origin", "missing", "missing", false); err == nil {
		t.Fatal("missing branch was pushed")
	}
}

func TestBranchesReportUpstreamCounts(t *testing.T) {
	ctx := context.Background()
	source := fixture(t)
	seed(t, source)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, source, "clone", "--bare", source, bare)
	r, err := Clone(ctx, bare, filepath.Join(t.TempDir(), "cloned"))
	if err != nil {
		t.Fatal(err)
	}
	git(t, r.Root, "config", "user.name", "Test Developer")
	git(t, r.Root, "config", "user.email", "test@example.invalid")
	write(t, r.Root, "local.txt", "local\n")
	git(t, r.Root, "add", ".")
	git(t, r.Root, "commit", "-m", "local work")
	branches, err := r.Branches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mainBranch := func() Branch {
		for _, branch := range branches {
			if branch.Name == "main" {
				return branch
			}
		}
		t.Fatal("main branch missing")
		return Branch{}
	}
	if main := mainBranch(); main.Upstream != "origin/main" || main.Ahead != 1 || main.Behind != 0 || main.UpstreamGone {
		t.Fatalf("after local commit = %+v; want ahead 1", main)
	}
	if main := mainBranch(); main.Commit.SHA != main.SHA || main.Commit.Subject != "local work" || main.Commit.Author == "" || main.Commit.Date == "" || main.Commit.Day == "" {
		t.Fatalf("branch tip metadata = %+v; want the commit it points at", main.Commit)
	}
	peer, err := Clone(ctx, bare, filepath.Join(t.TempDir(), "peer"))
	if err != nil {
		t.Fatal(err)
	}
	git(t, peer.Root, "config", "user.name", "Test Developer")
	git(t, peer.Root, "config", "user.email", "test@example.invalid")
	write(t, peer.Root, "peer.txt", "peer\n")
	git(t, peer.Root, "add", ".")
	git(t, peer.Root, "commit", "-m", "peer work")
	if err := peer.Push(ctx, "origin", "main", "main", false); err != nil {
		t.Fatal(err)
	}
	if err := r.Fetch(ctx, "origin"); err != nil {
		t.Fatal(err)
	}
	branches, err = r.Branches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if main := mainBranch(); main.Ahead != 1 || main.Behind != 1 {
		t.Fatalf("after diverging = %+v; want ahead 1 behind 1", main)
	}
	git(t, r.Root, "config", "branch.main.merge", "refs/heads/vanished")
	branches, err = r.Branches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if main := mainBranch(); !main.UpstreamGone || main.Ahead != 0 || main.Behind != 0 {
		t.Fatalf("after upstream removal = %+v; want gone", main)
	}
}

func remoteHas(refs []Branch, name string) bool {
	for _, ref := range refs {
		if ref.Name == name {
			return true
		}
	}
	return false
}
