package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type PurgeCmd struct {
	Verbose bool `flag:"" help:"Print additional information"`
}

func (cmd *PurgeCmd) Run(context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	fullManifest, skippedFiles, err := collectActiveManifests(homeDir)
	if err != nil {
		return err
	}

	deletedStores, err := purgeStore(fullManifest)
	if err != nil {
		return err
	}
	if cmd.Verbose {
		for file, err := range skippedFiles {
			fmt.Printf("Skipped %s: %s\n", file, color.YellowString(err.Error()))
			slog.Warn("skipped manifest during purge", "path", file, "error", err)
		}
		for store, versions := range deletedStores {
			fmt.Printf("Deleted %s: %s\n", store, color.YellowString(strings.Join(versions, ", ")))
			slog.Info("purged stored versions", "binary", store, "versions", versions)
		}
	}
	if len(deletedStores) == 0 {
		fmt.Println(color.GreenString("Stores are clean. No stores purged."))
	} else if len(deletedStores) == 1 {
		fmt.Println(color.GreenString("Successfully purged 1 store"))
	} else {
		fmt.Printf(color.GreenString("Successfully purged %d stores\n"), len(deletedStores))
	}
	slog.Info("purge completed", "stores", len(deletedStores))
	return nil
}

func collectActiveManifests(homeDir string) (manifest.Manifest, map[string]error, error) {
	fullManifest := manifest.Manifest{Binaries: make(map[string]manifest.BinaryEntry)}
	skippedFiles := make(map[string]error)
	if walkErr := filepath.WalkDir(homeDir, func(path string, dir fs.DirEntry, err error) error {
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

		fullManifest, err = mergeActiveManifestVersions(*man, fullManifest)
		if err != nil {
			return fmt.Errorf("merging active manifests: %w", err)
		}

		return nil
	}); walkErr != nil {
		return manifest.Manifest{}, skippedFiles, fmt.Errorf("walking dir: %w", walkErr)
	}
	return fullManifest, skippedFiles, nil
}

func purgeStore(fullManifest manifest.Manifest) (map[string][]string, error) {
	deletedStores := make(map[string][]string)
	binDirs, err := os.ReadDir(store.Root())
	if err != nil {
		return nil, fmt.Errorf("reading binary stores: %w", err)
	}

	for _, dirEntry := range binDirs {
		if !dirEntry.IsDir() {
			continue
		}
		entry, inManifest := fullManifest.Binaries[dirEntry.Name()]
		deletedVersions, err := purgeBinaryStore(dirEntry.Name(), entry, inManifest)
		if err != nil {
			return nil, err
		}
		if len(deletedVersions) > 0 {
			deletedStores[dirEntry.Name()] = deletedVersions
		}
	}
	return deletedStores, nil
}

func purgeBinaryStore(name string, entry manifest.BinaryEntry, inManifest bool) ([]string, error) {
	dirPath := filepath.Join(store.Root(), name)
	versionDirs, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("reading directory entry: %w", err)
	}

	var deletedVersions []string
	for _, versionDir := range versionDirs {
		if !versionDir.IsDir() {
			continue
		}
		_, versionInManifest := entry.Versions[versionDir.Name()]
		if inManifest && versionInManifest {
			continue
		}
		if err := store.Remove(name, versionDir.Name()); err != nil {
			return nil, fmt.Errorf("removing version %s: %w", versionDir.Name(), err)
		}
		deletedVersions = append(deletedVersions, versionDir.Name())
	}
	if !inManifest {
		if err := os.Remove(dirPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("removing empty binary entry: %w", err)
		}
	}
	return deletedVersions, nil
}

func mergeActiveManifestVersions(newManifest, fullManifest manifest.Manifest) (manifest.Manifest, error) {
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
		fullVal.Versions[newManifest.Binaries[key].Active] = versEntry
	}
	return fullManifest, nil
}
