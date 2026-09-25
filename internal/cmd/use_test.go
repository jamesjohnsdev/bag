package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/cmd"
	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

// installTestBinaryVersions stores two versions of name and records a
// manifest entry with activeVersion active, without linking binDir at all -
// exercising UseCmd from a state where nothing has been linked yet.
func installTestBinaryVersions(t *testing.T, name, activeVersion, otherVersion string) {
	t.Helper()

	versions := map[string]manifest.VersionEntry{}
	for _, v := range []string{activeVersion, otherVersion} {
		src := filepath.Join(t.TempDir(), "src-bin")
		if err := os.WriteFile(src, []byte("fake "+name+" "+v), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := store.InstallLocal(name, v, src); err != nil {
			t.Fatalf("InstallLocal(%s) error = %v", v, err)
		}
		versions[v] = manifest.VersionEntry{Source: src}
	}

	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.AddBinary(manPath, name, manifest.BinaryEntry{
		Type:     "binary",
		Active:   activeVersion,
		Versions: versions,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestUseCmdRunNonBagLeavesShimInPlace(t *testing.T) {
	_, binDir := setupHome(t)
	installTestBinaryVersions(t, "foo", "v1", "v2")

	useCmd := &cmd.UseCmd{Binary: "foo", Version: "v2"}
	if err := useCmd.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
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
		t.Errorf("symlink target = %q, want bag executable %q (shim, not a version path)", got, exe)
	}

	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := manifest.GetBinaryDetails(manPath, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Active != "v2" {
		t.Errorf("Active = %q, want %q", entry.Active, "v2")
	}
}

func TestUseCmdRunBagRelinksToVersion(t *testing.T) {
	_, binDir := setupHome(t)
	installTestBinaryVersions(t, "bag", "v1", "v2")
	if err := store.LinkToPath("bag", "v1", binDir); err != nil {
		t.Fatalf("LinkToPath() error = %v", err)
	}

	useCmd := &cmd.UseCmd{Binary: "bag", Version: "v2"}
	if err := useCmd.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got, err := os.Readlink(filepath.Join(binDir, "bag"))
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if want := store.BinaryPath("bag", "v2"); got != want {
		t.Errorf("symlink target = %q, want store path %q (direct version link, not a shim)", got, want)
	}

	manPath, _, err := manifest.Get(false)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := manifest.GetBinaryDetails(manPath, "bag")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Active != "v2" {
		t.Errorf("Active = %q, want %q", entry.Active, "v2")
	}
}
