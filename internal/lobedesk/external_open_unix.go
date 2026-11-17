//go:build !windows

package lobedesk

import (
	"fmt"
	"os/exec"
	"strings"
)

type externalOpenDiagnostic struct {
	text strings.Builder
}

func (b *externalOpenDiagnostic) Write(p []byte) (int, error) {
	n := len(p)
	if remaining := (16 << 10) - b.text.Len(); remaining > 0 {
		b.text.Write(p[:min(len(p), remaining)])
	}
	return n, nil
}

func runExternalOpen(cmd *exec.Cmd) error {
	var diagnostic externalOpenDiagnostic
	cmd.Stderr = &diagnostic
	if err := cmd.Run(); err != nil {
		if details := strings.TrimSpace(diagnostic.text.String()); details != "" {
			return fmt.Errorf("%w: %s", err, details)
		}
		return err
	}
	return nil
}
