package cmd

import (
	"context"
	"fmt"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type RemoveCmd struct {
	Name string `arg:"" help:"The name of the binary being removed"`
}

func (cmd *RemoveCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	man, err := manifest.Parse(ws.manPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}
	entry, exists := man.Binaries[cmd.Name]
	if !exists {
		return fmt.Errorf("%s not found in manifest", cmd.Name)
	}

	if err := store.Unlink(cmd.Name, ws.binDir); err != nil {
		return fmt.Errorf("removing sym link: %w", err)
	}
	if err := store.Remove(cmd.Name, entry.Active); err != nil {
		_ = store.LinkToPath(cmd.Name, entry.Active, ws.binDir)
		return fmt.Errorf("removing binary: %w", err)
	}

	// not reverting binary changes - figure out handling later
	// broken state if fails
	if err := manifest.RemoveBinary(ws.manPath, cmd.Name); err != nil {
		return fmt.Errorf("removing manifest entry: %w", err)
	}
	if err := manifest.RemoveLockEntry(manifest.FindLock(ws.manPath), cmd.Name); err != nil {
		_ = manifest.AddBinary(ws.manPath, cmd.Name, entry)
		return fmt.Errorf("removing lock entry: %w", err)
	}

	fmt.Printf("successfully removed: %s", cmd.Name)
	return nil
}
