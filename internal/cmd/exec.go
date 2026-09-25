package cmd

import (
	"context"
	"fmt"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

// ExecCmd runs a managed binary by name, resolving its active version the
// same way the PATH shim does. Useful directly, and shares its core with
// the argv0 dispatch in main.go.
type ExecCmd struct {
	Name string   `arg:"" help:"Name of the managed tool to run"`
	Args []string `arg:"" optional:"" passthrough:"" help:"Arguments to pass through"`
}

func (c *ExecCmd) Run(ctx context.Context) error {
	return RunExec(c.Name, c.Args)
}

// RunExec resolves the active version of a managed binary and execs it,
func RunExec(name string, args []string) error {
	_, version, err := resolveActiveVersion(name)
	if err != nil {
		return err
	}
	return execProcess(store.BinaryPath(name, version), args)
}

func resolveActiveVersion(name string) (manPath, version string, err error) {
	localPath, _, err := manifest.Get(true)
	if err != nil {
		return "", "", err
	}
	if entry, err := manifest.GetBinaryDetails(localPath, name); err == nil && entry.Active != "" {
		return localPath, entry.Active, nil
	}

	globalPath, _, err := manifest.Get(false)
	if err != nil {
		return "", "", err
	}
	entry, err := manifest.GetBinaryDetails(globalPath, name)
	if err != nil || entry.Active == "" {
		if localPath == globalPath {
			return "", "", fmt.Errorf(
				"bag: no version of %q configured (checked %s); run `bag add %s` or `bag tool add %s`",
				name, globalPath, name, name,
			)
		}
		return "", "", fmt.Errorf(
			"bag: no version of %q configured (checked %s and %s); run `bag add %s` or `bag tool add %s`",
			name, localPath, globalPath, name, name,
		)
	}
	return globalPath, entry.Active, nil
}
