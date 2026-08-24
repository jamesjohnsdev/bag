package manifest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/BurntSushi/toml"
)

type LockFile struct {
	Entries map[string]LockEntry
}

type LockEntry struct {
	Version string `toml:"version"`
	Hash    string `toml:"hash"`
}

func ParseLock(path string) (*LockFile, error) {
	// create blank Lock entry map with string
	entries := make(map[string]LockEntry)
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
	defer file.Close()
	return toml.NewEncoder(file).Encode(lf.Entries)
}

func AddLockEntry(lockPath, name string, entry LockEntry) error {
	lockfile, err := ParseLock(lockPath)
	if err != nil {
		return fmt.Errorf("parsing lockfile: %w", err)
	}
	lockfile.Entries[name] = entry
	return WriteLock(lockPath, lockfile)
}
