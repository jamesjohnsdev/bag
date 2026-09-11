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
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	man, err := manifest.Parse(ws.manPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// TODO: add support to identify current version being used
	fmt.Printf("Stored versions of %s:\n", cmd.BinaryName)
	entry := man.Binaries[cmd.BinaryName]
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
