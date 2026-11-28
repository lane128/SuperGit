//go:build !(darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || windows)

package delivery

import (
	"errors"
	"os"
)

func lockTaskStoreFile(*os.File) error {
	return errors.ErrUnsupported
}
