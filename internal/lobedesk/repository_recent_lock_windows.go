package lobedesk

import (
	"os"

	"golang.org/x/sys/windows"
)

func lockRepositoryRecents(file *os.File) error {
	var at windows.Overlapped
	return windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &at)
}
