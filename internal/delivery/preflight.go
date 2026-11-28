// Package delivery provides checks for the release workflow.
package delivery

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"supergit/internal/updater"
)

// PreflightOptions identifies the release checkout, tag, and signing keys.
type PreflightOptions struct {
	RepoDir     string // default "."
	Branch      string // default "main"
	Remote      string // default "origin"
	Tag         string
	PublicKey   string
	PrivKeyPath string
}

// PreflightCheck records the result of one release check.
type PreflightCheck struct {
	Name   string
	Passed bool
	Detail string
}

// PreflightReport contains the checks in the order they were performed.
type PreflightReport struct {
	Checks []PreflightCheck
}

// OK reports whether a nonempty report has only passing checks.
func (r PreflightReport) OK() bool {
	if len(r.Checks) == 0 {
		return false
	}
	for _, check := range r.Checks {
		if !check.Passed {
			return false
		}
	}
	return true
}

var errPreflightGit = errors.New("cannot execute Git preflight")

// Preflight checks a release checkout against the freshly fetched branch,
// verifies the release tag, and checks the signing key pair.
func Preflight(ctx context.Context, opts PreflightOptions) (PreflightReport, error) {
	if opts.RepoDir == "" {
		opts.RepoDir = "."
	}
	if opts.Branch == "" {
		opts.Branch = "main"
	}
	if opts.Remote == "" {
		opts.Remote = "origin"
	}

	git := preflightGit{ctx: ctx, repoDir: opts.RepoDir}
	var pub ed25519.PublicKey
	checks := []struct {
		name string
		run  func() error
	}{
		{"branch", func() error {
			branch, err := git.run("branch", "--show-current")
			if err != nil {
				return err
			}
			if branch != opts.Branch {
				return fmt.Errorf("current branch is %q, expected %q", branch, opts.Branch)
			}
			return nil
		}},
		{"clean", func() error {
			status, err := git.run("status", "--porcelain", "--untracked-files=normal")
			if err != nil {
				return err
			}
			if status != "" {
				return errors.New("worktree has uncommitted changes")
			}
			return nil
		}},
		{"synced", func() error { return git.checkSynced(opts.Remote, opts.Branch) }},
		{"tag", func() error { return git.checkTag(opts.Tag) }},
		{"public_key", func() error {
			var err error
			pub, err = updater.ParsePublicKey(opts.PublicKey)
			return err
		}},
		{"private_key", func() error { return checkPreflightPrivateKey(opts.PrivKeyPath, pub) }},
	}

	report := PreflightReport{Checks: make([]PreflightCheck, 0, len(checks))}
	for _, check := range checks {
		err := check.run()
		if errors.Is(err, errPreflightGit) {
			return PreflightReport{}, err
		}
		result := PreflightCheck{Name: check.name, Passed: err == nil}
		if err != nil {
			result.Detail = err.Error()
		}
		report.Checks = append(report.Checks, result)
	}
	return report, nil
}

type preflightGit struct {
	ctx     context.Context
	repoDir string
}

func (git preflightGit) run(args ...string) (string, error) {
	cmd := exec.CommandContext(git.ctx, "git", append([]string{"-C", git.repoDir}, args...)...)
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	if ctxErr := git.ctx.Err(); ctxErr != nil {
		return "", fmt.Errorf("%w: git %s: %w", errPreflightGit, args[0], ctxErr)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// Git diagnostics may include credentials from a remote URL.
		return "", fmt.Errorf("git %s failed: %w", args[0], exitErr)
	}
	return "", fmt.Errorf("%w: git %s: %w", errPreflightGit, args[0], err)
}

func (git preflightGit) resolve(ref string) (string, error) {
	sha, err := git.run("rev-parse", "--verify", "--end-of-options", ref)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", ref, err)
	}
	return sha, nil
}

func (git preflightGit) checkSynced(remote, branch string) error {
	branchRef := "refs/heads/" + branch
	if _, err := git.run("check-ref-format", branchRef); err != nil {
		return fmt.Errorf("validate branch name %q: %w", branch, err)
	}
	if _, err := git.run("fetch", "--quiet", "--no-tags", "--", remote, branchRef); err != nil {
		return err
	}
	head, err := git.resolve("HEAD")
	if err != nil {
		return err
	}
	tracked, err := git.resolve("refs/remotes/" + remote + "/" + branch)
	if err != nil {
		return err
	}
	// A custom fetch refspec can leave the standard tracking ref stale.
	fetched, err := git.resolve("FETCH_HEAD")
	if err != nil {
		return err
	}
	if head != tracked || head != fetched {
		return fmt.Errorf("HEAD is not synced with %s/%s", remote, branch)
	}
	return nil
}

func (git preflightGit) checkTag(tag string) error {
	if _, err := updater.ParseVersion(strings.TrimPrefix(tag, "v")); err != nil {
		return fmt.Errorf("invalid release tag: %w", err)
	}
	tagRef := "refs/tags/" + tag
	if _, err := git.run("check-ref-format", tagRef); err != nil {
		return fmt.Errorf("validate tag name %q: %w", tag, err)
	}
	tagCommit, err := git.resolve(tagRef + "^{}")
	if err != nil {
		return err
	}
	head, err := git.resolve("HEAD")
	if err != nil {
		return err
	}
	if tagCommit != head {
		return fmt.Errorf("tag %q does not point to HEAD", tag)
	}
	return nil
}

func checkPreflightPrivateKey(path string, pub ed25519.PublicKey) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat private key: %w", err)
	}
	if info.Mode() != 0o600 {
		return fmt.Errorf("private key must be a regular file with mode 0600 (got %v)", info.Mode())
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	raw, err := base64.StdEncoding.AppendDecode(nil, bytes.TrimSpace(encoded))
	if err != nil {
		return errors.New("invalid private key base64 encoding")
	}
	if len(raw) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid private key length %d", len(raw))
	}
	// Recompute the public half from the seed before trusting the stored key.
	priv := ed25519.NewKeyFromSeed(raw[:ed25519.SeedSize])
	if !priv.Equal(ed25519.PrivateKey(raw)) {
		return errors.New("private key contains an inconsistent public key")
	}
	if !pub.Equal(priv.Public()) {
		return errors.New("private key does not match the configured public key")
	}
	return nil
}
