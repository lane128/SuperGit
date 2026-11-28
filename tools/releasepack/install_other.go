//go:build !darwin

package main

import (
	"errors"
	"os"
)

func lockReleaseOutput(string) (*os.File, error) {
	return nil, errors.New("packaging macOS .app bundles requires macOS ditto")
}
