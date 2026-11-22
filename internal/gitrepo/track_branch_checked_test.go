package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func checkedTrackFixture(t *testing.T) (*Repository, string) {
	t.Helper()
	source := fixture(t)
	seed(t, source)
	git(t, source, "checkout", "-b", "topic")
	write(t, source, "base.txt", "remote base\n")
	write(t, source, "remote.txt", "remote tip\n")
	git(t, source, "add", ".")
	git(t, source, "commit", "-m", "remote topic")
	git(t, source, "checkout", "main")
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, source, "clone", "--bare", source, bare)
	r, err := Clone(context.Background(), bare, filepath.Join(t.TempDir(), "clone"))
	if err != nil {
		t.Fatal(err)
	}
	return r, strings.TrimSpace(git(t, r.Root, "rev-parse", "refs/remotes/origin/topic"))
}

// Losing the displayed-tip check would allow moved or deleted candidates to act.
func TestTrackBranchCheckedRejectsChangedTargetWithoutMutation(t *testing.T) {
	for _, mode := range []string{"invalid", "empty", "moved", "deleted", "collision", "dirty", "staged", "untracked", "ignored"} {
		t.Run(mode, func(t *testing.T) {
			r, sha := checkedTrackFixture(t)
			switch mode {
			case "invalid":
				sha = "not-an-object"
			case "empty":
				sha = ""
			case "moved":
				git(t, r.Root, "update-ref", "refs/remotes/origin/topic", "HEAD")
			case "deleted":
				git(t, r.Root, "update-ref", "-d", "refs/remotes/origin/topic")
			case "collision":
				git(t, r.Root, "branch", "destination")
				git(t, r.Root, "config", "branch.destination.remote", "origin")
				git(t, r.Root, "config", "branch.destination.merge", "refs/heads/main")
			case "dirty":
				write(t, r.Root, "base.txt", "preserve dirty\n")
			case "staged":
				write(t, r.Root, "base.txt", "preserve staged\n")
				git(t, r.Root, "add", "base.txt")
			case "untracked":
				write(t, r.Root, "remote.txt", "preserve untracked\n")
			case "ignored":
				write(t, r.Root, ".git/info/exclude", "remote.txt\n")
				write(t, r.Root, "remote.txt", "preserve ignored\n")
			}
			snapshot := func() string {
				var files strings.Builder
				for _, name := range []string{"base.txt", "remote.txt", "private.txt", ".git/config", ".git/index"} {
					b, err := os.ReadFile(filepath.Join(r.Root, name))
					if err != nil && !os.IsNotExist(err) {
						t.Fatal(err)
					}
					files.WriteString(name)
					files.Write(b)
				}
				return git(t, r.Root, "rev-parse", "HEAD") + git(t, r.Root, "symbolic-ref", "HEAD") + git(t, r.Root, "show-ref") + files.String()
			}
			before := snapshot()
			err := r.TrackBranchChecked(context.Background(), "destination", "origin/topic", sha)
			if err == nil {
				t.Fatal("unsafe remote checkout accepted")
			}
			if mode == "invalid" || mode == "empty" || mode == "moved" || mode == "deleted" {
				if !errors.Is(err, ErrSwitchTargetChanged) {
					t.Fatalf("unrecognizable changed target: %v", err)
				}
			}
			if after := snapshot(); after != before {
				t.Fatal("rejected checkout changed HEAD, refs, configuration, index or working bytes")
			}
		})
	}
}

func TestTrackBranchCheckedCarriesNonConflictingChanges(t *testing.T) {
	r, sha := checkedTrackFixture(t)
	write(t, r.Root, "notes.txt", "carry across track\n")
	if err := r.TrackBranchChecked(context.Background(), "my-topic", "origin/topic", sha); err != nil {
		t.Fatalf("non-conflicting remote checkout failed: %v", err)
	}
	if got := strings.TrimSpace(git(t, r.Root, "branch", "--show-current")); got != "my-topic" {
		t.Fatalf("branch = %q, want my-topic", got)
	}
	if b, err := os.ReadFile(filepath.Join(r.Root, "notes.txt")); err != nil || string(b) != "carry across track\n" {
		t.Fatalf("non-conflicting bytes changed: %q, %v", b, err)
	}
}

func TestTrackBranchCheckedCreatesExactUpstream(t *testing.T) {
	r, sha := checkedTrackFixture(t)
	if err := r.TrackBranchChecked(context.Background(), "my-topic", "origin/topic", sha); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		args []string
		want string
	}{
		{[]string{"rev-parse", "HEAD"}, sha},
		{[]string{"branch", "--show-current"}, "my-topic"},
		{[]string{"rev-parse", "--abbrev-ref", "@{upstream}"}, "origin/topic"},
		{[]string{"config", "branch.my-topic.remote"}, "origin"},
		{[]string{"config", "branch.my-topic.merge"}, "refs/heads/topic"},
	} {
		if got := strings.TrimSpace(git(t, r.Root, check.args...)); got != check.want {
			t.Fatalf("%v = %q, want %q", check.args, got, check.want)
		}
	}
}
