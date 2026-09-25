package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

type WorkSpace struct {
	binDir  string
	manPath string
}

// workSpace resolves the manifest path and bin dir to operate on. binDir is always
// ~/.local/bin regardless of scope - only manPath resolution differs. When local is
// true, manifest.Get walks up from the cwd looking for a project bag.toml, falling
// back to (and auto-creating) the global one if none is found; that fallback is what
// the caller's `global` return value flags.
func workSpace(local bool) (WorkSpace, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return WorkSpace{}, fmt.Errorf("checking home dir: %w", err)
	}
	binDir := filepath.Join(homeDir, ".local/bin")
	manPath, _, err := manifest.Get(local)
	if err != nil {
		return WorkSpace{}, err
	}
	return WorkSpace{
		binDir:  binDir,
		manPath: manPath,
	}, nil
}

func IsKnown(parser *kong.Kong, command string) bool {
	for _, knownCmd := range parser.Model.Children {
		if command == knownCmd.Name {
			return true
		}
	}
	return false
}
