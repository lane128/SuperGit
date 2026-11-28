package main

import (
	"errors"
	"os"
	"syscall"
)

func lockReleaseOutput(path string) (*os.File, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	for {
		err := syscall.Flock(int(dir.Fd()), syscall.LOCK_EX)
		if err == nil {
			return dir, nil
		}
		if !errors.Is(err, syscall.EINTR) {
			dir.Close()
			return nil, err
		}
	}
}
