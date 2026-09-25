package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

// InitCmd is not necessary for initialising a global bag.
// bag will be initialised automatically without this.
type InitCmd struct{}

// Run should pretty much always return an error
// workSpace will automatically create files
func (c *InitCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return initManifest(ws.manPath)
}

type InitToolCmd struct{}

func (c *InitToolCmd) Run(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return initManifest(filepath.Join(cwd, manifest.ManName))
}

// initManifest writes a fresh manifest and lock file at manifestPath, shared by InitCmd
// and InitToolCmd.
func initManifest(manifestPath string) error {
	if _, err := os.Stat(manifestPath); err == nil {
		return fmt.Errorf("already initialised: %s exists", manifestPath)
	}
	if err := manifest.Write(manifestPath, &manifest.Manifest{
		Commands: map[string]string{},
		Binaries: map[string]manifest.BinaryEntry{},
	}); err != nil {
		return err
	}

	lockPath := manifest.FindLock(manifestPath)
	err := manifest.WriteLock(lockPath, &manifest.LockFile{
		Entries: map[string]map[string]manifest.LockEntry{},
	})
	if err == nil {
		fmt.Println("Intialised successfully")
	}
	return err
}
