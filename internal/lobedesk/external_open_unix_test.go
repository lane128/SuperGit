//go:build !windows

package lobedesk

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalOpenFailureKeepsBoundedDiagnosticOutput(t *testing.T) {
	launcher := filepath.Join(t.TempDir(), "failed-launcher")
	if err := os.WriteFile(launcher, []byte("#!/bin/sh\nprintf 'No application is associated with this file\\n' >&2\ni=0\nwhile [ $i -lt 5000 ]; do printf 'diagnostic line\\n' >&2; i=$((i+1)); done\nexit 3\n"), 0700); err != nil {
		t.Fatal(err)
	}
	err := runExternalOpen(exec.Command(launcher))
	if err == nil || !strings.Contains(err.Error(), "No application is associated") || !strings.Contains(err.Error(), "exit status 3") {
		t.Fatalf("launch failure lost its cause: %v", err)
	}
	if len(err.Error()) > 17000 {
		t.Fatalf("launch diagnostic is unbounded: %d bytes", len(err.Error()))
	}
}
