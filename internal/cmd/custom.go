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
	// TODO: handle local manifests
	ws, err := workSpace()
	if err != nil {
		return false, fmt.Errorf("loading workspace: %w", err)
	}
	man, err := manifest.Parse(ws.manPath)
	if err != nil {
		return false, fmt.Errorf("parsing manifest: %w", err)
	}
	custCmd, ok := man.Commands[baseCmd]
	if !ok {
		return false, nil // will pass through for kong to error
	}
	shArgs := append([]string{"-c", custCmd + ` "$@"`, "--"}, args...)
	c := exec.Command("sh", shArgs...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return true, c.Run()
}
