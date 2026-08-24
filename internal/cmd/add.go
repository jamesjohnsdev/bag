package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

func ErrGlobeInit(err error) error {
	return fmt.Errorf("creating global manifest: %w", err)
}

type AddCmd struct {
	Local  bool   `flag:"" help:"Install a local binary"`
	Source string `arg:"" help:"Path or remote source"`
}

func (cmd *AddCmd) Run() error {
	if cmd.Local {
		version := "local"
		binName := filepath.Base(cmd.Source)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("checking home dir: %w", err)
		}
		binDir := filepath.Join(homeDir, ".local/bin")
		hash, err := store.InstallLocal(binName, version, cmd.Source)
		if err != nil {
			return fmt.Errorf("installing locally: %w", err)
		}
		if err := store.LinkToPath(binName, version, binDir); err != nil {
			return fmt.Errorf("installing locally: %w", err)
		}
		manPath, _, err := manifest.FindManifest()
		if err != nil {
			return err
		}
		// auto-create global manifest if not yet exists
		if _, err := os.Stat(manPath); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(manPath), 0755); err != nil {
				return ErrGlobeInit(err)
			}
			if err := manifest.Write(manPath, &manifest.Manifest{}); err != nil {
				return ErrGlobeInit(err)
			}
			if err := manifest.WriteLock(manifest.FindLock(manPath), &manifest.LockFile{}); err != nil {
				return ErrGlobeInit(err)
			}
		}
		binaryEntry := manifest.BinaryEntry{
			Source:  cmd.Source,
			Version: version,
		}
		if err := manifest.AddBinary(manPath, binName, binaryEntry); err != nil {
			return err
		}
		lockPath := manifest.FindLock(manPath)
		lockEntry := manifest.LockEntry{
			Version: version,
			Hash:    hash,
		}
		if err := manifest.AddLockEntry(lockPath, binName, lockEntry); err != nil {
			return err
		}
		fmt.Printf("installed %s successfully\n", binName)
	}
	// remote path
	return nil
}
