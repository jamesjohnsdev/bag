package manifest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/BurntSushi/toml"
)

type LockFile struct {
	// keyed by binary name -> version -> entry
	Entries map[string]map[string]LockEntry
}

type LockEntry struct {
	Hash string `toml:"hash"`
}

func ParseLock(path string) (*LockFile, error) {
	// create blank Lock entry map with string
	entries := make(map[string]map[string]LockEntry)
	if _, err := toml.DecodeFile(path, &entries); err != nil {
		// if no lock file, treat as if blank
		if errors.Is(err, fs.ErrNotExist) {
			return &LockFile{Entries: entries}, nil
		}
		return nil, err
	}
	return &LockFile{Entries: entries}, nil
}

func WriteLock(path string, lf *LockFile) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	return toml.NewEncoder(file).Encode(lf.Entries)
}

// AddLockEntry records the hash for a specific version of a binary
func AddLockEntry(lockPath, name, version string, entry LockEntry) error {
	lockfile, err := ParseLock(lockPath)
	if err != nil {
		return fmt.Errorf("parsing lockfile: %w", err)
	}
	if lockfile.Entries[name] == nil {
		lockfile.Entries[name] = make(map[string]LockEntry)
	}
	lockfile.Entries[name][version] = entry
	return WriteLock(lockPath, lockfile)
}

// RemoveLockEntry removes all recorded versions for a binary
func RemoveLockEntry(lockPath, name string) error {
	lockfile, err := ParseLock(lockPath)
	if err != nil {
		return fmt.Errorf("parsing lockfile: %w", err)
	}
	delete(lockfile.Entries, name)
	return WriteLock(lockPath, lockfile)
}
