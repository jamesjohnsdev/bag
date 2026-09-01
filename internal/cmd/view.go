package cmd

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/jamesjohnsdev/bag/internal/manifest"
)

type ViewCmd struct {
	Name string `arg:"" help:"Name of the binary to view"`
}

func (cmd *ViewCmd) Run(context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	man, err := manifest.Parse(ws.manPath)
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}

	entry, ok := man.Binaries[cmd.Name]
	if !ok {
		return fmt.Errorf("unable to find manifest entry for %s", cmd.Name)
	}

	// TODO: improve the results output here.
	fmt.Printf("Source: %s\n", color.BlueString(entry.Source))
	fmt.Printf("Type: %s\n", color.BlueString(entry.Type))
	fmt.Printf("Version: %s\n", color.BlueString(entry.Version))

	return nil
}
