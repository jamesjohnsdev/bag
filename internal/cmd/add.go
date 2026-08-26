package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type AddCmd struct {
	Local  bool   `flag:"" help:"Install a local binary"`
	Source string `arg:"" help:"Path or remote source"`
}

func (cmd *AddCmd) Run() error {
	ws, err := workSpace()

	var (
		binName     string
		version     string
		binaryEntry manifest.BinaryEntry
		hash        string
	)

	// download binary
	if cmd.Local {
		version = "local"
		binName = filepath.Base(cmd.Source)
		hash, err = store.InstallLocal(binName, version, cmd.Source)
		if err != nil {
			return fmt.Errorf("installing locally: %w", err)
		}
		if err := store.LinkToPath(binName, version, ws.binDir); err != nil {
			return fmt.Errorf("installing locally: %w", err)
		}

		binaryEntry = manifest.BinaryEntry{
			Source:  cmd.Source,
			Version: version,
		}
	}

	// remote path
	err = postInstall(ws.manPath, binName, version, hash, binaryEntry)
	if err != nil {
		return fmt.Errorf("post-install: %w", err)
	}
	fmt.Printf("installed %s successfully\n", binName)
	return nil
}

// postInstall handles general chores required for manifest and lock after binary installation
func postInstall(manPath, binName, version, hash string, binaryEntry manifest.BinaryEntry) error {
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
	return nil
}
