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
	ws, err := workSpace(false)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return whichBinary(ws.manPath, cmd.Name)
}

// whichBinary prints the stored path of a manifest entry's active version, shared by
// WhichCmd and its project-scoped tool equivalent.
func whichBinary(manPath, name string) error {
	man, err := manifest.Parse(manPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	binEntry, ok := man.Binaries[name]
	if !ok {
		return fmt.Errorf("no manifest entry recorded for %s", name)
	}
	if binEntry.Active == "" {
		return fmt.Errorf("no active version recorded for %s", name)
	}

	if store.BinaryExists(name, binEntry.Active) {
		fmt.Printf("%s\n", store.BinaryPath(name, binEntry.Active))
		return nil
	}

	return fmt.Errorf("no binary found for %s", name)
}
