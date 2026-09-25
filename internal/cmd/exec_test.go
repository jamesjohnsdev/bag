package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

// setupHome isolates HOME and XDG_DATA_HOME so tests never touch the
// developer's real ~/.local/bin, ~/.config/bag or ~/.local/share/bag. A
// package-local copy of the cmd_test helper of the same name: unexported
// identifiers (resolveActiveVersion, linkBinary) aren't reachable from the
// black-box cmd_test package this file's siblings use.
func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Cleanup(func() {
		_ = filepath.Walk(home, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			_ = os.Chmod(path, 0o755)
			return nil
		})
	})
	return home
}

// writeLocalManifest writes a project-local bag.toml with the given binary
// entries directly under dir, bypassing manifest.Get so the test controls
// exactly what exists without relying on its auto-create side effect.
func writeLocalManifest(t *testing.T, dir string, binaries map[string]manifest.BinaryEntry) string {
	t.Helper()
	path := filepath.Join(dir, manifest.ManName)
	if err := manifest.Write(path, &manifest.Manifest{Binaries: binaries}); err != nil {
		t.Fatalf("manifest.Write() error = %v", err)
	}
	return path
}

func TestResolveActiveVersionGlobalOnly(t *testing.T) {
	setupHome(t)

	globalPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.AddBinary(globalPath, "foo", manifest.BinaryEntry{
		Type:   manifest.BinaryType,
		Active: "1.0.0",
		Versions: map[string]manifest.VersionEntry{
			"1.0.0": {Source: "src"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// cwd has no bag.toml anywhere upward, so resolution must fall through
	// to the global manifest.
	t.Chdir(t.TempDir())

	manPath, version, err := resolveActiveVersion("foo")
	if err != nil {
		t.Fatalf("resolveActiveVersion() error = %v", err)
	}
	if manPath != globalPath {
		t.Errorf("manPath = %q, want %q", manPath, globalPath)
	}
	if version != "1.0.0" {
		t.Errorf("version = %q, want %q", version, "1.0.0")
	}
}

func TestResolveActiveVersionLocalOverridesGlobal(t *testing.T) {
	setupHome(t)

	globalPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.AddBinary(globalPath, "foo", manifest.BinaryEntry{
		Type:     manifest.BinaryType,
		Active:   "1.0.0",
		Versions: map[string]manifest.VersionEntry{"1.0.0": {Source: "src"}},
	}); err != nil {
		t.Fatal(err)
	}

	projectDir := t.TempDir()
	localPath := writeLocalManifest(t, projectDir, map[string]manifest.BinaryEntry{
		"foo": {
			Type:     manifest.BinaryType,
			Active:   "2.0.0",
			Versions: map[string]manifest.VersionEntry{"2.0.0": {Source: "src"}},
		},
	})

	// Run from a subdirectory to also exercise the upward walk.
	subDir := filepath.Join(projectDir, "sub", "deeper")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(subDir)

	manPath, version, err := resolveActiveVersion("foo")
	if err != nil {
		t.Fatalf("resolveActiveVersion() error = %v", err)
	}
	if manPath != localPath {
		t.Errorf("manPath = %q, want local %q", manPath, localPath)
	}
	if version != "2.0.0" {
		t.Errorf("version = %q, want %q", version, "2.0.0")
	}
}

func TestResolveActiveVersionFallsThroughWhenLocalEntryMissing(t *testing.T) {
	setupHome(t)

	globalPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.AddBinary(globalPath, "foo", manifest.BinaryEntry{
		Type:     manifest.BinaryType,
		Active:   "1.0.0",
		Versions: map[string]manifest.VersionEntry{"1.0.0": {Source: "src"}},
	}); err != nil {
		t.Fatal(err)
	}

	projectDir := t.TempDir()
	// A local bag.toml exists, but doesn't mention "foo" - resolution must
	// still fall through to global, not just error.
	writeLocalManifest(t, projectDir, map[string]manifest.BinaryEntry{
		"bar": {Type: manifest.BinaryType, Active: "9.9.9", Versions: map[string]manifest.VersionEntry{"9.9.9": {}}},
	})
	t.Chdir(projectDir)

	manPath, version, err := resolveActiveVersion("foo")
	if err != nil {
		t.Fatalf("resolveActiveVersion() error = %v", err)
	}
	if manPath != globalPath {
		t.Errorf("manPath = %q, want global %q", manPath, globalPath)
	}
	if version != "1.0.0" {
		t.Errorf("version = %q, want %q", version, "1.0.0")
	}
}

func TestResolveActiveVersionUnresolvedNoLocalManifest(t *testing.T) {
	setupHome(t)
	t.Chdir(t.TempDir())

	_, _, err := resolveActiveVersion("ghost")
	if err == nil {
		t.Fatal("expected error for unconfigured binary")
	}
	if !strings.Contains(err.Error(), `no version of "ghost" configured`) {
		t.Errorf("err = %v, want message naming the binary", err)
	}
	// No local manifest was ever found, so local and global collapse to the
	// same path - the message must not repeat it as "X and X".
	if strings.Count(err.Error(), "bag.toml") != 1 {
		t.Errorf("err = %v, want the manifest path mentioned once, not duplicated", err)
	}
}

func TestResolveActiveVersionUnresolvedWithLocalManifest(t *testing.T) {
	setupHome(t)

	projectDir := t.TempDir()
	writeLocalManifest(t, projectDir, map[string]manifest.BinaryEntry{})
	t.Chdir(projectDir)

	_, _, err := resolveActiveVersion("ghost")
	if err == nil {
		t.Fatal("expected error for unconfigured binary")
	}
	if !strings.Contains(err.Error(), "and") {
		t.Errorf("err = %v, want both local and global manifest paths mentioned", err)
	}
}

func TestExecCmdRunUnresolved(t *testing.T) {
	setupHome(t)
	t.Chdir(t.TempDir())

	execCmd := &ExecCmd{Name: "ghost"}
	err := execCmd.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for unconfigured binary")
	}
	if !strings.Contains(err.Error(), `no version of "ghost" configured`) {
		t.Errorf("err = %v, want message naming the binary", err)
	}
}

func TestLinkBinaryBagUsesDirectVersionSymlink(t *testing.T) {
	setupHome(t)
	binDir := t.TempDir()

	// linkBinary("bag", ...) must go through LinkToPath, which requires the
	// version to actually be installed in the store.
	if err := linkBinary("bag", "1.0.0", binDir); err == nil {
		t.Fatal("expected error linking an uninstalled version")
	}

	src := filepath.Join(t.TempDir(), "src-bin")
	if err := os.WriteFile(src, []byte("fake bag"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InstallLocal("bag", "1.0.0", src); err != nil {
		t.Fatalf("InstallLocal() error = %v", err)
	}

	if err := linkBinary("bag", "1.0.0", binDir); err != nil {
		t.Fatalf("linkBinary() error = %v", err)
	}

	got, err := os.Readlink(filepath.Join(binDir, "bag"))
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if want := store.BinaryPath("bag", "1.0.0"); got != want {
		t.Errorf("linkBinary(\"bag\") target = %q, want store path %q", got, want)
	}
}

func TestLinkBinaryNonBagUsesShim(t *testing.T) {
	setupHome(t)
	binDir := t.TempDir()

	// Unlike "bag", nothing needs to be installed in the store - the shim
	// only ever points at the bag executable itself.
	if err := linkBinary("foo", "1.0.0", binDir); err != nil {
		t.Fatalf("linkBinary() error = %v", err)
	}

	got, err := os.Readlink(filepath.Join(binDir, "foo"))
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		t.Fatal(err)
	}
	if got != exe {
		t.Errorf("linkBinary(\"foo\") target = %q, want bag executable %q", got, exe)
	}
}
