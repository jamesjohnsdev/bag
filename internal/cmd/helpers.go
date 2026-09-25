package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type WorkSpace struct {
	binDir  string
	manPath string
	// global is true when manPath resolved to the global manifest - either
	// because local resolution wasn't requested, or it was and no project
	// bag.toml was found anywhere upward from the cwd.
	global bool
}

// workSpace resolves the manifest path and bin dir to operate on. binDir is always
// ~/.local/bin regardless of scope - only manPath resolution differs. When local is
// true, manifest.Get walks up from the cwd looking for a project bag.toml, falling
// back to (and auto-creating) the global one if none is found; that fallback is what
// ws.global flags.
func workSpace(local bool) (WorkSpace, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return WorkSpace{}, fmt.Errorf("checking home dir: %w", err)
	}
	binDir := filepath.Join(homeDir, ".local/bin")
	manPath, global, err := manifest.Get(local)
	if err != nil {
		return WorkSpace{}, err
	}
	return WorkSpace{
		binDir:  binDir,
		manPath: manPath,
		global:  global,
	}, nil
}

// requireLocal errors if ws fell back to the global manifest.
func (ws WorkSpace) requireLocal() error {
	if ws.global {
		return fmt.Errorf("no local bag found; run `bag tool init` first")
	}
	return nil
}

// linkBinary points binDir/name at version. Every managed binary except
// "bag" itself uses the PATH shim (store.LinkShim), which resolves its
// active version dynamically and so never needs re-pointing when version
// changes. "bag" keeps the old direct-to-version symlink (store.LinkToPath)
// since there's no case for a per-project bag version - see exec.go.
func linkBinary(name, version, binDir string) error {
	if name == "bag" {
		return store.LinkToPath(name, version, binDir)
	}
	return store.LinkShim(name, binDir)
}

func IsKnown(parser *kong.Kong, command string) bool {
	for _, knownCmd := range parser.Model.Children {
		if command == knownCmd.Name {
			return true
		}
	}
	return false
}
