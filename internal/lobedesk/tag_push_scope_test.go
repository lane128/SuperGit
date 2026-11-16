package lobedesk

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"supergit/internal/gitrepo"
)

func TestTagPushRefusalUsesCurrentLanguageAndKeepsGitOutput(t *testing.T) {
	for _, language := range []locale{localeChinese, localeEnglish} {
		a := &app{locale: language}
		for _, target := range []struct {
			err error
			key string
		}{
			{gitrepo.ErrDirtySubmodule, "dirtySubmoduleGuidance"},
			{gitrepo.ErrSubmoduleTargetChanged, "submoduleTargetChanged"},
			{gitrepo.ErrForcePushTargetChanged, "forcePushTargetChanged"},
			{gitrepo.ErrPushTargetChanged, "pushTargetChanged"},
			{gitrepo.ErrPushUpstreamFailed, "pushUpstreamFailed"},
			{gitrepo.ErrRemoteDeleteTargetChanged, "remoteDeleteTargetChanged"},
			{gitrepo.ErrTagPushTargetChanged, "tagPushTargetChanged"},
			{gitrepo.ErrRemoteTargetChanged, "remoteTargetChanged"},
		} {
			if got := a.operationErrorDescription(target.err); got != a.t(target.key) {
				t.Fatalf("%s: missing localized recovery: %q", language, got)
			}
			joined := errors.Join(target.err, errors.New("git show-ref: fixture failure"))
			if got := a.operationErrorDescription(joined); !strings.HasPrefix(got, a.t(target.key)) || !strings.Contains(got, "git show-ref: fixture failure") {
				t.Fatalf("%s: lost original Git output: %q", language, got)
			}
		}
	}
}

func TestTagPushConfirmationKeepsObjectAndRemote(t *testing.T) {
	for _, change := range []string{"annotation", "remote"} {
		t.Run(change, func(t *testing.T) {
			a, _, root := worktreeApp(t)
			run := func(args ...string) string {
				t.Helper()
				out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
				if err != nil {
					t.Fatalf("fixture Git: %v: %s", err, out)
				}
				return string(out)
			}
			first, second := filepath.Join(t.TempDir(), "first.git"), filepath.Join(t.TempDir(), "second.git")
			run("init", "--bare", first)
			run("init", "--bare", second)
			run("remote", "add", "origin", first)
			run("tag", "-a", "--no-sign", "-m", "original", "release")
			var err error
			a.remotes, err = a.repo.Remotes(a.ctx)
			if err != nil {
				t.Fatal(err)
			}
			tags, err := a.repo.Tags(a.ctx)
			if err != nil || len(tags) != 1 {
				t.Fatalf("fixture tag: %v", err)
			}
			a.confirmPushTag(tags[0])
			if !a.confirm.open || !strings.Contains(a.confirm.message, shortSHA(tags[0].ObjectSHA)) {
				t.Fatal("tag push confirmation does not identify original object")
			}
			key := "tagPushTargetChanged"
			if change == "annotation" {
				run("tag", "-f", "-a", "--no-sign", "-m", "replacement", "release")
			} else {
				run("remote", "set-url", "origin", second)
				key = "remoteTargetChanged"
			}
			done := make(chan func(), 1)
			a.runOnMain = func(fn func()) {
				release := make(chan struct{})
				done <- func() { defer close(release); fn() }
				<-release
			}
			a.confirm.action()
			runContinuation(t, done)
			for _, bare := range []string{first, second} {
				out, err := exec.Command("git", "-C", bare, "for-each-ref", "--format=%(refname) %(objectname)").CombinedOutput()
				if err != nil || strings.TrimSpace(string(out)) != "" {
					t.Fatalf("old tag confirmation wrote remote refs: %s %v", out, err)
				}
			}
			found := false
			for _, n := range a.notices[toastMain] {
				if strings.HasPrefix(n.description, a.t(key)) {
					found = true
				}
			}
			if !found {
				t.Fatal("tag push refusal lacks localized recovery")
			}
		})
	}
}
