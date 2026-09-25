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
	ws, err := workSpace(false)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return removeBinary(ws.manPath, ws.binDir, cmd.Name)
}

// removeBinary unlinks and removes a stored binary and its manifest/lock entries,
// shared by RemoveCmd and its project-scoped tool equivalent.
func removeBinary(manPath, binDir, name string) error {
	man, err := manifest.Parse(manPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}
	entry, exists := man.Binaries[name]
	if !exists {
		return fmt.Errorf("%s not found in manifest", name)
	}

	if err := store.Unlink(name, binDir); err != nil {
		return fmt.Errorf("removing sym link: %w", err)
	}
	if err := store.Remove(name, entry.Active); err != nil {
		_ = store.LinkToPath(name, entry.Active, binDir)
		return fmt.Errorf("removing binary: %w", err)
	}

	// not reverting binary changes - figure out handling later
	// broken state if fails
	if err := manifest.RemoveBinary(manPath, name); err != nil {
		return fmt.Errorf("removing manifest entry: %w", err)
	}
	if err := manifest.RemoveLockEntry(manifest.FindLock(manPath), name); err != nil {
		_ = manifest.AddBinary(manPath, name, entry)
		return fmt.Errorf("removing lock entry: %w", err)
	}

	fmt.Printf("successfully removed: %s", name)
	return nil
}
