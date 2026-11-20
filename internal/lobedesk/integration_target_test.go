package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"supergit/internal/gitrepo"
)

func TestBranchIntegrationFooterKeepsBothConfirmationTargets(t *testing.T) {
	for _, kind := range []string{"merge", "rebase"} {
		for _, change := range []string{"incoming", "current"} {
			t.Run(kind+"/"+change, func(t *testing.T) {
				a, _, root := worktreeApp(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				a.ctx = ctx
				run := func(args ...string) string {
					t.Helper()
					out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
					if err != nil {
						t.Fatalf("fixture Git: %v: %s", err, out)
					}
					return string(out)
				}
				add := func(path string) {
					t.Helper()
					if err := os.WriteFile(filepath.Join(root, path), []byte(path+"\n"), 0600); err != nil {
						t.Fatal(err)
					}
					run("add", "--", path)
					run("commit", "-m", path)
				}
				run("switch", "-c", "topic")
				add("first.txt")
				selected := strings.TrimSpace(run("rev-parse", "HEAD"))
				add("second.txt")
				newer := strings.TrimSpace(run("rev-parse", "HEAD"))
				run("branch", "topic-next")
				run("switch", "main")
				run("branch", "-f", "topic", selected)
				var err error
				a.status, err = a.repo.Status(a.ctx)
				if err != nil {
					t.Fatal(err)
				}
				selectedBranch := gitrepo.Branch{Name: "topic", SHA: selected}
				label := a.t("mergeIntoCurrent")
				if kind == "rebase" {
					label = a.t("rebaseOnto")
				}
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					a.branchDetailActions(c, branchDetailSelection{kind: branchDetailLocal, branch: selectedBranch})
				}, 960, 640)
				button, ok := tt.Find(label)
				if !ok {
					t.Fatalf("%s footer action missing", kind)
				}
				tt.ClickAt(button.X+button.W/2, button.Y+button.H/2)
				if !a.confirm.open {
					t.Fatal("integration confirmation missing")
				}
				if !strings.Contains(a.confirm.message, "topic ("+shortSHA(selected)+")") || !strings.Contains(a.confirm.message, "main") {
					t.Fatalf("confirmation should name the incoming commit and current branch: %q", a.confirm.message)
				}
				if change == "incoming" {
					run("branch", "-f", "topic", newer)
				} else {
					run("switch", "-c", "other")
					// Refreshing UI state must not retarget the pending action.
					a.status, err = a.repo.Status(a.ctx)
					if err != nil {
						t.Fatal(err)
					}
				}
				head, refs, index := run("rev-parse", "HEAD"), run("for-each-ref", "--format=%(refname) %(objectname)"), run("ls-files", "--stage")
				done := make(chan func(), 1)
				a.runOnMain = func(fn func()) {
					release := make(chan struct{})
					done <- func() { defer close(release); fn() }
					<-release
				}
				a.confirm.action()
				runContinuation(t, done)
				if change == "incoming" {
					if strings.TrimSpace(run("rev-parse", "HEAD")) != selected {
						t.Fatal("confirmation followed changed branch instead of original commit")
					}
					if _, err := os.Stat(filepath.Join(root, "second.txt")); !os.IsNotExist(err) {
						t.Fatal("integration included later unconfirmed changes")
					}
				} else {
					if run("rev-parse", "HEAD") != head || run("for-each-ref", "--format=%(refname) %(objectname)") != refs || run("ls-files", "--stage") != index {
						t.Fatal("old confirmation integrated into a different current branch")
					}
					found := false
					for _, n := range a.notices[toastMain] {
						if n.description == a.t("integrationTargetChanged") {
							found = true
						}
					}
					if !found {
						t.Fatal("changed-current refusal lacks localized recovery")
					}
				}
			})
		}
	}
}
