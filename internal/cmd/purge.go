package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

type PurgeCmd struct{}

func (cmd *PurgeCmd) Run(context.Context) error {
	var fullManifest manifest.Manifest
	skippedFiles := make(map[string]error)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	if err = filepath.WalkDir(homeDir, func(path string, dir fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dir.IsDir() || dir.Name() != "bag.toml" {
			return nil
		}

		_, err = os.Stat(path)
		if err != nil {
			skippedFiles[path] = err
			return nil
		}

		man, err := manifest.Parse(path)
		if err != nil {
			return fmt.Errorf("parsing manifest: %w", err)
		}

		fullManifest, err = joinActiveManifests(*man, fullManifest)
		if err != nil {
			return fmt.Errorf("joining active manifests: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walking dir: %w", err)
	}
	// then for every directory in store, compare if it exists in composite manifest
	// if !exists: -> Remove All
	// if exists: check version subdirectories and remove if not found in similar one.
	// at the end: print deleted stuff, and also print skipped files
	panic("not implemented")
}

func joinActiveManifests(newManifest, fullManifest manifest.Manifest) (manifest.Manifest, error) {
	for key := range fullManifest.Binaries {
		binEntry := fullManifest.Binaries[key]
		newEntry, err := manifest.RemoveInactiveEntries(binEntry)
		if err != nil {
			return manifest.Manifest{}, fmt.Errorf("removing inactive entries: %w", err)
		}
		fullManifest.Binaries[key] = newEntry
	}

	for key := range newManifest.Binaries {
		fullVal, ok := fullManifest.Binaries[key]
		if !ok {
			binEntry := newManifest.Binaries[key]
			newEntry, err := manifest.RemoveInactiveEntries(binEntry)
			if err != nil {
				return manifest.Manifest{}, fmt.Errorf("removing inactive entries: %w", err)
			}
			fullManifest.Binaries[key] = newEntry
			continue
		}
		versEntry, err := manifest.GetActiveVersEntry(newManifest.Binaries[key])
		if err != nil {
			return manifest.Manifest{}, err
		}
		// This silently ignores case where version source doesn't match
		// not sure if I care enough about this though
		fullVal.Versions[key] = versEntry
	}
	return fullManifest, nil
}
