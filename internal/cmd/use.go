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
	ws, err := workSpace(false)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return useBinaryVersion(ws.manPath, ws.binDir, cmd.Binary, cmd.Version)
}

// useBinaryVersion switches the active version of a known manifest entry, shared by
// UseCmd and its project-scoped tool equivalent.
func useBinaryVersion(manPath, binDir, binary, version string) error {
	binEntry, err := manifest.GetBinaryDetails(manPath, binary)
	if err != nil {
		return fmt.Errorf("getting binary details: %w", err)
	}
	if _, ok := binEntry.Versions[version]; !ok {
		return fmt.Errorf("version %s not found in manifest", version)
	}
	oldVersion := binEntry.Active

	if err := store.Unlink(binary, binDir); err != nil {
		return fmt.Errorf("removing old symlink: %w", err)
	}
	if err := store.LinkToPath(binary, version, binDir); err != nil {
		_ = store.LinkToPath(binary, oldVersion, binDir)
		return fmt.Errorf("creating symlink: %w", err)
	}

	binEntry.Active = version
	if err := manifest.AddBinary(manPath, binary, binEntry); err != nil {
		_ = store.Unlink(binary, binDir)
		_ = store.LinkToPath(binary, oldVersion, binDir)
		return fmt.Errorf("updating manifest: %w", err)
	}

	fmt.Printf("Using version: %v", color.GreenString(version))
	return nil
}
