package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

// RunCustom parses custom commands as set by user and runs them
// not to be included in root register
func RunCustom(baseCmd string, args []string) (handled bool, err error) {
	custCmd, ok, err := resolveCustomCommand(baseCmd)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil // will pass through for kong to error
	}
	shArgs := append([]string{"-c", custCmd + ` "$@"`, "--"}, args...)
	c := exec.Command("sh", shArgs...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return true, c.Run()
}

// resolveCustomCommand looks up baseCmd in the local manifest first (same
// as resolveActiveVersion for binaries), falling back to the global one.
func resolveCustomCommand(baseCmd string) (custCmd string, ok bool, err error) {
	localPath, _, err := manifest.Get(true)
	if err != nil {
		return "", false, fmt.Errorf("loading workspace: %w", err)
	}
	localMan, err := manifest.Parse(localPath)
	if err != nil {
		return "", false, fmt.Errorf("parsing manifest: %w", err)
	}
	if custCmd, ok := localMan.Commands[baseCmd]; ok {
		return custCmd, true, nil
	}

	globalPath, _, err := manifest.Get(false)
	if err != nil {
		return "", false, fmt.Errorf("loading workspace: %w", err)
	}
	if globalPath == localPath {
		return "", false, nil
	}
	globalMan, err := manifest.Parse(globalPath)
	if err != nil {
		return "", false, fmt.Errorf("parsing manifest: %w", err)
	}
	custCmd, ok = globalMan.Commands[baseCmd]
	return custCmd, ok, nil
}
