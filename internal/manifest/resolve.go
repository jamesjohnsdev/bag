package manifest

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	globDir  = "bag"
	ManName  = "bag.toml"
	lockName = ".bag-lock"
)

func ErrGlobeInit(err error) error {
	return fmt.Errorf("creating global manifest: %w", err)
}

// Get will find the path of the manifest file, and create the manifest and lock if missing.
func Get(local bool) (path string, global bool, err error) {
	path, global, err = findManifestPath(local)
	if err != nil {
		return "", global, fmt.Errorf("finding manifest path: %w", err)
	}
	err = createIfMissing(path)
	if err != nil {
		return "", global, fmt.Errorf("creating manifest: %w", err)
	}
	return path, global, err
}

// FindManifestPath will return the path of the closest bag.toml file
// if not found in recursive search, will default to home ~/.config/bag/
// Specify whether local or global is wanted in param
// returns path + boolean whether it is global manifest
func findManifestPath(local bool) (path string, global bool, err error) {
	if local {
		workDir, err := os.Getwd()
		if err != nil {
			return "", false, fmt.Errorf("finding working dir: %w", err)
		}
		for {
			candidate := filepath.Join(workDir, ManName)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, false, nil
			}
			parent := filepath.Dir(workDir)
			if parent == workDir {
				// reached filesystem root
				break
			}
			workDir = parent
		}
	}
	// fall back to global
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("finding home dir: %w", err)
	}
	return filepath.Join(home, ".config", globDir, ManName), true, nil
}

// FindLock gets the path of the lockfile
// It only returns the linked lockfile given a bag.toml path
func FindLock(path string) string {
	parentDir := filepath.Dir(path)
	lockPath := filepath.Join(parentDir, lockName)
	return lockPath
}

// createIfMissing checks the manifest path, and if missing, creates a new manifest
func createIfMissing(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return ErrGlobeInit(err)
		}
		if err := Write(path, &Manifest{}); err != nil {
			return ErrGlobeInit(err)
		}
		if err := WriteLock(FindLock(path), &LockFile{}); err != nil {
			return ErrGlobeInit(err)
		}
	}
	return nil
}
