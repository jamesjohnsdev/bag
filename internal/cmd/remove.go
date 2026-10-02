package cmd

import (
	"context"
	"fmt"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type RemoveCmd struct {
	Name  string `arg:"" help:"The name of the binary being removed"`
	Purge bool   `flag:"" help:"Removes underlying stored binary files"`
}

func (cmd *RemoveCmd) Run(ctx context.Context) error {
	ws, err := workSpace(false)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	if cmd.Purge {
		entry, err := findManifestEntry(ws.manPath, cmd.Name)
		if err != nil {
			return fmt.Errorf("getting entry: %w", err)
		}
		if err = purgeBinary(cmd.Name, ws.binDir, entry); err != nil {
			return fmt.Errorf("purging binary: %w", err)
		}
	}
	return removeBinary(ws.manPath, cmd.Name)
}

type RemoveToolCmd RemoveCmd

func (cmd *RemoveToolCmd) Run(ctx context.Context) error {
	ws, err := workSpace(true)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	if err = ws.requireLocal(); err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	if cmd.Purge {
		entry, err := findManifestEntry(ws.manPath, cmd.Name)
		if err != nil {
			return fmt.Errorf("getting entry: %w", err)
		}
		if err = purgeBinary(cmd.Name, ws.binDir, entry); err != nil {
			return fmt.Errorf("purging binary: %w", err)
		}
	}
	return removeBinary(ws.manPath, cmd.Name)
}

// purgeBinary unlinks and removes a stored binary
func purgeBinary(name, binDir string, entry manifest.BinaryEntry) error {
	if err := store.Unlink(name, binDir); err != nil {
		return fmt.Errorf("removing sym link: %w", err)
	}
	if err := store.Remove(name, entry.Active); err != nil {
		_ = store.LinkToPath(name, entry.Active, binDir)
		return fmt.Errorf("removing binary: %w", err)
	}
	return nil
}

func findManifestEntry(manPath, name string) (entry manifest.BinaryEntry, err error) {
	man, err := manifest.Parse(manPath)
	if err != nil {
		return manifest.BinaryEntry{}, fmt.Errorf("parsing manifest: %w", err)
	}
	entry, exists := man.Binaries[name]
	if !exists {
		return manifest.BinaryEntry{}, fmt.Errorf("%s not found in manifest", name)
	}
	return entry, nil
}

// removeBinary removes a binary from the manifest and lock file
func removeBinary(manPath, name string) error {
	entry, err := findManifestEntry(manPath, name)
	if err != nil {
		return fmt.Errorf("getting manifest entry: %w", err)
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
