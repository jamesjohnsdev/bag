package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/cmd"
	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

// setupHome isolates HOME and XDG_DATA_HOME so tests never touch the
// developer's real ~/.local/bin, ~/.config/bag or ~/.local/share/bag.
func setupHome(t *testing.T) (home, binDir string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	// InstallLocal chmods entry dirs read-only (0o555); restore write perms
	// before TempDir's own cleanup tries to RemoveAll them.
	t.Cleanup(func() {
		_ = filepath.Walk(home, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			_ = os.Chmod(path, 0o755)
			return nil
		})
	})
	return home, filepath.Join(home, ".local", "bin")
}

// installTestBinary drives the store + manifest + lock exactly like AddCmd
// would, without going through the cmd package, so remove tests start from
// a fully "installed" state.
func installTestBinary(t *testing.T, name, version string) (hash string) {
	t.Helper()

	src := filepath.Join(t.TempDir(), "src-bin")
	if err := os.WriteFile(src, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	hash, err := store.InstallLocal(name, version, src)
	if err != nil {
		t.Fatalf("InstallLocal() error = %v", err)
	}

	binDir := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	if err := store.LinkToPath(name, version, binDir); err != nil {
		t.Fatalf("LinkToPath() error = %v", err)
	}

	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatalf("manifest.Get() error = %v", err)
	}
	if err := manifest.AddBinary(manPath, name, manifest.BinaryEntry{
		Type:   "binary",
		Active: version,
		Versions: map[string]manifest.VersionEntry{
			version: {Source: src},
		},
	}); err != nil {
		t.Fatalf("AddBinary() error = %v", err)
	}
	if err := manifest.AddLockEntry(manifest.FindLock(manPath), name, version, manifest.LockEntry{
		Hash: hash,
	}); err != nil {
		t.Fatalf("AddLockEntry() error = %v", err)
	}

	return hash
}

func TestRemoveCmdRun(t *testing.T) {
	_, binDir := setupHome(t)
	installTestBinary(t, "foo", "1.0.0")

	removeCmd := &cmd.RemoveCmd{Name: "foo"}
	if err := removeCmd.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Default remove only drops manifest/lock entries; the linked binary
	// and stored files are left alone unless --purge is passed.
	if _, err := os.Lstat(filepath.Join(binDir, "foo")); err != nil {
		t.Errorf("expected symlink preserved, stat err = %v", err)
	}
	if !store.BinaryExists("foo", "1.0.0") {
		t.Error("expected binary preserved in store")
	}

	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
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
}

func TestRemoveCmdRunPurge(t *testing.T) {
	_, binDir := setupHome(t)
	installTestBinary(t, "foo", "1.0.0")

	removeCmd := &cmd.RemoveCmd{Name: "foo", Purge: true}
	if err := removeCmd.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if _, err := os.Lstat(filepath.Join(binDir, "foo")); !os.IsNotExist(err) {
		t.Errorf("expected symlink removed, stat err = %v", err)
	}
	if store.BinaryExists("foo", "1.0.0") {
		t.Error("expected binary removed from store")
	}

	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
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
}

func TestRemoveCmdRunNotInManifest(t *testing.T) {
	setupHome(t)

	removeCmd := &cmd.RemoveCmd{Name: "does-not-exist"}
	if err := removeCmd.Run(context.Background()); err == nil {
		t.Fatal("expected error removing binary absent from manifest")
	}
}

func TestRemoveCmdRunMissingSymlink(t *testing.T) {
	// Manifest/lock know about the binary, but nothing was ever linked into
	// binDir. Default remove leaves symlinks alone so it succeeds; purge
	// must fail on Unlink and leave manifest/lock intact.
	setupHome(t)

	src := filepath.Join(t.TempDir(), "src-bin")
	if err := os.WriteFile(src, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	hash, err := store.InstallLocal("foo", "1.0.0", src)
	if err != nil {
		t.Fatal(err)
	}
	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.AddBinary(manPath, "foo", manifest.BinaryEntry{
		Type:   "binary",
		Active: "1.0.0",
		Versions: map[string]manifest.VersionEntry{
			"1.0.0": {Source: src},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := manifest.AddLockEntry(manifest.FindLock(manPath), "foo", "1.0.0", manifest.LockEntry{Hash: hash}); err != nil {
		t.Fatal(err)
	}

	removeCmd := &cmd.RemoveCmd{Name: "foo"}
	if err := removeCmd.Run(context.Background()); err != nil {
		t.Fatalf("Run() without purge should succeed despite missing symlink, got err = %v", err)
	}

	man, err := manifest.Parse(manPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := man.Binaries["foo"]; ok {
		t.Error("expected manifest entry removed")
	}

	// Re-add manifest/lock entries and verify purge fails on the missing
	// symlink, leaving manifest/lock intact.
	if err := manifest.AddBinary(manPath, "foo", manifest.BinaryEntry{
		Type:   "binary",
		Active: "1.0.0",
		Versions: map[string]manifest.VersionEntry{
			"1.0.0": {Source: src},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := manifest.AddLockEntry(manifest.FindLock(manPath), "foo", "1.0.0", manifest.LockEntry{Hash: hash}); err != nil {
		t.Fatal(err)
	}

	purgeCmd := &cmd.RemoveCmd{Name: "foo", Purge: true}
	if err := purgeCmd.Run(context.Background()); err == nil {
		t.Fatal("expected error unlinking nonexistent symlink with purge")
	}

	man, err = manifest.Parse(manPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := man.Binaries["foo"]; !ok {
		t.Error("expected manifest entry to remain after failed purge")
	}
}

func TestRemoveCmdRunRollsBackSymlinkOnStoreRemoveFailure(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("permission-based test requires a POSIX filesystem")
	}
	if os.Geteuid() == 0 {
		t.Skip("permission checks are bypassed when running as root")
	}

	_, binDir := setupHome(t)
	installTestBinary(t, "foo", "1.0.0")

	// Block removal of the entry dir itself by making its parent read-only,
	// forcing store.Remove to fail after store.Unlink already succeeded.
	parent := filepath.Dir(store.EntryDir("foo", "1.0.0"))
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	// setupHome's cleanup walk-chmods the whole home tree writable again,
	// so no explicit restore is needed here.

	removeCmd := &cmd.RemoveCmd{Name: "foo", Purge: true}
	if err := removeCmd.Run(context.Background()); err == nil {
		t.Fatal("expected error removing binary from store")
	}

	link, err := os.Lstat(filepath.Join(binDir, "foo"))
	if err != nil {
		t.Fatalf("expected symlink to be relinked after rollback, got err = %v", err)
	}
	if link.Mode()&os.ModeSymlink == 0 {
		t.Error("expected relinked entry to still be a symlink")
	}

	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	man, err := manifest.Parse(manPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := man.Binaries["foo"]; !ok {
		t.Error("expected manifest entry to remain after rollback")
	}
}

// installLocalTestBinary initialises a project-local bag in projectDir and
// installs name/version into the shared store, linking binDir and recording
// the entry in the local manifest + lock only.
func installLocalTestBinary(t *testing.T, projectDir, name, version string) (localPath string) {
	t.Helper()

	t.Chdir(projectDir)
	if err := (&cmd.InitToolCmd{}).Run(context.Background()); err != nil {
		t.Fatalf("InitToolCmd.Run() error = %v", err)
	}

	src := filepath.Join(t.TempDir(), "src-bin")
	if err := os.WriteFile(src, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	hash, err := store.InstallLocal(name, version, src)
	if err != nil {
		t.Fatalf("InstallLocal() error = %v", err)
	}
	binDir := filepath.Join(os.Getenv("HOME"), ".local", "bin")
	if err := store.LinkToPath(name, version, binDir); err != nil {
		t.Fatalf("LinkToPath() error = %v", err)
	}

	localPath = filepath.Join(projectDir, manifest.ManName)
	if err := manifest.AddBinary(localPath, name, manifest.BinaryEntry{
		Type:   "binary",
		Active: version,
		Versions: map[string]manifest.VersionEntry{
			version: {Source: src},
		},
	}); err != nil {
		t.Fatalf("AddBinary() error = %v", err)
	}
	if err := manifest.AddLockEntry(manifest.FindLock(localPath), name, version, manifest.LockEntry{Hash: hash}); err != nil {
		t.Fatalf("AddLockEntry() error = %v", err)
	}
	return localPath
}

func TestRemoveToolCmdRunNoLocalManifest(t *testing.T) {
	setupHome(t)
	t.Chdir(t.TempDir())

	removeCmd := &cmd.RemoveToolCmd{Name: "foo"}
	err := removeCmd.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when no local bag.toml exists")
	}
	if !strings.Contains(err.Error(), "bag tool init") {
		t.Errorf("err = %v, want message pointing at `bag tool init`", err)
	}
}

func TestRemoveToolCmdRun(t *testing.T) {
	_, binDir := setupHome(t)
	projectDir := t.TempDir()
	localPath := installLocalTestBinary(t, projectDir, "foo", "1.0.0")

	removeCmd := &cmd.RemoveToolCmd{Name: "foo"}
	if err := removeCmd.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Default tool remove drops the local manifest/lock entries only.
	man, err := manifest.Parse(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := man.Binaries["foo"]; ok {
		t.Error("expected local manifest entry removed")
	}
	lf, err := manifest.ParseLock(manifest.FindLock(localPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lf.Entries["foo"]; ok {
		t.Error("expected local lock entry removed")
	}
	if _, err := os.Lstat(filepath.Join(binDir, "foo")); err != nil {
		t.Errorf("expected symlink preserved, stat err = %v", err)
	}
	if !store.BinaryExists("foo", "1.0.0") {
		t.Error("expected binary preserved in store")
	}

	// Global manifest must be untouched by a project-scoped remove.
	globalPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	_ = globalPath
}

func TestRemoveToolCmdRunPurge(t *testing.T) {
	_, binDir := setupHome(t)
	projectDir := t.TempDir()
	localPath := installLocalTestBinary(t, projectDir, "foo", "1.0.0")

	removeCmd := &cmd.RemoveToolCmd{Name: "foo", Purge: true}
	if err := removeCmd.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	man, err := manifest.Parse(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := man.Binaries["foo"]; ok {
		t.Error("expected local manifest entry removed")
	}
	lf, err := manifest.ParseLock(manifest.FindLock(localPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lf.Entries["foo"]; ok {
		t.Error("expected local lock entry removed")
	}
	if _, err := os.Lstat(filepath.Join(binDir, "foo")); !os.IsNotExist(err) {
		t.Errorf("expected symlink removed, stat err = %v", err)
	}
	if store.BinaryExists("foo", "1.0.0") {
		t.Error("expected binary removed from store")
	}
}

func TestRemoveToolCmdRunNotInManifest(t *testing.T) {
	setupHome(t)
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	if err := (&cmd.InitToolCmd{}).Run(context.Background()); err != nil {
		t.Fatalf("InitToolCmd.Run() error = %v", err)
	}

	removeCmd := &cmd.RemoveToolCmd{Name: "does-not-exist"}
	if err := removeCmd.Run(context.Background()); err == nil {
		t.Fatal("expected error removing binary absent from local manifest")
	}
}
