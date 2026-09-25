package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesjohnsdev/bag/internal/cmd"
	"github.com/jamesjohnsdev/bag/internal/manifest"
)

func TestAddToolCmdRunNoLocalManifest(t *testing.T) {
	setupHome(t)

	// No project bag.toml exists anywhere upward from this cwd.
	t.Chdir(t.TempDir())

	src := filepath.Join(t.TempDir(), "src-bin")
	if err := os.WriteFile(src, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	addCmd := &cmd.AddToolCmd{Source: src, Local: true, Name: "foo"}
	err := addCmd.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when no local bag.toml exists")
	}
	if !strings.Contains(err.Error(), "bag tool init") {
		t.Errorf("err = %v, want message pointing at `bag tool init`", err)
	}

	globalPath, _, gerr := manifest.Get(false)
	if gerr != nil {
		t.Fatal(gerr)
	}
	man, perr := manifest.Parse(globalPath)
	if perr != nil {
		t.Fatal(perr)
	}
	if _, ok := man.Binaries["foo"]; ok {
		t.Error("expected nothing written to the global manifest on refusal")
	}
}

func TestAddToolCmdRunSucceedsAfterInit(t *testing.T) {
	_, binDir := setupHome(t)

	projectDir := t.TempDir()
	t.Chdir(projectDir)

	initCmd := &cmd.InitToolCmd{}
	if err := initCmd.Run(context.Background()); err != nil {
		t.Fatalf("InitToolCmd.Run() error = %v", err)
	}

	src := filepath.Join(t.TempDir(), "src-bin")
	if err := os.WriteFile(src, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	addCmd := &cmd.AddToolCmd{Source: src, Local: true, Name: "foo"}
	if err := addCmd.Run(context.Background()); err != nil {
		t.Fatalf("AddToolCmd.Run() error = %v", err)
	}

	localPath := filepath.Join(projectDir, manifest.ManName)
	man, err := manifest.Parse(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := man.Binaries["foo"]; !ok {
		t.Error("expected entry written to the local manifest")
	}

	globalPath, _, gerr := manifest.Get(false)
	if gerr != nil {
		t.Fatal(gerr)
	}
	globalMan, perr := manifest.Parse(globalPath)
	if perr != nil {
		t.Fatal(perr)
	}
	if _, ok := globalMan.Binaries["foo"]; ok {
		t.Error("expected nothing written to the global manifest")
	}

	// The shim, unlike the manifest entry, is scope-agnostic - it always
	// lands in the shared binDir and points at the bag executable itself.
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
		t.Errorf("shim target = %q, want bag executable %q", got, exe)
	}
}
