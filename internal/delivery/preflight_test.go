package delivery

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestPreflightPassesCleanSyncedTaggedCheckout(t *testing.T) {
	requirePrivateKeyModeSupport(t)
	opts, _ := newPreflightFixture(t)
	t.Chdir(opts.RepoDir)
	opts.RepoDir = ""

	report := runPreflight(t, opts)
	if !report.OK() {
		t.Fatalf("preflight failed: %+v", report.Checks)
	}
}

func TestPreflightRejectsDirtyWorktree(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	// An untracked file should count even if the user hides it in git status.
	preflightFixtureGit(t, opts.RepoDir, "config", "status.showUntrackedFiles", "no")
	if err := os.WriteFile(filepath.Join(opts.RepoDir, "untracked"), []byte("dirty\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	requirePreflightFailure(t, opts, "clean")
}

func TestPreflightRejectsStaleRemote(t *testing.T) {
	opts, remote := newPreflightFixture(t)
	other := t.TempDir()
	preflightFixtureGit(t, other, "clone", "--branch", "main", remote, ".")
	configurePreflightFixtureAuthor(t, other)
	preflightFixtureGit(t, other, "commit", "--allow-empty", "-m", "Advance remote")
	preflightFixtureGit(t, other, "push", "origin", "main")

	t.Run("unmapped tracking ref", func(t *testing.T) {
		preflightFixtureGit(t, opts.RepoDir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/alternate/*")
		requirePreflightFailure(t, opts, "synced")
	})
	t.Run("default tracking ref", func(t *testing.T) {
		preflightFixtureGit(t, opts.RepoDir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
		requirePreflightFailure(t, opts, "synced")
	})
}

func TestPreflightIgnoresGitRepositoryRouting(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	other := t.TempDir()
	preflightFixtureGit(t, other, "init", "--initial-branch=unrelated")
	if err := os.WriteFile(filepath.Join(opts.RepoDir, "release-change"), []byte("dirty\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)

	report := requirePreflightFailure(t, opts, "clean")
	for _, name := range []string{"branch", "synced", "tag"} {
		if check := preflightCheck(t, report, name); !check.Passed {
			t.Errorf("%s did not inspect the chosen checkout: %s", name, check.Detail)
		}
	}
}

func TestPreflightFixtureGitIgnoresGitDir(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	other := t.TempDir()
	preflightFixtureGit(t, other, "init", "--initial-branch=unrelated")
	config := filepath.Join(other, ".git", "config")
	before, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	preflightFixtureGit(t, opts.RepoDir, "config", "fixture.marker", "written to the selected checkout")
	after, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("fixture wrote its configuration to the unrelated repository")
	}
	intended, err := os.ReadFile(filepath.Join(opts.RepoDir, ".git", "config"))
	if err != nil || !strings.Contains(string(intended), "written to the selected checkout") {
		t.Errorf("requested repository did not receive fixture marker: %v", err)
	}
}

func TestPreflightIgnoresGitIndexOverride(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	file := filepath.Join(opts.RepoDir, "alternate-index-only")
	if err := os.WriteFile(file, []byte("not in the release index\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(t.TempDir(), "index")
	t.Setenv("GIT_INDEX_FILE", index)
	cmd := exec.CommandContext(t.Context(), "git", "-C", opts.RepoDir, "add", "alternate-index-only")
	cmd.Env = append(preflightGitEnvironment(cmd.Environ()), "GIT_INDEX_FILE="+index)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add alternate index: %v\n%s", err, out)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}

	report := runPreflight(t, opts)
	if check := preflightCheck(t, report, "clean"); !check.Passed {
		t.Fatalf("checkout was judged using the alternate index: %s", check.Detail)
	}
}

func TestPreflightOmitsCredentialsFromRemoteFailure(t *testing.T) {
	opts, remote := newPreflightFixture(t)
	const marker = "synthetic-password"
	opts.Remote = "https://fixture:" + marker + "@example.invalid/repo"
	// Exercise Git's URL handling using only the local fixture remote.
	preflightFixtureGit(t, opts.RepoDir, "config", "url."+filepath.ToSlash(remote)+".insteadOf", opts.Remote)

	report := requirePreflightFailure(t, opts, "synced")
	for _, check := range report.Checks {
		if strings.Contains(check.Detail, marker) {
			t.Errorf("%s check exposed remote credentials", check.Name)
		}
	}
}

func TestPreflightFetchDisablesTerminalPrompt(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	setPreflightUploadPack(t, opts.RepoDir, `test "$GIT_TERMINAL_PROMPT" = 0 || exit 41
# Force OpenSSH away from the controlling terminal, without a usable askpass program.
test "$SSH_ASKPASS_REQUIRE" = force && test -n "$SSH_ASKPASS" && test ! -x "$SSH_ASKPASS" || exit 42
exec git-upload-pack "$@"`)
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	t.Setenv("SSH_ASKPASS", "/bin/sh")

	report := runPreflight(t, opts)
	if check := preflightCheck(t, report, "synced"); !check.Passed {
		t.Fatalf("non-interactive fetch failed: %s", check.Detail)
	}
}

func TestPreflightFetchTimeoutFailsOnlySyncedCheck(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	setPreflightUploadPack(t, opts.RepoDir, "exec sleep 5")
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
	defer cancel()

	report, err := preflight(ctx, opts, 250*time.Millisecond)
	if err != nil {
		t.Fatalf("expected a timed-out check, not a preflight error: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("fetch did not return before the caller deadline")
	}
	assertPreflightCheckOrder(t, report)
	check := preflightCheck(t, report, "synced")
	if report.OK() || check.Passed || !strings.Contains(check.Detail, "timed out") {
		t.Fatalf("expected a timed-out synced check: %+v", check)
	}
	for _, name := range []string{"branch", "clean", "tag", "public_key"} {
		if check := preflightCheck(t, report, name); !check.Passed {
			t.Errorf("fetch timeout affected %s: %s", name, check.Detail)
		}
	}
}

func setPreflightUploadPack(t *testing.T, repo, body string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("upload-pack fixture uses a Unix shell")
	}
	path := filepath.Join(t.TempDir(), "upload-pack")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	quoted := "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
	preflightFixtureGit(t, repo, "config", "remote.origin.uploadpack", quoted)
}

func TestPreflightRejectsWrongBranch(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	preflightFixtureGit(t, opts.RepoDir, "switch", "-c", "topic")

	requirePreflightFailure(t, opts, "branch")
}

func TestPreflightRejectsTagNotOnHead(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	preflightFixtureGit(t, opts.RepoDir, "commit", "--allow-empty", "-m", "Advance past tag")
	preflightFixtureGit(t, opts.RepoDir, "push", "origin", "main")

	report := requirePreflightFailure(t, opts, "tag")
	if !preflightCheck(t, report, "synced").Passed {
		t.Fatal("expected the checkout to be synced after pushing the new commit")
	}
}

func TestPreflightRejectsMismatchedPrivateKey(t *testing.T) {
	requirePrivateKeyModeSupport(t)
	opts, _ := newPreflightFixture(t)
	opts.PublicKey = writePreflightFixtureKey(t, filepath.Join(t.TempDir(), "other.key"))

	requirePreflightFailure(t, opts, "private_key")
}

func TestPreflightRedactsPrivateKeyPath(t *testing.T) {
	const sentinel = "synthetic-private-key-path"
	err := checkPreflightPrivateKey(filepath.Join(t.TempDir(), sentinel), nil)
	if err == nil || err.Error() == "" {
		t.Fatal("expected a private key file access error with detail")
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Error("private key check exposed the synthetic path")
	}
}

func TestPreflightRejectsPrivateKeyDirectory(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	opts.PrivKeyPath = t.TempDir()
	report := requirePreflightFailure(t, opts, "private_key")
	if check := preflightCheck(t, report, "private_key"); !strings.Contains(check.Detail, "regular file") {
		t.Errorf("private key directory check = %+v", check)
	}
}

func TestPreflightRejectsWorldReadablePrivateKey(t *testing.T) {
	requirePrivateKeyModeSupport(t)
	opts, _ := newPreflightFixture(t)
	if err := os.Chmod(opts.PrivKeyPath, 0o644); err != nil {
		t.Fatal(err)
	}

	requirePreflightFailure(t, opts, "private_key")
}

func TestPreflightReportsIndependentFailures(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	opts.Remote = "missing-remote"
	opts.Tag = "v9.8.7"
	opts.PrivKeyPath = filepath.Join(t.TempDir(), "missing.key")

	report := requirePreflightFailure(t, opts, "synced")
	for _, name := range []string{"tag", "private_key"} {
		if check := preflightCheck(t, report, name); check.Passed || check.Detail == "" {
			t.Errorf("expected %s to fail with detail: %+v", name, check)
		}
	}
	if !preflightCheck(t, report, "public_key").Passed {
		t.Error("expected the public key check to pass")
	}
}

func TestPreflightRejectsInvalidVersionTag(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	opts.Tag = "not-a-version"
	preflightFixtureGit(t, opts.RepoDir, "tag", opts.Tag)

	requirePreflightFailure(t, opts, "tag")
}

func TestPreflightRejectsTagRevisionExpression(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	preflightFixtureGit(t, opts.RepoDir, "tag", "v1.2.3-rc.1")
	opts.Tag = "v1.2.3-rc.1^0"

	requirePreflightFailure(t, opts, "tag")
}

func TestPreflightRejectsBranchRefspecWithoutWritingRef(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	opts.Branch = "main:refs/heads/unexpected"

	requirePreflightFailure(t, opts, "synced")
	cmd := exec.CommandContext(t.Context(), "git", "-C", opts.RepoDir, "branch", "--list", "unexpected")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatal("preflight created a branch from the fetch refspec")
	}
}

func TestPreflightRejectsInvalidPublicKey(t *testing.T) {
	opts, _ := newPreflightFixture(t)
	opts.PublicKey = "invalid"

	requirePreflightFailure(t, opts, "public_key")
}

func TestPreflightRejectsMalformedPrivateKey(t *testing.T) {
	requirePrivateKeyModeSupport(t)
	opts, _ := newPreflightFixture(t)
	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "encoding", data: "invalid private key encoding fixture!"},
		{name: "length", data: base64.StdEncoding.EncodeToString([]byte("short"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(opts.PrivKeyPath, []byte(tc.data), 0o600); err != nil {
				t.Fatal(err)
			}
			report := requirePreflightFailure(t, opts, "private_key")
			if strings.Contains(preflightCheck(t, report, "private_key").Detail, tc.data) {
				t.Error("private key check exposed the key file contents")
			}
		})
	}
}

func TestPreflightRejectsCorruptedPrivateKeySeed(t *testing.T) {
	requirePrivateKeyModeSupport(t)
	opts, _ := newPreflightFixture(t)
	encoded, err := os.ReadFile(opts.PrivKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		t.Fatal(err)
	}
	raw[0] ^= 1 // Keep the embedded public key intact.
	if err := os.WriteFile(opts.PrivKeyPath, []byte(base64.StdEncoding.EncodeToString(raw)), 0o600); err != nil {
		t.Fatal(err)
	}

	requirePreflightFailure(t, opts, "private_key")
}

func TestPreflightReturnsErrorWhenGitCannotStart(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if report, err := Preflight(context.Background(), PreflightOptions{}); err == nil || report.OK() {
		t.Fatalf("expected an execution error and a failing report, got error %v and report %+v", err, report)
	}
}

func TestPreflightEmptyReportFails(t *testing.T) {
	if (PreflightReport{}).OK() {
		t.Fatal("empty report passed")
	}
}

func newPreflightFixture(t *testing.T) (opts PreflightOptions, remote string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(root, "gitconfig"))
	opts.RepoDir = filepath.Join(root, "worktree")
	remote = filepath.Join(root, "remote.git")
	preflightFixtureGit(t, root, "init", "--initial-branch=main", opts.RepoDir)
	configurePreflightFixtureAuthor(t, opts.RepoDir)
	preflightFixtureGit(t, opts.RepoDir, "commit", "--allow-empty", "-m", "Initial commit")
	opts.Tag = "v1.2.3"
	preflightFixtureGit(t, opts.RepoDir, "tag", "-a", opts.Tag, "-m", "Release fixture")
	preflightFixtureGit(t, root, "init", "--bare", "--initial-branch=main", remote)
	preflightFixtureGit(t, opts.RepoDir, "remote", "add", "origin", remote)
	preflightFixtureGit(t, opts.RepoDir, "push", "origin", "main", opts.Tag)

	opts.PrivKeyPath = filepath.Join(root, "update.key")
	opts.PublicKey = writePreflightFixtureKey(t, opts.PrivKeyPath)
	return opts, remote
}

func configurePreflightFixtureAuthor(t *testing.T, repo string) {
	t.Helper()
	preflightFixtureGit(t, repo, "config", "user.name", "Preflight Test")
	preflightFixtureGit(t, repo, "config", "user.email", "preflight@example.invalid")
}

func preflightFixtureGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = preflightGitEnvironment(cmd.Environ())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writePreflightFixtureKey(t *testing.T, path string) string {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.StdEncoding.EncodeToString(priv)
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(pub)
}

func requirePrivateKeyModeSupport(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not report Unix mode 0600")
	}
}

func runPreflight(t *testing.T, opts PreflightOptions) PreflightReport {
	t.Helper()
	report, err := Preflight(t.Context(), opts)
	if err != nil {
		t.Fatalf("preflight returned an error: %v", err)
	}
	assertPreflightCheckOrder(t, report)
	return report
}

func assertPreflightCheckOrder(t *testing.T, report PreflightReport) {
	t.Helper()
	var names []string
	for _, check := range report.Checks {
		names = append(names, check.Name)
	}
	if want := []string{"branch", "clean", "synced", "tag", "public_key", "private_key"}; !slices.Equal(names, want) {
		t.Fatalf("check names = %v, want %v", names, want)
	}
}

func requirePreflightFailure(t *testing.T, opts PreflightOptions, name string) PreflightReport {
	t.Helper()
	report := runPreflight(t, opts)
	check := preflightCheck(t, report, name)
	if report.OK() || check.Passed || check.Detail == "" {
		t.Fatalf("expected %s to fail with detail: %+v", name, report.Checks)
	}
	return report
}

func preflightCheck(t *testing.T, report PreflightReport, name string) PreflightCheck {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("missing %s check", name)
	return PreflightCheck{}
}
