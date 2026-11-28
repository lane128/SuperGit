//go:build !(darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || windows)

package delivery

import (
	"errors"
	"os"
)

func openTaskStoreLock(string) (*os.File, error) {
	return nil, errors.ErrUnsupported
}

func lockTaskStoreFile(*os.File) error {
	return errors.ErrUnsupported
}
