package desktop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"supergit/internal/gitrepo"
	"testing"
)

func TestPushConfirmationRejectsChangedRemote(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return string(out)
	}
	ctx := context.Background()
	root := t.TempDir()
	r, err := gitrepo.Init(ctx, filepath.Join(root, "repo"))
	if err != nil {
		t.Fatal(err)
	}
	git(r.Root, "config", "user.name", "Test")
	git(r.Root, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(r.Root, "file"), []byte("base\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git(r.Root, "add", ".")
	git(r.Root, "commit", "-m", "base")
	oldRemote, newRemote := filepath.Join(root, "confirmed.git"), filepath.Join(root, "unconfirmed.git")
	git(root, "init", "--bare", oldRemote)
	git(root, "init", "--bare", newRemote)
	if err := r.AddRemote(ctx, "origin", oldRemote); err != nil {
		t.Fatal(err)
	}
	remotes, err := r.Remotes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a := &app{repo: r, data: snapshot{remotes: remotes}}
	target := a.capture(actionDraft{Kind: actionPush, First: "origin", Second: "main", Third: "main"})
	git(r.Root, "remote", "set-url", "origin", newRemote)
	if err := target.Draft.run(ctx, r); err == nil {
		t.Error("push used unconfirmed remote configuration")
	}
	if got := git(newRemote, "for-each-ref", "refs/heads"); got != "" {
		t.Error("unconfirmed remote updated")
	}
	git(r.Root, "remote", "set-url", "origin", oldRemote)
	if err := target.Draft.run(ctx, r); err != nil {
		t.Fatalf("confirmed remote refused: %v", err)
	}
}
