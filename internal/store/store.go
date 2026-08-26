package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func InstallFromReader(name, version, source string, r io.ReadCloser) (string, error) {
	defer r.Close()

	if !isSafeName(name) {
		return "", fmt.Errorf("invalid binary name: %q", name)
	}
	if err := os.MkdirAll(EntryDir(name, version), 0o755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}

	dstFile, err := os.Create(BinaryPath(name, version))
	if err != nil {
		return "", fmt.Errorf("creating destination file: %w", err)
	}
	defer dstFile.Close()

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

// LinkToPath links a stored binary to path
func LinkToPath(name, version, binDir string) error {
	if !isSafeName(name) {
		return fmt.Errorf("invalid binary name: %q", name)
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
