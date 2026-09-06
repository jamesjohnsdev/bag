package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
)

type Metadata struct {
	Source      string    `toml:"source"`
	Hash        string    `toml:"hash"`
	InstalledAt time.Time `toml:"installed_at"`
}

func isSafeName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	return filepath.Base(name) == name
}

func Root() string {
	basePath := os.Getenv("XDG_DATA_HOME")
	if basePath == "" {
		home, _ := os.UserHomeDir()
		basePath = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(basePath, "bag", "store")
}

func EntryDir(name, version string) string {
	return filepath.Join(Root(), name, version)
}

func BinaryPath(name, version string) string {
	return filepath.Join(EntryDir(name, version), name)
}

func BinaryExists(name, version string) bool {
	if _, err := os.Stat(BinaryPath(name, version)); err != nil {
		return false
	}
	return true
}

// MetadataPath generates the expected path of the metadata file
func MetadataPath(name, version string) string {
	return filepath.Join(EntryDir(name, version), "metadata.toml")
}

func MetadataExists(name, version string) bool {
	if _, err := os.Stat(MetadataPath(name, version)); err != nil {
		return false
	}
	return true
}

func ReadMetadata(name, version string) (Metadata, error) {
	metadata := Metadata{}
	if !MetadataExists(name, version) {
		return metadata, errors.New("metadata file not found")
	}
	// not a fan of how `MetadataExists` already invocates MetadataPath
	// TODO: refactor somehow
	_, err := toml.DecodeFile(MetadataPath(name, version), &metadata)
	if err != nil {
		return metadata, fmt.Errorf("reading metadata: %w", err)
	}
	return metadata, nil
}

// WriteMetadata generates metadata file within a known directory
// Hash is stored as raw string, without sha256 prefix
func WriteMetadata(name, version string, metadata Metadata) error {
	file, err := os.Create(MetadataPath(name, version))
	if err != nil {
		return fmt.Errorf("creating metadata file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()
	if err := toml.NewEncoder(file).Encode(metadata); err != nil {
		return fmt.Errorf("encoding metadata file: %w", err)
	}
	return nil
}

// Install local adds a locally stored binary to a bag
// hash returned with `sha256:` prefix
func InstallLocal(name, version, srcPath string) (sha256hash string, err error) {
	if !isSafeName(name) {
		return "", fmt.Errorf("invalid bianry name: %q", name)
	}
	if !isSafeName(version) {
		return "", fmt.Errorf("invalid version: %q", version)
	}
	if BinaryExists(name, version) {
		metadata, err := ReadMetadata(name, version)
		if err != nil {
			return "", err
		}
		return "sha256:" + metadata.Hash, nil
	}
	// 0755 = rwxr-xr-x
	if err := os.MkdirAll(EntryDir(name, version), 0o755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = srcFile.Close()
	}()

	dstFile, err := os.Create(BinaryPath(name, version))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = dstFile.Close()
	}()

	hash := sha256.New()
	writer := io.MultiWriter(dstFile, hash)
	if _, err := io.Copy(writer, srcFile); err != nil {
		return "", fmt.Errorf("copying binary: %w", err)
	}
	// need to wait until after copy to encode
	hashStr := hex.EncodeToString(hash.Sum(nil))
	if err := WriteMetadata(name, version, Metadata{
		Source:      "",
		Hash:        hashStr,
		InstalledAt: time.Now(),
	}); err != nil {
		return "", fmt.Errorf("writing metadata: %w", err)
	}

	// 0555 = r-xr-xr-x
	if err := os.Chmod(BinaryPath(name, version), 0o555); err != nil {
		return "", fmt.Errorf("chmod binary: %w", err)
	}
	if err := os.Chmod(EntryDir(name, version), 0o555); err != nil {
		return "", fmt.Errorf("chmod entry dir: %w", err)
	}

	return "sha256:" + hashStr, nil
}

func InstallFromReader(name, version, source string, r io.ReadCloser) (result string, err error) {
	defer func() {
		if cerr := r.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("closing source reader: %w", cerr))
		}
	}()

	if !isSafeName(name) {
		return "", fmt.Errorf("invalid binary name: %q", name)
	}
	if !isSafeName(version) {
		return "", fmt.Errorf("invalid version: %q", version)
	}

	if BinaryExists(name, version) {
		metadata, err := ReadMetadata(name, version)
		if err != nil {
			return "", err
		}
		return "sha256:" + metadata.Hash, nil
	}
	if err := os.MkdirAll(EntryDir(name, version), 0o755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}

	dstFile, err := os.Create(BinaryPath(name, version))
	if err != nil {
		return "", fmt.Errorf("creating destination file: %w", err)
	}
	defer func() {
		if cerr := dstFile.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("closing destination file: %w", cerr))
		}
	}()

	hash := sha256.New()
	writer := io.MultiWriter(dstFile, hash)
	if _, err := io.Copy(writer, r); err != nil {
		return "", fmt.Errorf("copying binary: %w", err)
	}

	hashStr := hex.EncodeToString(hash.Sum(nil))
	if err = WriteMetadata(name, version, Metadata{
		Source:      source,
		Hash:        hashStr,
		InstalledAt: time.Now(),
	}); err != nil {
		return "", fmt.Errorf("writing metadata: %w", err)
	}

	// 0555 = r-xr-xr-x
	if err := os.Chmod(BinaryPath(name, version), 0o555); err != nil {
		return "", fmt.Errorf("chmod binary: %w", err)
	}
	if err := os.Chmod(EntryDir(name, version), 0o555); err != nil {
		return "", fmt.Errorf("chmod entry dir: %w", err)
	}

	return "sha256:" + hashStr, nil
}

// InstallFromDir installs an archive's fully extracted tree (see
// provider.Resolution.Dir/BinaryRelPath) at EntryDir(name, version),
// preserving its directory structure so multi-file toolchains (e.g. a Go SDK
// needing src/, pkg/ alongside bin/go) keep their layout intact. binaryRelPath
// is the path of the resolved executable relative to srcDir.
func InstallFromDir(name, version, source, srcDir, binaryRelPath string) (result string, err error) {
	if !isSafeName(name) {
		return "", fmt.Errorf("invalid binary name: %q", name)
	}
	if !isSafeName(version) {
		return "", fmt.Errorf("invalid version: %q", version)
	}

	if BinaryExists(name, version) {
		metadata, err := ReadMetadata(name, version)
		if err != nil {
			return "", err
		}
		return "sha256:" + metadata.Hash, nil
	}

	target := EntryDir(name, version)
	// Archive contents are extracted into a reserved "tree" subdirectory,
	// never placed directly under target: an archive can legitimately
	// contain a top-level entry that collides with name (e.g. the Go SDK
	// tarball wraps everything in a "go/" dir, and binName is naturally
	// "go" too) which would otherwise collide with the symlink created
	// below.
	treeDir := filepath.Join(target, "tree")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}

	if err := os.Rename(srcDir, treeDir); err != nil {
		if !errors.Is(err, syscall.EXDEV) {
			return "", fmt.Errorf("moving extracted tree: %w", err)
		}
		// srcDir and the store live on different filesystems - fall back to
		// a recursive copy instead of an atomic rename.
		if err := os.MkdirAll(treeDir, 0o755); err != nil {
			return "", fmt.Errorf("creating directory: %w", err)
		}
		if err := os.CopyFS(treeDir, os.DirFS(srcDir)); err != nil {
			return "", fmt.Errorf("copying extracted tree: %w", err)
		}
		if err := os.RemoveAll(srcDir); err != nil {
			return "", fmt.Errorf("removing staged tree: %w", err)
		}
	}

	realBin := filepath.Join(treeDir, filepath.FromSlash(binaryRelPath))
	hashStr, err := hashFile(realBin)
	if err != nil {
		return "", err
	}

	if err := WriteMetadata(name, version, Metadata{
		Source:      source,
		Hash:        hashStr,
		InstalledAt: time.Now(),
	}); err != nil {
		return "", fmt.Errorf("writing metadata: %w", err)
	}

	// BinaryPath is hardcoded to target/name everywhere else (BinaryExists,
	// LinkToPath, Unlink, Remove) - symlink it to the real binary under
	// tree/ so none of those need to change.
	linkName := BinaryPath(name, version)
	rel, err := filepath.Rel(target, realBin)
	if err != nil {
		return "", fmt.Errorf("resolving binary path: %w", err)
	}
	if err := os.Symlink(rel, linkName); err != nil {
		return "", fmt.Errorf("linking binary: %w", err)
	}

	// installed = read-only, matching InstallFromReader/InstallLocal - strip
	// write bits tree-wide. Dirs keep exec so they stay traversable/readable.
	err = filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(path, 0o555)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.Chmod(path, info.Mode().Perm()&^0o222)
	})
	if err != nil {
		return "", fmt.Errorf("making install read-only: %w", err)
	}

	return "sha256:" + hashStr, nil
}

// hashFile returns the sha256 hash (hex-encoded, no prefix) of the file at path.
func hashFile(path string) (result string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("closing %s: %w", path, cerr))
		}
	}()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", fmt.Errorf("hashing %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// LinkToPath links a stored binary to path
func LinkToPath(name, version, binDir string) error {
	if !isSafeName(name) {
		return fmt.Errorf("invalid binary name: %q", name)
	}
	if !isSafeName(version) {
		return fmt.Errorf("invalid version: %q", version)
	}
	// 0755 = rwxr-xr-x
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("creating bin dir: %w", err)
	}
	src := BinaryPath(name, version)
	dst := filepath.Join(binDir, name)
	if err := os.Symlink(src, dst); err != nil {
		return fmt.Errorf("linking binary to path: %w", err)
	}
	return nil
}

func Unlink(name, binDir string) error {
	if !isSafeName(name) {
		return fmt.Errorf("invalid binary name: %q", name)
	}
	//src := BinaryPath(name, version)
	linkName := filepath.Join(binDir, name)
	_, err := os.Readlink(linkName)
	if err != nil {
		return fmt.Errorf("reading sym link: %w", err)
	}
	if err := os.Remove(linkName); err != nil {
		return fmt.Errorf("removing link: %w", err)
	}
	return nil
}

func Remove(name, version string) error {
	if !isSafeName(name) {
		return fmt.Errorf("supplied name '%s' not safe", name)
	}
	if !isSafeName(version) {
		return fmt.Errorf("supplied version '%s' is not safe", version)
	}
	entryDir := EntryDir(name, version)
	if _, err := os.Stat(entryDir); err != nil {
		return fmt.Errorf("making entry dir writable: %w", err)
	}
	// A tree install leaves nested directories read-only too, so every
	// directory under entryDir needs write permission restored before
	// RemoveAll can unlink the files inside it - chmod-ing entryDir alone
	// only unblocks removal of entryDir's direct children.
	err := filepath.WalkDir(entryDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(path, 0o755)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("making entry dir writable: %w", err)
	}
	if err := os.RemoveAll(entryDir); err != nil {
		return fmt.Errorf("removing directory %s: %w", entryDir, err)
	}
	return nil
}
