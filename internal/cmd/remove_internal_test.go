package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

// installRemoveTestBinary drives the store + manifest + lock exactly like
// AddCmd would, returning the global manifest path, binDir and entry for
// direct purgeBinary/removeBinary/findManifestEntry tests.
func installRemoveTestBinary(t *testing.T, name, version string) (manPath, binDir string, entry manifest.BinaryEntry) {
	t.Helper()

	home := os.Getenv("HOME")
	binDir = filepath.Join(home, ".local", "bin")

	src := filepath.Join(t.TempDir(), "src-bin")
	if err := os.WriteFile(src, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	hash, err := store.InstallLocal(name, version, src)
	if err != nil {
		t.Fatalf("InstallLocal() error = %v", err)
	}
	if err := store.LinkToPath(name, version, binDir); err != nil {
		t.Fatalf("LinkToPath() error = %v", err)
	}

	var global bool
	manPath, global, err = manifest.Get(false)
	if err != nil {
		t.Fatalf("manifest.Get() error = %v", err)
	}
	_ = global
	entry = manifest.BinaryEntry{
		Type:   "binary",
		Active: version,
		Versions: map[string]manifest.VersionEntry{
			version: {Source: src},
		},
	}
	if err := manifest.AddBinary(manPath, name, entry); err != nil {
		t.Fatalf("AddBinary() error = %v", err)
	}
	if err := manifest.AddLockEntry(manifest.FindLock(manPath), name, version, manifest.LockEntry{Hash: hash}); err != nil {
		t.Fatalf("AddLockEntry() error = %v", err)
	}
	return manPath, binDir, entry
}

func TestFindManifestEntrySuccess(t *testing.T) {
	setupHome(t)
	manPath, _, entry := installRemoveTestBinary(t, "foo", "1.0.0")

	got, err := findManifestEntry(manPath, "foo")
	if err != nil {
		t.Fatalf("findManifestEntry() error = %v", err)
	}
	if got.Active != entry.Active {
		t.Errorf("Active = %q, want %q", got.Active, entry.Active)
	}
}

func TestFindManifestEntryNotFound(t *testing.T) {
	setupHome(t)
	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := findManifestEntry(manPath, "does-not-exist"); err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestFindManifestEntryParseError(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bag.toml")
	if err := os.WriteFile(badPath, []byte("not valid toml [[[["), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := findManifestEntry(badPath, "foo"); err == nil {
		t.Fatal("expected parsing error")
	}
}

func TestRemoveBinarySuccess(t *testing.T) {
	setupHome(t)
	manPath, binDir, _ := installRemoveTestBinary(t, "foo", "1.0.0")

	if err := removeBinary(manPath, "foo"); err != nil {
		t.Fatalf("removeBinary() error = %v", err)
	}

	man, err := manifest.Parse(manPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := man.Binaries["foo"]; ok {
		t.Error("expected manifest entry removed")
	}
	lf, err := manifest.ParseLock(manifest.FindLock(manPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lf.Entries["foo"]; ok {
		t.Error("expected lock entry removed")
	}

	// removeBinary must not touch the symlink or store; that is purgeBinary's job.
	if _, err := os.Lstat(filepath.Join(binDir, "foo")); err != nil {
		t.Errorf("expected symlink preserved, stat err = %v", err)
	}
	if !store.BinaryExists("foo", "1.0.0") {
		t.Error("expected binary preserved in store")
	}
}

func TestRemoveBinaryNotInManifest(t *testing.T) {
	setupHome(t)
	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}

	if err := removeBinary(manPath, "does-not-exist"); err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestRemoveBinaryRollsBackManifestOnLockFailure(t *testing.T) {
	setupHome(t)
	manPath, _, _ := installRemoveTestBinary(t, "foo", "1.0.0")

	// Corrupt the lock file so RemoveLockEntry fails after the manifest
	// entry has already been deleted; removeBinary must restore it.
	if err := os.WriteFile(manifest.FindLock(manPath), []byte("not valid toml [[[["), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := removeBinary(manPath, "foo"); err == nil {
		t.Fatal("expected error removing corrupt lock entry")
	}

	man, err := manifest.Parse(manPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := man.Binaries["foo"]; !ok {
		t.Error("expected manifest entry restored after lock failure")
	}
}

func TestPurgeBinarySuccess(t *testing.T) {
	setupHome(t)
	_, binDir, entry := installRemoveTestBinary(t, "foo", "1.0.0")

	if err := purgeBinary("foo", binDir, entry); err != nil {
		t.Fatalf("purgeBinary() error = %v", err)
	}

	if _, err := os.Lstat(filepath.Join(binDir, "foo")); !os.IsNotExist(err) {
		t.Errorf("expected symlink removed, stat err = %v", err)
	}
	if store.BinaryExists("foo", "1.0.0") {
		t.Error("expected binary removed from store")
	}
}

func TestPurgeBinaryMissingSymlink(t *testing.T) {
	setupHome(t)

	src := filepath.Join(t.TempDir(), "src-bin")
	if err := os.WriteFile(src, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InstallLocal("foo", "1.0.0", src); err != nil {
		t.Fatal(err)
	}
	entry := manifest.BinaryEntry{
		Type:   "binary",
		Active: "1.0.0",
		Versions: map[string]manifest.VersionEntry{
			"1.0.0": {Source: src},
		},
	}
	binDir := filepath.Join(os.Getenv("HOME"), ".local", "bin")

	if err := purgeBinary("foo", binDir, entry); err == nil {
		t.Fatal("expected error purging with missing symlink")
	}
	if !store.BinaryExists("foo", "1.0.0") {
		t.Error("expected store entry preserved after failed purge")
	}
}

func TestPurgeBinaryRollsBackSymlinkOnStoreRemoveFailure(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("permission-based test requires a POSIX filesystem")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission checks are bypassed when running as root")
	}

	setupHome(t)
	_, binDir, entry := installRemoveTestBinary(t, "foo", "1.0.0")

	parent := filepath.Dir(store.EntryDir("foo", "1.0.0"))
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}

	if err := purgeBinary("foo", binDir, entry); err == nil {
		t.Fatal("expected error removing binary from store")
	}

	link, err := os.Lstat(filepath.Join(binDir, "foo"))
	if err != nil {
		t.Fatalf("expected symlink relinked after rollback, got err = %v", err)
	}
	if link.Mode()&os.ModeSymlink == 0 {
		t.Error("expected relinked entry to still be a symlink")
	}
}
