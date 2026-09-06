package store_test

import (
	"os"
	"path/filepath"
	"strings"
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

// writeSrcTree builds a temp directory tree from a relative-path -> content
// map; paths listed in exec are written with the executable bit set.
func writeSrcTree(t *testing.T, files map[string]string, exec map[string]bool) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o644)
		if exec[rel] {
			mode = 0o755
		}
		if err := os.WriteFile(full, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	return dir
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

func TestInstallFromDirPreservesTree(t *testing.T) {
	setStoreRoot(t)

	srcDir := writeSrcTree(t, map[string]string{
		"bin/go":           "GOBIN",
		"src/foo.go":       "package foo",
		"pkg/tool/compile": "toolbin",
	}, map[string]bool{"bin/go": true})

	hash, err := store.InstallFromDir("go", "1.27.1", "https://example.com/go.tar.gz", srcDir, "bin/go")
	if err != nil {
		t.Fatalf("InstallFromDir() error = %v", err)
	}
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("hash = %q, want sha256: prefix", hash)
	}
	if !store.BinaryExists("go", "1.27.1") {
		t.Fatal("expected binary to exist after InstallFromDir")
	}

	// BinaryPath() must resolve through the symlink indirection to the real,
	// nested binary.
	content, err := os.ReadFile(store.BinaryPath("go", "1.27.1"))
	if err != nil {
		t.Fatalf("reading binary via BinaryPath(): %v", err)
	}
	if string(content) != "GOBIN" {
		t.Errorf("binary content = %q, want GOBIN", content)
	}

	// Siblings must have survived alongside the binary, not been discarded.
	// (They live under EntryDir/tree/... - the extracted tree is kept out
	// of EntryDir's own root so archive content can never collide with the
	// name symlink; see TestInstallFromDirTopLevelNameCollision.)
	sibling, err := os.ReadFile(filepath.Join(store.EntryDir("go", "1.27.1"), "tree", "src", "foo.go"))
	if err != nil {
		t.Fatalf("sibling file missing: %v", err)
	}
	if string(sibling) != "package foo" {
		t.Errorf("sibling content = %q, want %q", sibling, "package foo")
	}
	tool, err := os.ReadFile(filepath.Join(store.EntryDir("go", "1.27.1"), "tree", "pkg", "tool", "compile"))
	if err != nil {
		t.Fatalf("nested sibling file missing: %v", err)
	}
	if string(tool) != "toolbin" {
		t.Errorf("nested sibling content = %q, want %q", tool, "toolbin")
	}

	// LinkToPath/Unlink must keep working through the symlink indirection.
	binDir := t.TempDir()
	if err := store.LinkToPath("go", "1.27.1", binDir); err != nil {
		t.Fatalf("LinkToPath() error = %v", err)
	}
	linked, err := os.ReadFile(filepath.Join(binDir, "go"))
	if err != nil {
		t.Fatalf("reading through PATH symlink: %v", err)
	}
	if string(linked) != "GOBIN" {
		t.Errorf("PATH symlink content = %q, want GOBIN", linked)
	}
	if err := store.Unlink("go", binDir); err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}
	if !store.BinaryExists("go", "1.27.1") {
		t.Error("expected binary to remain in store after Unlink")
	}

	// Remove must succeed despite the recursively read-only tree left behind
	// by InstallFromDir (regression: a single top-level chmod is not enough
	// once nested directories are themselves read-only).
	if err := store.Remove("go", "1.27.1"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if store.BinaryExists("go", "1.27.1") {
		t.Error("expected binary to be removed from store")
	}
}

func TestInstallFromDirFlatBinary(t *testing.T) {
	setStoreRoot(t)

	srcDir := writeSrcTree(t, map[string]string{"tool": "ELF"}, map[string]bool{"tool": true})

	if _, err := store.InstallFromDir("tool", "1.0.0", "src", srcDir, "tool"); err != nil {
		t.Fatalf("InstallFromDir() error = %v", err)
	}

	content, err := os.ReadFile(store.BinaryPath("tool", "1.0.0"))
	if err != nil {
		t.Fatalf("reading binary via BinaryPath(): %v", err)
	}
	if string(content) != "ELF" {
		t.Errorf("binary content = %q, want ELF", content)
	}
}

// TestInstallFromDirTopLevelNameCollision guards against the case that
// broke the initial implementation: the Go SDK tarball wraps everything in
// a top-level "go/" directory, and the binary is naturally named "go" too
// (bag add ... --name go), so a naive rename of the extracted tree straight
// into EntryDir would collide with the "go" name symlink InstallFromDir
// itself needs to create there.
func TestInstallFromDirTopLevelNameCollision(t *testing.T) {
	setStoreRoot(t)

	srcDir := writeSrcTree(t, map[string]string{
		"go/bin/go":  "GOBIN",
		"go/src/foo": "package foo",
	}, map[string]bool{"go/bin/go": true})

	if _, err := store.InstallFromDir("go", "1.27.1", "src", srcDir, "go/bin/go"); err != nil {
		t.Fatalf("InstallFromDir() error = %v", err)
	}
	content, err := os.ReadFile(store.BinaryPath("go", "1.27.1"))
	if err != nil {
		t.Fatalf("reading binary via BinaryPath(): %v", err)
	}
	if string(content) != "GOBIN" {
		t.Errorf("binary content = %q, want GOBIN", content)
	}
}

func TestInstallFromDirAlreadyInstalled(t *testing.T) {
	setStoreRoot(t)

	first := writeSrcTree(t, map[string]string{"bin/tool": "first"}, map[string]bool{"bin/tool": true})
	hash1, err := store.InstallFromDir("tool", "1.0.0", "src", first, "bin/tool")
	if err != nil {
		t.Fatalf("InstallFromDir() error = %v", err)
	}

	second := writeSrcTree(t, map[string]string{"bin/tool": "second"}, map[string]bool{"bin/tool": true})
	hash2, err := store.InstallFromDir("tool", "1.0.0", "src", second, "bin/tool")
	if err != nil {
		t.Fatalf("InstallFromDir() error = %v", err)
	}
	if hash1 != hash2 {
		t.Errorf("expected cached hash on reinstall, got %q then %q", hash1, hash2)
	}
	content, err := os.ReadFile(store.BinaryPath("tool", "1.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "first" {
		t.Errorf("expected original install to remain untouched, got %q", content)
	}
}

func TestInstallFromDirInvalidName(t *testing.T) {
	setStoreRoot(t)
	srcDir := writeSrcTree(t, map[string]string{"tool": "ELF"}, map[string]bool{"tool": true})

	if _, err := store.InstallFromDir("../evil", "1.0.0", "src", srcDir, "tool"); err == nil {
		t.Fatal("expected error for unsafe name")
	}
	if _, err := store.InstallFromDir("tool", "../evil", "src", srcDir, "tool"); err == nil {
		t.Fatal("expected error for unsafe version")
	}
}
