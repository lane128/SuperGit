package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"supergit/internal/delivery"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: delivery <preflight|verify-release> [flags]")
		return 2
	}

	switch args[0] {
	case "preflight":
		return runPreflight(args[1:], stdout, stderr)
	case "verify-release":
		return runVerifyRelease(args[1:], stdout, stderr, nil)
	default:
		fmt.Fprintf(stderr, "unknown subcommand %q\n", args[0])
		return 2
	}
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
