package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"supergit/internal/delivery"
	"supergit/internal/updater"
)

func TestRunVerifyReleaseReportsAndExits(t *testing.T) {
	args, archivePath, digest := verifyReleaseFixture(t)
	verifySignature := func(context.Context, string) error { return nil }
	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	const manifestDetail = "version 1.2.3, build 42"
	sizeDetail := fmt.Sprintf("%d bytes", info.Size())
	passPrefix := "PASS manifest: " + manifestDetail + "\nPASS size: " + sizeDetail + "\n"
	passThroughZip := passPrefix + "PASS sha256: " + digest + "\nPASS ed25519: signature verified\nPASS zip_safety: zip safe\n"

	t.Run("text success", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := runVerifyRelease(args[1:], &stdout, &stderr, verifySignature)
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("exit code = %d, want 0; stdout = %q, stderr = %q", code, &stdout, &stderr)
		}
		if !strings.HasPrefix(stdout.String(), passThroughZip+"PASS extract: ") ||
			!strings.Contains(stdout.String(), "SuperGit.app\nPASS codesign: code signature verified\n") {
			t.Errorf("missing ordered passing checks with details; stdout = %q", &stdout)
		}
		for _, want := range []string{"Version 1.2.3", "Build 42", "Archive " + archivePath, "SHA-256 " + digest, "Bundle ", "SuperGit.app"} {
			if !strings.Contains(stdout.String(), want) {
				t.Errorf("missing %q; stdout = %q", want, &stdout)
			}
		}
	})

	t.Run("JSON success", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := runVerifyRelease(append(args[1:], "-json"), &stdout, &stderr, verifySignature)
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("exit code = %d, want 0; stdout = %q, stderr = %q", code, &stdout, &stderr)
		}
		var report delivery.ReleaseVerification
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("decode stdout as a single JSON report: %v; stdout = %q", err, &stdout)
		}
		if report.Version != "1.2.3" || report.Build != 42 || report.ArchivePath != archivePath || report.SHA256 != digest || len(report.Checks) != 7 || filepath.Base(report.BundlePath) != "SuperGit.app" {
			t.Errorf("unexpected release report: %+v", report)
		}
		for _, check := range report.Checks {
			if !check.Passed || check.Detail == "" {
				t.Errorf("expected a passing check with detail: %+v", check)
			}
		}
	})

	t.Run("multiline codesign failure", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := runVerifyRelease(args[1:], &stdout, &stderr, func(context.Context, string) error {
			return errors.New("signature rejected\nin subcomponent")
		})
		if code != 1 || stderr.Len() != 0 {
			t.Fatalf("exit code = %d, want 1; stdout = %q, stderr = %q", code, &stdout, &stderr)
		}
		if !strings.HasPrefix(stdout.String(), passThroughZip+"PASS extract: ") ||
			!strings.Contains(stdout.String(), "SuperGit.app\nFAIL codesign: ") ||
			!strings.Contains(stdout.String(), "in subcomponent") || strings.Count(stdout.String(), "\n") != 7 {
			t.Errorf("expected one line per check with the codesign detail; stdout = %q", &stdout)
		}
	})

	t.Run("tampered archive", func(t *testing.T) {
		data, err := os.ReadFile(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		data[0] ^= 1
		if err := os.WriteFile(archivePath, data, 0o600); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 1 || stderr.Len() != 0 {
			t.Fatalf("exit code = %d, want 1; stdout = %q, stderr = %q", code, &stdout, &stderr)
		}
		if !strings.HasPrefix(stdout.String(), passPrefix+"FAIL sha256: ") || strings.Count(stdout.String(), "\n") != 3 {
			t.Errorf("expected the completed and failed checks; stdout = %q", &stdout)
		}

		stdout.Reset()
		if code := run(append(args, "-json"), &stdout, &stderr); code != 1 || stderr.Len() != 0 {
			t.Fatalf("exit code = %d, want 1; stdout = %q, stderr = %q", code, &stdout, &stderr)
		}
		var report delivery.ReleaseVerification
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("decode stdout as a single JSON report: %v; stdout = %q", err, &stdout)
		}
		if len(report.Checks) != 3 ||
			report.Checks[0] != (delivery.PreflightCheck{Name: "manifest", Passed: true, Detail: manifestDetail}) ||
			report.Checks[1] != (delivery.PreflightCheck{Name: "size", Passed: true, Detail: sizeDetail}) ||
			report.Checks[2].Name != "sha256" || report.Checks[2].Passed || report.Checks[2].Detail == "" {
			t.Errorf("unexpected release checks: %+v", report.Checks)
		}
	})

	t.Run("missing public key", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run([]string{"verify-release", "-out", t.TempDir(), "-json"}, &stdout, &stderr); code != 2 {
			t.Fatalf("exit code = %d, want 2; stdout = %q, stderr = %q", code, &stdout, &stderr)
		}
		if stdout.Len() != 0 || !strings.Contains(stderr.String(), "-pubkey is required") {
			t.Errorf("expected a usage error on stderr; stdout = %q, stderr = %q", &stdout, &stderr)
		}
	})
}

func verifyReleaseFixture(t *testing.T) (args []string, archivePath, digest string) {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"TMPDIR", "TEMP", "TMP"} {
		t.Setenv(name, root)
	}
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for _, file := range []struct{ name, contents string }{
		{"SuperGit.app/Contents/Info.plist", `<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>cc.supermix.supergit</string>
<key>CFBundleExecutable</key><string>supergit</string>
<key>CFBundleShortVersionString</key><string>1.2.3</string>
<key>CFBundleVersion</key><string>42</string>
</dict></plist>`},
		{"SuperGit.app/Contents/MacOS/supergit", "fixture executable"},
	} {
		entry, err := zw.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, file.contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	archivePath = filepath.Join(root, "fixture.zip")
	if err := os.WriteFile(archivePath, archive.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive.Bytes())
	digest = hex.EncodeToString(sum[:])
	manifest, err := json.Marshal(updater.Manifest{
		Version: "1.2.3",
		Build:   42,
		Platforms: map[string]updater.PlatformAsset{"darwin-amd64": {
			URL:     "https://example.invalid/releases/unavailable.zip",
			SHA256:  digest,
			Ed25519: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, sum[:])),
			Size:    int64(archive.Len()),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, "fixture.json")
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	return []string{"verify-release", "-out", root, "-manifest", manifestPath, "-archive", archivePath,
		"-platform", "darwin-amd64", "-pubkey", base64.StdEncoding.EncodeToString(pub)}, archivePath, digest
}

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
