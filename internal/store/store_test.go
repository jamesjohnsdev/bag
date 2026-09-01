package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/store"
)

func setStoreRoot(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	// InstallLocal chmods entry dirs read-only (0o555); restore write perms
	// before TempDir's own cleanup tries to RemoveAll them.
	t.Cleanup(func() {
		_ = filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			_ = os.Chmod(path, 0o755)
			return nil
		})
	})
}

func writeSrcBinary(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "src-bin")
	if err := os.WriteFile(f, []byte("binary content"), 0o755); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestUnlink(t *testing.T) {
	setStoreRoot(t)
	binDir := t.TempDir()

	if _, err := store.InstallLocal("foo", "1.0.0", writeSrcBinary(t)); err != nil {
		t.Fatalf("InstallLocal() error = %v", err)
	}
	if err := store.LinkToPath("foo", "1.0.0", binDir); err != nil {
		t.Fatalf("LinkToPath() error = %v", err)
	}

	if err := store.Unlink("foo", binDir); err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}

	if _, err := os.Lstat(filepath.Join(binDir, "foo")); !os.IsNotExist(err) {
		t.Errorf("expected symlink to be removed, stat err = %v", err)
	}
	// Unlink must only remove the symlink, not the store entry it points to.
	if !store.BinaryExists("foo", "1.0.0") {
		t.Error("expected binary to remain in store after Unlink")
	}
}

func TestUnlinkMissingSymlink(t *testing.T) {
	setStoreRoot(t)
	binDir := t.TempDir()

	if err := store.Unlink("missing", binDir); err == nil {
		t.Fatal("expected error unlinking nonexistent symlink")
	}
}

func TestUnlinkInvalidName(t *testing.T) {
	setStoreRoot(t)

	if err := store.Unlink("../evil", t.TempDir()); err == nil {
		t.Fatal("expected error for unsafe name")
	}
}

func TestRemove(t *testing.T) {
	setStoreRoot(t)

	if _, err := store.InstallLocal("foo", "1.0.0", writeSrcBinary(t)); err != nil {
		t.Fatalf("InstallLocal() error = %v", err)
	}
	if !store.BinaryExists("foo", "1.0.0") {
		t.Fatal("setup: binary should exist before Remove")
	}

	if err := store.Remove("foo", "1.0.0"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if store.BinaryExists("foo", "1.0.0") {
		t.Error("expected binary to be removed from store")
	}
	if _, err := os.Stat(store.EntryDir("foo", "1.0.0")); !os.IsNotExist(err) {
		t.Errorf("expected entry dir to be removed, stat err = %v", err)
	}
}

func TestRemoveMissingEntry(t *testing.T) {
	setStoreRoot(t)

	if err := store.Remove("missing", "1.0.0"); err == nil {
		t.Fatal("expected error removing nonexistent entry")
	}
}

func TestRemoveInvalidName(t *testing.T) {
	setStoreRoot(t)

	if err := store.Remove("../evil", "1.0.0"); err == nil {
		t.Fatal("expected error for unsafe name")
	}
	if err := store.Remove("foo", "../evil"); err == nil {
		t.Fatal("expected error for unsafe version")
	}
}
