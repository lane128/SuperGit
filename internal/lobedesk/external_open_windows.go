package lobedesk

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var shellExecutePath = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")

// ShellExecute opens an independent desktop process. In particular, cmd.exe
// gets a usable console rather than the GUI process's absent standard handles.
func runExternalOpen(cmd *exec.Cmd) error {
	if cmd.Err != nil {
		return cmd.Err
	}
	file, err := syscall.UTF16PtrFromString(cmd.Path)
	if err != nil {
		return err
	}
	args := make([]string, 0, len(cmd.Args)-1)
	for _, arg := range cmd.Args[1:] {
		args = append(args, syscall.EscapeArg(arg))
	}
	parameters, err := syscall.UTF16PtrFromString(strings.Join(args, " "))
	if err != nil {
		return err
	}
	directory, err := syscall.UTF16PtrFromString(cmd.Dir)
	if err != nil {
		return err
	}
	result, _, _ := shellExecutePath.Call(0, 0, uintptr(unsafe.Pointer(file)), uintptr(unsafe.Pointer(parameters)), uintptr(unsafe.Pointer(directory)), 1)
	if result <= 32 {
		return fmt.Errorf("Windows could not open the path (ShellExecute code %d)", result)
	}
	return nil
}
