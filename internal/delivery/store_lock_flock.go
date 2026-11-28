//go:build darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd

package delivery

import (
	"os"
	"syscall"
)

func openTaskStoreLock(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
}

func lockTaskStoreFile(file *os.File) error {
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
		if err != syscall.EINTR {
			return err
		}
	}
}
