package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"supergit/internal/delivery"
)

func TestRunPreflightReportsChecksAndExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("preflight requires Unix private key permissions")
	}
	args := preflightFixtureArgs(t)
	const allPass = "PASS branch\nPASS clean\nPASS synced\nPASS tag\nPASS public_key\nPASS private_key\n"

	t.Run("all checks pass", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 0 {
			t.Fatalf("exit code = %d, want 0; stdout = %q, stderr = %q", code, &stdout, &stderr)
		}
		if stdout.String() != allPass || stderr.Len() != 0 {
			t.Errorf("stdout = %q, want %q; stderr = %q, want empty", &stdout, allPass, &stderr)
		}
	})

	t.Run("a failed check still reports later checks", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(append(args, "-branch", "other"), &stdout, &stderr); code != 1 {
			t.Fatalf("exit code = %d, want 1; stdout = %q, stderr = %q", code, &stdout, &stderr)
		}
		lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
		if len(lines) != 6 || lines[0] != `FAIL branch: current branch is "main", expected "other"` || lines[5] != "PASS private_key" || stderr.Len() != 0 {
			t.Errorf("expected the failed branch check with detail and subsequent checks; stdout = %q, stderr = %q", &stdout, &stderr)
		}
	})

	t.Run("JSON report", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run(append(args, "-json"), &stdout, &stderr); code != 0 {
			t.Fatalf("exit code = %d, want 0; stdout = %q, stderr = %q", code, &stdout, &stderr)
		}
		var report delivery.PreflightReport
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("decode stdout as a single JSON report: %v; stdout = %q", err, &stdout)
		}
		if !report.OK() || len(report.Checks) != 6 || report.Checks[0].Name != "branch" || stderr.Len() != 0 {
			t.Errorf("expected all six passing checks in JSON; stdout = %q, stderr = %q", &stdout, &stderr)
		}
	})

	t.Run("Git cannot start", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		var stdout, stderr bytes.Buffer
		if code := run(append(args, "-json"), &stdout, &stderr); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if stdout.Len() != 0 || !strings.Contains(stderr.String(), "cannot execute Git preflight") {
			t.Errorf("expected a Git error on stderr only; stdout = %q, stderr = %q", &stdout, &stderr)
		}
	})
}

func TestRunPreflightRequiresTagAndKeys(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"missing tag", []string{"preflight", "-pubkey=x", "-privkey=key.path"}, "-tag"},
		{"missing public key", []string{"preflight", "-tag=v1.2.3", "-privkey=key.path"}, "-pubkey"},
		{"missing private key", []string{"preflight", "-tag=v1.2.3", "-pubkey=x"}, "-privkey"},
		{"invalid flag value", []string{"preflight", "-json=maybe"}, "-json"},
		{"unexpected argument", []string{"preflight", "extra"}, "extra"},
		{"unknown subcommand", []string{"unknown"}, "unknown"},
		{"missing subcommand", nil, "preflight"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(tc.args, &stdout, &stderr); code != 2 {
				t.Fatalf("exit code = %d, want 2; stdout = %q, stderr = %q", code, &stdout, &stderr)
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), tc.want) {
				t.Errorf("expected %q in stderr only; stdout = %q, stderr = %q", tc.want, &stdout, &stderr)
			}
		})
	}
}

func TestRunPreflightRedactsPrivateKeyPath(t *testing.T) {
	const sentinel = "synthetic-private-key-argument"
	args := append(preflightFixtureArgs(t), "-privkey", filepath.Join(t.TempDir(), sentinel))

	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			command := args
			if format == "json" {
				command = append(command, "-json")
			}
			var stdout, stderr bytes.Buffer
			if code := run(command, &stdout, &stderr); code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if strings.Contains(stdout.String(), sentinel) || strings.Contains(stderr.String(), sentinel) {
				t.Error("preflight output exposed the synthetic private key argument")
			}
			if stderr.Len() != 0 {
				t.Error("expected a failed check report on stdout only")
			}
			if format == "text" {
				if !strings.Contains(stdout.String(), "FAIL private_key: ") {
					t.Error("expected a failed private_key check with detail")
				}
				return
			}

			var report delivery.PreflightReport
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatalf("decode stdout as a single JSON report: %v", err)
			}
			if report.OK() || len(report.Checks) != 6 {
				t.Fatal("expected a failing report with all six checks")
			}
			check := report.Checks[5]
			if check.Name != "private_key" || check.Passed || check.Detail == "" {
				t.Error("expected the last check to be a failed private_key check with detail")
			}
		})
	}
}

func TestRunPreflightFixtureIgnoresGitRouting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("preflight requires Unix private key permissions")
	}
	blockedPath := filepath.Join(t.TempDir(), "not-a-git-directory")
	if err := os.WriteFile(blockedPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR",
		"GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_NAMESPACE", "GIT_PREFIX",
	} {
		t.Setenv(name, blockedPath)
	}

	args := preflightFixtureArgs(t)
	var stdout, stderr bytes.Buffer
	if code := run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout = %q, stderr = %q", code, &stdout, &stderr)
	}
}

func preflightFixtureArgs(t *testing.T) []string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(root, "gitconfig"))
	repo := filepath.Join(root, "checkout")
	remote := filepath.Join(root, "remote.git")
	git := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = root
		for _, entry := range cmd.Environ() {
			name, _, _ := strings.Cut(entry, "=")
			switch strings.ToUpper(name) {
			case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR",
				"GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_NAMESPACE", "GIT_PREFIX":
				continue
			}
			cmd.Env = append(cmd.Env, entry)
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "--initial-branch=main", repo)
	git("-C", repo, "-c", "user.name=Preflight Test", "-c", "user.email=preflight@example.invalid", "commit", "--allow-empty", "-m", "Initial commit")
	git("-C", repo, "tag", "v1.2.3")
	git("clone", "--bare", repo, remote)
	git("-C", repo, "remote", "add", "origin", remote)

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(root, "update.key")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(privateKey)), 0o600); err != nil {
		t.Fatal(err)
	}
	return []string{"preflight", "-repo", repo, "-tag", "v1.2.3",
		"-pubkey", base64.StdEncoding.EncodeToString(publicKey), "-privkey", keyPath}
}
