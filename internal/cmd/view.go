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

func (cmd *ViewCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return viewBinary(ws.manPath, cmd.Name)
}

// viewBinary prints the active version details of a manifest entry, shared by ViewCmd
// and its project-scoped tool equivalent.
func viewBinary(manPath, name string) error {
	entry, err := manifest.GetBinaryDetails(manPath, name)
	if err != nil {
		return fmt.Errorf("getting binary details: %w", err)
	}

	activeVersion, ok := entry.Versions[entry.Active]
	if !ok {
		return fmt.Errorf("no active version recorded for %s", name)
	}

	// TODO: improve the results output here.
	fmt.Printf("Source: %s\n", color.BlueString(activeVersion.Source))
	fmt.Printf("Type: %s\n", color.BlueString(string(entry.Type)))
	fmt.Printf("Version: %s\n", color.BlueString(entry.Active))

	return nil
}
