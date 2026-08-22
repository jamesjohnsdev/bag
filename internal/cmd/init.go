package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

type InitCmd struct{}

func (c *InitCmd) Run() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(cwd, manifest.ManName)
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
	err = manifest.WriteLock(lockPath, &manifest.LockFile{
		Entries: map[string]manifest.LockEntry{},
	})
	if err == nil {
		fmt.Println("Intialised successfully")
	}
	return err
}
