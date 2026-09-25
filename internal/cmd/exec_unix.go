//go:build !windows

package cmd

import (
	"os"
	"syscall"
)

// execProcess replaces the current process image with the binary at path,
// so signals, exit codes, and stdio pass through untouched.
func execProcess(path string, args []string) error {
	argv := append([]string{path}, args...)
	return syscall.Exec(path, argv, os.Environ())
}
