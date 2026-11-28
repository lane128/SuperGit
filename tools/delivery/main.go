package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"supergit/internal/delivery"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: delivery <preflight|verify-release|start|check|advance|status|block|resume|revoke|authorize|queue> [flags]")
		return 2
	}

	switch args[0] {
	case "preflight":
		return runPreflight(args[1:], stdout, stderr)
	case "verify-release":
		return runVerifyRelease(args[1:], stdout, stderr, nil)
	case "start", "check", "advance", "status", "block", "resume", "revoke", "authorize", "queue":
		return runTaskCommand(args[0], args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown subcommand %q\n", args[0])
		return 2
	}
}

func runTaskCommand(command string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	id := flags.String("id", "", "task ID")
	root := flags.String("state", "", "delivery state directory (defaults to SUPERGIT_DELIVERY_STATE or the user config directory)")

	type operation func(*delivery.Store, time.Time) error
	var perform operation
	var validate func() error
	var required []string
	if command != "status" {
		required = append(required, "id")
	}
	update := func(mutate func(*delivery.Task, time.Time) error) operation {
		return func(store *delivery.Store, now time.Time) error {
			task, err := store.Update(*id, func(task *delivery.Task) error { return mutate(task, now) })
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(stdout, task.ID, task.Phase, task.Status)
			return err
		}
	}

	switch command {
	case "start":
		var request delivery.StartRequest
		flags.StringVar(&request.Request, "request", "", "request description")
		flags.StringVar(&request.Repository, "repo", "", "repository identifier")
		flags.StringVar(&request.BaseBranch, "branch", "main", "base branch")
		flags.StringVar(&request.AuthorizedBy, "by", "", "authorizer")
		flags.StringVar(&request.VersionPolicy, "version-policy", "", "version policy (patch, minor, or exact)")
		flags.StringVar(&request.Version, "version", "", "authorized version")
		flags.StringVar(&request.Channel, "channel", "", "release channel")
		acceptance := flags.String("acceptance", "", "comma-separated acceptance items")
		flags.BoolVar(&request.Release, "release", true, "authorize release")
		required = append(required, "request", "by", "version-policy")
		validate = func() error {
			if !request.Release {
				return errors.New("-release=false is unsupported")
			}
			return nil
		}
		perform = func(store *delivery.Store, now time.Time) error {
			for item := range strings.SplitSeq(*acceptance, ",") {
				if item = strings.TrimSpace(item); item != "" {
					request.AcceptanceItems = append(request.AcceptanceItems, item)
				}
			}
			task, err := delivery.NewTask(*id, request, now)
			if err != nil {
				return err
			}
			if err := store.Create(task); err != nil {
				return err
			}
			_, err = fmt.Fprintln(stdout, task.ID, task.Phase)
			return err
		}
	case "check":
		var check delivery.Check
		flags.StringVar(&check.Name, "name", "", "check name")
		flags.StringVar(&check.Candidate, "candidate", "", "candidate covered by the check")
		flags.BoolVar(&check.Passed, "passed", false, "record a passing check")
		flags.StringVar(&check.Evidence, "evidence", "", "check evidence")
		flags.StringVar(&check.ReleaseCommit, "release-commit", "", "merged commit covered by release evidence")
		planRevision := flags.Int("release-plan-revision", -1, "release plan revision covered by release evidence")
		required = append(required, "name")
		perform = update(func(task *delivery.Task, now time.Time) error {
			if *planRevision >= 0 {
				check.ReleasePlanRevision = planRevision
			}
			check.RecordedAt = now
			return task.RecordCheck(check)
		})
	case "advance":
		to := flags.String("to", "", "next task phase")
		candidate := flags.String("candidate", "", "candidate for this transition")
		required = append(required, "to")
		perform = func(store *delivery.Store, now time.Time) error {
			task, err := store.AdvanceTask(*id, delivery.Phase(*to), *candidate, now)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(stdout, task.Phase)
			return err
		}
	case "status":
		jsonOutput := flags.Bool("json", false, "print status as JSON")
		perform = func(store *delivery.Store, _ time.Time) error {
			return writeTaskStatus(stdout, store, *id, *jsonOutput)
		}
	case "block":
		reason := flags.String("reason", "", "reason the task is blocked")
		required = append(required, "reason")
		perform = update(func(task *delivery.Task, now time.Time) error { return task.Block(*reason, now) })
	case "resume":
		perform = update((*delivery.Task).Resume)
	case "revoke":
		by := flags.String("by", "", "person revoking authorization")
		required = append(required, "by")
		perform = update(func(task *delivery.Task, now time.Time) error { return task.Revoke(*by, now) })
	case "authorize":
		var revision delivery.AuthorizationRevision
		flags.StringVar(&revision.By, "by", "", "authorizer")
		flags.StringVar(&revision.Reason, "reason", "", "reason for the revision")
		flags.BoolVar(&revision.Release, "release", false, "authorize release")
		flags.StringVar(&revision.VersionPolicy, "version-policy", "", "version policy (patch, minor, or exact)")
		flags.StringVar(&revision.Version, "version", "", "authorized version")
		required = append(required, "by", "reason", "version-policy")
		perform = update(func(task *delivery.Task, now time.Time) error { return task.AuthorizeRevision(revision, now) })
	case "queue":
		acquire := flags.Bool("acquire", false, "acquire the release window")
		release := flags.Bool("release", false, "release the release window")
		by := flags.String("by", "", "maintainer recovering a blocked release window")
		reason := flags.String("reason", "", "reason for recovering a blocked release window")
		validate = func() error {
			if *acquire == *release {
				return errors.New("exactly one of -acquire and -release is required")
			}
			return nil
		}
		perform = func(store *delivery.Store, now time.Time) error {
			if *acquire {
				if err := store.AcquireReleaseWindow(*id, now); err != nil {
					return err
				}
				_, err := fmt.Fprintln(stdout, *id, "acquired the release window")
				return err
			}
			err := store.ReleaseReleaseWindow(*id)
			if errors.Is(err, delivery.ErrReleaseWindowBlocked) {
				err = store.RecoverReleaseWindow(*id, *by, *reason)
			}
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(stdout, *id, "released the release window")
			return err
		}
	}

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "%s: unexpected argument %q\n", command, flags.Arg(0))
		return 2
	}
	for _, name := range required {
		if strings.TrimSpace(flags.Lookup(name).Value.String()) == "" {
			fmt.Fprintf(stderr, "%s: -%s is required\n", command, name)
			return 2
		}
	}
	if validate != nil {
		if err := validate(); err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", command, err)
			return 2
		}
	}

	state, err := deliveryStateRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", command, err)
		return 1
	}
	state, err = validateStateRoot(state)
	if err != nil {
		fmt.Fprintf(stderr, "%s: invalid delivery state directory: %v\n", command, err)
		return 2
	}
	if err := perform(delivery.NewStore(state), time.Now().UTC()); err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", command, err)
		if errors.Is(err, delivery.ErrRecoveryDetailsRequired) {
			return 2
		}
		return 1
	}
	return 0
}

func deliveryStateRoot(root string) (string, error) {
	if root != "" {
		return root, nil
	}
	if root = os.Getenv("SUPERGIT_DELIVERY_STATE"); root != "" {
		return root, nil
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "SuperGit", "delivery"), nil
}

func validateStateRoot(root string) (string, error) {
	state, err := validateStateDirectory(root)
	if err != nil {
		return "", err
	}
	if _, err := validateStateDirectory(filepath.Join(state, "tasks")); err != nil {
		return "", fmt.Errorf("tasks directory: %w", err)
	}
	return state, nil
}

// validateStateDirectory finds the destination's checkout even when the process
// starts elsewhere. A worktree's .git entry may be a directory or a file.
func validateStateDirectory(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	ancestor, suffix := absolute, ""
	for {
		_, err := os.Lstat(ancestor)
		if err == nil {
			break
		}
		parent := filepath.Dir(ancestor)
		if !errors.Is(err, os.ErrNotExist) || parent == ancestor {
			return "", err
		}
		suffix = filepath.Join(filepath.Base(ancestor), suffix)
		ancestor = parent
	}
	ancestor, err = filepath.EvalSymlinks(ancestor)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(ancestor)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", ancestor)
	}
	for dir := ancestor; ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return "", fmt.Errorf("state must be outside Git repository %q", dir)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if filepath.Dir(dir) == dir {
			return filepath.Join(ancestor, suffix), nil
		}
	}
}

type taskSummary struct {
	ID      string          `json:"id"`
	Phase   delivery.Phase  `json:"phase"`
	Status  delivery.Status `json:"status"`
	Version string          `json:"version"`
	Tag     string          `json:"tag"`
}

type taskStatusResponse struct {
	ID            string                     `json:"id"`
	Request       string                     `json:"request"`
	Repository    string                     `json:"repository"`
	BaseBranch    string                     `json:"base_branch"`
	Tier          string                     `json:"tier"`
	Phase         delivery.Phase             `json:"phase"`
	Status        delivery.Status            `json:"status"`
	Candidate     string                     `json:"candidate"`
	Version       string                     `json:"version"`
	Tag           string                     `json:"tag"`
	PR            int                        `json:"pr"`
	BlockedReason string                     `json:"blocked_reason"`
	Authorization taskStatusAuthorization    `json:"authorization"`
	Checks        map[string]taskStatusCheck `json:"checks"`
	HistoryCount  int                        `json:"history_count"`
}

type taskStatusAuthorization struct {
	AuthorizedBy        string `json:"authorized_by"`
	Release             bool   `json:"release"`
	Revoked             bool   `json:"revoked"`
	ReleasePlanRevision int    `json:"release_plan_revision"`
	VersionPolicy       string `json:"version_policy"`
	Version             string `json:"version"`
}

type taskStatusCheck struct {
	Name                string         `json:"name"`
	Phase               delivery.Phase `json:"phase"`
	Candidate           string         `json:"candidate"`
	Passed              bool           `json:"passed"`
	Evidence            string         `json:"evidence"`
	RecordedAt          time.Time      `json:"recorded_at"`
	ReleaseCommit       string         `json:"release_commit,omitempty"`
	ReleasePlanRevision *int           `json:"release_plan_revision,omitempty"`
}

func writeTaskStatus(out io.Writer, store *delivery.Store, id string, jsonOutput bool) error {
	if id == "" {
		tasks, err := store.List()
		if err != nil {
			return err
		}
		if jsonOutput {
			summaries := make([]taskSummary, 0, len(tasks))
			for _, task := range tasks {
				summaries = append(summaries, taskSummary{task.ID, task.Phase, task.Status, task.Version, task.Tag})
			}
			return json.NewEncoder(out).Encode(summaries)
		}
		for _, task := range tasks {
			if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", task.ID, task.Phase, task.Status, task.Version, task.Tag); err != nil {
				return err
			}
		}
		return nil
	}

	task, err := store.Load(id)
	if err != nil {
		return err
	}
	if jsonOutput {
		checks := make(map[string]taskStatusCheck, len(task.Checks))
		for name, check := range task.Checks {
			checks[name] = taskStatusCheck{
				Name: check.Name, Phase: check.Phase, Candidate: check.Candidate,
				Passed: check.Passed, Evidence: check.Evidence, RecordedAt: check.RecordedAt,
				ReleaseCommit: check.ReleaseCommit, ReleasePlanRevision: check.ReleasePlanRevision,
			}
		}
		return json.NewEncoder(out).Encode(taskStatusResponse{
			ID: task.ID, Request: task.Request, Repository: task.Repository, BaseBranch: task.BaseBranch,
			Tier: task.Tier, Phase: task.Phase, Status: task.Status, Candidate: task.Candidate,
			Version: task.Version, Tag: task.Tag, PR: task.PR, BlockedReason: task.BlockedReason,
			Authorization: taskStatusAuthorization{
				AuthorizedBy:        task.Authorization.AuthorizedBy,
				Release:             task.Authorization.Release,
				Revoked:             task.Authorization.RevokedAt != nil,
				ReleasePlanRevision: task.Authorization.ReleasePlanRevision,
				VersionPolicy:       task.Authorization.VersionPolicy,
				Version:             task.Authorization.Version,
			},
			Checks: checks, HistoryCount: len(task.History),
		})
	}
	if _, err := fmt.Fprintf(out, "Task %s\nPhase %s\nStatus %s\nCandidate %q\nVersion %q\nTag %q\nPR %d\nBlocked reason %q\nAuthorization release=%t revoked=%t\nHistory entries %d\n",
		task.ID, task.Phase, task.Status, task.Candidate, task.Version, task.Tag, task.PR, task.BlockedReason,
		task.Authorization.Release, task.Authorization.RevokedAt != nil, len(task.History)); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(task.Checks)) {
		check := task.Checks[name]
		if _, err := fmt.Fprintf(out, "Check %q phase=%s candidate=%q passed=%t evidence=%q\n", name, check.Phase, check.Candidate, check.Passed, check.Evidence); err != nil {
			return err
		}
	}
	return nil
}

func runPreflight(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("preflight", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var opts delivery.PreflightOptions
	flags.StringVar(&opts.RepoDir, "repo", ".", "Git checkout directory")
	flags.StringVar(&opts.Branch, "branch", "main", "release branch")
	flags.StringVar(&opts.Remote, "remote", "origin", "Git remote name")
	flags.StringVar(&opts.Tag, "tag", "", "release tag (required)")
	flags.StringVar(&opts.PublicKey, "pubkey", "", "base64 update signing public key (required)")
	flags.StringVar(&opts.PrivKeyPath, "privkey", "", "update signing private key file path (required)")
	jsonOutput := flags.Bool("json", false, "print the report as JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "preflight: unexpected argument %q\n", flags.Arg(0))
		return 2
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{"tag", opts.Tag},
		{"pubkey", opts.PublicKey},
		{"privkey", opts.PrivKeyPath},
	} {
		if required.value == "" {
			fmt.Fprintf(stderr, "preflight: -%s is required\n", required.name)
			return 2
		}
	}

	report, err := delivery.Preflight(context.Background(), opts)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := writePreflightReport(stdout, report, *jsonOutput); err != nil {
		fmt.Fprintf(stderr, "preflight: write report: %v\n", err)
		return 1
	}
	if !report.OK() {
		return 1
	}
	return 0
}

func runVerifyRelease(args []string, stdout, stderr io.Writer, verifySignature func(context.Context, string) error) int {
	flags := flag.NewFlagSet("verify-release", flag.ContinueOnError)
	flags.SetOutput(stderr)
	opts := delivery.VerifyOptions{VerifySignature: verifySignature}
	flags.StringVar(&opts.OutDir, "out", "", "release output directory (required)")
	flags.StringVar(&opts.ArchivePath, "archive", "", "archive path (default from the manifest)")
	flags.StringVar(&opts.ManifestPath, "manifest", "", "manifest path (default update.json in the output directory)")
	flags.StringVar(&opts.PublicKey, "pubkey", "", "base64 update signing public key (required)")
	flags.StringVar(&opts.Platform, "platform", "darwin-arm64", "manifest platform to verify")
	jsonOutput := flags.Bool("json", false, "print the report as JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "verify-release: unexpected argument %q\n", flags.Arg(0))
		return 2
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{"out", opts.OutDir},
		{"pubkey", opts.PublicKey},
	} {
		if required.value == "" {
			fmt.Fprintf(stderr, "verify-release: -%s is required\n", required.name)
			return 2
		}
	}

	report, verifyErr := delivery.VerifyRelease(context.Background(), opts)
	if err := writeReleaseReport(stdout, report, *jsonOutput, verifyErr == nil); err != nil {
		fmt.Fprintf(stderr, "verify-release: write report: %v\n", err)
		return 1
	}
	if verifyErr != nil {
		return 1
	}
	return 0
}

func writeReleaseReport(stdout io.Writer, report delivery.ReleaseVerification, jsonOutput, verified bool) error {
	if jsonOutput {
		return json.NewEncoder(stdout).Encode(report)
	}
	if err := writeChecks(stdout, report.Checks); err != nil || !verified {
		return err
	}
	_, err := fmt.Fprintf(stdout, "Version %s\nBuild %d\nArchive %s\nSHA-256 %s\nBundle %s\n",
		report.Version, report.Build, report.ArchivePath, report.SHA256, report.BundlePath)
	return err
}

func writePreflightReport(stdout io.Writer, report delivery.PreflightReport, jsonOutput bool) error {
	if jsonOutput {
		return json.NewEncoder(stdout).Encode(report)
	}
	return writeChecks(stdout, report.Checks)
}

func writeChecks(stdout io.Writer, checks []delivery.PreflightCheck) error {
	lineBreaks := strings.NewReplacer("\r", `\r`, "\n", `\n`)
	for _, check := range checks {
		status := "PASS"
		if !check.Passed {
			status = "FAIL"
		}
		line := status + " " + check.Name
		if check.Detail != "" {
			line += ": " + lineBreaks.Replace(check.Detail)
		}
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}
