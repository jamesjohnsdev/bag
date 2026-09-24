package cmd

import (
	"context"
	"fmt"

	"github.com/fatih/color"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type UseCmd struct {
	Binary  string `arg:"" help:"Name of the binary"`
	Version string `arg:"" help:"Version to use"`
}

func (cmd *UseCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	binEntry, err := manifest.GetBinaryDetails(ws.manPath, cmd.Binary)
	if err != nil {
		return fmt.Errorf("getting binary details: %w", err)
	}
	if _, ok := binEntry.Versions[cmd.Version]; !ok {
		return fmt.Errorf("version %s not found in manifest", cmd.Version)
	}
	oldVersion := binEntry.Active

	if err := store.Unlink(cmd.Binary, ws.binDir); err != nil {
		return fmt.Errorf("removing old symlink: %w", err)
	}
	if err := store.LinkToPath(cmd.Binary, cmd.Version, ws.binDir); err != nil {
		_ = store.LinkToPath(cmd.Binary, oldVersion, ws.binDir)
		return fmt.Errorf("creating symlink: %w", err)
	}

	binEntry.Active = cmd.Version
	if err := manifest.AddBinary(ws.manPath, cmd.Binary, binEntry); err != nil {
		_ = store.Unlink(cmd.Binary, ws.binDir)
		_ = store.LinkToPath(cmd.Binary, oldVersion, ws.binDir)
		return fmt.Errorf("updating manifest: %w", err)
	}

	fmt.Printf("Using version: %v", color.GreenString(cmd.Version))
	return nil
}
