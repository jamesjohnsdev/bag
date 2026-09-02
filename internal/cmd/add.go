package cmd

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/jamesjohnsdev/bag/internal/httpclient"
	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/provider"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type AddCmd struct {
	Source  string `arg:"" help:"Path or remote source"`
	Local   bool   `flag:"" help:"Install a local binary"`
	Name    string `flag:"" help:"Override the name of the binary"`
	Version string `flag:"" help:"If direct link or local, set the version, otherwise choose version to download"`
}

func (cmd *AddCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	var (
		binName     string
		version     string
		binaryEntry manifest.BinaryEntry
		hash        string
	)

	// download binary
	if cmd.Local {
		version = "local"
		binName = filepath.Base(cmd.Source)
		if cmd.Version != "" {
			version = cmd.Version
		}
		if cmd.Name != "" {
			binName = cmd.Name
		}
		hash, err = store.InstallLocal(binName, version, cmd.Source)
		if err != nil {
			return fmt.Errorf("installing locally: %w", err)
		}

	} else {
		client := httpclient.New()
		provider, err := provider.Dispatch(cmd.Source, client)
		if err != nil {
			return fmt.Errorf("dispatching provider: %w", err)
		}
		src, err := url.Parse(cmd.Source)
		if err != nil {
			return fmt.Errorf("parsing source: %w", err)
		}
		if cmd.Version != "" {
			version = cmd.Version
		}
		if cmd.Name != "" {
			binName = cmd.Name
		}
		resolution, err := provider.Resolve(ctx, *src, binName, version)
		if err != nil {
			return fmt.Errorf("resolving: %w", err)
		}
		if resolution.BinaryName == "" ||
			resolution.BinaryName == "." ||
			resolution.BinaryName == ".." ||
			filepath.Base(resolution.BinaryName) != resolution.BinaryName {
			return fmt.Errorf("invalid binary name: %q", resolution.BinaryName)
		}
		binName, version = resolution.BinaryName, resolution.ResolvedVersion
		hash, err = store.InstallFromReader(binName, version, cmd.Source, resolution.Reader)
		if err != nil {
			return fmt.Errorf("installing from reader: %w", err)
		}
	}
	// TODO: consider moving this binaryEntry and LinkToPath to within postInstall
	binaryEntry = manifest.BinaryEntry{
		Type:   "binary",
		Active: version,
		Versions: map[string]manifest.VersionEntry{
			version: {Source: cmd.Source},
		},
	}
	if err := store.LinkToPath(binName, version, ws.binDir); err != nil {
		return fmt.Errorf("installing: %w", err)
	}

	err = postInstall(ws.manPath, binName, version, hash, binaryEntry)
	if err != nil {
		return fmt.Errorf("post-install: %w", err)
	}
	fmt.Printf("installed %s successfully\n", binName)
	return nil
}

// postInstall handles general chores required for manifest and lock after binary installation
func postInstall(manPath, binName, version, hash string, binaryEntry manifest.BinaryEntry) error {
	if err := manifest.AddBinary(manPath, binName, binaryEntry); err != nil {
		return fmt.Errorf("adding manifest entry: %w", err)
	}
	lockPath := manifest.FindLock(manPath)
	lockEntry := manifest.LockEntry{
		Hash: hash,
	}
	if err := manifest.AddLockEntry(lockPath, binName, version, lockEntry); err != nil {
		_ = manifest.RemoveBinary(manPath, binName)
		return fmt.Errorf("adding lockfile entry: %w", err)
	}
	return nil
}
