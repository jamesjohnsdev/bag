package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jamesjohnsdev/bag/internal/manifest"
)

type WorkSpace struct {
	binDir  string
	manPath string
}

func workSpace() (WorkSpace, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return WorkSpace{}, fmt.Errorf("checking home dir: %w", err)
	}
	binDir := filepath.Join(homeDir, ".local/bin")
	manPath, _, err := manifest.Get(false)
	if err != nil {
		return WorkSpace{}, err
	}
	return WorkSpace{
		binDir:  binDir,
		manPath: manPath,
	}, nil
}
