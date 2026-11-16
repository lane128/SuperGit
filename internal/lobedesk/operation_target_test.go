package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestOperationControlsRejectReplacementOfSameKind(t *testing.T) {
	for _, control := range []string{"abort", "skip", "continue"} {
		t.Run(control, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			run := func(args ...string) string {
				t.Helper()
				out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
				if err != nil {
					t.Fatalf("fixture Git: %v: %s", err, out)
				}
				return string(out)
			}
			write := func(text string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			run("switch", "-c", "topic")
			write("first\n")
			run("commit", "-am", "first")
			first := strings.TrimSpace(run("rev-parse", "HEAD"))
			write("second\n")
			run("commit", "-am", "second")
			run("switch", "main")
			write("ours\n")
			run("commit", "-am", "ours")
			if err := a.repo.Integrate(a.ctx, "cherry-pick", first); err == nil {
				t.Fatal("fixture lacks first conflict")
			}
			var err error
			a.operation, err = a.repo.Operation(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			if control == "abort" {
				a.confirmAbortOperation("cherry-pick")
			} else if control == "skip" {
				a.confirmSkipOperation("cherry-pick")
			}
			if control != "continue" && !a.confirm.open {
				t.Fatal("operation confirmation missing")
			}
			run("cherry-pick", "--abort")
			if err := a.repo.Integrate(a.ctx, "cherry-pick", "topic"); err == nil {
				t.Fatal("fixture lacks replacement conflict")
			}
			before, err := a.repo.Operation(a.ctx)
			if err != nil || before == a.operation {
				t.Fatalf("operation did not change: %v", err)
			}
			if control == "continue" {
				write("replacement resolution\n")
				run("add", "--", "base.txt")
			}
			// A refreshed card must not retarget the pending confirmation.
			if control == "skip" {
				a.operation = before
			}
			head, index := run("rev-parse", "HEAD"), run("ls-files", "--stage")
			bytes, err := os.ReadFile(filepath.Join(root, "base.txt"))
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan func(), 1)
			a.runOnMain = func(fn func()) {
				release := make(chan struct{})
				done <- func() { defer close(release); fn() }
				<-release
			}
			if control == "continue" {
				a.continueOperation("cherry-pick")
			} else {
				a.confirm.action()
			}
			runContinuation(t, done)
			after, err := a.repo.Operation(a.ctx)
			if err != nil || before != after {
				t.Fatalf("old %s confirmation changed replacement operation: %+v -> %+v, %v", control, before, after, err)
			}
			if run("rev-parse", "HEAD") != head || run("ls-files", "--stage") != index {
				t.Fatal("refusal changed HEAD or Index")
			}
			afterBytes, err := os.ReadFile(filepath.Join(root, "base.txt"))
			if err != nil || string(bytes) != string(afterBytes) {
				t.Fatal("refusal changed working bytes")
			}
			found := false
			for _, n := range a.notices[toastMain] {
				if strings.HasPrefix(n.description, a.t("operationTargetChanged")) {
					found = true
				}
			}
			if !found {
				t.Fatal("operation refusal lacks localized recovery")
			}
		})
	}
}

func TestOperationControlRefusalUsesCurrentLanguage(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		a := &app{locale: language}
		if got := a.operationErrorDescription(gitrepo.ErrOperationTargetChanged); got != a.t("operationTargetChanged") {
			t.Fatalf("%s: no localized operation recovery: %q", language, got)
		}
	}
}
