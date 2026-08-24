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

// FindManifest will return the path of the closest bag.toml file
// if not found in recursive search, will default to home ~/.config/bag/
// Specify whether local or global is wanted in param
// returns path + boolean whether it is global manifest
func FindManifest(local bool) (path string, global bool, err error) {
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
