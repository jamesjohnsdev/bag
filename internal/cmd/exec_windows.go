//go:build windows

package cmd

import "fmt"

// execProcess is not yet implemented on Windows: syscall.Exec has no
// equivalent there, and the PATH shim itself needs a different mechanism
// (a generated wrapper, since Windows lacks the symlink+argv0 trick this
// package relies on elsewhere). Stub kept so the Windows build compiles.
func execProcess(path string, args []string) error {
	return fmt.Errorf("bag: managed binaries are not yet supported on Windows")
}
