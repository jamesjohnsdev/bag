package cmd

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/jamesjohnsdev/bag/internal/httpclient"
	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/provider"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type UpdateCmd struct {
	Name string `arg:"" help:"Name of the binary to view"`
}

// initial implementation of update will only handle known sources
func (cmd *UpdateCmd) Run(ctx context.Context) error {
	ws, err := workSpace()
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	// for future use
	var version string

	man, err := manifest.Parse(ws.manPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}
	entry, ok := man.Binaries[cmd.Name]
	if !ok {
		return fmt.Errorf("unable to find %s in manifest", cmd.Name)
	}
	oldVersion := entry.Active

	src, err := url.Parse(entry.Versions[entry.Active].Source)
	if err != nil {
		return fmt.Errorf("parsing stored 'source': %w", err)
	}
	if provider.DirectURL(*src) {
		return fmt.Errorf("don't currently support direct urls")
	}
	client := httpclient.New()

	prov, err := provider.Dispatch(src.String(), client)
	if err != nil {
		return fmt.Errorf("dispatching: %w", err)
	}

	resolution, err := prov.Resolve(ctx, *src, cmd.Name, version)
	if err != nil {
		return fmt.Errorf("resolving: %w", err)
	}
	if resolution.BinaryName == "" ||
		resolution.BinaryName == "." ||
		resolution.BinaryName == ".." ||
		filepath.Base(resolution.BinaryName) != resolution.BinaryName {
		return fmt.Errorf("invalid binary name: %q", resolution.BinaryName)
	}
	if resolution.BinaryName != cmd.Name {
		return fmt.Errorf("resolved binary name %q does not match %q", resolution.BinaryName, cmd.Name)
	}
	version = resolution.ResolvedVersion
	hash, err := store.InstallFromReader(cmd.Name, version, src.String(), resolution.Reader)
	if err != nil {
		return fmt.Errorf("installing from reader: %w", err)
	}

	if err := store.Unlink(cmd.Name, ws.binDir); err != nil {
		return fmt.Errorf("removing old symlink: %w", err)
	}

	if err := store.LinkToPath(cmd.Name, resolution.ResolvedVersion, ws.binDir); err != nil {
		return fmt.Errorf("installing: %w", err)
	}

	// manifest updates
	entry.Versions[resolution.ResolvedVersion] = manifest.VersionEntry{
		Source: src.String(),
	}
	entry.Active = resolution.ResolvedVersion
	// manifest + lockfile changes
	if err := postInstall(ws.manPath, cmd.Name, version, hash, entry); err != nil {
		_ = store.Unlink(cmd.Name, ws.binDir)
		_ = store.LinkToPath(cmd.Name, oldVersion, ws.binDir)
		return fmt.Errorf("post-install: %w", err)
	}
	fmt.Printf("successfully updated %s\n", color.GreenString(cmd.Name))
	return nil
}
