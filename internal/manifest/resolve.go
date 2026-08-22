package manifest

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	globDir  = "bag"
	manName  = "bag.toml"
	lockName = ".bag-lock"
)

// FindManifest will return the path of the closest bag.toml file
// if not found in recursive search, will default to home ~/.config/bag/
// returns path + boolean whether it is global manifest
func FindManifest() (path string, global bool, err error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", false, err
	}
	for {
		candidate := filepath.Join(workDir, manName)
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
	// fall back to global
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}
	return filepath.Join(home, ".config", globDir, manName), true, nil
}

// FindLock gets the path of the lockfile
// It only returns the linked lockfile given a bag.toml path
func FindLock(path string) (string, error) {
	parentDir := filepath.Dir(path)
	lockPath := filepath.Join(parentDir, lockName)
	// TODO: possibly drop this check if not necessary
	_, err := os.Stat(lockPath)
	if err != nil {
		return "", fmt.Errorf("find lock file: %w", err)
	}
	return lockPath, nil
}
