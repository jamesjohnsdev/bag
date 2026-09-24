package cmd

import (
	"context"

	"github.com/alecthomas/kong"
)

type command interface {
	Run(ctx context.Context) error
}

type CLI struct {
	Version kong.VersionFlag `help:"Print version and exit"`

	ManInstall ManInstallCmd `cmd:"" name:"man-install" help:"Install man page for local use"`

	Init     InitCmd     `cmd:"" help:"Initialise a bag"`
	Add      AddCmd      `cmd:"" help:"Add a binary"`
	Update   UpdateCmd   `cmd:"" help:"Update a binary"`
	Use      UseCmd      `cmd:"" help:"Select a specific version to use"`
	List     ListCmd     `cmd:"" help:"List all stored versions of a binary"`
	Manifest ManifestCmd `cmd:"" help:"Interact with the manifest"`
	Lock     LockCmd     `cmd:"" help:"Interact with the lock file"`
	Remove   RemoveCmd   `cmd:"" help:"Remove an installed binary"`
	View     ViewCmd     `cmd:"" help:"View details of an installed binary"`
	Which    WhichCmd    `cmd:"" help:"Find the stored path of a binary"`
	Tool     ToolCmd     `cmd:"" help:"Manage project tools"`
}

// TODO: Consider changing definitions for commented out ones
var (
	// _ command = (*ManInstallCmd)(nil)
	_ command = (*InitCmd)(nil)
	_ command = (*AddCmd)(nil)
	_ command = (*UpdateCmd)(nil)
	_ command = (*ListCmd)(nil)
	_ command = (*RemoveCmd)(nil)
	_ command = (*ViewCmd)(nil)
	_ command = (*WhichCmd)(nil)
	_ command = (*ToolCmd)(nil)
	_ command = (*UseCmd)(nil)

	_ command = (*ManifestCheckCmd)(nil)
	_ command = (*ManifestListCmd)(nil)
	_ command = (*ManifestViewCmd)(nil)
	_ command = (*ManifestToolCheckCmd)(nil)
	_ command = (*ManifestToolListCmd)(nil)
	_ command = (*ManifestToolViewCmd)(nil)

	_ command = (*LockCheckCmd)(nil)
	_ command = (*LockViewCmd)(nil)
	_ command = (*LockToolCheckCmd)(nil)
	_ command = (*LockToolViewCmd)(nil)
)
