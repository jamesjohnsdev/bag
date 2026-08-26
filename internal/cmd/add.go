package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type AddCmd struct {
	Local  bool   `flag:"" help:"Install a local binary"`
	Source string `arg:"" help:"Path or remote source"`
}

func (cmd *AddCmd) Run() error {
	// TODO: some of this could be moved else where as repeated logic
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("checking home dir: %w", err)
	}
	binDir := filepath.Join(homeDir, ".local/bin")
	manPath, _, err := manifest.Get(false)
	if err != nil {
		return err
	}

	if cmd.Local {
		version := "local"
		binName := filepath.Base(cmd.Source)
		hash, err := store.InstallLocal(binName, version, cmd.Source)
		if err != nil {
			return fmt.Errorf("installing locally: %w", err)
		}
		if err := store.LinkToPath(binName, version, binDir); err != nil {
			return fmt.Errorf("installing locally: %w", err)
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
