package cmd

import (
	"context"
	"fmt"

	"github.com/fatih/color"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

type ListCmd struct {
	BinaryName string `kong:"arg" help:"The name of the binary to list"`
}

func (cmd ListCmd) Run(ctx context.Context) error {
	ws, err := workSpace(false)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}
	return listBinaryVersions(ws.manPath, cmd.BinaryName)
}

// listBinaryVersions prints the stored versions of a manifest entry, shared by ListCmd
// and its project-scoped tool equivalent.
func listBinaryVersions(manPath, binaryName string) error {
	man, err := manifest.Parse(manPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// TODO: add support to identify current version being used
	fmt.Printf("Stored versions of %s:\n", binaryName)
	entry := man.Binaries[binaryName]
	for version := range entry.Versions {
		if entry.Active == version {
			fmt.Printf("\u2022 %s (current)\n", color.GreenString(version))
		} else {
			fmt.Printf("\u2022 %s\n", color.BlueString(version))
		}
	}

	// TODO: add a flag to show structured full information for all versions available

	return nil
}
