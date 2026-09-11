package cmd

import (
	"context"
	"fmt"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type WhichCmd struct {
	Name string `arg:"" help:"The name of the binary to find"`
}

func (cmd *WhichCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	man, err := manifest.Parse(ws.manPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	binEntry, ok := man.Binaries[cmd.Name]
	if !ok {
		return fmt.Errorf("no manifest entry recorded for %s", cmd.Name)
	}
	if binEntry.Active == "" {
		return fmt.Errorf("no active version recorded for %s", cmd.Name)
	}

	if store.BinaryExists(cmd.Name, binEntry.Active) {
		fmt.Printf("%s\n", store.BinaryPath(cmd.Name, binEntry.Active))
		return nil
	}

	return fmt.Errorf("no binary found for %s", cmd.Name)
}
