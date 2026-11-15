package desktop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"supergit/internal/gitrepo"
	"testing"
)

func TestTagActionCapturesSelectedCommitAndDispatchesLocalCreation(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.invalid")
	git("commit", "--allow-empty", "-m", "selected older commit")
	selected := git("rev-parse", "HEAD")
	git("commit", "--allow-empty", "-m", "current HEAD")
	head := git("rev-parse", "HEAD")
	ctx := context.Background()
	r, err := gitrepo.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	status, err := r.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a := &app{repo: r, page: "history", selectedSHA: selected, data: snapshot{status: status}}
	a.guard.Switch()
	a.startForm(actionDraft{Kind: actionCreateTag, Second: selected})
	a.form.First = "release/候选"
	if a.page != "action" || !a.form.editable() || a.form.network() {
		t.Fatal("tag did not enter an editable local form")
	}
	target := a.capture(a.form)
	a.form.First, a.form.Second, a.selectedSHA = "other", head, head
	if target.Draft.First != "release/候选" || target.Draft.Second != selected {
		t.Fatal("confirmed name/selected SHA changed with form or history")
	}
	for _, value := range []string{"refs/tags/release/候选", selected, "本地", "轻量", "不覆盖"} {
		if !strings.Contains(target.Draft.description(), value) {
			t.Fatalf("confirmation omits %q", value)
		}
	}
	if !target.valid(r, &a.guard) || target.Root != r.Root {
		t.Fatal("confirmation did not capture repository")
	}
	if err := target.Draft.run(ctx, r); err != nil {
		t.Fatal(err)
	}
	if git("rev-parse", "refs/tags/release/候选") != selected || git("cat-file", "-t", "refs/tags/release/候选") != "commit" || git("rev-parse", "HEAD") != head {
		t.Fatal("dispatcher created wrong target/type or moved HEAD")
	}
	// Existing HEAD protection must also apply to this new dispatcher case.
	target.Draft.First = "must-not-exist"
	git("commit", "--allow-empty", "-m", "external advance")
	if err := target.Draft.run(ctx, r); err == nil || !strings.Contains(err.Error(), "已变化") {
		t.Fatalf("changed HEAD was accepted: %v", err)
	}
	if git("tag", "--list", "must-not-exist") != "" {
		t.Fatal("stale confirmation wrote a tag")
	}
	a.guard.Switch()
	if target.valid(r, &a.guard) {
		t.Fatal("old repository generation accepted")
	}
}

func TestTagFormCancellationDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	r, err := gitrepo.Init(context.Background(), filepath.Join(dir, "repo"))
	if err != nil {
		t.Fatal(err)
	}
	a := &app{repo: r, page: "history"}
	a.startForm(actionDraft{Kind: actionCreateTag, First: "cancelled", Second: strings.Repeat("a", 40)})
	a.cancelAction()
	if a.page != "history" || a.confirmation != nil || a.guard.writing {
		t.Fatal("form cancellation did not return without a write")
	}
	out, err := exec.Command("git", "-C", r.Root, "tag", "--list").CombinedOutput()
	if err != nil || len(out) != 0 {
		t.Fatalf("cancel wrote a tag: %s %v", out, err)
	}
}
