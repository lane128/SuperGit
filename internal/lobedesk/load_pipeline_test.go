package lobedesk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// gitCallLog puts a logging shim for git first on PATH, so an open can be
// inspected by the commands it runs rather than by guessed timings.
func gitCallLog(t *testing.T) string {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	shimDir := t.TempDir()
	log := filepath.Join(shimDir, "calls.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$SUPERGIT_GIT_LOG\"\nexec \"$SUPERGIT_REAL_GIT\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(shimDir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SUPERGIT_GIT_LOG", log)
	t.Setenv("SUPERGIT_REAL_GIT", realGit)
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func gitCalls(t *testing.T, log, prefix string) []string {
	t.Helper()
	data, err := os.ReadFile(log)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var found []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, " "+prefix) || strings.HasPrefix(line, prefix) {
			found = append(found, line)
		}
	}
	return found
}

func initLoadFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %q: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-b", "main")
	runGit("config", "user.name", "Test Developer")
	runGit("config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "--", "base.txt")
	runGit("commit", "-m", "initial")
	return dir
}

func loadRepository(t *testing.T, root string) *app {
	t.Helper()
	a := &app{ctx: context.Background(), page: "history", locale: localeEnglish}
	done := make(chan func(), 64)
	a.runOnMain = func(fn func()) { done <- fn }
	a.readRepository(root, true)
	deadline := time.After(20 * time.Second)
	for a.loading || len(done) > 0 {
		select {
		case fn := <-done:
			fn()
		case <-deadline:
			t.Fatal("opening the repository never finished")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	return a
}

// Opening a repository already reads the working tree status for the worktree
// row. The local watcher's baseline must reuse that read: a second `git status`
// doubles the slowest part of opening a large repository.
func TestOpeningRepositoryRunsStatusOnce(t *testing.T) {
	root := initLoadFixture(t)
	log := gitCallLog(t)
	a := loadRepository(t, root)
	if a.repo == nil || a.loadErr != "" {
		t.Fatalf("repository did not open: %q", a.loadErr)
	}
	if calls := gitCalls(t, log, "status"); len(calls) != 1 {
		t.Fatalf("opening ran %d status commands, want 1:\n%s", len(calls), strings.Join(calls, "\n"))
	}
	if len(a.commits) != 1 || a.commits[0].Subject != "initial" {
		t.Fatalf("history was not applied: %+v", a.commits)
	}
	if a.localWatch.stamp == "" {
		t.Fatal("opening did not record the local baseline")
	}
}
