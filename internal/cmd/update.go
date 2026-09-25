package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/fatih/color"

	"github.com/jamesjohnsdev/bag/internal/httpclient"
	"github.com/jamesjohnsdev/bag/internal/manifest"
	"github.com/jamesjohnsdev/bag/internal/output"
	"github.com/jamesjohnsdev/bag/internal/provider"
	"github.com/jamesjohnsdev/bag/internal/store"
)

type UpdateCmd struct {
	Name string `arg:"" help:"Name of the binary to view"`
}

func (cmd *UpdateCmd) Run(ctx context.Context) error {
	ws, err := workSpace(false)
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}
	return updateBinary(ctx, ws.manPath, ws.binDir, cmd.Name)
}

// updateBinary resolves the latest version for a known manifest entry and installs it,
// shared by UpdateCmd and its project-scoped tool equivalent.
// initial implementation of update will only handle known sources
func updateBinary(ctx context.Context, manPath, binDir, name string) error {
	var (
		version  string // for future use
		isScript bool
	)

	man, err := manifest.Parse(manPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}
	entry, ok := man.Binaries[name]
	if !ok {
		return fmt.Errorf("unable to find %s in manifest", name)
	}
	oldVersion := entry.Active
	if entry.Type == manifest.ScriptType {
		isScript = true
	}

	src, err := url.Parse(entry.Versions[entry.Active].Source)
	if err != nil {
		return fmt.Errorf("parsing stored 'source': %w", err)
	}
	// TODO: direct-URL sources (including scripts) have no resolvable version to
	// compare against, so update can't tell whether content changed without
	// fetching and hashing it - and even then, a changed hash needs a version
	// to record, which may mean prompting the user to bump it manually.
	if provider.DirectURL(*src) {
		return fmt.Errorf("don't currently support direct urls")
	}
	// stored source may still carry a pinned "@version" tag from before sources were
	// persisted clean (or from a manually edited manifest); strip it so update always
	// resolves against the latest release
	*src = provider.StripVersionTag(*src)
	client := httpclient.New()

	prov, err := provider.Dispatch(src.String(), client, isScript)
	if err != nil {
		return fmt.Errorf("dispatching: %w", err)
	}

	output.Statusf("resolving %s...", name)
	resolution, err := prov.Resolve(ctx, *src, name, version)
	if err != nil {
		return fmt.Errorf("resolving: %w", err)
	}
	defer func() {
		if resolution.Cleanup != nil {
			cerr := resolution.Cleanup()
			if cerr != nil {
				err = errors.Join(err, fmt.Errorf("resolution cleanup: %w", cerr))
			}
		}
	}()
	if resolution.BinaryName == "" ||
		resolution.BinaryName == "." ||
		resolution.BinaryName == ".." ||
		filepath.Base(resolution.BinaryName) != resolution.BinaryName {
		return fmt.Errorf("invalid binary name: %q", resolution.BinaryName)
	}
	if resolution.BinaryName != name {
		return fmt.Errorf("resolved binary name %q does not match %q", resolution.BinaryName, name)
	}
	if resolution.ResolvedVersion == oldVersion {
		if resolution.Reader != nil {
			_ = resolution.Reader.Close()
		}
		fmt.Printf("%s is already up to date (%s)\n", color.GreenString(name), oldVersion)
		return nil
	}
	version = resolution.ResolvedVersion
	var hash string
	if resolution.Dir != "" {
		hash, err = store.InstallFromDir(name, version, src.String(), resolution.Dir, resolution.BinaryRelPath)
		if err != nil {
			return fmt.Errorf("installing from dir: %w", err)
		}
	} else {
		hash, err = store.InstallFromReader(name, version, src.String(), resolution.Reader)
		if err != nil {
			return fmt.Errorf("installing from reader: %w", err)
		}
	}

	if err := store.Unlink(name, binDir); err != nil {
		return fmt.Errorf("removing old symlink: %w", err)
	}

	if err := store.LinkToPath(name, resolution.ResolvedVersion, binDir); err != nil {
		return fmt.Errorf("installing: %w", err)
	}

	// manifest updates
	entry.Versions[resolution.ResolvedVersion] = manifest.VersionEntry{
		Source: src.String(),
	}
	entry.Active = resolution.ResolvedVersion
	// manifest + lockfile changes
	if err := postInstall(manPath, name, version, hash, entry); err != nil {
		_ = store.Unlink(name, binDir)
		_ = store.LinkToPath(name, oldVersion, binDir)
		return fmt.Errorf("post-install: %w", err)
	}
	fmt.Printf("successfully updated %s\n", color.GreenString(name))
	return nil
}
