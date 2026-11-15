//go:build !unix

package gitrepo

import "os/exec"

func configureProcess(cmd *exec.Cmd) {}
