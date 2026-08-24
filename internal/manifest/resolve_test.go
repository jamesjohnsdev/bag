package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

// touch creates an empty file at path.
func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFindManifest(t *testing.T) {
	t.Run("finds bag.toml in cwd", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, filepath.Join(dir, "bag.toml"))
		t.Chdir(dir)

		path, global, err := manifest.FindManifest(true)
		if err != nil {
			t.Fatal(err)
		}
		if global {
			t.Error("global = true, want false")
		}
		if want := filepath.Join(dir, "bag.toml"); path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
	})

	t.Run("finds bag.toml in parent directory", func(t *testing.T) {
		parent := t.TempDir()
		child := filepath.Join(parent, "subdir")
		if err := os.Mkdir(child, 0o755); err != nil {
			t.Fatal(err)
		}
		touch(t, filepath.Join(parent, "bag.toml"))
		t.Chdir(child)

		path, global, err := manifest.FindManifest(true)
		if err != nil {
			t.Fatal(err)
		}
		if global {
			t.Error("global = true, want false")
		}
		if want := filepath.Join(parent, "bag.toml"); path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
	})

	t.Run("cwd bag.toml takes precedence over parent", func(t *testing.T) {
		parent := t.TempDir()
		child := filepath.Join(parent, "subdir")
		if err := os.Mkdir(child, 0o755); err != nil {
			t.Fatal(err)
		}
		touch(t, filepath.Join(parent, "bag.toml"))
		touch(t, filepath.Join(child, "bag.toml"))
		t.Chdir(child)

		path, _, err := manifest.FindManifest(true)
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(child, "bag.toml"); path != want {
			t.Errorf("path = %q, want %q", path, want)
		}
	})

	t.Run("falls back to global when no bag.toml found", func(t *testing.T) {
		t.Chdir(t.TempDir())

		_, global, err := manifest.FindManifest(true)
		if err != nil {
			t.Fatal(err)
		}
		if !global {
			t.Error("global = false, want true")
		}
	})

	t.Run("local = false skips search and returns global", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, filepath.Join(dir, "bag.toml"))
		t.Chdir(dir)

		_, global, err := manifest.FindManifest(false)
		if err != nil {
			t.Fatal(err)
		}
		if !global {
			t.Error("global = false, want true")
		}
	})
}

func TestFindLock(t *testing.T) {
	t.Run("returns lock path when file exists", func(t *testing.T) {
		dir := t.TempDir()
		manifestPath := filepath.Join(dir, "bag.toml")
		lockPath := filepath.Join(dir, ".bag-lock")
		touch(t, lockPath)

		got := manifest.FindLock(manifestPath)
		if got != lockPath {
			t.Errorf("FindLock() = %q, want %q", got, lockPath)
		}
	})

	t.Run("lock is always sibling of manifest", func(t *testing.T) {
		dir := t.TempDir()
		manifestPath := filepath.Join(dir, "bag.toml")
		touch(t, filepath.Join(dir, ".bag-lock"))

		got := manifest.FindLock(manifestPath)
		if filepath.Dir(got) != dir {
			t.Errorf("lock dir = %q, want %q", filepath.Dir(got), dir)
		}
	})
}
