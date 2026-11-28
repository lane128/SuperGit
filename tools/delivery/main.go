package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"supergit/internal/delivery"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: delivery preflight [flags]")
		return 2
	}

	switch args[0] {
	case "preflight":
		return runPreflight(args[1:], stdout, stderr)
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

func writePreflightReport(stdout io.Writer, report delivery.PreflightReport, jsonOutput bool) error {
	if jsonOutput {
		return json.NewEncoder(stdout).Encode(report)
	}
	for _, check := range report.Checks {
		status := "PASS"
		if !check.Passed {
			status = "FAIL"
		}
		line := status + " " + check.Name
		if check.Detail != "" {
			line += ": " + check.Detail
		}
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}
