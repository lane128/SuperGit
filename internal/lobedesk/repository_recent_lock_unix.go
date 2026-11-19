//go:build darwin || linux

package lobedesk

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockRepositoryRecents(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}
