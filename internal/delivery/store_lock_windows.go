package delivery

import (
	"os"
	"syscall"
	"unsafe"
)

var lockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")

func lockTaskStoreFile(file *os.File) error {
	// Use a synchronous exclusive lock on the first byte of the lock file.
	const exclusiveLock = 0x2
	var offset syscall.Overlapped
	ok, _, err := lockFileEx.Call(file.Fd(), exclusiveLock, 0, 1, 0, uintptr(unsafe.Pointer(&offset)))
	if ok == 0 {
		return err
	}
	return nil
}
